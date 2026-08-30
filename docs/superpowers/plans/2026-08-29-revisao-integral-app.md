# Revisão integral do app Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Tornar todos os fluxos do Optimizer previsíveis, observáveis e seguros, eliminando estados presos e garantindo aplicação reversível.

**Architecture:** O backend Go continua sendo a autoridade para coleta e alterações do Windows. O frontend vanilla terá estados explícitos de idle/loading/success/error, uma única função por fluxo e bindings conferidos contra o HTML. Cada fase terá testes de regressão antes da implementação e validação visual pelo Orca.

**Tech Stack:** Go, Wails v2, HTML/CSS/JavaScript vanilla, Node test runner, Orca computer-use.

**Spec:** `docs/superpowers/specs/2026-08-29-revisao-integral-app-design.md`

## Global Constraints

- Simulação nunca grava no Registro ou na rede.
- Alterações reais continuam passando por elevação, histórico e rollback existentes.
- Nenhuma chamada do Windows pode deixar a UI bloqueada indefinidamente.
- Sensores ausentes devem aparecer como indisponíveis, nunca como valores inventados.
- Preservar mudanças locais existentes e não executar comandos destrutivos durante testes.

### Task 1: Auditoria de bindings e estados de carregamento

**Files:**
- Modify: `cmd/optimizerui/frontend/dist/index.html`, `cmd/optimizerui/frontend/dist/app.js`
- Test: `cmd/optimizerui/frontend/app.test.cjs`

- [ ] Escrever teste que verifica que os seletores usados pelos fluxos públicos existem no HTML e que Startup/Discos/DNS deixam o estado de loading após resolução.
- [ ] Rodar o teste e confirmar falha se houver seletor divergente ou estado preso.
- [ ] Corrigir somente os seletores, handlers e transições de estado identificados.
- [ ] Rodar o teste específico e a suíte frontend completa.

### Task 2: Telemetria compatível com o Gerenciador de Tarefas

**Files:**
- Modify: `internal/telemetry/provider.go`, `cmd/optimizerui/frontend/dist/app.js`
- Test: `internal/telemetry/*_test.go`, `cmd/optimizerui/frontend/app.test.cjs`

- [ ] Escrever teste para PID real, percentual limitado entre 0 e 100 e falha estruturada sem zerar a tela inteira.
- [ ] Rodar o teste para observar RED.
- [ ] Implementar coleta por contador percentual formatado e normalização por processadores lógicos.
- [ ] Garantir polling sem sobreposição e limpeza do timer ao trocar de aba.
- [ ] Rodar testes Go/frontend e validar dois ciclos reais pelo Orca.

### Task 3: Perfis, aplicação e rollback

**Files:**
- Modify: `cmd/optimizerui/frontend/dist/app.js`, `cmd/optimizerui/app.go`
- Test: `cmd/optimizerui/frontend/app.test.cjs`, `cmd/optimizerui/app_test.go`

- [ ] Escrever testes para simulação sem escrita, aplicação com controles restaurados após erro e atualização dos cards após sucesso.
- [ ] Rodar os testes e confirmar as falhas esperadas.
- [ ] Corrigir handlers e bindings sem ampliar permissões dos perfis.
- [ ] Validar aplicação apenas em modo seguro/simulação durante a inspeção visual.

### Task 4: Rede, MTU/MSS e DNS

**Files:**
- Modify: `internal/netdiag/*.go`, `cmd/optimizerui/frontend/dist/app.js`, `cmd/optimizerui/frontend/dist/index.html`
- Test: `internal/netdiag/*_test.go`, `cmd/optimizerui/frontend/app.test.cjs`

- [ ] Escrever testes para limites de MTU, cálculo MSS IPv4/IPv6, adaptador ausente e comando de simulação.
- [ ] Rodar os testes em RED.
- [ ] Implementar validação e mensagens de erro/retry sem alterar configurações no teste.
- [ ] Validar DNS ativo, editor e simulação pelo Orca.

### Task 5: Diagnóstico, histórico, terminal e inspector

**Files:**
- Modify: `cmd/optimizerui/frontend/dist/app.js`, `cmd/optimizerui/frontend/dist/app.css`, `cmd/optimizerui/app.go`
- Test: `cmd/optimizerui/frontend/app.test.cjs`, `cmd/optimizerui/app_test.go`

- [ ] Escrever testes para histórico, visualizador de comandos, abertura não obstrutiva e fechamento de estados de erro.
- [ ] Rodar os testes em RED.
- [ ] Implementar drawer lateral, mensagens uniformes e ações de retry sem abrir consoles visíveis.
- [ ] Validar as telas Diagnóstico e Histórico pelo Orca.

### Task 6: Verificação integrada e entrega

- [ ] Rodar `go test -p 1 ./...`.
- [ ] Rodar `go vet ./...`, `node --test cmd/optimizerui/frontend/app.test.cjs`, `node --check cmd/optimizerui/frontend/dist/app.js` e `git diff --check`.
- [ ] Compilar e instalar com `scripts\instalar.ps1`.
- [ ] Abrir o app pelo Orca e verificar loading, visão geral, telemetria, rede, perfis, diagnóstico e histórico.
- [ ] Registrar limitações ambientais sem declarar sucesso não verificado.
