# LicenseWrapper Nil Check Fixes - Implementation Plan

**Date:** 2025-11-03
**Status:** Planning
**Related Research:** [2025-11-03-licensewrapper-pointer-nil-checks.md](../research/2025-11-03-licensewrapper-pointer-nil-checks.md)

## Executive Summary

After converting `LicenseWrapper` from value type to pointer type, approximately 60 locations in the codebase now require nil checks to prevent panics. This plan outlines a comprehensive, phased approach to add proper nil safety while maintaining code quality and test coverage.

**Scope:** Fix ALL ~60 identified locations (critical, high-risk, and medium-risk)
**Estimated Effort:** 4-6 hours
**Risk Level:** Low (defensive changes with comprehensive testing)

## Problem Statement

When `LicenseWrapper` was a value type, it was safe to call methods on it even if the underlying V1/V2 fields were nil. After converting to pointer type:

1. **Nil pointer panics** - Calling `license.IsV1()` on a nil pointer panics before reaching the method
2. **Direct field access bugs** - Code accessing `license.V1` without checking if it's a V1 license
3. **Unprotected Get* calls** - Methods like `GetAppSlug()` called without nil checks

Three risk levels identified:
- **Critical (4 locations):** Direct field access bugs causing wrong data access
- **High (30 locations):** Function entry points missing nil validation
- **Medium (25+ locations):** Version checks and Get* calls without nil guards

## Good Patterns to Follow

Based on recent fixes in the pointer conversion, these are the recommended patterns:

### Pattern 1: Function Entry Validation (pkg/license/signature.go:52)

```go
func VerifyLicenseWrapper(wrapper *licensewrapper.LicenseWrapper) (*licensewrapper.LicenseWrapper, error) {
    if wrapper == nil || (!wrapper.IsV1() && !wrapper.IsV2()) {
        return nil, errors.New("license wrapper contains no license")
    }
    // Safe to use wrapper methods
    endpoint := wrapper.GetEndpoint()
}
```

**Use when:** Functions receive license as parameter
**Benefits:** Fails fast, clear error message, protects entire function body

### Pattern 2: Early Return with Default (pkg/docker/registry/registry.go:149)

```go
func getRegistryProxyInfoFromLicense(license *licensewrapper.LicenseWrapper) *RegistryProxyInfo {
    defaultInfo := getDefaultRegistryProxyInfo()

    if license == nil || (!license.IsV1() && !license.IsV2()) {
        return defaultInfo
    }

    endpoint := license.GetEndpoint()
    // ... continue with license processing
}
```

**Use when:** Function can return sensible default without license
**Benefits:** Graceful degradation, no error propagation needed

### Pattern 3: Inline Check (pkg/template/config_context.go:336)

```go
licenseAppSlug := ""
if ctx.license != nil && (ctx.license.IsV1() || ctx.license.IsV2()) {
    licenseAppSlug = ctx.license.GetAppSlug()
}
```

**Use when:** License is optional in specific context
**Benefits:** Minimal code change, clear intent, safe default

### Pattern 4: Version-Specific Field Access

```go
// WRONG - Always accesses V1 field
if license.IsV1() || license.IsV2() {
    data := license.V1  // ❌ Panic if V2!
}

// RIGHT - Check version before accessing field
if license != nil && license.IsV1() {
    data := license.V1
} else if license != nil && license.IsV2() {
    data := license.V2
}
```

## Implementation Phases

### Phase 1: Critical Direct Field Access Bugs (4 locations)

**Priority:** Immediate
**Risk:** High - These cause data corruption or panics with V2 licenses
**Estimated Time:** 30 minutes

#### 1.1 pkg/supportbundle/spec.go:744

**Current Code:**
```go
if license.IsV1() || license.IsV2() {
    s := serializer.NewSerializerWithOptions(...)
    var b bytes.Buffer
    if err := s.Encode(license.V1, &b); err != nil {  // ❌ Always V1!
        logger.Errorf("Failed to marshal license: %v", err)
    } else {
        collectors = append(collectors, &troubleshootv1beta2.Collect{...})
    }
}
```

