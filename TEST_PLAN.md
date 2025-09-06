# Test Plan

## Unit
- `internal/spore`
  - Pack/Verify success.
  - Verify fails when manifest signature is altered.
  - Verify fails when binary content changed.
- `internal/repo`
  - Put returns same digest for same file.
  - Path returns correct location.
- `internal/fabric`
  - PublishPlan reaches subscribers.
  - RegisterEndpoint replaces previous for same NodeID.
  - Endpoints returns copy (no external mutation).

## Integration
- Build fake workload that returns `ok` on `/health` and text on `/hello`.
- Pack → Publish → Run (3 agents, 2 instances).
- `curl` `/{app}/hello` 10x; observe alternating responses (RR).

## Update
- Build new workload version (print different message).
- Pack → Publish new digest → PublishPlan with new digest.
- Within ~2s warmup, edge still serves 200; older instance processes exit on each node.
