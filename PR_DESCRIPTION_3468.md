# PR Description — Support PodDisruptionBudget (PDB) for workload availability protection

## Summary
This PR adds support for configuring PodDisruptionBudget (PDB) in KubeSphere-managed workloads to prevent excessive simultaneous voluntary disruptions and improve service availability during node drain, maintenance, and autoscaling events.

Closes #3468.

## Problem
Without PDB support in workload configuration, users cannot define disruption safety constraints (`minAvailable` or `maxUnavailable`) through KubeSphere-managed resources. During voluntary evictions, too many pods may be disrupted at once, causing temporary service unavailability or degraded SLOs.

## Motivation
PDB is a standard Kubernetes availability control and is critical for production reliability:
- protects replicas during node drain and rolling operations
- enforces controlled disruption rates
- reduces outage risk for stateless and stateful apps

Feature request reference:
- Kubernetes docs: https://kubernetes.io/docs/tasks/run-application/configure-pdb/

## Proposed Changes
1. Add optional PDB configuration in workload management flow.
2. Support both policy styles:
   - `minAvailable`
   - `maxUnavailable`
3. Generate and reconcile `policy/v1` PodDisruptionBudget objects aligned with workload selectors.
4. Keep backward compatibility by making PDB disabled by default unless explicitly configured.
5. Expose PDB status/validation feedback to users when config is invalid or ineffective.

## Scope
- In scope:
  - API/spec extensions for PDB settings
  - controller/reconciler creation and update of PDB resources
  - UI/API validation for mutually exclusive fields and value correctness
- Out of scope:
  - changes to Kubernetes eviction semantics
  - unrelated workload scaling behavior

## Compatibility & Upgrade
- Existing workloads continue to behave unchanged if no PDB is configured.
- New PDB support should be opt-in and non-breaking.
- Reconciliation should be idempotent for existing deployments.

## Risk Assessment
- **Risk**: incorrectly scoped selectors could create ineffective or overly restrictive PDBs.
- **Risk**: misconfigured strict PDB values could block planned maintenance operations.
- **Mitigation**:
  - strict validation of selector and field combinations
  - clear user-facing warnings/messages
  - documentation for operational best practices

## Validation Plan
- Unit tests:
  - spec validation (`minAvailable` vs `maxUnavailable`)
  - PDB object generation and update logic
  - selector mapping correctness
- Integration/E2E tests:
  - workload with PDB survives controlled eviction scenarios
  - no-PDB workloads remain unchanged
  - update/delete lifecycle for PDB-managed workloads
- Manual checks:
  1. create workload + PDB via KubeSphere
  2. drain node and observe disruption limits
  3. verify service continuity and pod replacement behavior

## Rollout
- Merge behind normal feature delivery path.
- Document how to configure PDB and recommend baseline values by workload type.
- Include release-note entry under workload availability enhancements.

## Checklist
- [x] Full local PR description drafted.
- [ ] Functional implementation completed.
- [ ] Unit/integration tests added and passing.
- [ ] User documentation updated.
