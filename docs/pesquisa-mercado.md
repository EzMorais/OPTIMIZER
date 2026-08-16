# Pesquisa de Mercado — SaaS de Otimização de PC Windows (Brasil)

> Metodologia: pesquisa web real (WebSearch/WebFetch), 40+ buscas distintas e ~10 fetches diretos de fontes primárias (GitHub LICENSE, páginas oficiais de preço da Stripe/Asaas/Norton, Banco Central, Câmara dos Deputados). Todo número marcado **[NÃO CONFIRMADO]** não pôde ser verificado em fonte primária — tratar como hipótese, não como fato para decisão de preço final.
>
> Cotação de referência usada nas conversões: **1 USD ≈ R$ 5,22** (agosto/2026, Investing.com). Conversões são só para dar ordem de grandeza.

---

## Sumário executivo

1. O concorrente mais perigoso não é pago — é o **Microsoft PC Manager**, gratuito, oficial, pré-confiável, e que já cobre ~70% do que os leigos esperam de um "otimizador" (limpeza, boost de memória, storage, saúde do sistema).
2. Todo o setor pago (CCleaner, IObit, Wise Care, AVG, Norton, iolo, Ashampoo) segue o mesmo molde: **"diagnóstico grátis, correção paga"** — e isso tem custo reputacional real (histórico de PUP, scareware-by-association, hack do CCleaner em 2017, notas baixíssimas em sites de reclamação).
3. As ferramentas open-source (WinUtil, Sophia Script, Win11Debloat, PowerToys) são **seguras como referência de conhecimento e até de código (licença MIT)**. Duas exceções importantes exigem cuidado jurídico: **Optimizer (hellzerg) é GPL-3.0** e **privacy.sexy é AGPL-3.0** — copyleft forte, não copiar código/dados, só usar como fonte de conhecimento (quais chaves de registro existem).
4. No Brasil, o pagamento recorrente está mudando rápido: **Pix Automático** já é obrigatório para bancos desde 16/06/2025, é sempre grátis para quem paga, e tarifa (se houver) é negociada entre a empresa e seu provedor de pagamento — não há tabela pública fixa.
5. Cobrança **semanal** existe no Brasil (ex.: apps de VPN, scanner), mas é fortemente associada a dark pattern e reclamação de cobrança indevida — recomendo tratar como oferta secundária, nunca como padrão/isca.
6. O wedge de produto mais defensável encontrado na pesquisa inteira: **nenhum concorrente — pago, gratuito ou open-source — oferece dois perfis lado a lado (Pessoal vs Trabalho)**. Isso é diferenciação real, não commodity.

---

## PARTE 1 — Concorrentes comerciais

### Tabela comparativa

| Produto | Free permanente? | Preço pago confirmado (USD, lista) | Equivalente BRL/mês (ref. R$5,22) | Cobrança | Dispositivos |
|---|---|---|---|---|---|
| **CCleaner** (Piriform/Gen Digital) | Sim, básico | Pro US$29,95–44,95/ano · Pro Plus US$64,95/ano · Premium (5 disp.) | ~R$13–19,50/mês (Pro) | Anual (mensal também existe na prática de mercado, não confirmei valor) | 1 / 3 / 5 |
| **Advanced SystemCare** (IObit) | Sim, básico | Pro US$19,99–29,99/ano (3 PCs) | ~R$8,70–13/mês | Anual | 3 |
| **Wise Care 365** | Sim, limitado | Pro US$29,95/ano (3 PCs) | ~R$13/mês | Anual | 3 |
| **AVG TuneUp** | Não — só trial 7–60 dias | US$65,99/ano (1) · US$79,99/ano (10) — 1º ano promo ~US$26–35 | ~R$28,70–34,70/mês (lista) | Anual | 1 / 10 |
| **Norton Utilities Ultimate** | Não (é add-on pago) | US$69,99/ano confirmado direto no site oficial (renovação, 10 dispositivos) | ~R$30,40/mês (10 disp. ≈ R$3/mês cada) | Anual | 10 |
| **iolo System Mechanic** | Não (trial) | Pro US$59,94/ano (lista) — comumente ~US$23,99 promo. + upsell separado de US$19,95/MÊS para "live agent" | ~R$26/mês (software) + ~R$104/mês (suporte) | Anual + upsell mensal de suporte | Varia |
| **Ashampoo WinOptimizer** | Sim, versão free separada e permanente (motor mais antigo, faltam ~39 itens) | ~US$30/ano lista (promo comum ~US$14) | ~R$13/mês (lista) | Anual | Varia |
| **Wintoys** | 100% grátis, sem tier pago | — | — | — | — |
| **Microsoft PC Manager** | 100% grátis, sem tier pago, nada bloqueado | — | — | — | — |

