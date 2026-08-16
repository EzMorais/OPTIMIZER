# Arquitetura de backend — SaaS de assinatura e licenciamento

> Pesquisa com verificação ao vivo de região/preço de provedores (Fly.io, Neon, Supabase) e das APIs reais do Windows para detecção de ambiente corporativo. Onde não foi possível confirmar um número exato (taxas Mercado Pago/Asaas, preço RDS), está marcado **"confirmar antes de lançar"**.

## 1. Stack do servidor

### Framework HTTP em Go

| Framework | Base | Prós | Contras |
|---|---|---|---|
| **Gin** (89k★) | net/http, contexto próprio | Ecossistema enorme, muitos exemplos em PT-BR | Contexto próprio exige adaptador para middlewares net/http puros |
| **Echo** (32,6k★) | net/http, contexto próprio | API limpa, router radix-tree rápido | Comunidade menor |
| **chi** (22,7k★) | net/http puro (`http.Handler` nativo) | Zero dependências, 100% compatível com qualquer middleware net/http (OpenTelemetry, JWT), ~1.000 linhas de core, usado em produção por Cloudflare/Heroku | Ecossistema de middlewares prontos menor |
| Fiber — **não recomendado** | fasthttp, não net/http | Muito rápido | **Lock-in**: SDKs de pagamento, libs JWT/OTel e o servidor de callback do login assumem net/http; adaptador tem overhead e bugs sutis |

**Recomendação**: `net/http` (stdlib, já roteia por método+wildcard desde Go 1.22) **+ chi** por cima só para agrupar rotas/middlewares. Se prioridade é velocidade de time júnior/exemplos prontos, Gin é alternativa legítima — evitar Fiber pelo lock-in.

### Banco de dados (Postgres) — presença confirmada no Brasil

| Provedor | Região BR? | Free tier | Custo inicial |
|---|---|---|---|
| **Neon** | **Sim — `aws-sa-east-1` (São Paulo), confirmado** | Permanente: 100 CU-h/mês, 0,5 GB, autoscale até 2 CU | US$0,106/CU-h + US$0,35/GB-mês; escala a zero quando ocioso |
| **Supabase** | **Sim — `sa-east-1`, confirmado** | Existe, pausa após inatividade | Pro ~US$25/mês (confirmar antes de lançar) — vem com Auth/Storage/Realtime prontos |
| DigitalOcean Managed PG | Não (mais próxima é NY) | Não | US$15,15/mês fixo, mas +120-150ms de latência do Brasil |
| AWS RDS | Sim — `sa-east-1` nativa | Não | São Paulo tem sobretaxa vs. `us-east-1` — cotar em calculator.aws antes de decidir |

**Recomendação**: **Neon em `aws-sa-east-1`** para a Fase 2 — free tier cobre o início, escala a zero em horário ocioso, branching de banco útil para CI/teste sem custo extra. Migrar para RDS/Aurora dedicado em `sa-east-1` só a partir de ~10-30 mil assinantes.

### Onde hospedar

**Fly.io tem região `gru` (São Paulo) confirmada** para compute — app e banco (Neon) ficam na mesma cidade, minimizando latência entre os dois. Limitação: `gru` não tem Postgres gerenciado nativo da Fly, por isso o banco fica no Neon/Supabase de qualquer forma.

- **Início (100–10k assinantes)**: Fly.io (`gru`) + Neon (`sa-east-1`). Paga por segundo, sem servidor ocioso.
- **Crescimento (10k–100k)**: mais máquinas Fly atrás do load balancer nativo, ou AWS ECS/Fargate em `sa-east-1` se precisar de mais controle de rede/VPC.
- **Alternativas sem região BR confirmada**: Render, Railway — DX mais simples, mas +150-200ms de latência.

### Estimativa de custo mensal (infraestrutura pura, sem taxa de pagamento nem equipe)

