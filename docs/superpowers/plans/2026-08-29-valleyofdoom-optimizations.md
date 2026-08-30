# Valleyofdoom PC-Tuning & AutoGpuAffinity Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Integrate evidence-based kernel, USB interrupt moderation, GPU MSI mode, background window rate, and gaming FSE tweaks inspired by `valleyofdoom/PC-Tuning`, `AutoGpuAffinity`, and `TimerResolution` into the Optimizer catalog and interface.

**Architecture:** Add curated, safe, and verifiable registry tweaks in `internal/tweaks/catalog.go`, update technical documentation in `docs/catalogo/`, and verify 100% test coverage across Go and Node frontend test runners.

**Tech Stack:** Go 1.26, Windows Registry (`winreg`), Wails v2 frontend, Node.js unit test runner (`app.test.cjs`), PowerShell installer.

**Spec:** `docs/catalogo/registro.md` and `docs/catalogo/hardware-energia-gpu.md`

## Global Constraints

- No placebo or harmful registry settings.
- Every tweak must have technical evidence and caveat when not recommended by default.
- Machine-level (`HKLM`) tweaks must require Administrator privileges (`RequiresAdmin = true`).
- 100% backwards compatibility and safe rollback support via transactional snapshots.

---

### Task 1: Add New Curated Tweaks to Catalog

**Files:**
- Modify: `internal/tweaks/catalog.go`
- Test: `internal/tweaks/catalog_test.go`

**Interfaces:**
- Consumes: `regtweak.Spec`, `regtweak.DWord`, `winreg.HKLM`, `winreg.HKCU`
- Produces: 4 new tweaks in `entries()`:
  - `entrada.usb-xhci-interrupt-moderation`
  - `sistema.background-window-message-rate`
  - `jogos.gamedvr-fse-behavior`
  - `hardware.msi-mode-gpu`

- [ ] **Step 1: Write the failing test for new tweaks in catalog_test.go**
- [ ] **Step 2: Run `go test ./internal/tweaks/...` to verify failure**
- [ ] **Step 3: Implement the 4 new tweaks in `internal/tweaks/catalog.go`**
- [ ] **Step 4: Run `go test ./internal/tweaks/...` to verify PASS**
- [ ] **Step 5: Commit changes**

---

### Task 2: Update Technical Catalog Documentation

**Files:**
- Modify: `docs/catalogo/registro.md`
- Modify: `docs/catalogo/hardware-energia-gpu.md`

**Interfaces:**
- Consumes: Documentation findings from `valleyofdoom/PC-Tuning`, `AutoGpuAffinity`, `TimerResolution`
- Produces: Detailed technical backing, benchmarks, keys, and caveats for each new tweak.

- [ ] **Step 1: Add XHCI IMOD, BackgroundWindowMessageRate, and GameDVR FSE sections to `docs/catalogo/registro.md`**
- [ ] **Step 2: Add MSI Mode and GPU affinity benchmarks to `docs/catalogo/hardware-energia-gpu.md`**
- [ ] **Step 3: Verify all evidence anchors match `Evidence` fields in `catalog.go`**
- [ ] **Step 4: Commit documentation updates**

---

### Task 3: Validate App Controller and Frontend Integration

**Files:**
- Modify: `cmd/optimizerui/app_test.go`
- Test: `cmd/optimizerui/frontend/app.test.cjs`

**Interfaces:**
- Consumes: `App.Diagnosticar`, `ItemUI`
- Produces: Verified item rendering and filtering in frontend.

- [ ] **Step 1: Run Go tests for optimizerui: `go test ./cmd/optimizerui/...`**
- [ ] **Step 2: Run frontend test runner: `node cmd/optimizerui/frontend/app.test.cjs`**
- [ ] **Step 3: Verify all 29+ frontend tests pass**
- [ ] **Step 4: Commit verification updates**

---

### Task 4: Recompile and Verify Installer

**Files:**
- Modify: `scripts/instalar.ps1`
- Test: Build and test execution

- [ ] **Step 1: Run silent installer `powershell -ExecutionPolicy Bypass -File scripts/instalar.ps1 -Silent`**
- [ ] **Step 2: Verify `Optimizer.exe` and `optimizerctl.exe` run properly**
- [ ] **Step 3: Run `optimizerctl listar` to confirm 55 tweaks listed**
- [ ] **Step 4: Commit final build state**

