# Kotsadm Migrations

# Development

## Okteto

### How To

#### Create/update a migration

1. From the migrations directory `okteto up`
2. Make the changes local.
3. Run `/schemahero plan`
4. You can check the plan file in `/migrations/plan.yaml` to make sure it looks correct.
5. Run `/schemahero apply` and check the output for correctness
