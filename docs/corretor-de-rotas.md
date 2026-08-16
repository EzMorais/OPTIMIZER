# Corretor de rotas (premium) — pesquisa e design

> Estudo de ExitLag, NoPing, WTFast e Mudfish/Haste, com verificação direta nos sites oficiais. Objetivo: entender a categoria e desenhar um "hub" ao vivo, decidindo o que é realista construir agora vs. depois.

## O que os concorrentes realmente fazem

Todos os quatro operam (ou operavam) uma **rede própria de servidores intermediários (overlay network)** espalhada geograficamente — o cliente escolhe automaticamente, pacote a pacote, um caminho alternativo até o servidor do jogo, geralmente via túnel UDP com multi-rota. **Nenhum "acelera a internet"** — eles contornam roteamento ruim do backbone da operadora do usuário.

| Produto | Rede | Preço mensal equivalente | Cobertura |
|---|---|---|---|
| **ExitLag** | 1.500+ servidores, 190+ países | US$9,99 avulso → US$6,25 anual → US$3,29 (multiplayer, 5 jogadores) | 3.000+ jogos/apps. Roteamento por IA + **Multi-Internet** (combina até 4 conexões) + Traffic Shaper |
| **NoPing** (brasileira) | 2.000 servidores, 150+ países, **PoPs dedicados em SP/RJ/Brasília/Fortaleza**, suporte nominal a Vivo/Claro/TIM/Oi/Brisanet | US$3,99 mensal → US$2,66 trimestral → US$1,97 anual | 3.000+ jogos. **Multi Connection** (até 5 rotas paralelas) + Multi Internet (até 6 provedores) + IP Block |
| **WTFast** | Não detalhado publicamente; "GPN" (não se chama VPN) | US$6,99–12,99 | 1.000+ jogos. Machine learning para seleção de rota; parcerias com Asus/TP-Link (integração em firmware de roteador) |
| **Mudfish/Haste** | 661 nós premium + 933 públicos gratuitos | US$0,01/mês (pré-pago por tráfego) a US$4,99 (assinatura) / Haste ~US$10 Pro + tier grátis por créditos | Server Chain + Multi Path Mode. **Indício (não 100% confirmado) de que ExitLag adquiriu a Haste** |

**Achado importante sobre o mercado brasileiro**: **NoPing praticamente não tem concorrência doméstica direta** — 3M+ usuários ativos, nota 4,9/5 em 20 mil avaliações, patrocina pro players (coldzera, Sacy, BRTT), 10 anos de mercado. É referência local forte — bom sinal de mercado provado, mas adversário difícil de deslocar num nicho pequeno.

## O painel em tempo real — referência de design

Nenhum concorrente publica screenshots detalhados nas páginas de marketing, mas a lógica de tela é reconstruível com confiança a partir do que cada um descreve como *feature*:

- **Seletor de jogo/servidor no topo** — cada jogo tem rota própria, o hub sempre parte desse contexto.
- **Comparação "antes vs depois" como elemento central** — par de números lado a lado (ex.: 87ms → 12ms; 68 → 130 FPS), vermelho para "antes", verde para "depois". É *o* KPI principal.
- **Métricas ao vivo em cartões**: ping atual, perda de pacotes (%), jitter (ms), às vezes FPS/CPU/GPU — atualizado continuamente durante a partida.
- **Toggle ON/OFF simples**, com status colorido (verde/amarelo/vermelho).
- **Lista de hops** (tipo traceroute visual) — mostra onde está o gargalo.
- **Visualização de Multi-Internet/Multi-Connection** quando várias rotas estão ativas simultaneamente.

**Referência de layout mais transparente publicamente**: **PingPlotter** — gráfico de linha do tempo de latência (histórico contínuo) + tabela hop-a-hop (IP/nome, ping min/max/médio, % perda por salto) + alerta visual no salto problemático ("Problem Detected on Hop 3"). É essencialmente o desenho por trás do "Network Analyzer" do ExitLag, mas documentado abertamente — usar como moodboard.

**Composição de tela recomendada**: topo = seletor de jogo + status geral + toggle; centro = par "ping normal vs otimizado" com gráfico de linha do tempo; abaixo = 3 indicadores secundários (perda, jitter, FPS/CPU/GPU); rodapé/aba avançada = tabela de hops.

## Realidade de infraestrutura

Confirma a preocupação inicial: **isso é uma operação de rede real**, não só software.

**Camada 1 — NaaS genérico/enterprise** (Zenlayer, Megaport, PacketFabric): rede própria com centenas de PoPs, gaming citado como caso de uso, mas **sem preço público, sob contrato**, volume mínimo — nível empresarial, caro e burocrático para começar.

**Camada 2 — infraestrutura específica de jogos**: **Subspace** é o mais próximo do que ExitLag/NoPing constroem por trás — "the dedicated network that makes online competitive gaming work", API de aceleração para desenvolvedores (modelo B2B — diferente do B2C do produto deste projeto). i3D.net é hospedagem/rede para estúdios, não para rotear tráfego de terceiros.

**Camada 3 — bootstrap com nuvem genérica** (DigitalOcean US$4-6/mês, Vultr): **ressalva crítica** — colocar uma VPS numa região não garante rota melhor. O ganho de latência que ExitLag/NoPing vendem vem de **peering e trânsito privilegiado negociado com operadoras/IXs locais** (por isso o NoPing lista Vivo/Claro/TIM/Oi nominalmente), não simplesmente "ter uma máquina lá". Uma VPS genérica pode até ter rota pior que a direta, dependendo do peering.