**Sobre cobrança semanal**: nenhum dos 8 concorrentes de utilitário de PC pesquisados cobra semanalmente. O modelo semanal aparece em categoria adjacente (apps mobile de VPN, scanner, edição) — ver Parte 3.

### Detalhamento por ferramenta

**CCleaner (Piriform, hoje Gen Digital)**
Free: limpeza de temporários, cache de navegador, privacidade limitada. Pro adiciona Performance Optimizer (hiberna programas em segundo plano), PC Health Check, atualização de drivers/software, limpeza agendada, monitoramento em tempo real e suporte prioritário. Garantia de reembolso de 30 dias.
- **Hack de 2017**: ataque de supply-chain confirmado — invasores entraram na rede da Piriform via TeamViewer (credenciais de vazamento anterior) e RDP, injetaram backdoor no instalador oficial distribuído entre agosto e setembro de 2017, infectando **2,3 milhões de usuários**; payload de segundo estágio mirou ~20 grandes empresas de tech (Google, Microsoft, Cisco, Intel), infectando 40 máquinas.
- **Reputação no Brasil**: status "Não Recomendada" no Reclame Aqui, menos de 50% das reclamações respondidas, maioria sobre cobrança indevida e dificuldade de cancelamento.
- **Padrão escuro confirmado**: portal de cancelamento retornando erro 404, ausência de e-mail de confirmação, renovação automática sem consentimento claro.

**Advanced SystemCare (IObit)**
Free cobre limpeza 1-click básica e proteção leve. Pro adiciona Full Detection, Anti-Tracking, limpeza automática de RAM, limpeza profunda de registro, AutoCare. Marketing usa alegações não verificáveis ("internet 300% mais rápida").
- **Reputação**: histórico de ser sinalizado como PUP por Malwarebytes, ESET, Panda, Avira, Kaspersky, Avast, Norton e Reason Core; disputa pública e antiga com Malwarebytes; reputação de empacotar bundleware.

**Wise Care 365**: 5 módulos (PC Checkup, System Cleaner, System Tuneup, proteção de privacidade, proteção em tempo real). Sem presença relevante no Reclame Aqui **[NÃO CONFIRMADO — baixa visibilidade no Brasil, não indica qualidade, só menor penetração local]**.

**AVG TuneUp (Gen Digital)**: sem versão free permanente — apenas trial (7-60 dias) que libera só diagnóstico e limpeza pontual. Modelo mais "fechado" do grupo.

**Norton Utilities Ultimate (Gen Digital)**: 5 ferramentas de limpeza + Processes/Startup Manager/Sleep Mode. Vendido com argumento de "menor custo por dispositivo" ao cobrir 10 aparelhos numa assinatura.

**iolo System Mechanic**: antivírus + otimizador + limpador de privacidade + reparador de registro + gerenciador de inicialização em um produto, com "Smart ActiveCare". **Maior red flag da Parte 1**: cobrança separada de US$19,95/mês por atendimento humano, podendo chegar a US$240/ano só de suporte. Nota de reputação **1,0/5 no ComplaintsBoard**.

**Ashampoo WinOptimizer**: único concorrente pago com free version permanente de verdade — mas com motor deliberadamente desatualizado, ~39 itens a menos. 19 módulos na versão completa.

**Wintoys**: gratuito, mantido por um único desenvolvedor independente, só via Microsoft Store. Cobre debloat de apps, toggles de privacidade numa tela só, ativação do plano "Ultimate Performance", GPU scheduling, gerência de serviços/inicialização e reparo do sistema. Interface em abas (Apps, Services, Performance, Health, Tweaks) é referência de UX limpa e replicável.