**Fix:**
```go
if license != nil && (license.IsV1() || license.IsV2()) {
    s := serializer.NewSerializerWithOptions(...)
    var b bytes.Buffer
    var encodeErr error

    if license.IsV1() {
        encodeErr = s.Encode(license.V1, &b)
    } else {
        encodeErr = s.Encode(license.V2, &b)
    }

    if encodeErr != nil {
        logger.Errorf("Failed to marshal license: %v", encodeErr)
    } else {
        collectors = append(collectors, &troubleshootv1beta2.Collect{
            Data: &troubleshootv1beta2.Data{
                CollectorMeta: troubleshootv1beta2.CollectorMeta{
                    CollectorName: "license.yaml",
                },
                Name: "kots/admin_console",
                Data: b.String(),
            },
        })
    }
}
```

**Changes:**
- Add nil check on license pointer
- Conditionally encode V1 or V2 based on version
- Support both license versions in support bundles

#### 1.2 pkg/reporting/preflight.go:84

**Current Code:**
```go
go func() {
    // ...
    if err := GetReporter().SubmitPreflightData(license.V1, appID, clusterID, ...); err != nil {
        logger.Debugf("failed to submit preflight data: %v", err)
        return
    }
}()
```

**Fix:**
```go
go func() {
    // Skip reporting if no license available
    if license == nil || (!license.IsV1() && !license.IsV2()) {
        logger.Debugf("skipping preflight data submission: no license available")
        return
    }

    var v1License *kotsv1beta1.License
    if license.IsV1() {
        v1License = license.V1
    } else {
        // TODO(Phase 4): Update reporting API to accept V2 licenses
        // For now, V2 licenses cannot be reported with current API
        logger.Debugf("skipping preflight data submission: V2 license not yet supported")
        return
    }

    if err := GetReporter().SubmitPreflightData(v1License, appID, clusterID, ...); err != nil {
        logger.Debugf("failed to submit preflight data: %v", err)
        return
    }
}()
```

**Changes:**
- Add nil check at start of goroutine
- Extract V1 license only for V1 licenses
- Document V2 limitation with TODO

#### 1.3 pkg/handlers/preflight.go:369

**Current Code:**
```go
go func() {
    // TODO(Phase 4): Update reporting.SubmitPreflightData to accept LicenseWrapper
    // Temporary workaround: Use .V1 for reporting
    if err := reporting.GetReporter().SubmitPreflightData(license.V1, foundApp.ID, ...); err != nil {
        logger.Debugf("failed to submit preflight data: %v", err)
        return
    }
}()
```

**Fix:**
```go
go func() {
    // Skip reporting if no license available
    if license == nil || (!license.IsV1() && !license.IsV2()) {
        logger.Debugf("skipping preflight data submission: no license available")
        return
    }

    var v1License *kotsv1beta1.License
    if license.IsV1() {
        v1License = license.V1
    } else {
        // TODO(Phase 4): Update reporting.SubmitPreflightData to accept LicenseWrapper
        // V2 licenses cannot be reported with current API
        logger.Debugf("skipping preflight data submission: V2 license not yet supported")
        return
    }

    if err := reporting.GetReporter().SubmitPreflightData(v1License, foundApp.ID, ...); err != nil {
        logger.Debugf("failed to submit preflight data: %v", err)
        return
    }
}()
```

**Changes:**
- Add nil check at start of goroutine
- Only extract V1 for V1 licenses
- Clarify TODO about Phase 4

#### 1.4 pkg/kotsadm/license.go:24,37

