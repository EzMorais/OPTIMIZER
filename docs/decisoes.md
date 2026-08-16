# Decisões de produto

## Requisitos do cliente (literais)

- Otimizar desde internet até registros do Windows, voltado a performance
- Todas as informações de forma simples; navegação completa e simples
- Uma ajuda para extrair o melhor desempenho do PC
- Escalável, vendido por mensalidade ou semanal
- Feito em **Go**, entregue como app executável (.exe)
- Aba separada para computadores de trabalho, voltada a outro público
- Estudar apps existentes e registros documentados pelo próprio Windows
- Cada otimização com explicação e interruptor liga/desliga; auto-intuitivo
- Versão free e paga, com o projeto dividido e as features premium na versão paga
- Recurso premium: corretor de rotas de internet (inspirado em ExitLag/NoPing), com hub ao vivo

## Decisões tomadas com o cliente

| Decisão | Escolha |
|---|---|
| Assistente de desempenho | Híbrido: diagnóstico e recomendação por regras determinísticas locais (offline, custo zero, nunca alucina). IA na nuvem só redige explicações e tira dúvidas — recurso premium. A IA nunca decide sozinha o que aplicar. |
| Cobrança | Brasil: PIX + cartão (Mercado Pago ou Asaas, a confirmar taxas exatas do Pix Automático antes de fechar). Camada de assinatura construída atrás de interface genérica. |
| Agressividade | Conservador com camada avançada opt-in. Padrão só aplica o que é seguro e reversível. Nunca desativa Defender, Windows Update ou mitigações de segurança — nem no avançado, nem no perfil pessoal, nem no premium. Ponto de restauração automático antes de qualquer alteração. |
| MVP | App pessoal completo primeiro (diagnóstico real, nota de saúde, catálogo com liga/desliga, aplicar/desfazer, as duas abas). Login e cobrança entram depois. |

## Princípios de produto (derivados da pesquisa de mercado)

Ver [`pesquisa-mercado.md`](./pesquisa-mercado.md) para o relatório completo. Decisões diretas dessa pesquisa:

- **Nunca inflar contagem de "problemas encontrados"** — é o mecanismo central do scareware e do padrão escuro replicado por toda a categoria paga (CCleaner, IObit, AVG, Norton, iolo). Mostrar números reais e medidos, sempre.
- **Limpeza manual, tweaks individuais com toggle, ponto de restauração automático e desfazer ficam grátis e ilimitados** — nunca atrás de paywall. É rede de segurança básica, não recurso premium.
- **Limpeza de registro não entra no catálogo como ganho de performance** — é placebo confirmado por múltiplas fontes; a própria Microsoft não recomenda.
- **O diferencial real**: duas abas lado a lado (Pessoal + Trabalho) — nenhum concorrente pesquisado (pago, grátis ou open-source) tem isso. Automação contínua e silenciosa é o que o Microsoft PC Manager (gratuito, oficial) estruturalmente não faz — é aí que o produto compete.
- **Licenças open-source**: WinUtil, Sophia Script, Win11Debloat e PowerToys são MIT (seguros até para reuso leve de código com atribuição). Optimizer (hellzerg) é GPL-3.0 e privacy.sexy é AGPL-3.0 — usar apenas como fonte de conhecimento (quais chaves/serviços existem), nunca copiar código ou dados. Catálogo reescrito do zero.
- **Preço sugerido (hipótese inicial, não testada com o público)**: semanal R$7,90 · mensal R$19,90 · anual R$149,90 · empresarial R$14,90/máquina/mês. Cobrança semanal só com cobrança visível, nunca como "trial disfarçado".

## Arquitetura decidida

Pesquisa completa e bibliotecas verificadas ao vivo em [`arquitetura-app-desktop.md`](./arquitetura-app-desktop.md) (app) e [`arquitetura-backend.md`](./arquitetura-backend.md) (backend). Resumo:

