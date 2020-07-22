package template

import (
	"fmt"
	"testing"

	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/kotskinds/multitype"
	"github.com/stretchr/testify/require"
	"go.undefinedlabs.com/scopeagent"
)

type depGraphTestCase struct {
	dependencies   map[string][]string
	testCerts      map[string][]string
	testKeys       map[string][]string
	resolveOrder   []string
	expectError    bool   //expect an error fetching head nodes
	expectNotFound string //expect this dependency not to be part of the head nodes

	name string
}

func TestDepGraph(t *testing.T) {
	tests := []depGraphTestCase{
		{
			dependencies: map[string][]string{
				"alpha":   {},
				"bravo":   {"alpha"},
				"charlie": {"bravo"},
				"delta":   {"alpha", "charlie"},
				"echo":    {},
			},
			resolveOrder: []string{"alpha", "bravo", "charlie", "delta", "echo"},
			name:         "basic_dependency_chain",
		},
		{
			dependencies: map[string][]string{
				"alpha": {"bravo"},
				"bravo": {"alpha"},
			},
			resolveOrder: []string{"alpha", "bravo"},
			expectError:  true,
			name:         "basic_circle",
		},
		{
			dependencies: map[string][]string{
				"alpha":   {},
				"bravo":   {"alpha"},
				"charlie": {"alpha"},
				"delta":   {"bravo", "charlie"},
				"echo":    {"delta"},
			},
			resolveOrder: []string{"alpha", "bravo", "charlie", "delta", "echo"},
			name:         "basic_forked_chain",
		},
		{
			dependencies: map[string][]string{
				"alpha":   {},
				"bravo":   {"alpha"},
				"charlie": {"alpha"},
				"delta":   {"bravo", "charlie", "foxtrot"},
				"echo":    {"delta"},
				"foxtrot": {},
			},
			resolveOrder:   []string{"alpha", "bravo", "charlie", "delta", "echo", "foxtrot"},
			expectNotFound: "delta",
			name:           "unresolved_dependency",
		},
		{
			dependencies: map[string][]string{
				"alpha":   {},
				"bravo":   {},
				"charlie": {"alpha"},
				"delta":   {"bravo"},
				"echo":    {"delta"},
			},
			resolveOrder: []string{"alpha", "bravo", "charlie", "delta", "echo"},
			name:         "two_chains",
		},
		{
			dependencies: map[string][]string{
				"alpha":   {},
				"bravo":   {"alpha"},
				"charlie": {"alpha", "bravo"},
				"delta":   {"alpha", "bravo", "charlie"},
				"echo":    {"alpha", "bravo", "charlie", "delta"},
				"foxtrot": {"alpha", "bravo", "charlie", "delta", "echo"},
				"golf":    {"alpha", "bravo", "charlie", "delta", "echo", "foxtrot"},
				"hotel":   {"alpha", "bravo", "charlie", "delta", "echo", "foxtrot", "golf"},
				"india":   {"alpha", "bravo", "charlie", "delta", "echo", "foxtrot", "golf", "hotel"},
				"juliet":  {"alpha", "bravo", "charlie", "delta", "echo", "foxtrot", "golf", "hotel", "india"},
				"kilo":    {"alpha", "bravo", "charlie", "delta", "echo", "foxtrot", "golf", "hotel", "india", "juliet"},
				"lima":    {"alpha", "bravo", "charlie", "delta", "echo", "foxtrot", "golf", "hotel", "india", "juliet", "kilo"},
				"mike":    {"alpha", "bravo", "charlie", "delta", "echo", "foxtrot", "golf", "hotel", "india", "juliet", "kilo", "lima"},
			},
			resolveOrder: []string{"alpha", "bravo", "charlie", "delta", "echo", "foxtrot", "golf", "hotel", "india", "juliet", "kilo", "lima", "mike"},
			name:         "pyramid",
		},
		{
			dependencies: map[string][]string{
				"alpha": {},
				"bravo": {},
			},
			resolveOrder: []string{"alpha", "bravo"},
			expectError:  false,
			name:         "not referenced", // items should eventually be resolved even if nothing depends on them
		},
		{
			dependencies: map[string][]string{
				"alpha": {},
				"bravo": {"alpha", "charlie"},
			},
			resolveOrder: []string{"alpha", "bravo"},
			expectError:  true,
			name:         "does_not_exist",
		},
		{
			dependencies: map[string][]string{
				"alpha":   {},
				"bravo":   {"alpha"},
				"charlie": {"bravo"},
				"delta":   {"alpha", "charlie"},
				"echo":    {},
			},
			testCerts: map[string][]string{
				"echo": {"certA"},
			},
			testKeys: map[string][]string{
				"delta": {"certA"},
			},
			resolveOrder: []string{"alpha", "bravo", "charlie", "echo", "delta"},
			name:         "basic_certs",
		},
		{
			dependencies: map[string][]string{
				"alpha":   {},
				"bravo":   {"alpha"},
				"charlie": {"bravo"},
				"delta":   {"alpha", "charlie"},
				"echo":    {},
			},
			testCerts: map[string][]string{
				"echo": {"certA"},
			},
			testKeys: map[string][]string{
				"delta": {"certA"},
			},
			resolveOrder:   []string{"alpha", "bravo", "charlie", "delta", "echo"},
			name:           "basic_certs_original_order",
			expectNotFound: "delta",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			scopetest := scopeagent.StartTest(t)
			defer scopetest.End()

			graph := depGraph{}
			for source, deps := range test.dependencies {
				graph.AddNode(source)
				for _, dep := range deps {
					graph.AddDep(source, dep)
				}
			}
			for source, certNames := range test.testCerts {
				for _, certName := range certNames {
					graph.AddCert(source, certName)
				}
			}
			for source, keys := range test.testKeys {
				for _, key := range keys {
					graph.AddKey(source, key)
				}
			}
			graph.resolveCertKeys()

			runGraphTests(t, test, graph)
		})

		t.Run(test.name+"+parse", func(t *testing.T) {
			scopetest := scopeagent.StartTest(t)
			defer scopetest.End()

			graph := depGraph{}

			groups := buildTestConfigGroups(test.dependencies, test.testCerts, test.testKeys, "templateStringStart", "templateStringEnd", true)

			err := graph.ParseConfigGroup(groups)
			require.NoError(t, err)

			runGraphTests(t, test, graph)
		})
	}
}

