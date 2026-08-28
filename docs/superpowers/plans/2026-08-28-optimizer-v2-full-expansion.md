# Optimizer 2.0 Full Expansion & Superpowers Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Expand Optimizer 2.0 with expanded telemetry (Disk I/O, Network Throughput, Sensor Export), Win11 AI Recall & High-Precision Timer optimizations, and enhanced real-time UI diagnostics.

**Architecture:** Extend `internal/telemetry` with I/O metrics, register 3 new catalog tweaks (`privacidade.win11-recall-off`, `jogos.hpet-synthetic-timers`, `sistema.vbs-hvci-audit`), implement report exporter in `cmd/optimizerui/app.go`, and wire into frontend MicroCanvas engine.

**Tech Stack:** Go 1.22+, Windows Native WMI / PDH / Winreg, Vanilla JS Canvas 2D, Node.js test runner.

---

## Task Decomposition

### Task 1: Extend Telemetry with Disk I/O & Network Throughput
**Files:**
- Modify: `internal/telemetry/types.go`
- Modify: `internal/telemetry/provider.go`
- Modify: `cmd/optimizerui/app.go`
- Modify: `cmd/optimizerui/app_test.go`

- [ ] **Step 1:** Add `DiskReadSpeedMBs`, `DiskWriteSpeedMBs`, `NetworkRxKBps`, `NetworkTxKBps` to `MetricSample` and `TelemetriaAoVivoUI`.
- [ ] **Step 2:** Update `WindowsLiveProvider` and `MockProvider` to sample disk and network speed.
- [ ] **Step 3:** Run `go test ./internal/telemetry/... ./cmd/optimizerui/...` to verify.

---

### Task 2: Add High-Impact Tweaks (Recall Off, Timer Sintonia, VBS Audit)
**Files:**
- Modify: `internal/tweaks/catalog.go`
- Modify: `internal/profiles/useprofiles.go`
- Modify: `internal/tweaks/catalog_test.go`

- [ ] **Step 1:** Implement `privacidade.win11-recall-off` (Disable Windows Recall/Snapshot telemetry in Win11 24H2).
- [ ] **Step 2:** Implement `jogos.hpet-synthetic-timers` (useplatformclock=false / synthetic timers optimization for games).
- [ ] **Step 3:** Run `go test ./internal/tweaks/...` to verify catalog counts and integrity.

---

### Task 3: Implement 1-Click Export System Report (JSON / Markdown)
**Files:**
- Modify: `cmd/optimizerui/app.go`
- Modify: `cmd/optimizerui/frontend/dist/index.html`
- Modify: `cmd/optimizerui/frontend/dist/app.js`

- [ ] **Step 1:** Add `ExportarRelatorioSistema()` in `app.go`.
- [ ] **Step 2:** Add "Exportar Relatório" button in `index.html` header and wire it in `app.js`.
- [ ] **Step 3:** Verify with `node cmd/optimizerui/frontend/app.test.cjs`.

---

### Task 4: Full Verification and Build
**Files:**
- Target: `Optimizer.exe`

- [ ] **Step 1:** Run `rtk go test -count=1 ./...`.
- [ ] **Step 2:** Run `node cmd/optimizerui/frontend/app.test.cjs`.
- [ ] **Step 3:** Build `Optimizer.exe` and commit to Git.
