# AGENTS

`cloud-init-portal` is a Go 1.22, standard-library-only web server for queuing and serving bare-metal cloud-init configurations. It builds as a single binary; templates, logs, and optional persisted state remain runtime files.

## Current repo state
- Go module: `cloud-init-portal`; `go.mod` has no third-party dependencies.
- App is a single-binary web server with an active-plus-FIFO provisioning queue and optional file-backed persistence via `STATE_FILE`.
- Cloud-init templates are loaded at startup from `./templates/*.yaml` and parsed with `missingkey=error`.
- UI + endpoints use stdlib `net/http`; the server listens on `0.0.0.0:8080`.

## Verified commands
- Requires Go 1.22 (from `go.mod`).
- Format: `gofmt -w *.go` or `make fmt`
- Test: `go test ./...` or `make test`
- Vet: `go vet ./...` or `make vet`
- Run locally: `go run .` or `make run`
- Build single binary: `go build -o cloud-init-portal .` or `make build`
- CI (`.github/workflows/go-ci.yml`) checks `gofmt -l .`, then runs `go test ./...`, `go vet ./...`, and `go build ./...`.
- Version file: `VERSION` (SemVer `MAJOR.MINOR.PATCH`).
- GitHub Actions workflow: `.github/workflows/version-bump.yml` bumps `VERSION` on merged PRs using exactly one label: `major`, `minor`, or `patch`.

## Package boundaries
- Root package `main` contains all server/runtime code in separate files:
  - `main.go` configuration, dependency wiring, routes, HTTP lifecycle, and graceful shutdown
  - `handlers.go` shared server/view/payload types
  - `ui_handlers.go` config/queue lifecycle handlers; `api_handlers.go` status, logs, and cloud-init endpoints
  - `views.go` provisioning/queue UI; `logs_views.go` event-history UI; keep UI changes consistent with `DESIGN.md`
  - `store.go` mutex-protected active config and persisted state integration
  - `queue.go` FIFO lifecycle and auto-promotion; `queue_persistence.go` atomic state-file writes
  - `templates_loader.go` runtime cloud-init template loader and code-defined box types
  - `validation.go`, `template_render.go`, and `base_url.go` focused input/rendering/URL helpers
  - `middleware.go` request IDs, request logging, and per-client fixed-window rate limiting
  - `provision_logger.go` append-only JSON-lines event logging plus filtered history reads
- Tests are colocated as `*_test.go`; HTTP tests use `net/http/httptest`, and shared helpers call `t.Helper()`.

## Runtime configuration
- `PUBLIC_BASE_URL` overrides generated user-data/meta-data URLs.
- `TRUST_PROXY_HEADERS=true` trusts the first `X-Forwarded-Proto` and `X-Forwarded-Host` values; enable only behind a trusted proxy.
- `STATE_FILE=/path/state.json` persists active, pending, completed, failed, and consumed state; without it, state lasts only for the process lifetime.
- `STATUS_RATE_LIMIT_PER_SEC` and `WRITE_RATE_LIMIT_PER_SEC` default to `6` and `3`; missing, nonnumeric, or nonpositive values fall back to those defaults.

## Operational notes
- Run from the repository root (or provide the same runtime layout): templates and `provision.log` use paths relative to the working directory.
- Startup fails if `./templates` is missing, contains no `.yaml` files, has an invalid template, or `STATE_FILE` contains invalid JSON.
- Logs append to `./provision.log`; it is gitignored and needs external rotation for long-running deployments.
- One config is active while additional configs wait FIFO. Completion after both `/user-data` and `/meta-data`, `POST /consume`, or `POST /force-replace` auto-promotes the next pending item; force replacement records the displaced item as failed without marking its hostname consumed.
- Consumed hostnames cannot be reused while retained in the in-memory or persisted state.
- Add cloud-init templates under `templates/`; available template fields are documented in `README.md`. Add hardware interface mappings in `DefaultBoxTypes()` rather than YAML.