### Microsoft PC Manager — análise aprofundada (o concorrente mais perigoso)

**O que É de verdade**: app genuíno da Microsoft, Microsoft Store, 100% gratuito, sem tier pago, sem chave de licença, nada esmaecido esperando upgrade.

**Funcionalidades reais confirmadas**:
- Cleanup & Boost: limpeza de temporários, apps de inicialização, memória, armazenamento, bloqueio de pop-up numa janela só;
- Health Check: limpa logs do sistema e arquivos recentes, desabilita apps de inicialização indesejados;
- Ferramentas avançadas: aceleração 1-click, gestão de armazenamento, verificação de vírus (integra Defender), gestão de pop-up;
- Barra flutuante de velocidade de internet em tempo real + liberação de RAM com 1 clique.

**O que ele estruturalmente NÃO faz (achado central para posicionamento)**:
- Não mexe em scheduling de kernel, drivers, ou I/O de baixo nível — ganhos descritos por reviewers como **modestos e temporários**;
- Menos profundidade forense que Task Manager/Autoruns;
- Não substitui SFC/DISM/chkdsk, rollback avançado de driver, ou gestão enterprise/frota;
- **Empurra o usuário para Edge/Bing** de forma "heavy-handed" — mesmo o concorrente "grátis e do bem" tem viés comercial embutido;
- Falta de transparência: nem sempre explica o que está mudando ou por quê.

**Implicação estratégica**: o PC Manager cobre a fatia "básica e honesta" de graça, tornando obsoleta a categoria clássica de "cleaner pago". A brecha real: automação contínua e silenciosa, dois perfis de uso (pessoal vs trabalho), remoção agressiva de bloatware/nudges da própria Microsoft, rollback comunicado com clareza, suporte em português para leigo, gestão multi-máquina.

**[NÃO CONFIRMADO]**: disponibilidade regional exata do PC Manager na Microsoft Store BR — checar manualmente antes de finalizar posicionamento.

### O mito do "registry cleaner" e o problema estrutural da categoria

**Limpeza de registro é, na prática, placebo para Windows moderno**. Mesmo centenas de entradas órfãs representam quantidade de dados irrisória e não geram impacto de performance mensurável (Malwarebytes Labs, "Digital Snake Oil", 2015). **A própria Microsoft não recomenda limpadores de registro** e alerta que limpeza malfeita pode quebrar configurações ou causar problemas de boot.

Isso explica a origem do padrão escuro "diagnóstico grátis, correção paga" — ferramentas scareware como PC Optimizer Pro, PC Utilities Pro e Ultimate PC Optimizer escaneiam uma máquina limpa e "encontram" dezenas/centenas de "problemas críticos". Classificadas como PUP/scareware por fornecedores de segurança — e o mecanismo visual (barra de progresso, contador, botão "Corrigir Tudo" bloqueado) é **visualmente idêntico** ao usado por CCleaner e IObit em suas versões free, contaminando a percepção da categoria inteira por associação.

### Padrões escuros consolidados (evidência coletada)

1. Contagem inflada de "problemas encontrados" — mecanismo core do scareware, replicado de forma mais branda por produtos legítimos como gancho de conversão free→pago.
2. Cancelamento difícil: portal quebrado, ausência de confirmação por e-mail, necessidade de insistir por dias (caso CCleaner).
3. Renovação automática silenciosa sem lembrete claro.
4. Alegações de marketing não verificáveis ("300% mais rápido").
5. Upsell de suporte "premium" como assinatura à parte e cara (iolo, US$19,95/mês — pior exemplo encontrado).
6. Histórico de bundleware/PUP nos instaladores free (IObit, historicamente).

---

## PARTE 2 — Ferramentas open-source de referência

### Tabela comparativa (licença confirmada via fetch direto quando indicado)