**Current Code:**
```go
func getLicenseSecretYAML(deployOptions *types.DeployOptions) (map[string][]byte, error) {
    docs := map[string][]byte{}
    s := json.NewYAMLSerializer(...)

    var b bytes.Buffer
    // Encode the actual license object (V1 or V2), not the wrapper
    if deployOptions.License.IsV1() {  // ❌ Panic if License is nil
        if err := s.Encode(deployOptions.License.V1, &b); err != nil {
            return nil, errors.Wrap(err, "failed to encode v1beta1 license")
        }
    } else if deployOptions.License.IsV2() {
        if err := s.Encode(deployOptions.License.V2, &b); err != nil {
            return nil, errors.Wrap(err, "failed to encode v1beta2 license")
        }
    } else {
        return nil, errors.New("no license to encode")
    }

    var license bytes.Buffer
    if err := s.Encode(kotsadmobjects.LicenseSecret(..., deployOptions.License.GetAppSlug(), ...), &license); err != nil {
        return nil, errors.Wrap(err, "failed to marshal license secret")
    }
    docs["secret-license.yaml"] = license.Bytes()

    return docs, nil
}
```

**Fix:**
```go
func getLicenseSecretYAML(deployOptions *types.DeployOptions) (map[string][]byte, error) {
    if deployOptions.License == nil {
        return nil, errors.New("deploy options license is nil")
    }

    if !deployOptions.License.IsV1() && !deployOptions.License.IsV2() {
        return nil, errors.New("no license to encode")
    }

    docs := map[string][]byte{}
    s := json.NewYAMLSerializer(...)

    var b bytes.Buffer
    // Encode the actual license object (V1 or V2), not the wrapper
    if deployOptions.License.IsV1() {
        if err := s.Encode(deployOptions.License.V1, &b); err != nil {
            return nil, errors.Wrap(err, "failed to encode v1beta1 license")
        }
    } else if deployOptions.License.IsV2() {
        if err := s.Encode(deployOptions.License.V2, &b); err != nil {
            return nil, errors.Wrap(err, "failed to encode v1beta2 license")
        }
    }

    var license bytes.Buffer
    if err := s.Encode(kotsadmobjects.LicenseSecret(..., deployOptions.License.GetAppSlug(), ...), &license); err != nil {
        return nil, errors.Wrap(err, "failed to marshal license secret")
    }
    docs["secret-license.yaml"] = license.Bytes()

    return docs, nil
}
```

**Changes:**
- Add nil check at function entry
- Check for valid license before processing
- GetAppSlug() is now safe to call

**Phase 1 Verification:**
```bash
# Build affected packages
go build ./pkg/supportbundle/...
go build ./pkg/reporting/...
go build ./pkg/handlers/...
go build ./pkg/kotsadm/...

# Run tests
go test ./pkg/supportbundle/...
go test ./pkg/reporting/...
go test ./pkg/handlers/...
go test ./pkg/kotsadm/...
```

---

### Phase 2: High-Risk Function Entry Points (30 locations)

**Priority:** High
**Risk:** Medium - Functions that accept licenses as parameters
**Estimated Time:** 2 hours

These functions should validate license at entry using Pattern 1 or Pattern 2.

#### 2.1 pkg/kotsutil/kots.go - 4 Functions

**Locations:**
- Line 122: `GetLicenseVersion()`
- Line 1732: `FindChannelIDInLicense()`
- Line 1758: `FindChannelInLicense()`
- Line 1787: `GetDefaultChannelIDFromLicense()`

**Current Pattern (GetLicenseVersion as example):**
```go
func (k *KotsKinds) GetLicenseVersion() string {
    if k.License.IsV2() {  // ❌ Panic if k.License is nil
        return "v1beta2"
    }
    return "v1beta1"
}
```

**Fix Pattern:**
```go
func (k *KotsKinds) GetLicenseVersion() string {
    if k.License == nil {
        return ""  // or "v1beta1" as default?
    }
    if k.License.IsV2() {
        return "v1beta2"
    }
    return "v1beta1"
}
```

**Apply to all 4 functions** in pkg/kotsutil/kots.go using appropriate nil handling.

#### 2.2 pkg/upstream/replicated.go - 7 Locations

