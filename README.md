# Optimizer

SaaS de otimização de performance para Windows — app desktop em Go, com assinatura semanal/mensal, versão free e premium, e perfis separados para uso pessoal e computador de trabalho.

## Status

Pesquisa técnica e de mercado concluída. **O app desktop já roda**: interface gráfica em Wails v2 sobre o motor completo — catálogo com estado real lido da máquina, aplicar/simular/verificar/desfazer com histórico auditável, ponto de restauração, as abas Pessoal e Trabalho e a medição de MTU da conexão.

O que ainda não existe: login/cobrança, automação contínua, corretor de rotas e o instalador assinado. Próximo passo em [`docs/decisoes.md`](./docs/decisoes.md#próximo-passo-físico).

### Instalar nesta máquina para testar

```
powershell -ExecutionPolicy Bypass -File scripts\instalar.ps1
```

Compila e instala em `%LOCALAPPDATA%\Programs\Optimizer` (por usuário, sem exigir administrador, sem serviço em segundo plano) e cria os atalhos. Para remover: `scripts\desinstalar.ps1` — que **não** desfaz as otimizações, isso se faz pelo próprio app antes.

## Por que este produto existe

A categoria "otimizador de PC" (CCleaner, IObit Advanced SystemCare, AVG TuneUp) tem reputação ruim merecida: alarme falso ("encontramos 3.847 problemas"), paywall no básico, histórico de PUP/malware. O concorrente mais perigoso hoje é gratuito e oficial — o **Microsoft PC Manager** — mas ele não faz automação contínua nem separa perfil pessoal de trabalho. A tese deste produto: medir de verdade, explicar em português claro, mostrar risco, sempre permitir desfazer — e cobrar pela automação contínua e pelas duas abas de uso, não pelo diagnóstico básico.

Ver [`docs/pesquisa-mercado.md`](./docs/pesquisa-mercado.md) para a pesquisa completa de concorrentes, licenças open-source e monetização no Brasil.

## Documentação

| Documento | Conteúdo |
|---|---|
| [`docs/decisoes.md`](./docs/decisoes.md) | Requisitos do cliente, decisões de produto e arquitetura, próximo passo |
| [`docs/pesquisa-mercado.md`](./docs/pesquisa-mercado.md) | Concorrentes, benchmark free/premium, pagamento no Brasil, licenças OSS |
| [`docs/catalogo-windows.md`](./docs/catalogo-windows.md) | Índice do catálogo técnico de otimizações (6 domínios, ver `docs/catalogo/`) |
| [`docs/catalogo/top-15-e-armadilhas.md`](./docs/catalogo/top-15-e-armadilhas.md) | As 15 otimizações de melhor ganho-real/risco + texto pronto de UI para tweaks recusados |
| [`docs/arquitetura-app-desktop.md`](./docs/arquitetura-app-desktop.md) | Framework de UI, acesso ao Windows, elevação, motor de tweaks, empacotamento, testes |
| [`docs/arquitetura-backend.md`](./docs/arquitetura-backend.md) | Stack do servidor, login do app, licenciamento, schema de dados, LGPD, roteiro |
| [`docs/corretor-de-rotas.md`](./docs/corretor-de-rotas.md) | Pesquisa de ExitLag/NoPing/WTFast e design do "hub de rede" premium |

## Decisões centrais (resumo)

- **App desktop**: Go + **Wails v2** (HTML/CSS/JS na UI, sem CGO). Elevação de privilégio sob demanda em lote — nunca um serviço sempre-elevado.
- **Motor de otimização**: cada tweak implementa a interface `Tweak` (`Check`/`Apply`/`Revert`/`Verify`), sempre código local e compilado. Catálogo de metadados pode atualizar via manifesto remoto assinado, mas nunca executa código de terceiros.
- **Backend**: Go + Postgres (Neon, `sa-east-1`), hospedado no Fly.io (`gru`, São Paulo). Login do app via OAuth Device Authorization Grant (RFC 8628) — igual Docker Desktop/GitHub CLI, funciona atrás de firewall corporativo.
- **Monetização**: free generoso (diagnóstico completo, tweaks manuais, ponto de restauração e desfazer sempre grátis) + premium (automação contínua, as duas abas simultâneas, histórico, corretor de rotas). Preço hipótese: semanal R$7,90 · mensal R$19,90 · anual R$149,90 · empresarial R$14,90/máquina/mês.
- **Segurança de produto**: conservador por padrão, camada avançada opt-in, nunca mexe em Defender/Windows Update/mitigações de segurança em nenhum perfil ou plano.
- **Rede**: o produto não vende "acelerador de internet". O ajuste de MTU só aparece depois de **medir** o maior pacote que atravessa a conexão (eco ICMP "não fragmentar", a mesma técnica do `ping -f -l`), e a UI mostra os comandos para o usuário conferir na mão — ver [`docs/catalogo/rede.md`](./docs/catalogo/rede.md#como-o-produto-implementa-medição-nunca-chute--internalnetdiag).

## Estrutura do repositório

```
cmd/optimizerui/         # app desktop (Wails v2): bindings Go + frontend embutido no .exe
cmd/optimizerctl/         # CLI interna de diagnóstico/dev — não distribuída ao usuário
internal/tweak/            # contrato Tweak, Meta (catálogo como dado) e Registry
internal/tweaks/            # catálogo embutido; regtweak/ implementa o padrão "tweak de registro"
internal/winreg/             # fronteira mockável com o registro (Live no Windows, Fake nos testes)
internal/engine/              # orquestra diagnosticar → aplicar → verificar → desfazer
internal/history/              # histórico JSONL append-only (é o que torna o desfazer possível)
internal/netdiag/               # medição de rede: MTU real do caminho + ajuste do adaptador
internal/restore/                # ponto de restauração (SRSetRestorePointW)
internal/elevate/ console/        # administrador e elevação sob demanda; UTF-8 no terminal
scripts/                           # instalar/desinstalar nesta máquina
docs/                              # pesquisa e decisões (ver tabela acima)
```

## Desenvolvimento

Requisitos: Go 1.26+. Verificado nesta máquina: Go 1.26.5, Node 24 (necessário para a Wails CLI mais adiante).

```
go build ./...
go test ./...
```

### CLI interna (`optimizerctl`)

Mesma engrenagem que a interface gráfica vai usar, exposta em linha de comando:

```
go run ./cmd/optimizerctl diagnosticar              # lê o estado real, não altera nada
go run ./cmd/optimizerctl listar --detalhado        # catálogo com ressalva e base técnica de cada item
go run ./cmd/optimizerctl aplicar --recomendados --simular
go run ./cmd/optimizerctl desfazer --tudo
go run ./cmd/optimizerctl mtu --detalhado           # mede o MTU do caminho e explica
```

Os testes usam um registro falso em memória — `go test ./...` não toca no Windows da máquina de desenvolvimento.
