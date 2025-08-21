# Research: Skip Deployment for V3 Embedded Cluster Initial Installs

## Summary
This research document covers the technical analysis for implementing a feature to skip application deployment for V3 Embedded Cluster initial installs while maintaining version records and admin console functionality.

## Current Implementation Analysis

### Deployment Flow
1. **Main deployment entry point**: `pkg/operator/operator.go:423`
   - The `DeployVersion` function orchestrates the deployment process
   - Calls `o.client.DeployApp(deployArgs)` to actually deploy the application
   - Updates downstream status based on deployment result

2. **Key components involved**:
   - `pkg/operator/operator.go`: Main operator logic that handles deployment
   - `pkg/operator/client/deploy.go`: Client interface for deployment operations
   - `pkg/store/kotsstore/version_store.go`: Manages version records and metadata
   - `pkg/embeddedcluster/util.go`: Contains embedded cluster utility functions
   - `pkg/util/util.go`: Contains `IsEmbeddedCluster()` detection function

### Embedded Cluster Detection
Current detection mechanism uses environment variables:
- `EMBEDDED_CLUSTER_ID`: Primary identifier for embedded cluster installations
- `EMBEDDED_CLUSTER_VERSION`: Version information
- `IsEmbeddedCluster()` function at `pkg/util/util.go:156` checks for `EMBEDDED_CLUSTER_ID`

### Version Management
The system creates and manages version records through:
- `CreateVersion`: Creates initial version records
- `SetDownstreamVersionStatus`: Updates version status (deployed, failed, etc.)
- Version metadata is stored including KotsKinds, installation specs, and embedded cluster configs

### Critical Path for Deployment
```
DeployApp() -> 
  Load app and version data ->
  Prepare manifests ->
  Call o.client.DeployApp() ->
  Update downstream status
```

## V3 Embedded Cluster Specifics

### Current V3 Detection Gaps
- No existing mechanism to differentiate V3 from other embedded cluster versions
- Environment variable approach would be consistent with existing patterns
- Need to distinguish initial install from upgrades

### Initial Install vs Upgrade Detection
Current code checks for:
- `previouslyDeployedSequence` to determine if this is an upgrade
- Sequence `-1` indicates no previous deployment (initial install)

## Implementation Considerations

### Detection Mechanism Options
1. **Environment Variable** (Recommended)
   - Add `IS_EMBEDDED_CLUSTER_V3=true` or similar
   - Consistent with existing pattern
   - Easy to set via Helm chart values

2. **Version Parsing**
   - Parse `EMBEDDED_CLUSTER_VERSION` to detect V3
   - More complex, requires version format stability

3. **ConfigMap/Annotation**
   - Use Kubernetes resources for detection
   - More complex, requires additional API calls

### Code Modification Points
1. **Primary change location**: `pkg/operator/operator.go:423`
   - Add condition check before `o.client.DeployApp()`
   - Return success without calling deployment

2. **Detection helper function**:
   - Add `IsV3EmbeddedClusterInitialInstall()` function
   - Check both V3 status and initial install condition

3. **Status updates**:
   - Ensure version status is properly set to "deployed"
   - Maintain all metadata creation flows

## Risks and Edge Cases

### Identified Risks
1. **Version Status Consistency**
   - Must ensure version appears as "deployed" in UI
   - Admin console queries rely on status field

2. **Upgrade Path**
   - Must not skip deployment during upgrades
   - Clear distinction between initial install and upgrade required

3. **Rollback Scenarios**
   - If V3 install fails, need clear rollback path
   - Version records must remain consistent

### Testing Requirements
1. **Unit Tests**:
   - Test V3 detection logic
   - Test initial install vs upgrade differentiation
   - Test version status updates

2. **Integration Tests**:
   - Full V3 EC install flow
   - Upgrade from V3 initial install
   - Admin console accessibility

## Dependencies
- No external library changes required
- No database schema changes needed
- No API contract changes

## Files to Modify
1. `pkg/operator/operator.go` - Main deployment logic
2. `pkg/util/util.go` - Add V3 detection helper
3. Test files for both components

## Backwards Compatibility
- Non-V3 embedded cluster installs unaffected
- Existing kURL installs unaffected
- Online installs unaffected