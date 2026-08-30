# Estabilização Completa do Optimizer — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Manter as 51 otimizações do catálogo e tornar linguagem, recomendações por hardware, bindings, aplicação, edição, simulação e rollback verificáveis e previsíveis.

**Architecture:** Go permanece autoridade para diagnóstico e alterações do Windows. O frontend vanilla terá contratos de binding únicos, estados explícitos e mensagens UTF-8. Recomendações serão calculadas pelo hardware detectado, mas sempre exigirão confirmação do cliente.

**Tech Stack:** Go 1.26, Wails v2, HTML/CSS/JavaScript vanilla, Node test runner, testes Windows e Orca.

**Spec:** `docs/superpowers/specs/2026-08-29-revisao-integral-app-design.md`

## Global Constraints

- Manter as 51 otimizações existentes e seus IDs públicos.
- Simulação nunca grava Registro, rede, serviços ou arquivos de configuração.
- Aplicação real exige confirmação, histórico, verificação e rollback.
- Sensor ausente deve ser exibido como indisponível, nunca como valor estimado.
- Recomendações são anexadas ao diagnóstico e não são aplicadas automaticamente.
- Todo texto da interface e documentação deve permanecer em UTF-8 válido.
- Não remover alterações locais nem executar comandos destrutivos.

### Task 1: Corrigir codificação e contrato visual

**Files:**
- Modify: `README.md`, `cmd/optimizerui/frontend/dist/index.html`, `cmd/optimizerui/frontend/dist/app.js`, `cmd/optimizerui/frontend/dist/app.css`, textos em `docs/`
- Test: `cmd/optimizerui/frontend/app.test.cjs`

- [ ] Escrever teste que rejeita sequências mojibake (`Ã`, `Â`, `â`) em HTML, JavaScript, CSS, README e textos de documentação alterados.
- [ ] Rodar o teste e confirmar que ele falha com os textos atuais.
- [ ] Converter os arquivos textuais para UTF-8 real preservando conteúdo, IDs, classes e scripts.
- [ ] Rodar teste de codificação, `node --check` e `git diff --check`.

### Task 2: Recomendações por computador

**Files:**
- Modify: `cmd/optimizerui/app.go`, `internal/telemetry/types.go`, `cmd/optimizerui/frontend/dist/app.js`
- Test: `cmd/optimizerui/app_test.go`, `internal/telemetry/telemetry_test.go`, `cmd/optimizerui/frontend/app.test.cjs`

- [ ] Escrever testes para GPU NVIDIA ausente, pouca RAM, GPU presente, hardware indisponível e preservação dos 51 IDs.
- [ ] Rodar os testes e confirmar as falhas esperadas.
- [ ] Implementar o perfil de hardware cacheado e a decisão `recomendado/motivoRecomendacao` sem remover itens do catálogo.
- [ ] Mostrar o motivo da recomendação no card e manter seleção pronta para confirmação, sem aplicação automática.
- [ ] Rodar testes Go/frontend e validar limites de recomendação.

### Task 3: Fluxos de leitura, edição e aplicação

**Files:**
- Modify: `cmd/optimizerui/frontend/dist/app.js`, `cmd/optimizerui/frontend/dist/index.html`
- Test: `cmd/optimizerui/frontend/app.test.cjs`

- [ ] Escrever testes de rejeição para Startup, Discos, MTU, DNS, histórico, perfil de rede e perfil de uso.
- [ ] Rodar os testes em RED.
- [ ] Corrigir cada handler com estado de loading, mensagem de erro e restauração garantida dos controles.
- [ ] Garantir que editores validem antes de aplicar e que a simulação não chame caminhos de escrita.
- [ ] Rodar a suíte frontend completa e a checagem de sintaxe.

### Task 4: Backend, rollback e telemetria

**Files:**
- Modify: `cmd/optimizerui/app.go`, `internal/engine/engine.go`, `internal/netdiag/mtutweak.go`, `internal/telemetry/provider.go`, `internal/telemetry/collector.go`
- Test: `cmd/optimizerui/app_test.go`, `internal/engine/engine_test.go`, `internal/netdiag/*_test.go`, `internal/telemetry/*_test.go`

- [ ] Escrever testes de reabertura do histórico, rollback de MTU, percentuais fora de faixa, PID/memória e sensores indisponíveis.
- [ ] Rodar os testes em RED.
- [ ] Implementar reidratação de tweaks dinâmicos, cancelamento/limites de operações e normalização no limite público.
- [ ] Preservar PID e memória na consolidação de benchmarks e separar indisponível de zero.
- [ ] Rodar `go test -p 1 ./...` e `go vet ./...`.

### Task 5: Validação integrada e aplicação Windows

**Files:**
- Modify: `scripts/instalar.ps1`, somente se a compilação/instalação exigir correção
- Test: todos os testes existentes e checklist manual documentado em `docs/superpowers/`

- [ ] Compilar com `go build ./...`.
- [ ] Executar testes Go, frontend, `go vet`, `node --check` e `git diff --check`.
- [ ] Instalar com `scripts\\instalar.ps1` sem alterar configurações de otimização automaticamente.
- [ ] Validar no Windows: diagnóstico, recomendados, simulação, edição de rede, DNS, MTU, aplicação confirmada, histórico e rollback.
- [ ] Validar linguagem visual e ausência de loading preso.
- [ ] Registrar limitações ambientais quando Orca/Windows não estiverem disponíveis.