| Ferramenta | Licença exata | Como confirmei | Modelo de catálogo | Rollback/undo |
|---|---|---|---|---|
| **ChrisTitusTech/winutil** | MIT | LICENSE no repo | PowerShell + WPF; tweaks dirigidos por config, abas (Install, Tweaks, Config, Updates) | Cria Ponto de Restauração automático antes de aplicar; botão "Undo Selected Tweaks"; recomenda System Restore completo para mudanças profundas |
| **hellzerg/optimizer** | **GPL-3.0** | fetch direto do LICENSE | C#/.NET WinForms; checkboxes por categoria | Não documentado com profundidade; prática comum: ponto de restauração manual |
| **O&O ShutUp10++** | Freeware proprietário (não é open-source) | busca + confirmação cruzada | Lista tipo "semáforo" (recomendado/não recomendado) por item | "Undo → Restore initial settings", "Undo all changes", exportação/importação de perfis |
| **privacy.sexy** | **AGPL-3.0** | fetch direto do LICENSE | Coleções em YAML por SO, nível de recomendação por script, arquitetura DDD — ótima referência de *schema* de dados | Cada script tem um script de reversão pareado no mesmo catálogo — reversibilidade é cidadã de primeira classe do modelo de dados |
| **Sophia Script for Windows** | MIT | fetch direto do LICENSE | Módulo PowerShell, 150+ funções, uma por tweak, só mecanismos documentados oficialmente | Contraparte de reversão/padrão por função |
| **Raphire/Win11Debloat** | MIT | busca | Script PowerShell + GUI opcional; flags por grupo de recurso | Backup automático do registro em JSON com timestamp; GUI "Restore Backup"; não reinstala apps removidos |
| **Microsoft PowerToys** | MIT | busca | Não é catálogo de tweaks — suíte modular independente | N/A — módulos são aditivos |

### Risco jurídico: conhecimento vs. cópia de código

**Princípio central**: nomes de chaves de registro, cmdlets e serviços do Windows são fatos/superfície de API — **não protegidos por direito autoral**. Um catálogo de "quais tweaks existem e o que fazem", construído lendo documentação oficial e observando essas ferramentas, é legalmente limpo como conhecimento. O risco está em **copiar código-fonte, prosa exata, ou arquivos de dados exatos**.

- **MIT (WinUtil, Sophia Script, Win11Debloat, PowerToys)** — permissivo, seguro até para reuso literal de trechos com atribuição.
- **GPL-3.0 (Optimizer/hellzerg)** — copyleft forte, incompatível com .exe comercial fechado. **Não copiar código**; usar só como fonte de estudo, reimplementar de forma independente ("clean room").
- **AGPL-3.0 (privacy.sexy)** — ainda mais restritivo (gatilho de rede/serviço). Como o produto é .exe desktop (não serviço de rede), o gatilho específico de "uso via rede" é menos diretamente aplicável, mas a obrigação de copyleft sobre distribuição continua valendo. Mesma regra: não copiar YAML/código.
- **O&O ShutUp10++** — sem código-fonte público; só referência de UX, nunca copiar textos de interface literalmente.

**Recomendação prática**: tratar WinUtil, Sophia Script, Win11Debloat e PowerToys como fontes seguras até para inspiração/reuso leve de código; tratar Optimizer e privacy.sexy como fontes de leitura apenas. Catálogo do zero, em Go, com dados e prosa autorais. **Não é aconselhamento jurídico formal** — confirmar com advogado especializado antes de lançar comercialmente.

---

## PARTE 3 — Monetização e pagamento no Brasil

### Preço psicológico de assinatura de consumo (2026, confirmado)

| Serviço | Preço mensal BR confirmado |
|---|---|
| Netflix (com anúncios) | R$ 20,90/mês |
| Netflix Padrão | R$ 44,90/mês |
| Netflix Premium | R$ 59,90/mês |
| Spotify Universitário | R$ 12,90/mês |
| Spotify Individual | R$ 23,90/mês |
| Spotify Duo | R$ 31,90/mês |
| Spotify Família | R$ 40,90/mês |
| Antivírus BR (faixa geral) | R$ 40–250/ano (≈ R$ 3,33–20,83/mês) |
| Avast (planos citados) | R$ 59–169/ano (≈ R$ 4,92–14,08/mês) |
| Kaspersky Plus | a partir de R$ 117,90/ano (≈ R$ 9,83/mês) |