| Item | 100 assinantes | 10.000 assinantes | 100.000 assinantes |
|---|---|---|---|
| Compute (Fly.io `gru`) | ~US$5-10 | ~US$40-80 | ~US$300-600 |
| Banco (Neon `sa-east-1`) | US$0 (free) | ~US$25-45 | ~US$400-900 (Scale ou RDS reservado) |
| E-mail transacional | US$0 (free) | ~US$20 | ~US$50-100 |
| Observabilidade/logs | US$0 (free) | ~US$25-40 | ~US$150-400 |
| CDN/WAF (Cloudflare) | US$0 | ~US$0-20 | ~US$20-200 |
| **Total** | **~US$10-20/mês (R$50-105)** | **~US$120-220/mês (R$625-1.150)** | **~US$1.000-2.500/mês (R$5.200-13.000)** |

Câmbio: R$5,22/US$. Estimativas de planejamento (assumindo ~50 chamadas/assinante/dia) — rodar teste de carga real antes de comprometer orçamento. Mesmo no cenário mais caro, infraestrutura não é o gargalo de custo em 100k assinantes — aquisição de cliente é.

## 2. Login do app desktop contra o servidor

O .exe **não** é um navegador — reimplementar login dentro dele é trabalho e risco desnecessários. Padrão correto: **OAuth 2.0 Device Authorization Grant (RFC 8628)** — o mesmo mecanismo do Docker Desktop, GitHub CLI, AWS CLI, Netflix em smart TV.

**Fluxo (o que aparece na tela)**:
1. Usuário clica "Entrar" no app.
2. App chama `POST /v1/auth/device/start`, recebe `device_code` (fica só com o app), `user_code` curto tipo `WDJB-MJHT`, e `verification_uri`.
3. App mostra: **"Abra seu navegador em app.empresa.com.br/entrar e digite: WDJB-MJHT"** — e já abre o navegador sozinho com o código pré-preenchido (`verification_uri_complete`); na prática o usuário só confirma um clique.
4. No navegador (ambiente que o usuário já confia), autoriza — login com Google, senha, etc.
5. O app pergunta discretamente ao servidor a cada ~5s (`POST /v1/auth/device/token`) "já autorizou?" — recebe `authorization_pending` até confirmar, depois os tokens.
6. Tela muda sozinha para "Conectado como fulano@email.com".

**Por que é o caminho certo para público leigo e para o modo trabalho**:
- Não exige abrir porta local (`localhost:PORT`) para capturar redirect — esse outro padrão comum quebra em máquinas corporativas com firewall restritivo/proxy, exatamente o cenário da aba "trabalho".
- Navegador já guarda sessão/senha salva — login em 1-2 cliques na segunda vez.
- Nenhuma senha passa pelo .exe — reduz superfície de ataque e desconfiança do leigo ("por que um otimizador está pedindo minha senha?").
- Simples em Go: dois `POST` + tela de espera com spinner.

## 3. Token de acesso e licenciamento

**Access token**: JWT curto, assinado com **EdDSA (Ed25519)** — assinatura pequena, verificação local rápida, `crypto/ed25519` nativo do Go.

```
claims: {
  sub, workspace_id,
  plan: "pro_mensal",              // ou "free"
  subscription_status: "active",
  subscription_expires_at,
  device_id: "sha256_do_fingerprint",
  iat, exp                          // exp curto: 15-30 min
}
```

- App valida a assinatura **localmente** (chave pública embutida) para decidir o que mostrar sem bater no servidor a cada clique — é só conveniência de UX; a proteção real está na seção 4.
- **Refresh token**: opaco, aleatório (256 bits), vida longa, guardado no banco **como hash**, com **rotação a cada uso** — cada troca invalida o anterior. Se um refresh token já usado reaparecer, é sinal de roubo/replay → servidor revoga toda a família daquele dispositivo.

**Limite de dispositivos**: tabela `devices` com `device_fingerprint_hash` — **nunca dado pessoal**. Fingerprint calculado no cliente (`MachineGuid` do registro + número de série da placa-mãe via WMI), tudo em SHA-256 antes de sair da máquina — servidor nunca vê hardware bruto. Semanal/individual = 1 dispositivo; mensal = 2-3. Autoatendimento ao estourar limite: "Você já usa em 3 PCs. Remover um?"

**Tolerância offline**: access token dura 15-30min, mas app deve continuar liberado por mais tempo — guarda-se localmente (protegido contra edição trivial) o timestamp da última confirmação bem-sucedida. Dentro de uma janela (ex.: 72h), libera premium mesmo sem rede. Passado o prazo, volta sozinho para o plano gratuito — nunca trava.

