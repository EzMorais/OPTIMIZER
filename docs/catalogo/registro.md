# Catálogo técnico — Registro do Windows

> Windows 11 24H2/25H2 (build 26100+) e Windows 10 22H2. Verificação combina documentação oficial (learn.microsoft.com) e leitura empírica ao vivo do registro numa máquina real build 26200. Onde os dois discordam, a leitura ao vivo prevalece e está sinalizada.

## Responsividade da interface

### MenuShowDelay
- **Chave**: `HKCU\Control Panel\Desktop`, valor `MenuShowDelay`, **REG_SZ** (string, não DWORD). Padrão comum ≈ `400` (ms). Otimizado: `0`–`10`.
- **Ganho**: menus em cascata (Explorer, apps Win32 clássicos) abrem na hora, sem espera ao passar o mouse.
- **Impacto**: Médio — um dos poucos tweaks com efeito imediatamente perceptível por qualquer leigo. **Risco**: Seguro. **Veredicto**: confirmado.
- **Ressalva**: só afeta menus do stack clássico Win32; não afeta menus de apps modernos (WinUI/UWP).
- **Fonte**: [SystemParametersInfoW — Microsoft Learn](https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-systemparametersinfow) (`SPI_SETMENUSHOWDELAY`)

### MouseHoverTime
- **Chave**: `HKCU\Control Panel\Mouse`, valor `MouseHoverTime`, **REG_SZ**. Padrão oficial = `400` ms. Otimizado: `0`–`100`.
- **Ganho**: dicas de ferramenta e pré-visualização reagem mais rápido ao parar o mouse.
- **Impacto**: Baixo (afeta só hover específico, não a sensação geral de velocidade). **Risco**: Seguro. **Veredicto**: confirmado — raro caso com valor default documentado oficialmente e explicitamente.
- **Fonte**: [SystemParametersInfoW](https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-systemparametersinfow) (`SPI_GETMOUSEHOVERTIME`)

### Animações — 4 chaves independentes (toggles separáveis)

| Chave | Caminho | Tipo | Padrão | O que faz |
|---|---|---|---|---|
| `UserPreferencesMask` | `HKCU\Control Panel\Desktop` | REG_BINARY (8 bytes) | variável | Máscara mestre de todos os efeitos de UI legados (sombra, fade, animação de combobox). Mapa de bits **nunca publicado oficialmente pela Microsoft** — reconstruído por engenharia reversa da comunidade, não 100% oficial. |
| `MinAnimate` | `HKCU\Control Panel\Desktop\WindowMetrics` | REG_SZ ("0"/"1") | `1` | Animação de minimizar/maximizar janela. Puramente estético. |
| `TaskbarAnimations` | `HKCU\...\Explorer\Advanced` | REG_DWORD | `1` | Animações da barra de tarefas. **Incerto se ainda controla o shell novo do Win11** (Taskbar/Start reescritos em XAML/Composition) — efeito mais completo e comprovado no Windows 10. |
| `VisualFXSetting` | `HKCU\...\Explorer\VisualEffects` | REG_DWORD | `0` | Toggle mestre: `0`=deixar o Windows decidir, `1`=melhor aparência, `2`=melhor desempenho, `3`=personalizado. Equivale ao botão da caixa "Opções de Desempenho". Mais robusto que editar `UserPreferencesMask` manualmente. |