**Leitura**: existe faixa "aceitável" clara para utilitário de consumo — abaixo de Spotify Individual (R$23,90) e na vizinhança de antivírus de entrada (R$10-20/mês). Acima disso, entra em território "streaming premium", exigindo entrega de valor muito mais explícita.

### Assinatura semanal no Brasil

- Existe e é praticada, mas **não** pelos concorrentes diretos de utilitário de PC (nenhum dos 8 cobra semanalmente). Aparece em categoria adjacente: apps de VPN mobile (~US$3,99/semana), scanner de documento.
- Caso real no Reclame Aqui: app "Scanner App Scan PDF e Docs" cobrando R$54,99/semana, com reclamação de cobrança não autorizada.
- Levantamento ICPEN: dark patterns em 75,7% dos sites/apps de assinatura analisados globalmente, 66,8% usando 2+ padrões simultâneos. Cobrança semanal fortemente associada a "trial disfarçado".
- Taxas: Pix normalmente menor que cartão recorrente (ordem de grandeza: Pix avulso a partir de 0,99% no Mercado Pago vs. cartão a partir de 3,99%; Stripe 1,19% Pix vs. 3,99%+R$0,39 cartão nacional). Cobrança semanal via cartão multiplica transações tarifadas 4-5x vs. mensal — Pix reduz bastante esse atrito se a cobrança for frequente.

**Recomendação**: cobrança semanal tem reputação de risco elevado no Brasil. Se oferecida, deve ser transparente (cobrança visível, nunca cartão de graça convertendo silenciosamente).

### Pix Automático (Banco Central) — status confirmado

- Regulado pela Resolução BCB nº 402, de 22/07/2024.
- **Lançamento oficial: 16 de junho de 2025** — a partir dessa data, instituições com contas transacionais são **obrigadas** a disponibilizar.
- **Sempre gratuito para quem paga** — Art. 96 da Resolução proíbe tarifa entre provedores relacionada a transações de Pix Automático.
- **Quem pode ser cobrado**: o recebedor (a empresa) pode pagar tarifa ao próprio provedor, por livre acordo comercial — sem tabela oficial fixa.
- Provedores confirmados:
  - **Asaas**: confirmado (blog oficial) que já oferece para PJ. Pix avulso: R$0,99 (promo 3 meses) / R$1,99 padrão; cartão recorrente R$0,49 + 1,99%. **[NÃO CONFIRMADO: tarifa específica do Pix Automático em si]**.
  - **Mercado Pago**: Planos de Assinatura aceitam cartão/Pix/boleto em semanal/quinzenal/mensal/trimestral/semestral/anual (confirmado). Pix avulso a partir de 0,99%, cartão a partir de 3,99%. **[NÃO CONFIRMADO: taxa específica do Pix Automático]**.
  - **Pagar.me/Stone**: menção indireta em blog terceiro — **[NÃO CONFIRMADO em fonte oficial]**.
  - **Stripe Brasil**: Pix geral 1,19% (hoje só por convite para empresas BR) + 0,7% do volume em Billing (confirmado via fetch oficial). **[NÃO CONFIRMADO se já suporta Pix Automático nomeadamente]**.
- **Alerta de qualidade de fonte**: números como "0,22%-0,35%" para Pix Automático aparecem replicados em múltiplos blogs de SEO citando uma "Resolução BC nº 422/2025" não verificável em fonte primária — cheiro de conteúdo gerado se replicando. **[NÃO CONFIRMADO]** até validar diretamente com um provedor (conta teste) antes de basear o modelo financeiro nisso.

### Regras do CDC para assinatura

