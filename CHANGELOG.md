# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased] - 2026-04-20

### Added

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
