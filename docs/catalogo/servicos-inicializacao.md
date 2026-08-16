# Catálogo técnico — Serviços, Inicialização e Bloatware

> Windows 11 24H2/25H2 (build 26100+) e Windows 10 22H2. Estados de serviço confirmados ao vivo via `Get-Service`/`Get-ScheduledTask` numa máquina real build 26200. **Achado central**: nomes de tarefas/serviços mudam entre builds — o produto deve consultar o estado real da máquina do usuário em tempo de execução, nunca aplicar uma lista fixa hard-coded.

## Perfis de recomendação

Todo item abaixo é marcado **Pessoal** / **Trabalho** / **Ambos** — no perfil Trabalho, a barra de segurança é mais alta (nunca tocar em nada que TI possa depender).

## Parte 1 — Seguros para desativar (com ressalvas)

| Serviço | Nome técnico | Ganho | Risco se desativar | Perfil |
|---|---|---|---|---|
| Fax | `Fax` | Nulo-Baixo (já Manual) | Nenhum | Ambos |
| Bluetooth Support | `bthserv` | Baixo | Quebra periféricos Bluetooth pareados — **checar antes** | Pessoal (checar em Trabalho) |
| Windows Insider | `wisvc` | Nulo | Nenhum (trava proteção contra entrar em canal beta por acidente) | Ambos |
| Retail Demo | `RetailDemo` | Nulo (já inativo de fábrica) | Nenhum | Ambos |
| Downloaded Maps Manager | `MapsBroker` | Baixo | Mapas offline param de atualizar | Ambos |
| Geolocation | `lfsvc` | Baixo (estava **Running** mesmo sendo Manual) | **Moderado** — quebra "Encontrar meu dispositivo" e fuso horário automático | Pessoal (Trabalho geralmente usa Find My Device via Intune) |
| Windows Biometric Service | `WbioSrvc` | Baixo | **Moderado** — quebra login por digital/rosto (Windows Hello) se em uso | Só Pessoal, após checar hardware biométrico |
| Secondary Logon | `seclogon` | Nulo-Baixo | Baixo (perde "Executar como") | Pessoal |
| Program Compatibility Assistant | `PcaSvc` | Baixo | Moderado — perde assistente de compatibilidade para software legado | Pessoal (manter em Trabalho com apps corporativos antigos) |
| Xbox services (Auth/GameSave/NetApi) | `XblAuthManager`, `XblGameSave`, `XboxNetApiSvc` | Baixo | Quebra login/multiplayer em jogos com Xbox Live, mesmo não-Microsoft | Pessoal não-gamer |
| Xbox Accessory Management | `XboxGipSvc` | Baixo | Quebra reconhecimento de controle Xbox USB/Bluetooth — **separar dos outros 3 na UI** | Pessoal, checar periférico |
| Mobile Hotspot | `icssvc` | Nulo-Baixo | Quebra hotspot/ICS | Ambos |
| Phone Service | `PhoneSvc` | Nulo-Baixo | Quebra Vínculo com o Celular (Phone Link) | Pessoal, checar uso |
| dmwappushservice | `dmwappushservice` | Nulo-Baixo | **Arriscado em PC gerenciado** — interfere com MDM/Intune | **Só Pessoal** |
| WalletService | `WalletService` | Nulo | Nenhum | Ambos |
| Touch Keyboard/Handwriting | `TabletInputService` | Baixo | Moderado — remove teclado virtual de acessibilidade | Pessoal, só sem tela touch |
| Smart Card | `SCardSvr` | Nulo-Baixo | **Arriscado no Brasil** — quebra certificado digital A3 (comum para e-CPF/e-CNPJ, contadores, MEIs) | Pessoal, checar certificado antes; nunca em Trabalho |
| Remote Registry | `RemoteRegistry` | Nulo (**já desativado de fábrica**) | — | Item de auditoria/verificação apenas, não otimização |
| Print Spooler | `Spooler` | Baixo (ganho é de segurança, não performance) | Ver discussão completa abaixo | Pessoal, só sem nenhuma impressora (nem virtual) |