// this makes sure that we test with each of the configOption types, in both Value and Default
func buildTestConfigGroups(dependencies, certs, keys map[string][]string, prefix string, suffix string, rotate bool) []kotsv1beta1.ConfigGroup {
	group := kotsv1beta1.ConfigGroup{}
	group.Items = make([]kotsv1beta1.ConfigItem, 0)
	counter := 0

	templateFuncs := []string{
		"{{repl ConfigOption \"%s\" }}",
		"{{repl ConfigOptionIndex \"%s\" }}",
		"{{repl ConfigOptionData \"%s\" }}",
		"repl{{ ConfigOptionEquals \"%s\" \"abc\" }}",
		"{{repl ConfigOptionNotEquals \"%s\" \"xyz\" }}{{repl DoesNotExistFunc }}",
	}

	if !rotate {
		//use only ConfigOption, not all 5
		templateFuncs = []string{
			"{{repl ConfigOption \"%s\" }}",
		}
	}

	totalDepItems := 0

	for source, deps := range dependencies {
		newItem := kotsv1beta1.ConfigItem{Type: "text", Name: source}
		depString := prefix
		for _, dep := range deps {
			depString += fmt.Sprintf(templateFuncs[totalDepItems%len(templateFuncs)], dep)
			totalDepItems++
		}

		if certNames, ok := certs[source]; ok {
			for _, certName := range certNames {
				depString += fmt.Sprintf("{{repl TLSCert \"%s\" }}", certName)
			}
		}

		if certNames, ok := keys[source]; ok {
			for _, certName := range certNames {
				depString += fmt.Sprintf("{{repl TLSKey \"%s\" }}", certName)
			}
		}

		depString += suffix

		if counter%2 == 0 {
			newItem.Value.StrVal = depString
			newItem.Value.Type = multitype.String
		} else {
			newItem.Default.StrVal = depString
			newItem.Default.Type = multitype.String
		}
		counter++

		group.Items = append(group.Items, newItem)
	}

	return []kotsv1beta1.ConfigGroup{group}
}

func runGraphTests(t *testing.T, test depGraphTestCase, graph depGraph) {
	depLen := len(graph.Dependencies)
	graphCopy, err := graph.Copy()
	require.NoError(t, err)

	for _, toResolve := range test.resolveOrder {
		available, err := graph.GetHeadNodes()
		if err != nil && test.expectError {
			// fmt.Printf("err: %s\n", err.Error())
			return
		}

		require.NoError(t, err, "toResolve: %s", toResolve)

		if test.expectNotFound != "" && toResolve == test.expectNotFound {
			require.NotContains(t, available, toResolve)
			return
		}

		require.Contains(t, available, toResolve)

		graph.ResolveDep(toResolve)
	}

	available, err := graph.GetHeadNodes()
	require.NoError(t, err)
	require.Empty(t, available)

	require.False(t, test.expectError, "Did not find expected error")

	require.Equal(t, depLen, len(graphCopy.Dependencies))
}
