# AGENTS

## Current repo state
- Go module: `cloud-init-portal`.
- App is a single-binary web server with in-memory provisioning state.
- Cloud-init templates are loaded at runtime from `./templates/*.yaml`.
- UI + endpoints are implemented in stdlib `net/http` handlers.

## Verified commands
- Format: `gofmt -w *.go`
- Test: `go test ./...`
- Run locally: `go run .`
- Build single binary: `go build -o cloud-init-portal .`
- Make targets:
  - `make build`
  - `make run`
  - `make test`
  - `make vet`
- Version file: `VERSION` (SemVer `MAJOR.MINOR.PATCH`).
- GitHub Actions workflow: `.github/workflows/version-bump.yml` bumps `VERSION` on merged PRs using exactly one label: `major`, `minor`, or `patch`.

## Package boundaries
- Root package `main` contains all server/runtime code in separate files:
  - `main.go` bootstrap + routing
  - `handlers.go` UI + HTTP handlers
  - `store.go` in-memory active config + consumption logic
  - `templates_loader.go` runtime cloud-init template loader + box type config
  - `provision_logger.go` append-only event logging

## Operational notes
- No database and no external service dependencies.
- Logs append to `./provision.log`.
- One active config at a time; consumed hostnames are tracked in-memory for process lifetime.
