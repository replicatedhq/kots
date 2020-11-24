package rbac

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type testLogger struct{}

func (l testLogger) Debug(msg string, args ...interface{}) {}

func Test_resourceMatchesString(t *testing.T) {
	log := testLogger{}

	match := resourceMatchesString(log, "platform/app/create", "platform/app/create")
	assert.True(t, match, "platform/app/create should match platform/app/create")
	match = resourceMatchesString(log, "platform/app/create", "platform/app/ccreate")
	assert.False(t, match, "platform/app/create should not match platform/app/ccreate")

	match = resourceMatchesString(log, "platform/app/*/delete", "platform/app/abcdef/delete")
	assert.True(t, match, "platform/app/*/delete should match platform/app/absdef/delete")
	match = resourceMatchesString(log, "platform/app/*/delete", "platform/app/abcdef/123/delete")
	assert.False(t, match, "platform/app/*/delete should not match platform/app/absdef/123/delete")

	match = resourceMatchesString(log, "**/read", "platform/app/abcdef/delete")
	assert.False(t, match, "**/read should not match platform/app/abcdef/delete")

	match = resourceMatchesString(log, "**/read", "platform/app/abcdef/read")
	assert.True(t, match, "**/read should match platform/app/abcdef/read")

	match = resourceMatchesString(log, "kots/app/**/read", "kots/app/theappid/channel/thechannelid/promote")
	assert.False(t, match, "kots/app/**/read should not match kots/app/theappid/channel/thechannelid/promote")
	match = resourceMatchesString(log, "kots/app/**/list", "kots/app/theappid/channel/thechannelid/promote")
	assert.False(t, match, "kots/app/**/list should not match kots/app/theappid/channel/thechannelid/promote")
	match = resourceMatchesString(log, "kots/app/**/promote", "kots/app/theappid/channel/thechannelid/promote")
	assert.True(t, match, "kots/app/**/promote should match kots/app/theappid/channel/thechannelid/promote")
}

func Test_isPatternMoreSpecific(t *testing.T) {
	yes := isPatternMoreSpecific("**/*", "platform/app/create")
	assert.True(t, yes, "platform/app/create is more specific than **/*")

	no := isPatternMoreSpecific("platform/app/create", "**/*")
	assert.False(t, no, "**/* is not more specific than platform/app/create")

	yes = isPatternMoreSpecific("platform/**/create", "platform/app/*/create")
	assert.False(t, no, "platform/app/*/create is more specific than platform/**/create")

	yes = isPatternMoreSpecific("**", "platform/app/*/license/**")
	assert.True(t, yes, "platform/app/*/license/** is more specific than **")

	no = isPatternMoreSpecific("platform/app/*/license/**", "**")
	assert.False(t, no, "** is not more specific than platform/app/*/license/**")
}

func Test_simplifyPattern(t *testing.T) {
	type pattern struct {
		original   string
		simplified string
	}
	patterns := []pattern{
		{
			original:   "**/*",
			simplified: "**",
		},
		{
			original:   "**",
			simplified: "**",
		},
		{
			original:   "platform/app/**/*/delete",
			simplified: "platform/app/**/delete",
		},
		{
			original:   "platform/app/*/delete",
			simplified: "platform/app/*/delete",
		},
	}
	for _, p := range patterns {
		assert.Equal(t, simplifyPattern(p.original), p.simplified)
	}
}
