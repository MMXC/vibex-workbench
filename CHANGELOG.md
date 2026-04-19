# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased] - 2026-04-20

### Added

- **E3 (Run Engine)**
  - E3-U1: Run state tracking — run-store.ts toolInvocations[], addToolInvocation() / updateToolInvocation()
  - E3-U2: RunStatusBar in Composer.svelte — running spinner + tool count / completed green check / failed red X
  - sse.ts: tool.called/completed/failed handlers integrate runStore tool tracking

- **E2 (Thread Management)**
  - E2-U1: Thread IndexedDB persistence via Dexie (WorkbenchDB, threads/artifacts tables)
  - E2-U2: ThreadList four-state UI — skeleton / empty / normal / error-retry
  - E2-U3: Thread switching triggers SSE reconnect (disconnect → connect)
  - `frontend/src/lib/db.ts` — Dexie database with IndexedDB schema
  - `frontend/src/lib/stores/thread-store.ts` — loadFromDB / addThread / updateThread / removeThread persistence

- **E1 (SSE Backend Integration)**
  - CF-1: Install vitest / @testing-library/svelte / @playwright/test for testing
  - CF-2: Fix right panel width 0px → 320px in WorkbenchShell.svelte
  - CF-3: Add onDestroy(() => sseConsumer.disconnect()) to prevent SSE connection leaks
  - CF-4: Create .env.example template with VITE_SSE_URL, frontend/.env in gitignore
  - E1-U1: SSE URL environment variable — replace all 3 hardcoded URLs with `import.meta.env.VITE_SSE_URL`
  - E1-U2: SSE exponential backoff retry — 3s→6s→12s→24s→48s, max 5 retries
  - Backend SSE mock server (Python, port 33335) — thread-based client registry with mock run executor
  - Frontend SSE consumer with full event handler mapping (Thread/Run/Tool/Artifact/Canvas events)
  - Three-column workbench layout (ThreadList / Canvas / ArtifactPanel / Composer)

### Changed

- WorkbenchShell grid layout: `grid-template-columns: 280px 1fr 320px`
- All SSE URLs now read from `VITE_SSE_URL` environment variable

### Fixed

- Right sidebar was 0px width — now renders at 320px
- SSE connections now properly disconnected on component destroy
- TS: add missing `Step` interface in generated types.ts
- SSE: fix `$effect` cleanup to prevent memory leaks and double connections