- **Bônus**: `EnableTransparency` em `HKCU\...\Themes\Personalize` (REG_DWORD, padrão `1`) controla o efeito Acrylic/Mica — mecanismo separado e mais moderno, não faz parte da UserPreferencesMask.
- **Impacto geral**: Baixo a Médio em hardware antigo/GPU integrada fraca; **placebo em qualquer PC comprado nos últimos 8-10 anos** — vale reclassificar como ganho real só em hardware muito fraco (<2015, Atom/Celeron, <4GB RAM) ou VM sem aceleração 3D. **Risco**: Seguro, com ressalva de acessibilidade (relatos não confirmados oficialmente de que desligar animações via `UserPreferencesMask` pode afetar dicas de tempo do Narrador/Lupa).
- **Nota de engenharia**: mudar `VisualFXSetting` sozinho às vezes não é refletido até reiniciar o `explorer.exe` ou usar a API `SystemParametersInfo` — não basta escrever o registro.
- **Veredicto**: confirmado_com_ressalva (mecanismo real, ganho hoje é marginal na maioria dos PCs).
- **Fonte**: sem página oficial dedicada ao mapa de bits — [Microsoft Q&A](https://learn.microsoft.com/en-us/answers/questions/2185159/registry-key-to-turn-on-show-animations-in-windows); leitura direta do registro.

### Efeitos avulsos (modo "Personalizado")

`DragFullWindows` (REG_SZ, `HKCU\Control Panel\Desktop`) · `ListviewAlphaSelect` e `ListviewShadow` (REG_DWORD, `Explorer\Advanced`) · `EnableAeroPeek` (REG_DWORD, `HKCU\...\DWM`). Todos puramente estéticos, custo de CPU/GPU irrelevante em hardware pós-2015.
- **Veredicto**: placebo em hardware moderno; confirmado_com_ressalva só em hardware muito antigo/fraco.

## Prioridade de CPU

### Win32PrioritySeparation
- **Chave**: `HKLM\SYSTEM\CurrentControlSet\Control\PriorityControl`, valor `Win32PrioritySeparation`, REG_DWORD. Padrão não mexido = `2`. A GUI oficial ("Configurações avançadas → Desempenho → Agendamento do processador") grava `0x26` (38) para "Programas" e `0x18` (24) para "Serviços em segundo plano".
- **Ganho**: controla se o Windows dá prioridade extra de CPU ao app em foco, e o tamanho/tipo das fatias de tempo.
- **Impacto**: Baixo para a maioria (padrão já favorece o app em uso); Médio em cenários específicos (carga pesada em segundo plano — encode, compilação — enquanto se usa o PC ativamente).
- **Risco**: Seguro nas duas opções oficiais (0x26/0x18); Moderado para valores "mágicos" customizados fora dessas combinações — sem respaldo documentado, efeito próximo de placebo sobre o padrão.
- **Veredicto**: confirmado_com_ressalva. **Nota importante**: a Microsoft não publica mais o detalhamento bit a bit desse valor — as poucas fontes de comunidade disponíveis se contradisseram entre si nos detalhes finos.
- **Fonte**: leitura direta do registro (padrão=2 confirmado); sem página oficial atual dedicada.

## MMCSS — SystemResponsiveness e NetworkThrottlingIndex

### SystemResponsiveness
- **Chave**: `HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Multimedia\SystemProfile`, valor `SystemResponsiveness`, REG_DWORD. Default histórico = `20`.
- **⚠️ Achado crítico confirmado pela documentação oficial**: *"values that are not evenly divisible by 10 are rounded down to the nearest multiple of 10. Values below 10 and above 100 are clamped to 20."* **O valor `0`, recomendado em praticamente todo guia de "otimização para gamers" na internet, é automaticamente grampeado para `20` pelo próprio Windows — o oposto do efeito pretendido.** O menor valor não descartado pelo clamp é `10`.
- **Recomendação de produto**: oferecer `10` como valor "otimizado", nunca `0`. Isso é um diferencial de credibilidade real.
- **Veredicto**: confirmado_com_ressalva.
- **Fonte**: [Multimedia Class Scheduler Service — Microsoft Learn](https://learn.microsoft.com/en-us/windows/win32/procthread/multimedia-class-scheduler-service) (citação literal do clamp).

### NetworkThrottlingIndex
- **Chave**: mesma árvore, valor `NetworkThrottlingIndex`, REG_DWORD. Default = `10` (0xA). Otimizado comum: `0xFFFFFFFF` (desativa o throttling).
- **Ganho**: remove o limite de ~10.000 pacotes/s que o Windows aplica quando há tarefa multimídia rodando, para reservar CPU para áudio/vídeo não engasgar.
- **Impacto**: Baixo para a maioria (só perceptível em rede Gigabit+ com tráfego pesado simultâneo à mídia); Médio em produção de áudio/vídeo ou transferências grandes durante VoIP/streaming.
- **Risco**: Seguro — pior caso é reintroduzir os raros engasgos que o mecanismo evita.
- **Veredicto**: confirmado_com_ressalva — mecanismo documentado originalmente por engenheiro sênior da Microsoft (Mark Russinovich), mas **não aparece mais na documentação oficial ativa** (só em conteúdo arquivado).
- **Fonte**: [Vista Multimedia Playback and Network Throughput — arquivo oficial Microsoft](https://learn.microsoft.com/en-us/archive/blogs/markrussinovich/vista-multimedia-playback-and-network-throughput).

## Perfis MMCSS Tasks (Games e Audio)

**Achado mais valioso desta seção**: a documentação oficial confirma que 2 das 5 chaves mais "tunadas" por guias de gaming **não têm efeito nenhum hoje**.

| Chave | Tipo | Default (Games/Audio) | Veredicto |
|---|---|---|---|
| `Scheduling Category` | REG_SZ | Medium/Medium | **confirmado** — único ajuste genuinamente funcional. `High` roda em prioridade 23-26; `Medium` em 16-22. Risco Moderado (High foi pensado para Pro Audio). |
| `Priority` | REG_DWORD (1-8) | 2/6 | confirmado_com_ressalva — doc oficial: *"For tasks with a Scheduling Category of High, this value is always treated as 2."* Guias que mandam mudar Category=High **e** Priority=8 desperdiçam metade do tweak. |
| `GPU Priority` | REG_DWORD (0-31) | 8/8 | **placebo** — doc oficial literal: *"This priority is not yet used."* |
| `SFIO Priority` | REG_SZ | Normal/Normal | **placebo** — doc oficial literal: *"This value is not used."* |
| `Background Only` | REG_SZ | False/True | confirmado (mecanismo real) mas sem caso de uso de otimização válido — não expor como toggle. |

Caminho base: `HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Multimedia\SystemProfile\Tasks\Games` (e `\Tasks\Audio`).
- **Fonte**: [Multimedia Class Scheduler Service — Microsoft Learn](https://learn.microsoft.com/en-us/windows/win32/procthread/multimedia-class-scheduler-service) (citações literais "not yet used" / "not used").

## Gerenciamento de memória

| Chave | Caminho | Veredicto | Nota |
|---|---|---|---|
| `LargeSystemCache` | `HKLM\...\Session Manager\Memory Management` | **obsoleto** | Página oficial atual de chaves de memória não menciona mais este valor — resíduo de Windows 2000/XP; GUI que o expõe só existe em Server. |
| `DisablePagingExecutive` | mesma árvore | **obsoleto** | Única doc oficial encontrada é página **arquivada do Windows Server 2003**, marcada `NOINDEX,NOFOLLOW` pela própria Microsoft. |
| `ClearPageFileAtShutdown` | mesma árvore | confirmado_com_ressalva | **Não é tweak de performance — é o oposto**: aumenta o tempo de desligamento (zera o pagefile por segurança). Muitos guias o listam erroneamente como otimização. Redundante em disco com BitLocker/criptografia. |
| `EnablePrefetcher` / `EnableSuperfetch` | `...\Memory Management\PrefetchParameters` | confirmado_com_ressalva | Ganho real bem menor em SSD/NVMe do que em HDD (era desenhado para disco mecânico). `EnableSuperfetch` só tem efeito se o serviço **SysMain** estiver rodando. |

## Latência de entrada (mouse e teclado)

| Chave | Caminho/Tipo | O que é | Veredicto |
|---|---|---|---|
| `MouseSensitivity` | `HKCU\Control Panel\Mouse`, REG_SZ, padrão `10` | Velocidade do ponteiro (1-20) | confirmado — preferência pessoal, não otimização |
| `SmoothMouseXCurve`/`YCurve`, `MouseSpeed`, `MouseThreshold1/2` | mesma chave | Aceleração do mouse ("Aprimorar precisão do ponteiro") | **confirmado — não é mito.** Mecanismo real e documentado (`SPI_GETMOUSE`); desativar (todos em `0`) dá mapeamento 1:1 consistente, preferido por jogadores competitivos. Alto impacto para quem se importa. |
| `KeyboardDelay` | `HKCU\Control Panel\Keyboard`, REG_SZ "0"-"3" | Delay antes de repetir tecla segurada | confirmado, mas **não é latência de tecla única** — não confundir com tempo de resposta do teclado |
| `KeyboardSpeed` | mesma chave, REG_SZ "0"-"31" | Taxa de repetição após o delay | confirmado, mesma ressalva |

- **Fonte**: [SystemParametersInfoW — Microsoft Learn](https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-systemparametersinfow) para todos os itens acima.

---

## Explorer — Thumbnails, Busca

### Cache de miniaturas
- **⚠️ Correção de nome**: a chave popularmente citada "DisableThumbnailCache" **não é o nome oficial**. A policy oficial (`NoCacheThumbNailPictures`) grava em `HKCU\Software\Microsoft\Windows\CurrentVersion\Policies\Explorer`, valor **`NoThumbnailCache`** (REG_DWORD).
- **Ganho**: economiza espaço e evita escrita de `thumbcache_*.db`, mas torna pastas de foto/vídeo mais lentas (recalcula sempre).
- **Impacto**: Baixo. **Veredicto**: confirmado_com_ressalva — troca ruim de UX para ganho hoje irrisório.
- Relacionado: `DisableThumbsDBOnNetworkFolders` (`HKCU\Software\Policies\Microsoft\Windows\Explorer`) evita thumbs.db em pastas de rede — mais relevante no perfil "trabalho".
- **Fonte**: [ADMX_WindowsExplorer Policy CSP](https://learn.microsoft.com/en-us/windows/client-management/mdm/policy-csp-admx-windowsexplorer).

### Indexação e busca (Windows Search)
- Busca em local **indexado** (Documentos, Desktop, Menu Iniciar) = quase instantânea (banco pronto, inclusive conteúdo de texto). Local **não indexado** = varredura ao vivo por nome apenas (padrão).
- `ConnectedSearchUseWeb`/policy `DoNotUseWebResults` (`HKLM\SOFTWARE\Policies\Microsoft\Windows\Windows Search`) — bloqueia resultados do Bing na busca local. Ganho de velocidade pequeno, valor real é privacidade/previsibilidade. Só Pro/Enterprise/Education. **Veredicto**: confirmado.
- **Excluir pastas da indexação** (relevante para perfil trabalho/dev — `node_modules`, `.git`): não existe chave simples de registro — usa a API `ISearchCrawlScopeManager` ou o atributo de pasta `FILE_ATTRIBUTE_NOT_CONTENT_INDEXED`. **Veredicto**: confirmado, ganho real para desenvolvedores (indexador competindo por I/O durante builds), mas **exige engenharia via API, não regedit puro**.
- `DisableBackoff` (`HKLM\SOFTWARE\Policies\Microsoft\Windows\Windows Search`) — **atenção, é o oposto de uma otimização**: habilitá-lo faz o indexador rodar a toda velocidade mesmo com o usuário usando o PC ativamente, piorando a responsividade em primeiro plano. Nunca oferecer como "ativar para otimizar".

## Telemetria

### AllowTelemetry
- **Chave**: `HKLM\SOFTWARE\Policies\Microsoft\Windows\DataCollection`, valor `AllowTelemetry`, REG_DWORD. `0`=Security, `1`=Basic (padrão desde 1903), `2`=Enhanced (**descontinuado no Win11/Server 2022+**), `3`=Full.
- **⚠️ Achado crítico**: o nível `0` (Security/desligado) **só é respeitado em Windows Enterprise, Education e Server** — confirmado literalmente na doc oficial. **Em Windows Home e Pro (o grosso do público leigo brasileiro), configurar `0` não desliga a telemetria — o mínimo (`1`) continua sendo enviado.** O produto deve deixar isso claro na UI, nunca prometer "telemetria zero" em Home/Pro.
- **Impacto real**: Nulo em performance — muda **o que** é enviado, não desliga o mecanismo de coleta.
- **Veredicto**: confirmado_com_ressalva — é item de **privacidade**, não de performance; não apresentar como ganho de velocidade.
- **Fonte**: [Configure Windows diagnostic data in your organization — Microsoft Learn](https://learn.microsoft.com/en-us/windows/privacy/configure-windows-diagnostic-data-in-your-organization).

### Serviço DiagTrack (Connected User Experiences and Telemetry)
- Desligar via `sc config DiagTrack start= disabled`. **Risco**: Moderado — frequentemente revertido por updates do Windows, pode quebrar Feedback Hub, não é a via oficialmente suportada (a MS recomenda usar `AllowTelemetry`, não desligar o serviço).
- **Veredicto**: confirmado_com_ressalva — ganho de privacidade, não de performance (serviço tem baixo duty-cycle, sem evidência de consumo relevante de CPU/disco).

## Hibernação e Fast Startup

### Fast Startup (HiberbootEnabled)
- **Chave**: `HKLM\SYSTEM\CurrentControlSet\Control\Session Manager\Power`, valor `HiberbootEnabled`, REG_DWORD, padrão `1`.
- **⚠️ A própria Microsoft desaconselha mexer**: documentação oficial afirma textualmente **"Disabling Fast Startup is not recommended"**.
- **Riscos documentados oficialmente**: problemas de dual-boot com Linux (partição NTFS pode ficar só leitura ou causar inconsistência); falhas de driver conhecidas (KB3211190, `IO_DUMP_DRIVER_LOAD_FAILURE`).
- **Recomendação de produto**: manter desligado por padrão como toggle; se oferecido, avisar "recomendado desligar apenas se você usa dual-boot com Linux ou tem problemas ao desligar", nunca apresentar como otimização universal.
- **Veredicto**: confirmado_com_ressalva.
- **Fonte**: [Fast startup causes hibernation or shutdown to fail — Microsoft Learn](https://learn.microsoft.com/en-us/troubleshoot/windows-client/setup-upgrade-and-drivers/fast-startup-causes-system-hibernation-shutdown-fail) (contém as citações "not recommended" e "enabled by default").

## Xbox Game Bar / Game DVR

| Chave | Escopo | Veredicto |
|---|---|---|
| `GameDVR_Enabled` (`HKCU\System\GameConfigStore`) | por usuário | confirmado_com_ressalva — efeito real mas variável por jogo/hardware; há relatos de "frame caps" causados especificamente pelo GameDVR |
| `AllowGameDVR` (`HKLM\SOFTWARE\Policies\Microsoft\Windows\GameDVR`) | política, máquina toda | confirmado_com_ressalva — útil para perfil "trabalho"/multiusuário; documentação pública oficial fraca (não está nas páginas Policy CSP padrão) |

## NTFS — disablelastaccess

- **Comando**: `fsutil behavior set disablelastaccess {0|1|2|3}` (0=on/usuário, 1=off/usuário, 2=on/sistema — padrão comum, 3=off/sistema).
- **Ganho real menor do que a fama sugere**: desde o Windows Vista, a atualização de "último acesso" já fica em cache até 1h e é aproveitada de carona com outras escritas — a Microsoft já mitigou boa parte do custo há 15+ anos.
- **Risco documentado oficialmente**: *"The disablelastaccess parameter can affect programs such as Backup and Remote Storage, which rely on this feature."*
- **Veredicto**: confirmado_com_ressalva.
- **Fonte**: [fsutil behavior — Microsoft Learn](https://learn.microsoft.com/en-us/windows-server/administration/windows-commands/fsutil-behavior).

## Mitigações Spectre/Meltdown/L1TF/MDS — item mais sensível do catálogo

- **Chaves**: `HKLM\SYSTEM\CurrentControlSet\Control\Session Manager\Memory Management`, `FeatureSettingsOverride` + `FeatureSettingsOverrideMask`, REG_DWORD. Valor `3`+`3` desativa Spectre v2/Meltdown originais; variantes posteriores (L1TF, MDS, Retbleed, BHI, Downfall...) somam bits adicionais espalhados por KBs específicas — **não há tabela oficial única e centralizada**.
- **Impacto de desativar**: MUITO variável. Blog oficial de segurança da MS (2018) reportava queda "de um dígito" em CPUs Skylake+/Win10; mais perceptível em CPUs Haswell ou mais antigas e em Server com I/O intenso. **Não há benchmark recente (2023-2025) rigoroso e verificável para Windows** — uma alegação de "≤2,1% overhead em 2024" foi checada e **não tem citação real por trás**, descartada.
- **Risco**: **Arriscado** — expõe a máquina a técnicas de vazamento de memória entre processos já demonstradas publicamente, com novas variantes descobertas até 2024.
- **Veredicto**: **perigoso** — deliberadamente não classificado como "confirmado_com_ressalva" apesar do mecanismo funcionar exatamente como documentado. Ganho incerto, risco de segurança bem estabelecido.
- **Companheiro obrigatório**: `Get-SpeculationControlSettings` (script oficial da Microsoft, `Install-Module SpeculationControl` ou baixar de `https://aka.ms/SpeculationControlPS`) — somente leitura, mostra status real de cada mitigação. **Incluir sempre como botão "verificar status" ao lado de qualquer controle relacionado.**
- **Fonte**: [KB4073119 — Windows client guidance for IT Pros](https://support.microsoft.com/en-us/topic/kb4073119-windows-client-guidance-for-it-pros-to-protect-against-silicon-based-microarchitectural-and-speculative-execution-side-channel-vulnerabilities-35820a8a-ae13-1299-88cc-357f104f5b11), [microsoft/SpeculationControl no GitHub](https://github.com/microsoft/SpeculationControl).

## Interrupções USB e Jitter de Entrada

### Moderação de Interrupções USB XHCI (`#usb-xhci-imod`)
- **Chave**: `HKLM\SYSTEM\CurrentControlSet\Services\USBXHCI\Parameters`, valor `InterruptModeration`, REG_DWORD. Padrão = `1` (ativado). Otimizado: `0` (desativado).
- **Ganho**: desativa a retenção em lote de interrupções no controlador host USB 3.0/3.1/3.2, despachando pacotes do mouse e teclado imediatamente para a fila ISR/DPC da CPU.
- **Impacto**: Alto em mouses de alta taxa de atualização (1000Hz, 4000Hz, 8000Hz), reduzindo dispersão temporal e micro-stutters no tracking.
- **Risco**: Baixo. Em CPUs com 2 núcleos muito antigas pode gerar leve aumento de carga de interrupções.
- **Veredicto**: confirmado (respaldado por pesquisas e benchmarks do `valleyofdoom/PC-Tuning` §11.39).

## Agendamento de Janelas e Mensagens Win11

### BackgroundWindowMessageRate (`#background-window-message-rate`)
- **Chave**: `HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Windows`, valor `BackgroundWindowMessageRate`, REG_DWORD. Padrão = ausente / `0`. Otimizado: `1`.
- **Ganho**: limita a taxa de despachos de mensagens de janela não prioritárias para processos fora de foco, liberando o thread de renderização em primeiro plano.
- **Impacto**: Médio em multitarefa com jogos e compiladores.
- **Risco**: Baixo. Não bloqueia downloads de background nem reprodução de áudio.
- **Veredicto**: confirmado no Windows 11 22H2+ (`valleyofdoom/PC-Tuning` §11.52).

## Subsistema Gráfico e Tela Cheia Real

### Comportamento Fullscreen Exclusive FSE (`#gamedvr-fse`)
- **Chave**: `HKCU\System\GameConfigStore`, valores `GameDVR_FSEBehaviorMode` (`2`), `GameDVR_HonorUserFSEBehaviorMode` (`1`), `GameDVR_FSEBehavior` (`2`), REG_DWORD.
- **Ganho**: impede a injeção do wrapper DWM de sobreposição quando um aplicativo DirectX solicita modo de tela cheia exclusivo, reduzindo latência de apresentação e eliminando descompassos de frame timing.
- **Impacto**: Alto em jogos competitivos.
- **Risco**: Baixo.
- **Veredicto**: confirmado (`valleyofdoom/PC-Tuning` §11.41 / §11.42).


---

## Regras para a UI (derivadas deste domínio)

1. `SystemResponsiveness` otimizado deve ser `10`, nunca `0` (clamp documentado oficialmente torna `0` inútil).
2. `GPU Priority` e `SFIO Priority` dos perfis MMCSS **não entram no catálogo** — a própria Microsoft os documenta como sem efeito.
3. `AllowTelemetry=0` deve exibir aviso "não confirmado em Windows Home/Pro" quando detectada essa edição.
4. Fast Startup permanece desligado por padrão (é o padrão de fábrica do Windows manter ligado) — se oferecido como toggle, com aviso específico de dual-boot/driver.
5. Mitigações de CPU (Spectre/Meltdown) nunca entram no perfil padrão nem no avançado sem confirmação explícita adicional ("digite para confirmar que entende o risco").
6. `NoThumbnailCache` deve usar o nome de chave correto — nunca "DisableThumbnailCache".

## Notas de incerteza principais

- Mapa de bits do `UserPreferencesMask` — nunca publicado oficialmente, reconstruído por engenharia reversa da comunidade.
- Efeito real das chaves de animação legadas sobre o shell novo (Start/Taskbar) do Windows 11 — plausível que seja parcial, não confirmado tecnicamente.
- Semântica bit a bit completa do `Win32PrioritySeparation` — sem página oficial atual, fontes de comunidade se contradisseram.
- Tabela numérica completa de `FeatureSettingsOverride` por CVE pós-2018 — espalhada em KBs individuais, não consolidada oficialmente; **não usar números específicos em produção sem checar a KB vigente**.
- Nome curto do serviço `DiagTrack` não aparece verbatim em documentação oficial (só o nome longo "Connected User Experiences and Telemetry").