**Vencimento/cancelamento**: via webhook do provedor (seção 9), `subscriptions.status` muda; endpoint de refresh para de emitir novos access tokens. Como o token atual é curto, o recurso desliga sozinho em ~30min — sem precisar de blacklist.

## 4. Proteger o pago sem depender só do .exe

**Princípio central**: qualquer verificação 100% dentro do binário pode, em tese, ser burlada (patch, hook, debugger). O valor real das funcionalidades pagas não pode estar *contido* no .exe — tem que estar *no servidor*, entregue só mediante confirmação de assinatura ativa a cada chamada:

- O **catálogo completo de otimizações avançadas** só é devolvido por `GET /v1/catalog/optimizations` se o token for válido **e** o servidor re-checar o status da assinatura no banco naquele instante — não confia em nenhuma flag que o cliente afirme sobre si mesmo.
- **Explicações por IA**: processamento acontece no servidor, resultado só é devolvido a quem tem assinatura confirmada — não há como "desbloquear" localmente porque o dado nem chega ao disco sem passar pelo servidor primeiro.
- A parte visual (esconder/mostrar botão premium) pode ser adulterada — mas isso só muda a interface, não entrega o conteúdo de valor, que nunca existiu localmente.
- Ofuscação/assinatura de código continuam valendo, mas como "elevar o custo do ataque", não como a proteção de verdade — que é arquitetural (autorização no servidor), não uma corrida de gato-e-rato no executável.

## 5. Modelagem de dados

Conceito unificador desde o início: **"workspace" (conta)**, não pessoa física direto. Pessoa comum = workspace de 1 membro; empresa = workspace de N — evita reescrever o schema na Fase 3.

```sql
accounts                  -- pessoa física logada
  id, email, auth_provider, created_at, deleted_at

workspaces                 -- "conta": pessoal (1) ou empresarial (N)
  id, name, kind (personal|business), created_at

workspace_members          -- já prepara o modo empresarial
  workspace_id, account_id, role (owner|admin|member), invited_at, joined_at

subscriptions
  id, workspace_id, plan_id, status (trialing|active|past_due|canceled),
  provider (mercadopago|asaas), provider_subscription_id,
  seats, current_period_end, created_at, canceled_at

devices
  id, workspace_id, device_fingerprint_hash, display_name,
  os_version, app_version, first_seen_at, last_seen_at, revoked_at

optimization_catalog        -- versionado
  id, slug, version, title, description, category, risk_level,
  applies_to (personal|work|both), min_plan_required,
  definition_jsonb, created_at, deprecated_at

applied_optimizations       -- histórico por device
  id, device_id, optimization_id, optimization_version,
  action (applied|reverted), status (success|failed|rolled_back),
  before_state_jsonb, applied_at, reverted_at, error_detail

payment_events               -- auditoria/reconciliação
  id, subscription_id, provider_event_id, amount, currency,
  status, paid_at, raw_payload_jsonb

anonymous_metrics             -- propositalmente SEM device_fingerprint_hash
  id, install_id_random, optimization_id, optimization_version,
  os_build, outcome, duration_ms, created_at
```

**Caminho para a versão empresarial**: como `devices` já pendura em `workspace_id` (não em `account_id`), e já existe `workspace_members` com papéis, o "painel de TI" da Fase 3 é **apenas uma nova superfície de leitura** filtrada por `workspace_id` + checagem `role=admin`. Sem migração estrutural, só telas e regras de autorização novas.

## 6. Privacidade e LGPD

**Coletar** (sempre anônimo, atrelado a `install_id_random` **diferente** do `device_fingerprint_hash` usado para licenciamento — separação proposital: se telemetria e licenciamento usassem o mesmo identificador, deixaria de ser dado anônimo e viraria dado pseudonimizado, sujeito à LGPD como dado pessoal):
- Métricas antes/depois de uma otimização (RAM, CPU, boot).
- Taxa de sucesso/falha por otimização, cruzada com build do Windows.
- Contagem agregada de uso.
- Diagnóstico de erro/crash — só com opt-in separado.

