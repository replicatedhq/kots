package rbac

import (
	"strings"

	"github.com/replicatedhq/kots/pkg/rbac/types"
)

type Logger interface {
	Debug(msg string, args ...interface{})
}

func CheckAccess(log Logger, policies []types.Policy, resource string) bool {
	// Multiple rsources can match, and the most specific will apply
	var highestAllowedPolicy string
	var highestDeniedPolicy string

	for _, policy := range policies {
		for _, allowedPolicy := range policy.Allowed {
			allowedPolicy = simplifyPattern(allowedPolicy)
			if resourceMatchesString(log, allowedPolicy, resource) {
				if isPatternMoreSpecific(highestAllowedPolicy, allowedPolicy) {
					highestAllowedPolicy = allowedPolicy
				}
			}
		}
	}

	for _, policy := range policies {
		for _, deniedPolicy := range policy.Denied {
			deniedPolicy = simplifyPattern(deniedPolicy)
			if resourceMatchesString(log, deniedPolicy, resource) {
				if isPatternMoreSpecific(highestDeniedPolicy, deniedPolicy) {
					highestDeniedPolicy = deniedPolicy
				}
			}
		}
	}

	// if the policies match, deny unless it's default
	if highestDeniedPolicy == "" && highestAllowedPolicy != "" {
		log.Debug("Allowing access because there is an allowed policy, and no denied policy")
		return true
	}

	if highestAllowedPolicy == highestDeniedPolicy {
		if highestDeniedPolicy == "" {
			log.Debug("Allowing and denied are equal, but denied is empty, denying access")
			return false
		}

		log.Debug("Denying access on conflicting access")
		return false
	}

	// note, this allows no allowed policies to default to deny all
	if isPatternMoreSpecific(highestAllowedPolicy, highestDeniedPolicy) {
		log.Debug("RBAC Policy Access denied from rule %q", highestDeniedPolicy)
		return false
	}

	log.Debug("RBAC Policy Access allowed from rule %q", highestAllowedPolicy)
	return true
}

// pattern = troubleshoot/**/read
// resource = app/123/troubleshoot/support-bundle/abc/read
func resourceMatchesString(log Logger, pattern string, resource string) bool {
	log.Debug("Checking if %q allows %q", pattern, resource)

	// Shortcut for optimization
	if pattern == resource || pattern == "**/*" || pattern == "**" {
		log.Debug("Allowing trivial case")
		return true
	}

	patternParts := strings.Split(pattern, "/")
	patternPartIdx := 0
	resourceParts := strings.Split(resource, "/")
	resourcePartIdx := 0

	// for (let resourcePart of resourceParts) {
	for patternPartIdx < len(patternParts) && resourcePartIdx < len(resourceParts) {
		patternPart := patternParts[patternPartIdx]
		resourcePart := resourceParts[resourcePartIdx]

		if patternPart == "**" {
			// pattern ending with "**" matches the tail of everything
			if patternPartIdx+1 == len(patternParts) {
				log.Debug("Pattern ends in \"**\", allowing a match")
				return true
			}

			// Advance resource index until it's completely consumed or we hit a matching part that follows "**"
			nextPatternPart := patternParts[patternPartIdx+1]
			if nextPatternPart == resourcePart {
				log.Debug("advancing past \"**/%s\" because it matches resource part", nextPatternPart)
				patternPartIdx += 2
				resourcePartIdx++
			} else {
				resourcePartIdx++
			}
		} else if patternPart == "*" || patternPart == resourcePart {
			patternPartIdx++
			resourcePartIdx++
		} else {
			log.Debug("denying because %q does not match %q", patternPart, resourcePart)
			return false
		}
	}

	if patternPartIdx == len(patternParts) && resourcePartIdx == len(resourceParts) {
		log.Debug("Allowing match because pattern and resource were both consumed")
		return true
	}

	log.Debug("denying, consumed %q pattern parts and %q resource parts", patternPartIdx, resourcePartIdx)

	return false
}

func simplifyPattern(pattern string) string {
	// Turn "**/*" into "**"

	patternParts := strings.Split(pattern, "/")

	newParts := []string{patternParts[0]}

	patternPartIdx := 1
	newPatternPartIdx := 1
	for patternPartIdx < len(patternParts) {
		patternPart := patternParts[patternPartIdx]
		if newParts[newPatternPartIdx-1] != "**" || patternPart != "*" {
			newParts = append(newParts, patternPart)
			newPatternPartIdx++
		}
		patternPartIdx++
	}

	return strings.Join(newParts, "/")
}

func isPatternMoreSpecific(previousPattern string, newPattern string) bool {
	// Note that this is used to decide between "allowed" and "denied" policies
	// where previousPattern == allowed and newPattern == denied,
	// so the precedence of base cases is important here.

	// this is the default state
	if previousPattern == "" {
		return true
	}

	// Anything is more specific than "everything"
	if previousPattern == "**" {
		return true
	}

	// "everything" is not more specific than anything
	if newPattern == "**" {
		return false
	}

	previousPatternParts := strings.Split(previousPattern, "/")
	newPatternParts := strings.Split(newPattern, "/")

	previousPatternHasGlob := stringInSlice("**", previousPatternParts)
	previousPatternWildcardCount := instancesOfStringInSlice("*", previousPatternParts)

	newPatternHasGlob := stringInSlice("**", newPatternParts)
	newPatternWildcardCount := instancesOfStringInSlice("*", newPatternParts)

	if previousPatternHasGlob && !newPatternHasGlob {
		// previousPattern = "test/**/read"
		// newPattern = test/something/read
		return true
	}

	if previousPatternWildcardCount > newPatternWildcardCount {
		// previousPattern = "test/*/something/*/read"
		// newPattern = "test/*/something"
		return true
	}

	return len(newPatternParts) > len(previousPatternParts)
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func instancesOfStringInSlice(a string, list []string) int {
	count := 0
	for _, b := range list {
		if b == a {
			count++
		}
	}

	return count
}