### Print Spooler — discussão ampliada
Histórico real de vulnerabilidades graves (família **PrintNightmare**, CVE-2021-34527 e correlatas — execução remota de código com privilégios SYSTEM), com novas CVEs relacionadas em 2022-2025. **A orientação oficial da Microsoft (KB5005010) nunca foi "desative o serviço"** — foi endurecer restrições de instalação de drivers (Point and Print). CISA/Microsoft recomendam desativar em **servidores que não imprimem**, não em clientes domésticos. Muitos usuários usam "Microsoft Print to PDF"/"Microsoft XPS Document Writer" (dependem do Spooler) sem perceber.
**Recomendação**: não incluir na lista "segura" por padrão — só como opção avançada condicionada a "nenhuma impressora detectada".

## Parte 2 — NUNCA desativar (verificado ao vivo, todos críticos)

`RpcSs`, `RpcEptMapper`, `DcomLaunch`, `BFE` (Base Filtering Engine — a própria MS diz que pará-lo "reduz significativamente a segurança e resulta em comportamento imprevisível de firewall/IPsec"; é alvo clássico de malware), `EventLog`, `PlugPlay`, `Power`, `ProfSvc`, `Winmgmt` (WMI), `Schedule` (Agendador de Tarefas — ironicamente executa a própria manutenção automática do Windows), `MpsSvc` (Firewall), `CryptSvc`, `gpsvc` (Cliente de Política de Grupo — o Windows normalmente **nem permite** desativar pela UI; forçar via registro causa falhas graves de logon), `LSM`, `Dhcp`, `Dnscache`, `wscsvc` (Central de Segurança — não é tecnicamente crítico, mas desativar faz o usuário **perder a visibilidade** de status do antivírus — risco de segurança indireto para exatamente o público leigo que este produto atende), `nsi`, `NlaSvc`.

### Windows Update (`wuauserv`) — caso especial
Tecnicamente não é boot-critical (dá para desativar sem o Windows travar na hora), e por isso muitos guias o incluem como "seguro" — mas o consenso de segurança e a posição pública da Microsoft são claramente contra desativação permanente (caso clássico: WannaCry/EternalBlue 2017, só afetou PCs sem update que já existia havia meses). O Windows também "briga de volta" — tarefas do Orquestrador de Atualizações reativam o serviço sozinhas, resultando em sistema instável que ora atualiza, ora não. Com Windows 10 22H2 fora de suporte regular desde 14/10/2025 (só ESU pago), isso é ainda mais crítico para quem está nessa versão.
**Recomendação**: nunca oferecer "desativar Windows Update" — só os controles nativos suportados (pausar por N dias, Horário Ativo).

### Windows Defender (`WinDefend`) — achado de engenharia importante
**Verificar `Get-Service WinDefend` para decidir se o antivírus está ativo é heurística NÃO CONFIÁVEL nos builds atuais.** Nesta máquina, `WinDefend` aparece Manual/Stopped mesmo com o Defender funcionando — builds recentes reorganizaram partes dele em serviços/processos protegidos adicionais que não seguem mais o padrão clássico "Automatic+Running". **O jeito correto**: `Get-MpComputerStatus` (módulo PowerShell Defender) ou a API do Windows Security Center. Tamper Protection (ligada por padrão) já bloqueia a maioria das tentativas de desativação por fora do app oficial.

## Parte 3 — SysMain (ex-Superfetch): veredicto por tipo de disco