- **Art. 49 — direito de arrependimento de 7 dias corridos**: confirmado, aplica-se a toda contratação pela internet/app. Incondicional, devolução imediata e monetariamente atualizada.
- **Cancelamento pelo "mesmo canal da contratação"**: vem do Decreto nº 11.034/2022 (regras do SAC), que se aplica apenas a **setores regulados** (telecom, energia, financeiro, aviação, saúde, transporte). **Um SaaS de otimização de PC genérico não está automaticamente coberto.**
- Princípios gerais do CDC (Art. 39 práticas abusivas, Art. 51 cláusulas abusivas, Art. 6º informação clara) são usados por analogia contra empresas que dificultam cancelamento, mesmo fora do Decreto do SAC.
- **[NÃO CONFIRMADO]**: lei federal já em vigor exigindo aviso prévio de 30 dias antes de renovação automática ≥12 meses para software em geral — indícios apontam para projeto de lei em tramitação, não lei sancionada.
- **Recomendação prática segura**: cancelamento self-service dentro do próprio app/site, mesmo canal da contratação, sem exigir contato humano nem CTA escondido.

---

## PARTE 4 — Benchmark free vs. premium e recomendação final

### O padrão "diagnóstico grátis, correção paga" é um padrão escuro?

Confirmado: é o padrão-fábrica de quase todo o setor pago.

| Ferramenta | Free mostra | Free permite corrigir? | Pago desbloqueia |
|---|---|---|---|
| CCleaner | limpeza básica, cache | manual sim; monitoramento/agendamento não | Performance Optimizer, tempo real, updates automáticos, agendamento |
| Advanced SystemCare | scan 1-click | básica sim; detecção completa/RAM automática não | Full Detection, Anti-Tracking, RAM automática, registro profundo |
| AVG TuneUp | diagnóstico (trial) | praticamente nada sem pagar | tudo |
| Wise Care 365 | scan | parcial | todos os módulos, 3 PCs |
| Ashampoo WinOptimizer | scan com motor antigo | sim, limitado (~39 itens a menos) | motor atual completo |
| **Microsoft PC Manager** | tudo | **sim, tudo** | não existe pago |

O mecanismo (scan → contador de "problemas" → ação bloqueada) é visualmente idêntico ao scareware puro, contaminando a percepção de toda a categoria. Como o PC Manager não joga esse jogo, fica posicionado como "o único que não tenta enganar" — risco reputacional estrutural para quem copiar o padrão clássico.

**O que fazer diferente**:
1. Nunca vender "a correção do problema que acabei de te mostrar". Limpeza manual e tweaks individuais (toggle + descrição clara, estilo privacy.sexy/ShutUp10) podem ser **grátis e ilimitados**.
2. Ponto de restauração automático + desfazer individual **grátis sempre**, nunca paywall.
3. Vender a camada que o PC Manager estruturalmente não constrói: automação contínua e silenciosa; **dois perfis lado a lado (Pessoal vs. Trabalho)**; histórico/relatório; suporte PT-BR; multi-máquina.

### O que as pessoas de fato pagam

Em todos os concorrentes, o eixo que separa free de pago é consistentemente:
1. **Automação contínua / monitoramento em tempo real** — item nº1 em CCleaner, IObit, AVG e Norton simultaneamente;
2. **Número de dispositivos cobertos** — Norton, CCleaner Plus/Premium e AVG usam contagem (1→3→5→10) como eixo de tier;
3. **Suporte** — diferenciador de tier mais alto, mas também maior risco de virar reclamação se monetizado agressivamente (caso iolo).

A função "limpeza" isolada tem valor real questionável — o valor percebido pago está mais em conveniência automatizada + tranquilidade/garantia (rollback) + suporte do que na limpeza em si.

### B2B / preço por máquina no Brasil

- Antivírus corporativo BR: R$15-50/usuário/mês; pequenas empresas a partir de R$500/ano, maiores R$12.000+/ano (ordem de grandeza, não fonte única).
- RMM internacional (referência): NinjaOne US$1,50-3,75/endpoint/mês (≈R$7,80-19,50); Atera cobra por técnico (US$129-219/mês).
- Não encontrado "otimizador de PC" B2B com preço público específico no Brasil — antivírus corporativo/RMM serve de proxy razoável.

---

## Recomendação de divisão FREE vs. PREMIUM