**File:** pkg/upstream/replicated.go
**Lines:** 210, 287, 477, 490, 497, 509, 524

**Fix Pattern:** Use Pattern 2 (early return) or Pattern 3 (inline check) depending on context.

Example for line 210:
```go
func someFunction(license *licensewrapper.LicenseWrapper) error {
    if license == nil || (!license.IsV1() && !license.IsV2()) {
        return errors.New("invalid license")
    }
    // Safe to use license
}
```

#### 2.3 pkg/replicatedapp/api.go - 2 Functions

**Locations:**
- Line 47: `GetLatestLicense()`
- Line 61: `makeLicenseURL()`

**Fix:** Add nil checks at function entry, return appropriate defaults or errors.

#### 2.4 pkg/handlers/license.go - 1 Function

**Location:** Line 217: `getLicenseEntitlements()`

**Fix Pattern:**
```go
func getLicenseEntitlements(license *licensewrapper.LicenseWrapper) []EntitlementField {
    if license == nil || (!license.IsV1() && !license.IsV2()) {
        return []EntitlementField{}
    }
    // Process entitlements
}
```

#### 2.5 Additional High-Priority Locations

Apply Pattern 1 or Pattern 2 to these locations:

- `pkg/kotsadm/app.go:98` - needsRegistry function
- `pkg/kotsadm/api.go:100` - API initialization
- `pkg/kotsadm/kotsadm.go:144` - getProtectedFields
- `pkg/kotsadm/main.go:485` - version check
- `pkg/kotsadm/configmaps.go:84, 147` - config map generation
- `pkg/kotsadm/objects/objects.go:54` - object creation
- `pkg/identity/identity.go:21, 35, 56` - identity config generation
- `pkg/kurl/kurl.go:48` - Kurl integration

**Phase 2 Verification:**
```bash
# Build all affected packages
go build ./pkg/kotsutil/...
go build ./pkg/upstream/...
go build ./pkg/replicatedapp/...
go build ./pkg/handlers/...
go build ./pkg/kotsadm/...
go build ./pkg/identity/...
go build ./pkg/kurl/...

# Run comprehensive tests
go test ./pkg/kotsutil/...
go test ./pkg/upstream/...
go test ./pkg/replicatedapp/...
go test ./pkg/handlers/...
go test ./pkg/kotsadm/...
go test ./pkg/identity/...
go test ./pkg/kurl/...
```

---

### Phase 3: Medium-Risk Version Checks and Get* Calls (25+ locations)

**Priority:** Medium
**Risk:** Low-Medium - Defensive checks in less critical paths
**Estimated Time:** 1.5 hours

These locations primarily use Pattern 3 (inline check) as they're in contexts where license is optional.

#### 3.1 Version Check Locations

**Files and Lines:**
- `pkg/upstream/helm.go:278, 381`
- `pkg/image/online.go:160`
- `pkg/util/util.go:203`
- `pkg/upstream/replicated.go:164, 178`
- `pkg/kotsutil/kots.go:1711, 1718`
- `pkg/base/replicated.go:57, 86, 117`
- `pkg/kotsadm/configmaps.go:132, 233`
- `pkg/midstream/license.go:22`

**Fix Pattern:**
```go
// Before
if license.IsV1() {
    // ...
}

// After
if license != nil && license.IsV1() {
    // ...
}
```

#### 3.2 Get* Method Call Locations

**Files and Lines:**
- `pkg/handlers/license.go:28, 88, 155`
- `pkg/handlers/download.go:153`
- `pkg/airgap/update.go:149`
- `pkg/kotsutil/kots.go:123, 1696, 1699, 1715, 1739, 1765`
- `pkg/upstream/replicated.go:211, 478, 498`

**Fix Pattern:**
```go
// Before
appSlug := license.GetAppSlug()

// After
appSlug := ""
if license != nil && (license.IsV1() || license.IsV2()) {
    appSlug = license.GetAppSlug()
}
```

