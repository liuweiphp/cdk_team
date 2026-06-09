# CDK Delete Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Allow admins to delete their own unused CDKs from CDK 管理.

**Architecture:** Add `CdkService.Delete` for ownership and status checks, expose it through `DELETE /api/admin/cdk/:id`, and add a guarded delete button in the Vue CDK table.

**Tech Stack:** Go, Gin, GORM, Vue 3, Element Plus, Vite.

---

### Task 1: Backend Delete Rule

**Files:**
- Create: `backend/service/cdk_service_test.go`
- Modify: `backend/service/cdk_service.go`

- [ ] Write failing tests for deleting own unused CDK, rejecting exchanged CDK, and rejecting team-shared CDK.
- [ ] Run `go test ./service -run 'TestCdkServiceDelete' -v` and verify RED.
- [ ] Implement `CdkService.Delete(id, currentUserID uint) error`.
- [ ] Run the same tests and verify GREEN.

### Task 2: API Route

**Files:**
- Modify: `backend/handler/admin_cdk_handler.go`
- Modify: `backend/router/router.go`

- [ ] Add `DELETE /api/admin/cdk/:id`.
- [ ] Parse path ID and call `CdkService.Delete`.
- [ ] Run `go test ./... -count=1`.

### Task 3: Frontend Delete Button

**Files:**
- Modify: `frontend/src/api/index.ts`
- Modify: `frontend/src/views/admin/CdkManageView.vue`

- [ ] Add `deleteCdk(id)` API helper.
- [ ] Add delete button only for `unused` rows.
- [ ] Confirm before deletion, call API, show success, and refresh table.
- [ ] Run `npm run build`.

### Task 4: Docker Verification

**Files:**
- Generated: `frontend/dist/**`

- [ ] Rebuild backend image.
- [ ] Restart backend and frontend containers.
- [ ] Verify `GET /readyz` returns ready.