### FREE (generoso — arma de confiança contra o Microsoft PC Manager)
- Diagnóstico completo sem contagem inflada de "problemas" — categorias reais e claras, nunca número alarmista;
- Limpeza manual sob demanda (temporários, cache) **ilimitada**;
- Tweaks individuais aplicados um a um, toggle + descrição em português simples — grátis;
- **Ponto de restauração automático + desfazer individual sempre grátis**;
- 1 perfil ativo por vez (Pessoal OU Trabalho, à escolha).

### PREMIUM (semanal/mensal/anual)
- Automação agendada e silenciosa (roda sozinho, sem abrir o app);
- **As duas abas simultâneas — Pessoal + Trabalho** — diferenciação central, não encontrada em nenhum concorrente;
- Histórico/relatório de execução;
- Atualização de driver/software (alto valor percebido, mas alto risco técnico/suporte — avaliar com cautela);
- Multi-PC (ex.: 3 dispositivos) na assinatura pessoal;
- Suporte prioritário PT-BR, humano — sem repetir o erro do iolo de cobrar suporte separado e agressivo.
- Corretor de rotas de internet (ver pesquisa dedicada).

### Empresarial/Frota (cobrança por máquina, separado)
- Dashboard central, relatórios consolidados, aplicação do perfil "Trabalho" padronizado em N máquinas, nota fiscal PJ única.

---

## Tabela de preço sugerida (BRL) — hipótese inicial, não validada por pesquisa de sensibilidade

| Plano | Preço sugerido | Equivalente mensal | Lógica / ancoragem |
|---|---|---|---|
| **Semanal** (oferecer com cautela) | R$ 7,90/semana | ≈ R$ 34/mês | Deliberadamente mais caro por mês-equivalente que o mensal — mesmo padrão do exemplo de VPN, mais moderado. Cobrança sempre visível, nunca trial disfarçado |
| **Mensal** | R$ 19,90/mês | R$ 19,90/mês | Entre o equivalente do IObit (~R$8,70-13) e do AVG TuneUp (~R$28,70-34,70); abaixo do Spotify Individual |
| **Anual** | R$ 149,90/ano | ≈ R$ 12,50/mês | ~37% de desconto vs. mensal×12 — alinhado ao padrão do setor |
| **Empresarial (por máquina)** | R$ 14,90/máquina/mês | — | Abaixo do teto de antivírus corporativo BR; dentro da faixa de RMM internacional; sugerido mínimo de 5 máquinas |

**Antes de fixar**: validar com pesquisa de sensibilidade de preço direta no público-alvo (Van Westendorp ou conjoint simples).

---

## Lista consolidada do que NÃO foi confirmado

1. Disponibilidade regional exata do Microsoft PC Manager na Microsoft Store BR.
2. Preço BRL oficial (tabela local) de CCleaner, IObit, Wise Care 365, AVG TuneUp, Norton Utilities e iolo — só preço USD de lista americana convertido por cotação de referência.
3. Tarifa específica do produto "Pix Automático" (separada de Pix avulso) para Asaas e Mercado Pago.
4. Se a Stripe Brasil já suporta Pix Automático como produto nomeado.
5. Se Pagar.me/Stone já oferece Pix Automático — só menção indireta em blog terceiro.
6. Números de taxa "0,22%-0,35%" para Pix Automático — possível conteúdo replicado/gerado, cita resolução não verificada.
7. Lei federal já em vigor exigindo aviso de 30 dias antes de renovação automática ≥12 meses para software em geral.
8. Toda a tabela de preços sugerida — hipótese ancorada em comparáveis, não pesquisa de sensibilidade direta.
9. Preço exato do tier "Ultimate" do Advanced SystemCare (IObit) — só o tier "Pro" foi confirmado.
10. Profundidade exata do sistema de "agressividade"/nível de recomendação do privacy.sexy — existe, mas não a escala completa/nomenclatura de cada nível.

---

## Fontes principais consultadas