#### 3.3 Combined Version Check and Get* Calls

Some locations need both fixes:

**Example from pkg/kotsutil/kots.go:1739:**
```go
// Before
func FindChannelIDInLicense(channelName string, license *licensewrapper.LicenseWrapper) (string, error) {
    if license.IsV1() {
        for _, channel := range license.V1.Spec.Channels {
            if channel.Name == channelName {
                return channel.ChannelID, nil
            }
        }
    } else if license.IsV2() {
        for _, channel := range license.V2.Spec.Channels {
            if channel.Name == channelName {
                return channel.ChannelID, nil
            }
        }
    }
    return "", errors.New("channel not found")
}

// After
func FindChannelIDInLicense(channelName string, license *licensewrapper.LicenseWrapper) (string, error) {
    if license == nil {
        return "", errors.New("license is nil")
    }

    if license.IsV1() {
        for _, channel := range license.V1.Spec.Channels {
            if channel.Name == channelName {
                return channel.ChannelID, nil
            }
        }
    } else if license.IsV2() {
        for _, channel := range license.V2.Spec.Channels {
            if channel.Name == channelName {
                return channel.ChannelID, nil
            }
        }
    }
    return "", errors.New("channel not found")
}
```

**Phase 3 Verification:**
```bash
# Build all remaining affected packages
go build ./pkg/upstream/...
go build ./pkg/image/...
go build ./pkg/util/...
go build ./pkg/base/...
go build ./pkg/midstream/...
go build ./pkg/airgap/...

# Run comprehensive tests
go test ./pkg/upstream/...
go test ./pkg/image/...
go test ./pkg/util/...
go test ./pkg/base/...
go test ./pkg/midstream/...
go test ./pkg/airgap/...
```

---

### Phase 4: CLI Package Fixes (Already Planned - Reference Only)

**Status:** Documented in research but deferred
**Reason:** Phase 4 TODO comments indicate broader refactoring planned

The research identified several CLI locations with TODO comments referencing "Phase 4":
- `cmd/kots/cli/install.go:366`
- `cmd/kots/cli/upload.go:135`
- `cmd/kots/cli/admin-console.go:246`
- `cmd/kots/cli/download.go:175`

These should be addressed as part of the broader Phase 4 refactoring, not this immediate nil-check fix.

---

## Testing Strategy

### Automated Testing

#### Unit Tests
```bash
# Run all unit tests
make test

# Run specific package tests
go test ./pkg/kotsutil/... -v
go test ./pkg/handlers/... -v
go test ./pkg/upstream/... -v
```

#### Integration Tests
```bash
# Run integration tests
make integration-test
```

#### Build Verification
```bash
# Verify all packages compile
go build ./pkg/...
go build ./cmd/...
```

### Manual Verification Scenarios

#### Scenario 1: Nil License Handling
1. Create KotsKinds with nil License (using EmptyKotsKinds)
2. Call GetLicenseVersion() - should return empty string or default, not panic
3. Call FindChannelIDInLicense() - should return error, not panic

#### Scenario 2: V1 License
1. Load app with V1 license
2. Generate support bundle - should include license.yaml
3. Submit preflight data - should work
4. Create license secret - should encode V1 license

#### Scenario 3: V2 License
1. Load app with V2 license
2. Generate support bundle - should include license.yaml with V2 data
3. Submit preflight data - should log "V2 not yet supported"
4. Create license secret - should encode V2 license

#### Scenario 4: Mixed Operations
1. Switch between nil, V1, and V2 licenses
2. Verify no panics in any operation
3. Verify correct data handling for each license type

### Regression Testing

After all phases, run:
```bash
# Full test suite
make test
make integration-test
make e2e-test

# Verify no new failures
git diff main --stat
```

---

## Success Criteria

### Automated Criteria

1. **No Build Failures**
   ```bash
   go build ./pkg/... ./cmd/...
   # Expected: exit code 0
   ```

