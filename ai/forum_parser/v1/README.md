# ForumParserService (`ai.forum_parser.v1`)

Unified contract for **crawled threads** and **CTOgram user discussions** (one `ThreadCard` / `Post` model).

## gRPC ↔ HTTP (forum-parser dev server)

| RPC | HTTP (current implementation) |
|-----|-------------------------------|
| `SearchThreads` | `POST /v1/search` |
| `GetThread` | `GET /v1/threads/{id}` |
| `ListThreadPosts` | `GET /v1/threads/{id}/posts` |
| `ListDiscussions` | `GET /v1/discussions` |
| `ListMyDiscussions` | `GET /v1/discussions/mine` |
| `CreateDiscussion` | `POST /v1/discussions` |
| `CreateReply` | `POST /v1/threads/{id}/replies` |

Legacy aliases still served over HTTP: `/v1/community/*` → same handlers.

## Auth metadata (mutating RPCs)

BFF should forward:

- `authorization: Bearer <jwt>`
- `x-user-id: <usr_id>` after token validation (matches `forum-parser` `TrustUserIDHeader`)

## Consumers

- **cg-ai** `services/forum-parser` — gRPC server (to implement after proto tag bump)
- **cg-bff** `bff-mobile` / `api-gateway` — REST for `ctogram_web`, gRPC client to forum-parser

Regenerate: `buf generate` from repo root.
