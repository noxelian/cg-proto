# cg-proto - CTOgram Contracts

This repository owns the versioned Protobuf contracts shared by CTOgram services. Read this file before editing any `.proto` file.

## Rules

- Additive changes only: never delete or renumber an existing field or RPC.
- Keep one coherent package per domain and avoid importing another business domain's message types.
- Generated Go under `gen/go` is committed and must only be changed by `buf generate`.
- Every release is tagged `v1.X.Y` and pushed before consumers are bumped.
- Consumers pin the GitHub module directly; never add local `replace` directives.
- Identity, tenant, amount and ownership fields are request data, not authority. Services must bind them to authenticated claims and source-of-truth records.
- Distinct semantics require distinct fields. For request listing, `organization_id` is organization feed-state context; `owner_organization_id` filters the request's authoritative owner organization.

## Validation

```bash
buf lint path/to/changed.proto
buf generate
go test ./...
go build ./...
git diff --check
```

The repository has historical global `buf format` and `buf lint` drift, so do
not turn an unrelated contract change into a repository-wide cleanup. Review both the source
`.proto` and generated diff, then tag and push the release before changing consumers.
