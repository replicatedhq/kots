package appstate

import "strings"

var (
	resourceKindNames [][]string
)

func registerResourceKindNames(names ...string) {
	resourceKindNames = append(resourceKindNames, names)
}

func getResourceKindCommonName(a string) string {
	for _, names := range resourceKindNames {
		for _, name := range names {
			if name == strings.ToLower(a) {
				return names[0]
			}
		}
	}
	return a
}