2. **All Unit Tests Pass**
   ```bash
   go test ./pkg/... ./cmd/...
   # Expected: PASS for all packages
   ```

3. **No Nil Pointer Panics**
   ```bash
   go test -race ./pkg/... ./cmd/...
   # Expected: no panic traces
   ```

4. **Integration Tests Pass**
   ```bash
   make integration-test
   # Expected: all tests pass
   ```

### Manual Criteria

1. **Code Review Checks**
   - [ ] Every `license.IsV1()` call has preceding nil check
   - [ ] Every `license.IsV2()` call has preceding nil check
   - [ ] Every `license.Get*()` call is protected
   - [ ] Every `license.V1` access is guarded by `IsV1()` check
   - [ ] Every `license.V2` access is guarded by `IsV2()` check
   - [ ] No direct field access without version verification

2. **Pattern Consistency**
   - [ ] Function entry points use Pattern 1 or Pattern 2
   - [ ] Inline checks use Pattern 3
   - [ ] Error messages are descriptive
   - [ ] Logging indicates when license is absent

3. **Backward Compatibility**
   - [ ] V1 licenses continue to work
   - [ ] V2 licenses work in all supported contexts
   - [ ] Nil license handling is graceful (no crashes)
   - [ ] EmptyKotsKinds() usage doesn't cause panics

---

## Implementation Checklist

### Phase 1: Critical Bugs (Immediate)
- [ ] Fix pkg/supportbundle/spec.go:744 - V1/V2 field access
- [ ] Fix pkg/reporting/preflight.go:84 - V1 field access
- [ ] Fix pkg/handlers/preflight.go:369 - V1 field access
- [ ] Fix pkg/kotsadm/license.go:24,37 - nil checks
- [ ] Run Phase 1 verification tests
- [ ] Commit: "Fix critical license field access bugs"

