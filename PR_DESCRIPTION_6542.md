# PR Description — Fix GKE member cluster import stuck at "Waiting for the cluster to join"

## Summary
This PR addresses a bug where importing a GKE cluster as a KubeSphere member cluster remains indefinitely in `Waiting for the cluster to join...` even though the provided kubeconfig can access the cluster successfully with standalone `kubectl`.

Closes #6542.

## Problem
Users can provide a valid token-based kubeconfig for a GKE cluster, and direct commands such as:

`kubectl get nodes --kubeconfig=<file>`

work correctly. However, the KubeSphere member-cluster join workflow remains stuck and ks-apiserver logs show request failures (for example `503`) when accessing cluster-scoped KubeSphere APIs through the imported cluster.

## Impact
- GKE clusters cannot be added as member clusters in affected setups.
- Multi-cluster management is blocked despite valid credentials/connectivity.
- User experience is confusing because CLI kubeconfig tests pass but join state does not progress.

## Root Cause Hypothesis
The imported cluster can be reached at Kubernetes API level, but KubeSphere join logic likely fails in one or more of these areas:
1. Endpoint reachability/translation for internal-only API server addresses from KubeSphere control-plane context.
2. Cluster registration handshake assumptions around KubeSphere API availability (`/kapis/...`) before initialization is complete.
3. Validation path coupling that treats transient/unavailable KubeSphere API on member side as hard join failure instead of staged readiness.
4. Token/identity scope checks not aligned with GKE service-account token usage pattern in this flow.

## Proposed Changes
1. Harden member-cluster join preflight checks:
   - explicitly separate Kubernetes API connectivity checks from KubeSphere API readiness checks.
2. Improve join-state transitions:
   - move from a single waiting state to actionable sub-states with concrete error reasons.
3. Add robust retry/backoff for KubeSphere API bootstrap checks on newly imported member clusters.
4. Enhance diagnostics:
   - include precise failure point (network/authn/authz/api readiness) in status and logs.
5. Ensure internal endpoint handling for imported clusters is validated from ks-apiserver runtime network perspective.

## Scope
- In scope:
  - member-cluster registration/join logic
  - status and error reporting for join progress
  - resilience improvements for GKE-like external clusters
- Out of scope:
  - redesign of general multi-cluster architecture
  - unrelated kubeconfig generation mechanisms

## Compatibility & Risk
- No breaking API changes intended.
- Existing successful import paths should remain unchanged.
- **Risk**: altered retry/state logic may affect join timing behavior.
- **Mitigation**:
  - keep default behavior for healthy clusters,
  - add focused tests for timeout/retry transitions,
  - preserve backward-compatible status fields where possible.

## Validation Plan
- Unit tests:
  - preflight result classification (k8s reachable vs kapis unavailable)
  - join-state transition logic and retry/backoff behavior
- Integration tests:
  - import cluster with reachable API and delayed KubeSphere readiness
  - import cluster using internal endpoint scenarios
  - verify eventual join completion and meaningful failure states
- Manual verification:
  1. Import GKE cluster with token-based kubeconfig.
  2. Confirm join progresses beyond waiting with clear status updates.
  3. Validate ks-apiserver logs provide actionable diagnostics when failing.

## Rollout
- Merge behind standard bugfix release process.
- Document troubleshooting guidance for GKE/internal endpoint imports.
- Include release note under multi-cluster/member-cluster reliability fixes.

## Checklist
- [x] Full local PR description drafted.
- [ ] Functional fix implemented.
- [ ] Unit/integration tests added and passing.
- [ ] User-facing troubleshooting notes updated.