**Nunca coletar**: nomes/conteúdo de arquivos, histórico de navegação, teclas digitadas, capturas de tela, lista completa de softwares instalados, documentos pessoais, tráfego de rede, geolocalização precisa, dados biométricos. **Atenção ao modo trabalho**: telemetria aí é mais sensível (poderia revelar inventário de software interno de uma empresa) — vem **desligada por padrão** quando detectado ambiente corporativo, só liga por decisão explícita do admin de TI, não do usuário final.

**Consentimento**: checkbox desmarcado por padrão, linguagem simples (não só link de termos), toggle sempre visível com efeito imediato, registro de timestamp+versão da política aceita (rastreabilidade para ANPD). Base legal: LGPD art. 6º (finalidade/minimização/transparência) e art. 12 (dado anonimizado irreversível não é dado pessoal).

## 7. Degradação sem servidor

Princípio: **local-first com reforço da nuvem** — já é a decisão de MVP da Fase 1 (funciona sem nuvem nenhuma). A nuvem só adiciona catálogo atualizado, licenciamento e IA por cima.

- Cache local do último catálogo baixado com sucesso — aplicar/desfazer funcionam 100% offline com o que já está salvo.
- Cache local do último access token válido + timestamp — libera premium na janela de tolerância, depois volta sozinho ao plano gratuito.
- Atualização em segundo plano na inicialização, nunca bloqueia a tela principal; falha mostra banner discreto: *"Sem conexão — usando dados salvos de 12/08"*.
- Retentativa com backoff exponencial + jitter, teto de ~5min.
- Telemetria em fila local limitada (arquivo/SQLite), envia quando a conexão volta, descarta os mais antigos se crescer demais.

## 8. Detectar ambiente corporativo gerenciado

