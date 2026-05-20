# AGENTS.md

This SDK provides a client and CLI tool for the ABAX Open API.

## Docs

When developing this SDK, use the API docs and specs:

- [Getting Started](./docs/getting-started.md)
- [OpenAPI Spec](./internal/oapi/abaxoapi/01-original.json)

## Developing

- Install tools: `mise install`

- Run tests: `mise run test`

- Lint: `mise run lint`

- Re-generate code: `mise run generate`

- Full CI build: `mise run build`

- Leave all version control and git to the user/developer. If you see a build error related to having a git diff, this is normal.