| Camada | Escolha |
|---|---|
| GUI do app desktop | **Wails v2** (v3 ainda em beta) — HTML/CSS/JS + Go, sem CGO, .exe pequeno |
| Acesso ao Windows | `golang.org/x/sys/windows` direto (registro, serviços); `yusufpapurcu/wmi` para WMI; evitar `powershell.exe` sempre que possível |
| Elevação de privilégio | Elevação sob demanda em lote (nunca serviço sempre-elevado — evita alvo permanente de escalonamento de privilégio) |
| Motor de tweaks | Interface `Tweak` (Check/Apply/Revert/Verify) sempre local e compilada; catálogo de metadados atualizável via manifesto remoto **assinado** (nunca código executável remoto) |
| Empacotamento | Inno Setup, assinatura de código (Azure Artifact Signing ou certificado OV/EV), auto-update via `minio/selfupdate`, submissão ao winget |
| Backend | Go + `net/http`/chi, Postgres (Neon, região `sa-east-1`/São Paulo), hospedado no Fly.io (região `gru`/São Paulo) |
| Login do app | OAuth 2.0 Device Authorization Grant (RFC 8628) — mesmo padrão do Docker Desktop/GitHub CLI, funciona atrás de firewall corporativo |
| Licenciamento | JWT curto (EdDSA) + refresh token rotativo; conteúdo premium sempre validado server-side, nunca só no binário |
| Detecção de ambiente corporativo | `NetGetJoinInformation`, `dsregcmd /status`, registro de Enrollments/GPO — nunca sobrescrever política de TI |

Catálogo técnico completo (6 domínios verificados, com testes empíricos ao vivo) em [`catalogo-windows.md`](./catalogo-windows.md). Pesquisa do corretor de rotas (recurso premium) em [`corretor-de-rotas.md`](./corretor-de-rotas.md).

## Ambiente de desenvolvimento verificado

- Windows 11 Pro build 26200 (25H2), x64, i7-8700 (6C/12T), 16 GB RAM
- Go 1.26.5 windows/amd64 · Node 24.18.1 · Python 3.14.6 · git 2.55
- Sem .NET, sem Rust instalado — descarta frameworks dependentes de WPF/WinUI/Tauri

## O que já está construído

O motor completo do produto roda hoje, sem interface gráfica, e é exercitado pela CLI interna `optimizerctl`:

| Camada | Estado |
|---|---|
| Contrato `Tweak` + catálogo como dado (`Meta`, manifesto remoto) | Pronto, com teste travando que manifesto remoto **não** cria otimização nova nem rebaixa exigência de administrador |
| Fronteira com o registro (`winreg`) | Pronta: `Live` no Windows, `Fake` em memória para teste — `go test ./...` não toca no Windows real |
| Catálogo inicial (9 itens de registro) | Pronto, com valor padrão do Windows, ressalva honesta e link para a seção do catálogo técnico que embasa cada item |
| Motor (diagnosticar → aplicar → verificar → desfazer, em lote, com simulação) | Pronto |
| Histórico JSONL append-only + desfazer/desfazer tudo | Pronto |
| Ponto de restauração (`SRSetRestorePointW`) | Implementado; **falta validar em VM** (nunca testado contra o System Restore da máquina de dev, conforme a regra de teste da arquitetura) |
| Medição de MTU do caminho + ajuste do adaptador | Pronto e validado ao vivo contra o `ping -f -l` do próprio Windows |
| Detecção de administrador e elevação sob demanda | Prontas |
| **Interface gráfica (Wails v2)** | **Pronta**: três telas (Otimizações com as abas Pessoal/Trabalho, Internet e Histórico), aplicar em lote, simular, desfazer tudo |

Decisões de implementação que nasceram aqui e viraram regra:

- **O produto mostra o número que mediu, não o padrão que ele supõe.** O texto de estado sempre inclui o valor lido ao vivo (`Medimos agora: MenuShowDelay = 100 (alvo 10)`) — descoberto na prática, quando o app dizia "está no padrão do Windows (0,4 s)" numa máquina cujo valor real era outro.
- **Nunca piorar um ajuste que o usuário já deixou melhor que o alvo** (`AlreadyOptimized`).
- **Desfazer é restaurar o estado anterior exato**, inclusive apagar o valor quando ele não existia antes — gravar o "valor padrão" no lugar não é a mesma coisa.
- **Só 2 dos 9 itens são recomendados por padrão.** Há teste automatizado impedindo que a proporção de "recomendados" cresça sem critério — é a barreira técnica contra o padrão escuro da categoria.

## Próximo passo físico

A interface gráfica foi feita **sem a Wails CLI**: o Wails v2 entrou como biblioteca Go, o frontend é HTML/CSS/JS puro embutido por `go:embed`, e o build é um `go build -tags "desktop,production"` comum. Isso dispensou Node no build e mantém o .exe em ~11 MB.

Em ordem de valor, o que vem agora:

1. **Itens de inicialização e plano de energia** — os dois maiores ganhos reais do top 15, ainda sem implementação.
2. **Validar o ponto de restauração em VM** — está implementado e exposto na UI, mas nunca foi executado contra o System Restore real.
3. **Empacotamento**: ícone e metadados de versão (go-winres), instalador Inno Setup e assinatura de código — sem isso o SmartScreen mostra "editor desconhecido".
4. **Login, assinatura e automação contínua.**
