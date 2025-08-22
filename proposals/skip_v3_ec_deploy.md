# Proposal: Skip Application Deployment for V3 Embedded Cluster Initial Installs

## TL;DR

Modify KOTS to skip the application deployment step during V3 Embedded Cluster initial installations while maintaining all version records and metadata, allowing the new V3 installer binary to handle application chart deployment directly. This change only affects initial installs; upgrades continue to deploy normally.

## The problem

This proposal is an iterative step toward the larger architectural goal of controlling application lifecycle management outside the cluster through the embedded-cluster binary, while ensuring KOTS continues to function end-to-end during the transition.

**Long-term vision**: Move application deployment and lifecycle management from in-cluster KOTS to the external embedded-cluster binary for better control, reliability, and user experience.

**Current iteration goal**: Enable the V3 embedded-cluster binary to handle initial application deployment while allowing KOTS to seamlessly take over lifecycle management post-install. This approach:

- **Maintains KOTS functionality**: Admin console, upgrades, configuration management remain intact
- **Enables iterative development**: Allows V3 installer development to proceed without breaking existing KOTS workflows  
- **Preserves upgrade path**: KOTS can manage subsequent deployments after initial install
- **Reduces scope**: Smaller, manageable change that keeps the system working end-to-end

**Current technical problem**: Both KOTS and the V3 installer attempt to deploy applications during initial install, creating resource conflicts and unclear ownership. We need KOTS to defer initial deployment to the V3 installer while maintaining all the version records and metadata necessary for post-install management.

**Evidence**: V3 EC installer development requires this separation to proceed with external lifecycle management while keeping KOTS operational for existing features.

## Prototype / design

The solution introduces a conditional deployment skip mechanism:

```
┌─────────────────┐
│  DeployVersion  │
└────────┬────────┘
         │
         ▼
┌─────────────────┐     ┌──────────────────┐
│ Is V3 EC Init?  │────▶│ Create Version   │
└────────┬────────┘ Yes │ Records/Metadata │
         │No            └───────────┬──────┘
         ▼                          │
┌─────────────────┐                 │
│ Deploy App      │                 │
└────────┬────────┘                 │
         │                          │
         ▼                          ▼
┌─────────────────────────────────────┐
│  Update Status to "Deployed"        │
└─────────────────────────────────────┘
```

Detection mechanism uses environment variables passed via Helm values:
- `IS_EMBEDDED_CLUSTER_V3=true` - Indicates V3 embedded cluster
- Combined with sequence check to determine initial install vs upgrade

## New Subagents / Commands

No new subagents or commands will be created for this change.

## Database

**No database changes required.**

All existing tables and schemas remain unchanged. Version records continue to be created with the same structure.

## Implementation plan

### Files to modify

1. **pkg/operator/operator.go**
   - Modify `DeployApp()` function around line 423
   - Add conditional check before `o.client.DeployApp()`
   - For V3 EC initial installs: skip deployment but call `setDeployResults` to insert record into `app_downstream_output` table
   - Ensure downstream deploy status is updated with successful deployment indication

2. **pkg/util/util.go**
   - Add `IsV3EmbeddedCluster()` function to check `IS_EMBEDDED_CLUSTER_V3` env var
   - Add `IsV3EmbeddedClusterInitialInstall()` helper combining V3 check and sequence validation

3. **pkg/operator/operator_test.go**
   - Add unit tests for V3 detection logic
   - Add tests for initial install vs upgrade scenarios
   - Verify status updates occur correctly

### External contracts

**No changes to external APIs or events.**

The operator continues to:
- Emit the same status update events
- Maintain the same gRPC interface
- Preserve all existing API contracts

### Implementation details

**Core deployment logic modification in `pkg/operator/operator.go`:**