- CCleaner: [findstack.com/products/ccleaner/pricing](https://findstack.com/products/ccleaner/pricing), [The Hacker News – CCleaner attack timeline](https://thehackernews.com/2018/04/ccleaner-malware-attack.html), [CrowdStrike – CCleaner backdoor](https://www.crowdstrike.com/en-us/blog/protecting-software-supply-chain-deep-insights-ccleaner-backdoor/), [Reclame Aqui – CCleaner](https://www.reclameaqui.com.br/empresa/ccleaner/)
- IObit: [iobit.com/en/compare/asc](https://www.iobit.com/en/compare/asc/), [MalwareTips – IObit reputação](https://malwaretips.com/threads/why-does-iobit-have-such-a-bad-reputation.85035/)
- Norton: [us.norton.com/pricing](https://us.norton.com/pricing) (fetch direto, confirmado)
- iolo: [ComplaintsBoard – iolo](https://www.complaintsboard.com/iolo-b193893)
- Ashampoo: [ashampoo.com/en-us/winoptimizer-free](https://www.ashampoo.com/en-us/winoptimizer-free)
- Microsoft PC Manager: [itechguides.com](https://www.itechguides.com/microsoft-pc-manager-is-available-via-the-microsoft-store-but-not-everywhere/), [windowsforum.com review](https://windowsforum.com/threads/microsoft-pc-manager-review-one-click-boost-cleanup-and-real-world-gains.398266/)
- Registry cleaner myth: [Malwarebytes Labs – Digital Snake Oil](https://www.malwarebytes.com/blog/news/2015/06/digital-snake-oil), [How-To Geek](https://www.howtogeek.com/the-truth-about-windows-registry-cleaners-and-why-people-still-use-them/)
- WinUtil: [github.com/ChrisTitusTech/winutil/blob/main/LICENSE](https://github.com/ChrisTitusTech/winutil/blob/main/LICENSE)
- Optimizer (hellzerg): [github.com/hellzerg/optimizer/blob/master/LICENSE](https://github.com/hellzerg/optimizer/blob/master/LICENSE) (fetch direto, GPL-3.0 confirmado)
- privacy.sexy: [github.com/undergroundwires/privacy.sexy/blob/master/LICENSE](https://github.com/undergroundwires/privacy.sexy/blob/master/LICENSE) (fetch direto, AGPL-3.0 confirmado)
- Sophia Script: [github.com/farag2/Sophia-Script-for-Windows](https://github.com/farag2/Sophia-Script-for-Windows) (fetch direto, MIT confirmado)
- Win11Debloat: [github.com/Raphire/Win11Debloat/wiki/Reverting-Changes](https://github.com/Raphire/Win11Debloat/wiki/Reverting-Changes)
- PowerToys: [github.com/microsoft/PowerToys/blob/main/LICENSE](https://github.com/microsoft/PowerToys/blob/main/LICENSE)
- Pix Automático: [Agência Brasil – lançamento](https://agenciabrasil.ebc.com.br/economia/noticia/2024-07/bc-define-que-pix-automatico-sera-lancado-em-junho-de-2025), [LegisWeb – Resolução BCB 402/2024](https://www.legisweb.com.br/legislacao/?id=462331)
- Asaas: [asaas.com/precos-e-taxas](https://www.asaas.com/precos-e-taxas) (fetch direto)
- Stripe Brasil: [stripe.com/br/pricing](https://stripe.com/br/pricing) (fetch direto)
- Mercado Pago: [mercadopago.com.br/ferramentas-para-vender/assinaturas](https://www.mercadopago.com.br/ferramentas-para-vender/assinaturas)
- CDC / cancelamento: [Decreto 11.034/2022 – gov.br](https://www.gov.br/pt-br/noticias/justica-e-seguranca/2022/10/entenda-as-novas-regras-do-sac-que-ja-estao-em-vigor), [ProConSP](https://www.procon.sp.gov.br/cobrancas-indesejadas/)
- Netflix/Spotify BR: [oficinadanet.com.br](https://www.oficinadanet.com.br/netflix/67930-netflix-planos-precos-onde-assistir)
- Antivírus corporativo BR: [simplessolucao.com.br](https://www.simplessolucao.com.br/melhores-antivirus-empresas-2025-preco-r), [ax4b.com](https://ax4b.com/quanto-custa-um-antivirus-corporativo/)
- RMM internacional: [aimultiple.com/rmm-pricing](https://aimultiple.com/rmm-pricing)
- Dark patterns: [ICPEN](https://www.icpen.org/news/1360)