### Phase 2: High-Risk Functions (High Priority)
- [ ] Fix pkg/kotsutil/kots.go:122 - GetLicenseVersion
- [ ] Fix pkg/kotsutil/kots.go:1732 - FindChannelIDInLicense
- [ ] Fix pkg/kotsutil/kots.go:1758 - FindChannelInLicense
- [ ] Fix pkg/kotsutil/kots.go:1787 - GetDefaultChannelIDFromLicense
- [ ] Fix pkg/upstream/replicated.go (7 locations)
- [ ] Fix pkg/replicatedapp/api.go (2 functions)
- [ ] Fix pkg/handlers/license.go:217
- [ ] Fix pkg/kotsadm/* (8 locations)
- [ ] Fix pkg/identity/identity.go (3 locations)
- [ ] Fix pkg/kurl/kurl.go:48
- [ ] Run Phase 2 verification tests
- [ ] Commit: "Add nil checks to high-risk license functions"

### Phase 3: Medium-Risk Checks (Medium Priority)
- [ ] Fix version checks in pkg/upstream/helm.go (2 locations)
- [ ] Fix version checks in pkg/image/online.go (1 location)
- [ ] Fix version checks in pkg/util/util.go (1 location)
- [ ] Fix version checks in pkg/upstream/replicated.go (2 locations)
- [ ] Fix version checks in pkg/kotsutil/kots.go (2 locations)
- [ ] Fix version checks in pkg/base/replicated.go (3 locations)
- [ ] Fix version checks in pkg/kotsadm/configmaps.go (2 locations)
- [ ] Fix version checks in pkg/midstream/license.go (1 location)
- [ ] Fix Get* calls in pkg/handlers/* (4 locations)
- [ ] Fix Get* calls in pkg/airgap/update.go (1 location)
- [ ] Fix Get* calls in pkg/kotsutil/kots.go (6 locations)
- [ ] Fix Get* calls in pkg/upstream/replicated.go (3 locations)
- [ ] Run Phase 3 verification tests
- [ ] Commit: "Add defensive nil checks to remaining locations"

### Final Verification
- [ ] Run full test suite: `make test`
- [ ] Run integration tests: `make integration-test`
- [ ] Manual testing: nil license scenario
- [ ] Manual testing: V1 license scenario
- [ ] Manual testing: V2 license scenario
- [ ] Code review checklist complete
- [ ] Documentation updated (if needed)

---

## Rollback Plan

If issues arise during implementation:

1. **Per-Phase Rollback:**
   ```bash
   git revert <phase-commit-sha>
   git push
   ```

2. **Full Rollback:**
   ```bash
   git revert HEAD~3..HEAD  # Revert all 3 phase commits
   git push
   ```

3. **Emergency Hotfix:**
   - Identify specific failing function
   - Apply minimal fix to that function only
   - Create hotfix commit
   - Continue with full plan after stabilization

---

## Risk Assessment

### Low Risk
- Pattern 3 (inline checks) - minimal behavioral change
- Locations with existing partial nil checks

### Medium Risk
- Pattern 1 (function entry validation) - changes error handling
- Locations that previously assumed license present

### High Risk
- Direct field access fixes (Phase 1) - changes data flow
- Functions called by multiple code paths

**Mitigation:** Comprehensive testing after each phase, not just at the end.

---

## Notes and Considerations

### Why Fix All ~60 Locations?

1. **Defensive Programming:** Prevent future panics as code evolves
2. **V2 License Support:** Ensure proper V2 license handling everywhere
3. **Code Quality:** Consistent nil checking patterns improve maintainability
4. **Reduce Technical Debt:** Address all pointer-related issues from conversion

### Why Phase the Implementation?

1. **Risk Management:** Fix critical bugs first, defer lower-risk changes
2. **Incremental Testing:** Verify each phase before proceeding
3. **Easier Review:** Smaller, focused commits are easier to review
4. **Bisectability:** If issues arise, easier to identify which change caused it

### Future Considerations

- **Phase 4 Refactoring:** The CLI TODO comments reference a broader refactoring
- **Reporting API Update:** Several locations note that reporting doesn't support V2 yet
- **Helper Functions:** Consider creating `RequireValidLicense()` helper for common pattern

---

## Appendix: Complete Location Reference

All ~60 locations from research document, organized by phase:

**Phase 1 - Critical (4):**
- pkg/supportbundle/spec.go:744
- pkg/reporting/preflight.go:84
- pkg/handlers/preflight.go:369
- pkg/kotsadm/license.go:24,37

**Phase 2 - High Risk (30):**
- pkg/kotsutil/kots.go:122, 1732, 1758, 1787
- pkg/upstream/replicated.go:210, 287, 477, 490, 497, 509, 524
- pkg/replicatedapp/api.go:47, 61
- pkg/handlers/license.go:217
- pkg/kotsadm/app.go:98
- pkg/kotsadm/api.go:100
- pkg/kotsadm/kotsadm.go:144
- pkg/kotsadm/main.go:485
- pkg/kotsadm/configmaps.go:84, 147
- pkg/kotsadm/objects/objects.go:54
- pkg/identity/identity.go:21, 35, 56
- pkg/kurl/kurl.go:48

**Phase 3 - Medium Risk (25+):**
- pkg/upstream/helm.go:278, 381
- pkg/image/online.go:160
- pkg/util/util.go:203
- pkg/upstream/replicated.go:164, 178, 211, 478, 498
- pkg/kotsutil/kots.go:123, 1696, 1699, 1711, 1715, 1718, 1739, 1765
- pkg/base/replicated.go:57, 86, 117
- pkg/kotsadm/configmaps.go:132, 233
- pkg/midstream/license.go:22
- pkg/handlers/license.go:28, 88, 155
- pkg/handlers/download.go:153
- pkg/airgap/update.go:149

**Phase 4 - Deferred (CLI):**
- cmd/kots/cli/install.go:366
- cmd/kots/cli/upload.go:135
- cmd/kots/cli/admin-console.go:246
- cmd/kots/cli/download.go:175

---

**End of Implementation Plan**