```go
func (o *Operator) DeployVersion(appID string, versionLabel string, sequence int64) (bool, error) {
    // Existing version loading and validation logic...
    
    // Check if this is a V3 EC initial install that should skip deployment
    if util.IsV3EmbeddedClusterInitialInstall(sequence) {
        logger.Infof("Skipping deployment for V3 Embedded Cluster initial install (sequence %d)", sequence)
        
        // Create successful deployment record for admin console
        emptyOutput := downstreamtypes.DownstreamOutput{
            DryrunStdout: base64.StdEncoding.EncodeToString([]byte("Skipped - deployed by V3 installer")),
            ApplyStdout:  base64.StdEncoding.EncodeToString([]byte("Skipped - deployed by V3 installer")),
        }
        err := store.GetStore().UpdateDownstreamDeployStatus(appID, clusterID, sequence, false, emptyOutput)
        if err != nil {
            return false, errors.Wrap(err, "failed to update downstream deploy status")
        }
        
        // Set version as deployed
        err = store.GetStore().SetDownstreamVersionStatus(appID, clusterID, sequence, storetypes.VersionDeployed, "")
        if err != nil {
            return false, errors.Wrap(err, "failed to set downstream version status")
        }
        
        return true, nil
    }
    
    // Proceed with normal deployment for all other cases
    return o.client.DeployApp(...)
}
```

**V3 detection utility in `pkg/util/util.go`:**

```go
func IsV3EmbeddedClusterInitialInstall(sequence int64) bool {
    return IsV3EmbeddedCluster() && sequence == 0
}

func IsV3EmbeddedCluster() bool {
    return os.Getenv("IS_EMBEDDED_CLUSTER_V3") == "true"
}
```

### Toggle strategy

**Environment variable toggle**: `IS_EMBEDDED_CLUSTER_V3`
- Set via Helm chart values during KOTS deployment
- No feature flags or entitlements required
- Clear on/off behavior with no gradual rollout needed

## Testing

### Unit tests
```go
// pkg/util/util_test.go
func TestIsV3EmbeddedCluster(t *testing.T) {
    // Test with env var set
    // Test with env var unset
    // Test with various values
}

// pkg/operator/operator_test.go  
func TestDeployVersionSkipsForV3ECInitialInstall(t *testing.T) {
    // Mock V3 EC environment
    // Verify DeployApp not called for sequence 0
    // Verify status still updated to "deployed"
}

func TestDeployVersionDeploysForV3ECUpgrade(t *testing.T) {
    // Mock V3 EC environment
    // Verify DeployApp IS called for sequence > 0
}
```

### Integration tests
NONE

### Compatibility tests (covered by E2E tests)
- Non-V3 embedded cluster installs continue to deploy
- kURL installations unaffected
- Online installations unaffected

## Monitoring & alerting

NONE

## Backward compatibility

**Fully backward compatible:**
- Only affects V3 EC initial installs when environment variable is set
- All existing installation types unchanged
- Version record format unchanged
- API contracts preserved

**Forward compatibility:**
- V3 installs can upgrade to future versions normally (manual test required)

## Migrations

**No special deployment handling required.**

This change can be deployed through normal channels:
1. Update KOTS image with new code
2. Set `IS_EMBEDDED_CLUSTER_V3=true` in Helm values for V3 deployments
3. Existing installations unaffected

## Trade-offs

**Optimizing for**: Clean separation of concerns between KOTS and V3 installer

**Trade-offs made**:
1. **Complexity**: Adding conditional logic to core deployment path
   - *Mitigation*: Well-isolated, clearly documented condition
2. **Testing surface**: Need to test both V3 and non-V3 paths
   - *Mitigation*: Comprehensive test coverage included
3. **Debugging**: Two different deployment mechanisms to understand
   - *Mitigation*: Clear logging of which path is taken

## Alternative solutions considered

1. **Remove KOTS from V3 entirely**
   - *Rejected*: Need admin console for upgrades and management for now
   - Would require significant changes and is out of scope for this iteration

2. **Version detection via Helm client**
   - *Rejected*: Would require significant changes and is out of scope for this iteration

## Research

Reference: [Skip V3 EC Deploy Research](./skip_v3_ec_deploy_research.md)

### Prior art in codebase
- Environment-based detection: `pkg/util/util.go:IsEmbeddedCluster()`
- Conditional deployment: License validation skip for airgap
- Sequence-based logic: Previous deployment detection in `operator.go:352`

### External references
- Helm chart environment variable patterns
- Kubernetes operator deployment strategies
- [Embedded Cluster V3 Architecture Docs](internal-link)

## Checkpoints (PR plan)

**Single PR approach** recommended due to small scope:

1. **Single PR containing**:
   - Core logic changes in operator.go and util.go
   - Complete test coverage
   - Documentation updates

The change is isolated enough that breaking it into multiple PRs would add unnecessary overhead without improving reviewability.