| O que checar | Como (chamável do Go) |
|---|---|
| Domínio AD clássico | `NetGetJoinInformation` (Win32, `Netapi32.dll`) → `NetSetupDomainName`/`NetSetupWorkgroupName`/`NetSetupUnjoined`. Via `golang.org/x/sys/windows`, sem cgo. |
| Azure AD/Entra join, hybrid join, MDM | `dsregcmd /status` via `os/exec`, parsear `AzureAdJoined`, `DomainJoined`, `EnterpriseJoined`. `MdmUrl` indica MDM no tenant, mas **não garante** que este dispositivo específico está gerenciado — é indício, não prova (aviso da própria doc oficial). |
| Enrollment específico Intune/MDM | Registro `HKLM\SOFTWARE\Microsoft\Enrollments\<GUID>\` — subchave com `ProviderID` preenchido (ex. "MS DM Server" = Intune) é o sinal mais confiável. Via `golang.org/x/sys/windows/registry`. |
| GPO aplicada | `HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Group Policy\History` (subchaves = GPOs já aplicadas) + `gpresult /r` via `os/exec` para diagnóstico completo. |
| Sinal geral de política ativa | `HKLM\SOFTWARE\Policies` e `HKCU\SOFTWARE\Policies` — onde GPO e MDM/CSP escrevem valores ADMX-backed. |

**Recomendação de engenharia**: não bastar "é empresa → não faça nada" — checar **por chave específica** no momento de aplicar cada otimização (a chave que seria alterada já está sob `Policies`?), pois uma máquina de domínio pode ter só um punhado de políticas configuradas. Mostrar uma vez, claro: *"Detectamos que este computador é gerenciado pela sua empresa. Algumas otimizações foram ocultadas porque a próxima atualização de política de TI as desfaria mesmo assim."*

## 9. Endpoints principais da API

| Método | Caminho | Envia | Devolve |
|---|---|---|---|
| POST | `/v1/auth/device/start` | `client_id`, `app_version` | `device_code`, `user_code`, `verification_uri(_complete)`, `expires_in`, `interval` |
| POST | `/v1/auth/device/token` | `device_code` | tokens **ou** `authorization_pending`/`slow_down`/`expired_token` (semântica RFC 8628) |
| POST | `/v1/auth/refresh` | `refresh_token`, `device_fingerprint_hash` | novo access token + refresh rotacionado; `403` se assinatura inativa ou limite de dispositivos excedido |
| GET | `/v1/subscription/status` | Bearer | `plan`, `status`, `current_period_end`, `seats_used/total`, `devices` resumido |
| GET | `/v1/catalog/optimizations?since_version=N&target=personal\|work` | Bearer | otimizações **já filtradas no servidor** pelo plano real |
| DELETE | `/v1/devices/{id}` | Bearer | libera assento (autoatendimento) |
| POST | `/v1/telemetry/batch` | array `{install_id_random, optimization_id, outcome, metrics}` — **sem token de conta**, rate-limited por `install_id` | `202 Accepted` |
| POST | `/v1/webhooks/payments/{provider}` | payload assinado Mercado Pago/Asaas | `200 OK` — atualiza `subscriptions`/`payment_events` |

## 10. Roteiro de entrega

**Fase 1 — app pessoal standalone** (prioridade já decidida)
- Instalador `.exe` **assinado digitalmente** (orçamento anual à parte — essencial porque otimizadores de PC são o tipo de app que SmartScreen barra por padrão sem assinatura).
- Motor 100% local: catálogo embutido, aplicar/desfazer com histórico local (SQLite/arquivo), aba pessoal funcional.
- Detecção básica de ambiente corporativo (seção 8) já aqui, mesmo sem nuvem — reaproveitado depois, evita "quebrar" otimizações de TI desde o dia 1.
- Zero dependência de internet.
- **Pronta quando**: instala e roda do zero em Windows limpo sem rede; aplica/desfaz com ganho mensurável; passa por beta testers leigos sem suporte manual; SmartScreen/antivírus não bloqueiam.

**Fase 2 — login + assinatura + premium real**
- Backend Go (seção 1) + Device Authorization Grant (seção 2) + JWT curto/refresh rotativo (seção 3) + interface genérica de pagamento (Mercado Pago/Asaas, webhooks).
- Catálogo/telemetria migram para o servidor; premium depende de confirmação server-side (seção 4).
- **Pronta quando**: usuário cria conta, assina via PIX/cartão, loga, vê premium aparecer; cancela e vê desaparecer no prazo esperado; app continua offline na janela de tolerância; **teste crítico**: modificar o `.exe` localmente não desbloqueia premium de verdade.

**Fase 3 — modo empresarial completo**
- Painel web para TI ver todas as máquinas do workspace, histórico agregado, políticas centralizadas.
- Faturamento por assento (`seats`, já modelado desde a Fase 2).
- Possível SSO corporativo (login via Entra ID da empresa).
- **Pronta quando**: empresa piloto gerencia N máquinas de uma conta, TI aplica política central refletida em todos os dispositivos, cobrança por assento fecha com o provedor.

---

## A confirmar antes de comprometer orçamento/contrato

- Taxas exatas atuais de PIX/cartão do Mercado Pago e Asaas (páginas de preço bloquearam acesso automatizado nesta pesquisa) — confirmar direto em mercadopago.com.br e asaas.com/precos.
- Preço exato do AWS RDS `db.t4g.micro/small` em `sa-east-1` — cotar em calculator.aws na migração da Fase 3.
- Preço atual do plano Pro da Supabase (usado ~US$25/mês de memória, não verificado ao vivo).

## Fontes

[Fly.io regions](https://fly.io/docs/reference/regions/) · [Fly.io pricing](https://fly.io/docs/about/pricing/) · [Neon pricing](https://neon.com/pricing) · [Neon regions](https://neon.com/docs/introduction/regions) · [Supabase regions](https://supabase.com/docs/guides/platform/regions) · [DigitalOcean managed databases](https://www.digitalocean.com/pricing/managed-databases) · [Railway pricing](https://railway.com/pricing) · [RFC 8628 — OAuth 2.0 Device Authorization Grant](https://datatracker.ietf.org/doc/html/rfc8628) · [Microsoft Learn — dsregcmd](https://learn.microsoft.com/en-us/entra/identity/devices/troubleshoot-device-dsregcmd) · [Microsoft Learn — NetGetJoinInformation](https://learn.microsoft.com/en-us/windows/win32/api/lmjoin/nf-lmjoin-netgetjoininformation) · [gin-gonic/gin](https://github.com/gin-gonic/gin) · [go-chi/chi](https://github.com/go-chi/chi) · [labstack/echo](https://github.com/labstack/echo) · [gofiber/fiber](https://github.com/gofiber/fiber)
