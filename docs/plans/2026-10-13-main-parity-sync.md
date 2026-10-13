# Main parity synchronization implementation plan

Goal: match main 1e2baa1 algorithms while preserving backend ownership, collect local budget changes, then merge into artemch and dev.

Architecture: existing React order screen, job queue and service owners. Excel stays in document-service. Main provides oracle behavior, never merge source.

1. Run local budget/stock regressions red, import saved patch, regenerate protobuf.
2. Add main-derived LeviSsime identity/volume/duplicate regressions, implement matching and warehouse owners.
3. Standalone recalculation in existing order UI/jobs, preserve actual quantities/comments.
4. Compare budgets, recommendations, supplier quantities, prices and warehouse merging with main on shared fixtures.
5. Sync docs/contracts; test:ui, make verify, security and review.
6. Commit feature, merge authorized artemch/dev, push with hooks and inspect CI/deployment.

Original dirty worktrees remain untouched. Never commit private workbooks or browser Excel processing.
