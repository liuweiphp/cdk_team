# User File Prefix Sequence Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let admins and users configure per-user generated filename prefixes, with generated filenames starting at 1001 after each prefix update.

**Architecture:** Add prefix and sequence fields to users, update import generation to allocate filenames from the importing user's sequence, add admin and self-service APIs for prefix updates, and expose both controls in the Vue admin UI.

**Tech Stack:** Go, Gin, GORM, MySQL migrations, Vue 3, Element Plus, Vite.

---

### Task 1: Data Model and Import Sequence

**Files:**
- Create: `backend/migration/0009_user_file_prefix_sequence.up.sql`
- Create: `backend/migration/0009_user_file_prefix_sequence.down.sql`
- Modify: `backend/model/user.go`
- Modify: `backend/service/test_helpers_test.go`
- Modify: `backend/service/redeem_item_service_test.go`
- Modify: `backend/service/redeem_item_service.go`

- [ ] Add failing tests for prefix `x` generating `x1001.txt`, `x1002.txt`, and sequence advancing to `1003`.
- [ ] Implement migration/model/test schema.
- [ ] Update import generation to use `users.file_prefix` and `users.file_sequence_next`.
- [ ] Run service tests.

### Task 2: Prefix Update APIs

**Files:**
- Modify: `backend/service/user_service.go`
- Modify: `backend/handler/admin_user_handler.go`
- Modify: `backend/handler/user_handler.go`
- Modify: `backend/router/router.go`
- Modify: `backend/service/user_service_test.go`

- [ ] Add tests for valid prefix reset and invalid prefix rejection.
- [ ] Add `UpdateFilePrefix` behavior.
- [ ] Add admin `file_prefix` support in `PATCH /api/admin/users/:id`.
- [ ] Add self-service `PUT /api/user/file-prefix`.
- [ ] Run backend tests.

### Task 3: Frontend Controls

**Files:**
- Modify: `frontend/src/api/index.ts`
- Modify: `frontend/src/views/admin/UserManageView.vue`
- Modify: `frontend/src/layouts/AdminLayout.vue`

- [ ] Add `updateMyFilePrefix`.
- [ ] Add file prefix field to user management dialog.
- [ ] Add sidebar “生成设置” dialog for current user.
- [ ] Confirm before saving prefix because sequence resets to 1001.
- [ ] Run frontend build.

### Task 4: Runtime Verification

**Files:**
- Generated: `frontend/dist/**`

- [ ] Run backend tests with `-count=1`.
- [ ] Run frontend build.
- [ ] Rebuild backend image and restart backend/frontend containers.
- [ ] Verify `/readyz`.
