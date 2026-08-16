# Catálogo técnico — Rede / Internet

> Windows 11 24H2/25H2 (build 26100+) e Windows 10 22H2. Verificado com `netsh interface tcp show global`, `Get-NetTCPSetting`, `Get-NetAdapterAdvancedProperty` ao vivo numa máquina real build 26200.

## 1. TCP Autotuning

- **Comando**: verificar `netsh interface tcp show global` (campo "Receive Window Auto-Tuning Level"); alterar `netsh interface tcp set global autotuninglevel=normal|restricted|highlyrestricted|disabled|experimental`.
- **Ganho de desativar**: nenhum — o oposto do que o tweak promete. Sem autotuning, uma conexão de 1 Gbps com 10ms de latência fica limitada a ~51 Mbps por conexão (fórmula oficial da Microsoft).
- **Veredicto**: **obsoleto** desde Windows Vista/Server 2008. O tweak "desative autotuning" nasceu de bugs de RWIN em roteadores domésticos de ~2005-2008, praticamente extintos há 15+ anos.
- **Risco**: Moderado (desativar reduz throughput real, o oposto do prometido).
- **Fonte**: [Network Adapter Performance Tuning in Windows Server](https://learn.microsoft.com/en-us/windows-server/networking/technologies/network-subsystem/net-sub-performance-tuning-nics).

### Chaves legadas Tcp1323Opts / TcpWindowSize
- **Veredicto**: **obsoleto**. `TcpWindowSize` está listado oficialmente entre os "parâmetros TCP obsoletos" — "não são mais suportados e são ignorados" desde Windows Server 2019 (mesma pilha do client). `Tcp1323Opts` não aparece mais na doc atual.
- **Risco**: Moderado (dá falsa sensação de ajuste; valor é ignorado).

## 2. RSS e RSC

| Item | Comando | Impacto | Veredicto |
|---|---|---|---|
| RSS (Receive Side Scaling) | `Get/Enable/Disable-NetAdapterRss` | Baixo para uso doméstico — benefício documentado é para servidores com muitas conexões simultâneas; **já vem ligado por padrão** | confirmado_com_ressalva |
| RSC (Receive Segment Coalescing) | `Get/Enable/Disable-NetAdapterRsc`, ou `netsh int tcp set global rsc=` | Baixo para uso doméstico; reduz CPU em transferências grandes, mas **exige suporte de hardware/driver — nem toda placa expõe** | confirmado_com_ressalva |

- **Achado empírico**: neste ambiente de teste (build 26200), RSC apareceu **desabilitado por padrão**, contradizendo o exemplo padrão da própria documentação Microsoft — confirma que o estado padrão depende do driver/placa, não é universal. O produto deve **ler o estado real**, nunca assumir.
- **Fonte**: [Introduction to Receive Side Scaling](https://learn.microsoft.com/en-us/windows-hardware/drivers/network/introduction-to-receive-side-scaling), [Overview of Receive Segment Coalescing](https://learn.microsoft.com/en-us/windows-hardware/drivers/network/overview-of-receive-segment-coalescing).

## 3. Provedor de controle de congestionamento

- **Comando**: `Get-NetTCPSetting` / `Set-NetTCPSetting -SettingName Internet -CongestionProvider CUBIC`.
- **⚠️ Contradição entre fontes oficiais**: a página `Set-NetTCPSetting` (desatualizada, conteúdo de 2016) afirma "clientes usam NewReno por padrão"; **verificação ao vivo neste Windows 11 build 26200 mostra CUBIC** no template "Internet" (com HyStart e Proportional Rate Reduction). Confiar na verificação ao vivo.
- **BBR2**: existe desde Win11 22H2 como experimental, mas com relatos consistentes de quebra de apps locais (adb, Steam, IDM) em 23H2/24H2. **Só Windows 11 22H2+, não existe no Windows 10.**
- **Veredicto**: confirmado_com_ressalva — CUBIC já é o padrão adequado, trocar manualmente não traz ganho para a maioria; BBR2 é arriscado para público leigo.

## 4. Nagle's Algorithm / TcpAckFrequency / TCPNoDelay

- **Chaves**: `HKLM\SYSTEM\CurrentControlSet\Services\Tcpip\Parameters\Interfaces\{GUID}\TcpAckFrequency` (REG_DWORD, padrão `2`; valor `0` popular em guias é **inválido**, tratado como 2) e `TCPNoDelay` (REG_DWORD, 0=Nagle ativo/padrão, 1=desativado).
- **⚠️ `TCPNoDelay` nunca foi documentada oficialmente pela Microsoft** — só existe em fóruns/blogs de terceiros. `TcpAckFrequency` tem KB oficial (328890), mas classificado sob doc de Windows Server.
- **Achado central**: a maioria dos jogos modernos usa **UDP**, não TCP, para tráfego em tempo real — nesse caso Nagle/delayed-ACK nunca influenciaram a latência do jogo, e o "efeito" percebido por quem aplicou é largamente coincidência/placebo. O tweak genuinamente importa em apps que usam TCP para tráfego interativo (RDP, SSH, emuladores, streaming de tela) — a própria Microsoft recomendava desativar Nagle para o MSMQ.
- **Veredicto**: confirmado_com_ressalva — mecanismo real, mas benefício para "gaming" moderno muito menor do que a fama sugere.
- **Risco**: Moderado (mais overhead de ACK, pode piorar conexões de alta latência/perda, como Wi-Fi ruim ou 4G).
- **Fonte**: [KB328890](https://learn.microsoft.com/en-us/troubleshoot/windows-server/networking/registry-entry-control-tcp-acknowledgment-behavior).

## 5. MTU

- **Comando**: `netsh interface ipv4 set subinterface "<Nome>" mtu=1492 store=persistent` / `Set-NetIPInterface -NlMtu`.
- **Veredicto**: **placebo** para Ethernet/Wi-Fi doméstico comum (MTU padrão 1500 já é ideal; Path MTU Discovery já negocia automaticamente). Só relevante em **PPPoE** (MTU efetivo 1492) ou **atrás de VPN/túnel** (confirmado empiricamente: interface VPN neste ambiente mostrou MTU=1280, bem abaixo do padrão).
- **Risco**: Moderado (MTU mal configurado causa fragmentação ou perda silenciosa de pacotes).
- **Path MTU Discovery** (`EnablePMTUDiscovery`, padrão `1`): deve **permanecer ligado sempre** — desativar força MTU fixo de 576 bytes para qualquer destino externo. **Veredicto**: confirmado (não expor como algo a "otimizar", só verificar se não foi desativado por engano).

### Como o produto implementa (medição, nunca chute) — `internal/netdiag`

Este é o item que virou a prova de conceito da regra de ouro do produto: **não existe botão "otimizar MTU"; existe uma medição, e o ajuste só aparece depois dela, com o número que justificou.**

**Técnica** — a mesma do `ping -f -l <tamanho>` do Windows: eco ICMP com o sinalizador *don't fragment* ligado. Busca binária entre 64 e 1472 bytes de dados acha o maior pacote que atravessa o caminho inteiro sem ser quebrado; o MTU é esse número **+ 28** (20 de cabeçalho IPv4 + 8 de ICMP). Caminho limpo resolve em 2 medições; caso ruim, ~12.

**Decisões de engenharia**:
- Usa `IcmpSendEcho` (iphlpapi.dll) em vez de chamar `ping.exe`: não paga criação de processo, não depende do idioma do Windows para interpretar a resposta, não bate na heurística de antivírus de "programa comum executando ferramenta de linha de comando escondido" — e, diferente de socket ICMP cru, **não exige administrador**.
- Cada tamanho é testado mais de uma vez antes de ser dado como reprovado: uma perda solta de pacote não pode virar diagnóstico errado.
- O adaptador de saída é descoberto pela rota real até o destino (`net.Dial` UDP não envia pacote, só faz o Windows escolher a rota), não por "pega a primeira placa".
- A escrita usa `SetIpInterfaceEntry` (API pública) com `SitePrefixLength = 0`, exigência documentada da Microsoft para IPv4 — não `netsh`.
- A UI mostra os dois comandos decisivos (`ping -n 1 -f -l N <destino>`, o maior que passa e o menor que falha) para o usuário conferir na mão. Transparência verificável em vez de "confie no número".

**Achado de produto (medido ao vivo nesta máquina, ago/2026)**: o caminho até `8.8.8.8` aceitava 1480, com o adaptador em 1500 — e o `ping` nativo do Windows confirmou o limite exatamente em 1452/1453 bytes. Mas o equipamento no caminho **avisava** ("precisa fragmentar"), então o Windows já se ajustava sozinho por destino via Path MTU Discovery. Daí a distinção que o diagnóstico faz e que nenhum concorrente pesquisado faz:

| Situação | O que significa | O que o produto diz |
|---|---|---|
| Pacote grande volta com aviso "precisa fragmentar" | PMTUD funcionando: o Windows se adapta sozinho por destino | Ajuste **opcional**, ganho pequeno — dizer que é urgente seria vender problema inventado |
| Pacote grande **some calado** (buraco negro de PMTUD) | O Windows não tem como descobrir e segue mandando grande demais | Ajuste **recomendado**: é o caso real de site que trava no meio, download que para, VPN instável |
| Adaptador já está abaixo de 1500 e tudo passa | A medição está limitada pelo próprio adaptador | Diz que **não dá para saber** se o caminho aceita mais; avisa que valor menor em VPN/PPPoE costuma ser proposital |
| Destino não responde a ping | Medição impossível | Diz isso e **não sugere nada** — não inventa MTU |

## 6. QoS Packet Scheduler — o mito dos "20% de banda reservada"

- **Política**: `Limitar largura de banda reservável` (GPO) / `HKLM\SOFTWARE\Policies\Microsoft\Windows\Psched`, valor `NonBestEffortLimit`.
- **O mito**: circula desde o Windows XP que o Windows "trava" 20% da internet para uso próprio. **Falso.** É apenas um teto que limita quanto apps que pedem explicitamente prioridade QoS (via API raramente usada) podem reservar juntas — banda não reservada no instante fica livre para tudo, incluindo downloads normais.
- **Veredicto**: **placebo** — um dos mitos de rede mais antigos e mais desmentidos do Windows (citado inclusive por Raymond Chen, engenheiro da Microsoft, como "placebo").
- **Fonte**: [ADMX_QOS Policy CSP](https://learn.microsoft.com/en-us/windows/client-management/mdm/policy-csp-admx-qos), [How-To Geek — Don't Touch "Limit Reservable Bandwidth"](https://www.howtogeek.com/730147/windows-10-myth-dont-touch-limit-reservable-bandwidth/).

## 7. DNS

- **Comando**: `Set-DnsClientServerAddress -ServerAddresses ("1.1.1.1","1.0.0.1")` (Cloudflare) ou `("8.8.8.8","8.8.4.4")` (Google).
- **Impacto real**: Baixo a Médio, **estritamente sobre latência de resolução de nomes** — nunca sobre throughput/velocidade de download. O ganho depende de quão bom já é o DNS do provedor do usuário.
- **Veredicto**: confirmado_com_ressalva — o produto deve comunicar essa distinção claramente para não prometer "internet mais rápida" de forma enganosa.

## 8. Adaptador de rede — energia

| Item | Impacto | Veredicto |
|---|---|---|
| "Permitir que o computador desligue este dispositivo" (desmarcar) | **Alto** — causa clássica confirmada de lag spikes intermitentes | confirmado |
| Interrupt Moderation | Médio — é **trade-off** CPU×latência, não ganho puro. A própria MS recomenda considerar o trade-off, não desativar sempre | confirmado_com_ressalva |
| Energy Efficient Ethernet (EEE/"Green Ethernet") | Médio — resolve micro-stuttering periódico em Ethernet cabeada quando presente | confirmado |

- **Nota de implementação**: `Set-NetAdapterPowerManagement` não expõe um parâmetro chamado literalmente "AllowComputerToTurnOffDevice" na lista oficial documentada — validar em bancada antes de automatizar essa checkbox específica.
- **Fonte**: [Get/Set-NetAdapterPowerManagement](https://learn.microsoft.com/en-us/powershell/module/netadapter/get-netadapterpowermanagement), [Interrupt Moderation](https://learn.microsoft.com/en-us/windows-hardware/drivers/network/interrupt-moderation).

## 9. Wi-Fi — modo de economia de energia do adaptador

- **Local**: Opções de Energia → Configurações avançadas → "Configurações de adaptador sem fio" → "Modo de economia de energia" → forçar Máximo desempenho.
- **Impacto real**: **Alto** — mais agressivo que em Ethernet (rádio é ciclado com mais frequência). Caso relatado: throughput de ~260 Mbps para ~430 Mbps ao forçar máximo desempenho.
- **Risco**: Moderado em notebook na bateria (reduz autonomia); Seguro em desktop/notebook sempre na tomada.
- **Veredicto**: confirmado.

## 10. Delivery Optimization

- **Chave**: `HKLM\SOFTWARE\Policies\Microsoft\Windows\DeliveryOptimization\DODownloadMode`, padrão `1` (LAN — compartilha com PCs na mesma rede/IP público). `0` = HTTP Only (desativa P2P). Valor `100` (Bypass) está **depreciado desde o Windows 11**.
- **Ganho**: elimina consumo de upload em segundo plano (o Windows agindo como "torrent" de atualizações para outros PCs) — relevante para internet limitada/compartilhada/com cota.
- **Alternativa mais fina**: `DOPercentageMaxForegroundBandwidth`/`DOMaxForegroundDownloadBandwidth` (limitar % ou KB/s) e `DOMonthlyUploadDataCap` (padrão 20 GB/mês) em vez de desativar por completo.
- **Veredicto**: confirmado.
- **Fonte**: [Delivery Optimization reference — Microsoft Learn](https://learn.microsoft.com/en-us/windows/deployment/do/waas-delivery-optimization-reference).

## 11. Reset de Winsock/TCP-IP

- **Comandos**: `netsh winsock reset` e `netsh int ip reset [log.txt]` (ambos exigem reinício). Mais drástico: "Redefinição de rede" pela GUI (reinstala adaptadores, **esquece Wi-Fi/VPN salvos**).
- **Impacto real**: Alto quando há corrupção real da pilha de rede (pós-malware, VPN/antivírus mal desinstalado, driver corrompido); **Nulo** quando o problema é outra coisa (Wi-Fi fraco, roteador, ISP, DNS) — nesses casos é "reiniciar e rezar" sem diagnóstico real.
- **Risco**: Moderado — `winsock reset` remove todos os LSPs de terceiros (VPNs, controle parental, softwares de segurança com filtro de rede podem parar até reinstalar).
- **Veredicto**: confirmado_com_ressalva — deve ser a **última** ferramenta do fluxo de diagnóstico, nunca um botão de "clique aqui para consertar a internet" oferecido de cara.
- **Fonte**: [Reset TCP/IP by Using the NetShell Utility](https://learn.microsoft.com/en-us/troubleshoot/windows-server/networking/reset-tcp-ip-net-shell).

---

## Regras para a UI (derivadas deste domínio)

1. Nunca oferecer "desativar autotuning TCP" nem as chaves legadas `Tcp1323Opts`/`TcpWindowSize` — são obsoletas e prejudicam throughput.
2. `TCPNoDelay`/Nagle deve vir com aviso "efeito real maior em RDP/SSH/emuladores; a maioria dos jogos usa UDP e não é afetada".
3. MTU customizado só deve ser sugerido quando detectado PPPoE ou VPN ativa — nunca como tweak geral de "mais velocidade".
4. QoS "reserva de 20%" não entra no catálogo como otimização — é mito confirmado.
5. Reset de Winsock/TCP-IP é a última etapa do fluxo de diagnóstico de rede, sempre com aviso de que VPN/proxy customizados serão perdidos.
6. Todo estado de rede (RSS, RSC, congestion provider) deve ser **lido ao vivo da máquina**, nunca assumido por padrão documentado — a pesquisa confirmou variação real entre placas/drivers.

## Notas de incerteza

- Provedor de congestionamento padrão: contradição entre a doc oficial desatualizada (NewReno) e a verificação ao vivo (CUBIC) — confiar na verificação ao vivo, mas validar em mais máquinas reais antes de expor como afirmação de marketing.
- GUIDs do powercfg para "Power Saving Mode" do Wi-Fi: amplamente citados por fontes de terceiros, sem página oficial dedicada encontrada — confirmar via `powercfg /q` antes de embutir GUIDs fixos.
- Efetividade de TcpAckFrequency/TCPNoDelay em jogos modernos: inferência técnica razoável (maioria UDP), não uma medição controlada direta.
- RSC habilitado/desabilitado por padrão: amostra de uma única máquina — depende de driver/fabricante, sempre ler o estado real.