## O que dar pra construir AGORA, sem rede própria

100% viável e honesto, zero custo de infraestrutura:

- **`tracert`/`pathping` nativos do Windows** contra o IP do servidor do jogo (capturável via `netstat` enquanto o jogo está aberto) — `pathping` já dá perda por salto e latência mais confiável (múltiplas amostras). Reproduz boa parte do "Network Analyzer" do ExitLag.
- **Ping em loop** (ICMP ou UDP no padrão do jogo) calculando média, jitter (desvio-padrão) e % de perda numa janela deslizante — o mesmo gráfico de linha do tempo que os concorrentes mostram, com medição real.
- **Diagnóstico "de quem é a culpa"**: comparar latência do 1º salto (roteador do usuário — se já ruim, é Wi-Fi/rede local) vs. primeiros 2-3 saltos (rede da operadora) vs. resto do caminho (backbone/destino — é aqui que uma rota alternativa ajudaria).
- **Sinais de rede local via Windows**: força do Wi-Fi (`netsh wlan show interfaces`), Wi-Fi vs. cabo, uso de banda por processo, MTU/driver desatualizado.
- **Ações sugeridas com 1 clique**: trocar DNS, testar cabo vs. Wi-Fi, alertar canal Wi-Fi congestionado.

**Em Go**: `golang.org/x/net/icmp` (biblioteca padrão) ou `pro-bing` (lib de mercado) para ping cross-platform; `pathping`/`tracert` nativos via `os/exec` para o V1. Nada disso exige rede própria — é diagnóstico honesto do caminho que já existe.

## Recomendação final

### (a) Fase 1/local — "Hub de Rede" honesto

- Status geral em badge colorido + jogo detectado automaticamente (processos/portas) + servidor identificado.
- Gráfico de linha do tempo ao vivo (2-5 min, 1x/s) de ping até o servidor, jitter como faixa ao redor da linha.
- 3 cartões: ping atual, perda de pacotes, jitter — mesmo padrão "antes/depois" dos concorrentes, mas honesto: "sua rede agora", sem alegar rota otimizada que o produto ainda não oferece.
- Tabela de hops expansível (estilo traceroute), com destaque automático no salto problemático.
- **Diagnóstico em linguagem simples**, gerado automaticamente: *"O problema parece estar no seu Wi-Fi"* ou *"A degradação começa dentro da rede da sua operadora — não há o que ajustar localmente"*. **Este é o diferencial real**: nenhum concorrente entrega diagnóstico claro hoje — eles vendem a solução, não a explicação.
- Ações com 1 clique: trocar DNS, testar cabo vs. Wi-Fi, ver canal Wi-Fi congestionado.

Vendável como premium já no V1: custo de infraestrutura zero (roda só na máquina do usuário), e cria a base de telemetria agregada/anonimizada (com consentimento) que seria valiosa para decidir *onde* colocar servidores próprios na fase seguinte.

### (b) Fase futura/ambiciosa — rota realmente otimizada via rede própria ou alugada

Só compensa quando forem verdadeiras, ao mesmo tempo: (1) base de assinantes grande o bastante para diluir custo fixo (concorrentes cobram US$2-10/usuário/mês — a conta só fecha com dezenas de milhares de ativos); (2) orçamento para 4-6 regiões estratégicas (SP, Fortaleza/Recife para cabos internacionais, Miami, próximo aos datacenters dos jogos mais jogados); (3) capacidade de negociar peering real com operadoras brasileiras — exatamente a vantagem que o NoPing já tem e não é fácil replicar rápido; (4) cobertura de jogos relevante (líderes cobrem 1.000-3.000+).

**Recomendação prática**: **não construir a rede do zero.** Faz mais sentido parceria/revenda de capacidade com provedor B2B (Subspace, Zenlayer) ou acordo white-label com NoPing/Mudfish (que já tem modelo pay-per-traffic barato), em vez de negociar contrato enterprise direto. Trilha realista: lançar (a) agora → usar a telemetria de latência coletada para provar, com dados reais, em quais rotas/regiões o problema é mais frequente entre os assinantes → só então decidir, com números em mãos, entre parceria ou POPs próprios via VPS barata (~US$5-6/mês/nó) nas 2-3 rotas mais problemáticas identificadas — testando o ganho real antes de prometer ao usuário final.

---

## Fontes

[exitlag.com](https://exitlag.com) · [ExitLag pricing](https://exitlag.com/en/pricing) · [noping.com](https://noping.com/) · [NoPing pricing](https://noping.com/pricing) · [wtfast.com](https://www.wtfast.com) · [mudfish.net](https://mudfish.net) · [Haste](https://haste-913349.webflow.io) · [Zenlayer](https://www.zenlayer.com) · [Megaport](https://www.megaport.com) · [PacketFabric](https://www.packetfabric.com) · [i3D.net](https://www.i3d.net) · [Subspace](https://www.subspace.com) · [DigitalOcean pricing](https://www.digitalocean.com/pricing/droplets) · [PingPlotter](https://www.pingplotter.com) · [LagoFast](https://www.lagofast.com/pt-br/)

**Não confirmado, revisar antes de citar publicamente**: detalhamento exato dos planos por tier do WTFast; confirmação formal da aquisição Haste↔ExitLag; contagem atual de regiões da Vultr.
