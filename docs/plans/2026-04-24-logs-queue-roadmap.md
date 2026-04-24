# Cloud-Init Portal Logs + Queue Roadmap

> For Hermes: execute in small, test-first increments. Keep single-binary, no SQL, and file-backed persistence.

## Goal
Add operational history visibility (logs page + logs API) and sequential multi-box provisioning (queue API + queue page) without introducing a database.

## Architecture
Use two file-backed stores:
1) Append-only JSONL event log for audit/history.
2) JSON snapshot state for queue and current active item.

Keep all endpoints on existing net/http server and preserve one-active-config behavior while allowing pending queued items.

## Constraints
- No SQL datastore.
- Self-contained single binary.
- Durable across restarts.
- API-first queue operations.
- UI pages for logs and queue visibility.

---

## Phase 1: Logs Foundation (History)

### Scope
- Structured JSONL logging.
- Logs API with filtering/pagination.
- Logs page for operators.

### Deliverables
- `provision_logger.go` supports JSONL write + query read.
- New endpoint: `GET /api/logs` (`limit`, `event`, `hostname`, optional `cursor`).
- New page: `GET /logs` showing latest events.
- Route wiring in `main.go`.

### Files
- Modify: `provision_logger.go`
- Modify: `api_handlers.go`
- Create/Modify: `ui_handlers.go` (logs page handler)
- Create/Modify: `views.go` or dedicated logs template file
- Modify: `main.go`
- Tests: `provision_logger_test.go`, new `logs_handlers_test.go`

### Acceptance Criteria
- Generated/served/consumed events appear in `/logs`.
- `/api/logs` returns JSON results and supports filters.
- Existing behavior unchanged for provisioning flow.

---

## Phase 2: Queue Core (Sequential Processing)

### Scope
- Queue item model.
- Persistent queue state file.
- Auto-promotion from pending -> active.

### Deliverables
- Queue item type with status lifecycle.
- Queue store APIs: enqueue/list/cancel/promote-next/complete-active.
- Integration with existing consume/force-replace lifecycle.

### Files
- Create: `queue.go`
- Create: `queue_persistence.go`
- Modify: `store.go` and handlers where active config is consumed/replaced
- Tests: `queue_test.go`, lifecycle integration tests

### Acceptance Criteria
- Multiple items can be queued while one is active.
- FIFO processing is deterministic.
- Queue persists restart.

---

## Phase 3: Queue API

### Scope
- API for enqueue and queue state inspection.

### Deliverables
- `POST /api/queue`
- `GET /api/queue`
- `GET /api/queue/{id}`
- `POST /api/queue/{id}/cancel`

### Files
- Modify/Create: `api_handlers.go` split into queue/log sections
- Tests: API handler tests for success + validation + edge cases

### Acceptance Criteria
- Dev can enqueue N boxes via API.
- Queue status and positions are queryable.

---

## Phase 4: Queue UI Page

### Scope
- Operator queue visibility and basic controls.

### Deliverables
- New page `GET /queue`.
- Sections: active item, pending list, recent completed/failed.
- Basic actions: refresh, cancel pending.

### Files
- Modify/Create: `ui_handlers.go`, templates/views
- Tests: render and handler method guards

### Acceptance Criteria
- Operators can see “current, next, done” at a glance.

---

## Phase 5: Hardening

### Scope
- Reliability, safety limits, operational behavior.

### Deliverables
- Max queue depth config.
- Idempotency key support for enqueue API.
- Log rotation strategy (size/time based).
- Additional rate limiting for queue writes.

### Acceptance Criteria
- Stable behavior under retry storms and long runtimes.

---

## Milestone Order
1. Phase 1 (logs) — immediate operator value
2. Phase 2 (queue core)
3. Phase 3 (queue API)
4. Phase 4 (queue UI)
5. Phase 5 (hardening)

## Immediate Next Step
Implement Phase 1 now on current branch with tests and push.