- **Chave/comando**: `Set-Service -Name SysMain -StartupType Disabled`.
- **HD mecânico (HDD)**: manter **ligado** — cenário original para o qual foi desenhado, ganho real ainda genuíno. Desde o Windows 8 já é ciente de SSD vs HDD e ajusta comportamento sozinho.
- **SSD/NVMe**: confirmado_com_ressalva — desativar é seguro, ganho real é marginal/beira o placebo para a maioria com RAM suficiente. Mito antigo de "gasta vida útil do SSD" não se sustenta mais (comportamento já é conservador em SSD desde o Win8).
- **Único efeito colateral real**: desativa o ReadyBoost (hoje irrelevante).
- **Recomendação de produto**: decisão **automática** baseada em `Get-PhysicalDisk`/`MediaType` (HDD vs SSD), não escolha manual do usuário leigo.
- **Fonte**: [Microsoft Q&A — should i disable sysmain](https://answers.microsoft.com/en-us/windows/forum/all/should-i-didable-sysmainsuperfetch-if-i-want-to/4b3ee632-ff65-4aac-9154-2a1b7a3a9576).

## Parte 4 — Programas de inicialização

- **Onde**: Configurações → Aplicativos → Inicialização, ou Gerenciador de Tarefas → aba "Aplicativos de Inicialização". Ambos leem chaves `Run`/`RunOnce` (HKLM/HKCU + Wow64) e pastas Startup.
- **⚠️ Não cobre tudo**: não inclui serviços, drivers, nem a maioria das tarefas agendadas — Autoruns (Sysinternals) mostra um conjunto bem mais completo.
- **Limiares oficiais exatos de "Impacto"** (confirmados em 2 documentos oficiais):
  - Baixo: <300ms CPU **e** <~300KB disco
  - Médio: 300ms-1000ms CPU **ou** ~300KB-3MB disco
  - Alto: >1s CPU **ou** >3MB disco
  - Calculado empiricamente pelo Windows observando boots reais, não estimativa teórica.
- **Nuance importante**: mede o efeito no login/pós-boot, não o boot bruto (POST+kernel+drivers, dominado por UEFI/SSD, praticamente não afetado por apps de usuário). A Microsoft não promete "quantos segundos" — recomenda abordagem incremental.
- **Maior ganho real vem de poucos "vilões"**: clientes de nuvem antigos, launchers de jogos, suítes de fabricante (Realtek, Dell/HP/Lenovo, Corsair iCUE, Logitech G Hub), agentes de update de terceiros — desativar dezenas de itens de Baixo impacto tem efeito cumulativo pequeno comparado a 2-3 de Alto impacto.
- **Veredicto**: confirmado — comunicar em segundos-reais-medidos, nunca prometer número sem medir na máquina do usuário.
- **Fonte**: [Configure startup applications in Windows](https://support.microsoft.com/en-us/windows/configure-startup-applications-in-windows-115a420a-0bff-4a6f-90e0-1934c844e473).

## Parte 5 — Bloatware pré-instalado (24H2/25H2)

- **Política nativa oficial**: "Remove Default Microsoft Store packages from the system" (`HKLM\SOFTWARE\Policies\Microsoft\Windows\Appx\RemoveDefaultMicrosoftStorePackages`, GPO — só Pro/Enterprise/Education) documenta ~25-26 apps que a própria Microsoft considera removíveis: Clipchamp, Copilot, Feedback Hub, Bing News/Weather, Fotos, Paint, Calculadora, Câmera, Bloco de Notas, Solitaire, Notas Autoadesivas, Tarefas, Gravador de Som, Outlook, Terminal, Assistência Rápida, componentes Xbox, Media Player, Ferramenta de Captura, Teams consumidor.
- **⚠️ Limitação importante**: a política só se aplica a **perfis de usuário novos** — não remove de contas existentes. Home precisa de PowerShell/winget.
- **Remoção universal**: `Get-AppxPackage -AllUsers <Nome> | Remove-AppxPackage`; provisionamento: `Get-AppxProvisionedPackage -Online | Where DisplayName -like "<Nome>*" | Remove-AppxProvisionedPackage -Online`; ou `winget uninstall --id <Id>`.
- **⚠️ Apps clássicos de "lista de bloatware" que JÁ NÃO EXISTEM mais** — não incluir no catálogo: Cortana (removida como app desde 22H2/23H2), Skype (Microsoft encerrou versão consumidor em maio/2025, substituído pelo Teams), 3D Viewer e Paint 3D (já removidos do provisionamento antes do 24H2).
- **Honestidade sobre impacto**: majoritariamente organização/preferência/privacidade, **não performance**. Apps UWP parados ficam "suspensos" pelo Windows — não consomem CPU/RAM relevante mesmo instalados. Ganho real de remover: espaço em disco modesto (dezenas de MB a poucos GB), menu mais limpo, superfície de update levemente menor.
- **Exceções com impacto real mensurável**: widget de Notícias/Clima e Teams consumidor (historicamente residente com o Windows) — desabilitar **inicialização automática/segundo plano** desses dois captura a maior parte do ganho real de performance desta categoria.
- **Blocos patrocinados no Menu Iniciar** (Spotify, TikTok, Prime Video etc.): a maioria **não vem instalada** — são atalhos que disparam download da Store ao clicar (publicidade). Não consomem recursos até serem clicados.
- **Veredicto**: confirmado_com_ressalva — reposicionar como "organização e privacidade", não "seu PC vai voar depois disso".

## Parte 6 — Tarefas agendadas de telemetria

**⚠️ Achado crítico de engenharia**: verificação ao vivo mostrou que nomes clássicos citados por guias antigos **já não existem** nesta build — `ProgramDataUpdater` e `KernelCeipTask` não foram encontradas; `Microsoft Compatibility Appraiser` virou **`Microsoft Compatibility Appraiser Exp`** (sufixo novo). Duas tarefas novas apareceram sem estar em listas antigas: `MareBackup` (backup de apps ao trocar de PC) e `SdbinstMergeDbTask` (manutenção local, **não é telemetria** — não classificar como item de privacidade).

**Correção importante**: `StartupAppTask`, citada por muitos guias genéricos como "telemetria", na verdade é *"Examina as entradas de inicialização e notifica se houver entradas em excesso"* — está ligada à Parte 4 (avisos de inicialização), **não** ao envio de dados à Microsoft.

| Caminho | Confirmada nesta build? | Função |
|---|---|---|
| `\Microsoft\Windows\Application Experience\Microsoft Compatibility Appraiser Exp` | Sim | Telemetria de compatibilidade (CEIP) |
| `\Microsoft\Windows\Application Experience\PcaPatchDbTask` | Sim | Atualiza base de compatibilidade |
| `\Microsoft\Windows\Application Experience\MareBackup` | Sim (nova) | Dados de apps para backup ao migrar de PC |
| `\Microsoft\Windows\Application Experience\ProgramDataUpdater` | **Não encontrada** — verificar antes de aplicar | Telemetria de uso (histórica) |
| `\Microsoft\Windows\Autochk\Proxy` | Sim | Info de chkdsk no boot |
| `\Microsoft\Windows\Customer Experience Improvement Program\Consolidator` | Sim | Consolida/envia dados CEIP |
| `\Microsoft\Windows\Customer Experience Improvement Program\UsbCeip` | Sim | Telemetria de dispositivos USB |
| `\Microsoft\Windows\Customer Experience Improvement Program\KernelCeipTask` | **Não encontrada** — verificar antes | Telemetria de kernel (histórica) |
| `\Microsoft\Windows\DiskDiagnostic\...DiskDiagnosticDataCollector` | Sim | Diagnóstico S.M.A.R.T. |
| `\Microsoft\Windows\Feedback\Siuf\DmClient` (+ OnScenarioDownload) | Sim | Motor de pesquisas de satisfação |
| `\Microsoft\Windows\Windows Error Reporting\QueueReporting` | Sim | Envia relatórios de erro/crash |
| `\Microsoft\Windows\Device Information\Device` | Sim | Censo/inventário de dispositivo |

- **Comando**: `Disable-ScheduledTask -TaskName "<nome>" -TaskPath "<caminho>\"`.
- **Impacto real**: **Nulo a Baixo em performance** — isso precisa ficar claro na UI. Valor real é **privacidade**, não velocidade (picos breves e esporádicos, não carga constante).
- **Veredicto**: confirmado_com_ressalva — **o produto deve consultar `Get-ScheduledTask` em tempo real na máquina do usuário**, nunca aplicar lista fixa hard-coded (nomes mudam entre builds/rings).

## Parte 7 — Windows Defender e exclusões de pasta

- **Comando**: `Add-MpPreference -ExclusionPath "<caminho>"`. Checar exclusões: `Get-MpPreference | Select ExclusionPath`.
- **⚠️ A Microsoft lista oficialmente pastas que NUNCA devem ser excluídas**: `C:\Windows\Temp`, `AppData\Local\Temp`, `C:\Users\` inteiro, extensões `.exe .dll .ps1 .js .zip .rar .7z`, processos `powershell.exe cmd.exe wscript.exe mshta.exe` — atacantes abusam justamente dessas pastas.

| Cenário | Impacto | Risco | Veredicto |
|---|---|---|---|
| Pasta de jogos (Steam/Epic/GOG detectada automaticamente) | Moderado — real durante download/patch de jogos grandes | Baixo-Moderado se restrito à pasta do launcher oficial; **alto** se pasta genérica de downloads (vetor comum de malware "jogo crackeado") | confirmado_com_ressalva |
| Pastas de desenvolvimento (`node_modules`, `.git`, WSL2 `.vhdx`, Docker) | **Moderado-Alto** — o ganho mais genuíno de toda a seção | Moderado — remove a camada que flagraria pacote npm/PyPI comprometido (ataques de supply-chain são vetor ativo, não hipotético) | confirmado_com_ressalva — exclusão granular por marcador (`.git`, `package.json`), nunca pasta de usuário inteira |
| Pasta Downloads | Baixo | **Alto** — maior probabilidade estatística de conter malware baixado | **perigoso** — não oferecer como toggle |
| Pasta Temp (usuário/sistema) | Baixo-Moderado | **Alto** — listada oficialmente pela Microsoft como "não excluir" | **perigoso** — não oferecer como toggle |

- **Veredicto geral**: ganho concentrado em 2 cenários específicos (jogos com launcher legítimo; dev com WSL2/Docker/node_modules). Fora disso, risco supera benefício. Nenhuma exclusão deve ser aplicada a pastas genéricas (Documentos, Downloads, Temp, Desktop, C:\ inteira) — sempre exigir confirmação explícita com aviso "isso reduz sua proteção contra vírus nesta pasta".
- **Fonte**: [Configure custom exclusions](https://learn.microsoft.com/en-us/defender-endpoint/microsoft-defender-antivirus-exclusions-configure), [Exclusions to avoid](https://learn.microsoft.com/en-us/defender-endpoint/defender-endpoint-exclusions-common-mistakes).

---

## Regras para a UI (derivadas deste domínio)

1. **Nunca desativar**: RPC, DCOM, BFE, EventLog, PlugPlay, Power, ProfSvc, WMI, Agendador de Tarefas, Firewall, CryptSvc, gpsvc, LSM, DHCP, DNS Client, Central de Segurança, Windows Update.
2. Estado real de serviços/tarefas deve ser **consultado ao vivo** (`Get-Service`/`Get-ScheduledTask`) na máquina do usuário — nunca hard-coded, nomes mudam entre builds.
3. `Get-Service WinDefend` **não é confiável** para saber se o Defender está ativo — usar `Get-MpComputerStatus`.
4. SysMain: decisão automática por tipo de disco (HDD liga, SSD é opcional com aviso de ganho marginal), nunca escolha manual cega.
5. Exclusões do Defender: nunca em pastas genéricas (Downloads, Temp, Documentos); só em pastas específicas detectadas automaticamente (biblioteca de launcher, marcador de projeto dev), sempre com confirmação e aviso de risco.
6. Bloatware: nunca vender remoção como "ganho de performance" — é organização/privacidade. Não incluir apps que já não existem (Cortana, Skype clássico, 3D Viewer/Paint 3D).
7. Print Spooler, Smart Card, Bluetooth, Biometria, Geolocalização: exigem checagem de hardware/uso real antes de oferecer o toggle, nunca pré-marcados.
8. Perfil Trabalho: excluir por padrão dmwappushservice, Smart Card e qualquer item que dependa de detecção de domínio/MDM.

## Notas de incerteza

- Verificação ao vivo é de **uma única máquina/build** (Win11 Pro build 26200, ramo 25H2) — estados podem variar por edição, canal de update, customização prévia. O produto deve fazer essa mesma checagem em tempo de execução, não confiar neste catálogo como valores fixos.
- `OneSyncSvc` não verificado ao vivo (nome sufixado por SID de usuário, não fixo).
- Recomendação de excluir `.vhdx` do WSL2/Docker do Defender: amplamente conhecida na comunidade, mas página oficial dedicada não localizada nesta pesquisa (orçamento de busca esgotado).
- `ProgramDataUpdater`/`KernelCeipTask` "não encontradas" pode ser mudança real recente ou particularidade desta máquina — tratar como "verificar antes de aplicar `/DISABLE`" (comando falha silenciosamente se a tarefa não existir, risco prático baixo).
- Limiares exatos de Impacto de Inicialização (300ms/300KB, 1000ms/3MB): confirmados em 2 páginas oficiais, mas ambas de manutenção pouco frequente (uma datada de 2018) — confiança alta na lógica, moderada nos números exatos permanecerem idênticos na build mais atual.
