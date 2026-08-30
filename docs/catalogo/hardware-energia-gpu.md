# Catálogo técnico — Hardware, Energia e GPU

> Windows 11 24H2/25H2 e Windows 10 22H2. GUIDs confirmados ao vivo via `powercfg /list`, `/query`, `/aliases` numa máquina real build 26200.

## 1. Planos de energia + "Desempenho Máximo" oculto (Ultimate Performance)

```
Equilibrado       = 381b4222-f694-41f0-9685-ff5bb260df2e  (SCHEME_BALANCED)
Alto Desempenho   = 8c5e7fda-e8bf-4a96-9a85-a6e23a8c635c  (SCHEME_MIN)
Economia          = a1841308-3541-4fab-bc81-f71556f20b4a  (SCHEME_MAX)
Ultimate Perf.    = e9a42b02-d5df-448d-aa00-03f14749eb61  (oculto por padrão)
```
Revelar: `powercfg -duplicatescheme e9a42b02-d5df-448d-aa00-03f14749eb61` → ativar o GUID retornado com `powercfg /setactive`.

- **Impacto real**: Baixo a Nulo para a maioria. Teste real (xda-developers, RTX, 4 jogos): em títulos GPU-bound (Cyberpunk 2077, Black Myth: Wukong, Forza Horizon 5) diferença "dentro da margem de erro" — em Forza até *ligeiramente pior*. Em título CPU-bound (Fortnite): ganho real, 101→108 FPS médio e 1% low de 72,3→83,4 FPS.
- **Custo**: mais calor e consumo em idle; desativa suspensão de HD, USB selective suspend e PCIe power management — **péssimo para notebook**.
- **⚠️ Nota de engenharia crítica**: nesta máquina de teste, `powercfg /list` mostrou **dois planos diferentes** chamados "Desempenho Máximo" com GUIDs diferentes do oficial (injetados por software de terceiro, ex. Driver Booster). **O produto deve sempre identificar planos por GUID, nunca por nome exibido** — nomes colidem.
- **Veredicto**: confirmado_com_ressalva — só oferecer no perfil avançado/desktop; nunca ativar automaticamente em notebook.
- **Fonte**: [Powercfg command-line options](https://learn.microsoft.com/en-us/windows-hardware/design/device-experiences/powercfg-command-line-options), [xda-developers — teste real](https://www.xda-developers.com/this-hidden-windows-11-power-plan-improved-my-games-performance/).

## 2. Estacionamento de núcleos (Core Parking)

```
Subgrupo SUB_PROCESSOR = 54533251-82be-4824-96c1-47b60b740d00
CPMINCORES  = 0cc5b647-c1df-4637-891a-dec35c318583
CPMAXCORES  = ea062031-0e34-4ff1-9b6d-eb1059334028
```
Desbloquear na UI: `powercfg -attributes SUB_PROCESSOR CPMINCORES -ATTRIB_HIDE`. Desativar parking: `powercfg /setacvalueindex scheme_current sub_processor CPMINCORES 100`.

- **Impacto real**: Baixo, situacional. Caso documentado onde realmente ajuda: produção de áudio profissional/DAW (cargas com picos súbitos sensíveis a latência). CPUs híbridas modernas (Intel 12ª gen+/Thread Director) já delegam parte da decisão ao hardware, reduzindo ainda mais a relevância manual.
- **Risco**: Moderado — fácil vender como "mais desempenho sempre" quando na real só aumenta consumo/calor sem ganho perceptível em uso comum.
- **Veredicto**: confirmado_com_ressalva — restringir a "ganho restrito a cargas de áudio profissional/tempo real"; padrão do Windows já é adequado para 95% dos usuários.
- **Fonte**: [CPMinCores](https://learn.microsoft.com/en-us/windows-hardware/customize/power-settings/options-for-core-parking-cpmincores), [CPMaxCores](https://learn.microsoft.com/en-us/windows-hardware/customize/power-settings/options-for-core-parking-cpmaxcores).

## 3. PCIe Link State Power Management (ASPM)

```
Subgrupo SUB_PCIEXPRESS = 501a4d13-42af-4429-9fd1-a8218c268e20
ASPM = ee12f906-d277-404b-b6da-e5fa1a576df5  (0=Desligado, 1=moderado, 2=máximo)
```
- **Impacto real**: Médio para latência/estabilidade em sistemas sensíveis; ajuste clássico em comunidades de input-lag/e-sports. Também citado como causa ocasional de instabilidade USB e microengasgos.
- **Risco**: Moderado — desligar é seguro em desktop; em notebook reduz autonomia sensivelmente. Efeito depende de placa-mãe/chipset/driver, não é universal.
- **Veredicto**: confirmado_com_ressalva — oferecer "desligar" só para desktop fixo, manter padrão em notebook.
- **Fonte**: [PCI Express settings overview](https://learn.microsoft.com/en-us/windows-hardware/customize/power-settings/pci-express-settings).

## 4. USB Selective Suspend

```
Subgrupo = 2a737441-1930-4402-8d77-b2bebba308a3
Setting  = 48e6b7a6-50f5-4782-a5d4-53bb8f07e226  (0=Desabilitado, 1=Ativado/padrão)
```
- **Uso real**: não é otimização de velocidade, é **correção de comportamento** — resolve delay de mouse "acordando" ou fone/interface de áudio USB que desconecta sozinho.
- **Risco**: Seguro, totalmente reversível.
- **Veredicto**: confirmado — a Microsoft recomenda manter ativado por padrão; desativar é troubleshooting legítimo, não otimização universal.

## 5. Hardware Accelerated GPU Scheduling (HAGS)

- **Chave**: `HKLM\SYSTEM\CurrentControlSet\Control\GraphicsDrivers`, valor `HwSchMode`, REG_DWORD. `1`=desativado forçado, `2`=ativado. Ausente = driver decide.
- **Veredicto honesto: DEPENDE, sem consenso único.** A própria Microsoft, no anúncio oficial, não promete ganho geral: a maioria dos apps já esconde o custo de agendamento via buffering entre frames. Há relatos de problemas reais (falhas em CUDA/múltiplas GPUs, congelamentos ocasionais) nos próprios comentários do anúncio da Microsoft. Comunidade Adobe Premiere: consenso de incerteza (evitar mexer para não interferir com CUDA).
- **⚠️ Achado recente relevante**: HAGS passou a ser **pré-requisito técnico** para NVIDIA DLSS Frame Generation em GPUs RTX 40-series+ — deixa de ser "opcional" nesses casos.
- **Veredicto**: confirmado_com_ressalva — apresentar como toggle informativo com aviso de "resultado variável"; detectar se algum recurso ativo (DLSS FG) exige antes de sugerir desativar.
- **Compatibilidade**: requer Windows 10 2004+ (build 19041+) — Win10 22H2 ✓, Win11 24H2/25H2 ✓.
- **Fonte**: [DirectX Developer Blog — HAGS](https://devblogs.microsoft.com/directx/hardware-accelerated-gpu-scheduling/).

## 6. Modo de Jogo (Game Mode)

- **Chave**: `HKCU\Software\Microsoft\GameBar`, valor `AutoGameModeEnabled`. Política de bloqueio: `AllowAutoGameMode=0`.
- **Não é placebo puro**: existe um perfil PPM oficial "GameMode" que pode forçar 100% de desempenho mínimo do processador — **mas é opt-in do fabricante (OEM)**, nem todo notebook tem isso realmente configurado mesmo com o toggle ligado. Separadamente, o Windows já prioriza a janela em foco com QoS "High" desde 1709, **independente do Modo de Jogo** — boa parte do "benefício" já acontece de qualquer forma.
- **Benefício mais consistente e garantido**: supressão de notificações e Windows Update durante o jogo, não uma "turbinada" de CPU/GPU inédita.
- **Veredicto**: confirmado_com_ressalva — efeito real mas modesto, parcialmente redundante com comportamento padrão do Windows.
- **Fonte**: [Processor power management options](https://learn.microsoft.com/en-us/windows-hardware/customize/power-settings/configure-processor-power-management-options), [Quality of Service - Win32 apps](https://learn.microsoft.com/en-us/windows/win32/procthread/quality-of-service).

## 7. TRIM e "Otimizar Unidades"

- **Comando**: verificar `fsutil behavior query DisableDeleteNotify` (0=TRIM ativo); forçar `Optimize-Volume -DriveLetter C -ReTrim`.
- **Comportamento oficial confirmado por tipo de mídia** (`Optimize-Volume`): HDD/VHD fixo → `-Analyze -Defrag`; SSD com TRIM → `-Retrim` (nunca desfragmenta); SSD sem TRIM/removível FAT → nenhuma operação. **Já é adaptativo e correto por padrão.**
- **Tarefa agendada nativa**: `\Microsoft\Windows\Defrag\ScheduledDefrag`, semanal, roda `defrag.exe -c -h -o -$`.
- **Impacto**: Alto para saúde/performance de longo prazo do SSD — mas é manutenção preventiva, não ganho instantâneo perceptível.
- **Veredicto**: confirmado.
- **Fonte**: [Optimize-Volume (Storage)](https://learn.microsoft.com/en-us/powershell/module/storage/optimize-volume).

## 8. Arquivo de Paginação (Pagefile) — o mito de "desative para ganhar RAM"

- **Verificar**: `Get-CimInstance Win32_ComputerSystem | Select AutomaticManagedPagefile`.
- **⚠️ Mito perigoso confirmado pela documentação oficial**: o pagefile **não é** só "extensão da RAM para quando acaba" — sustenta o "system commit charge" mesmo com RAM sobrando, e é usado para dump de memória em caso de pane. A Microsoft afirma textualmente que desativar sem reserva suficiente resulta em **"congelamento, travamentos e outras falhas"** — não há ganho de desempenho compensatório algum.
- **Veredicto**: confirmado_com_ressalva — **NUNCA oferecer "desativar pagefile" como otimização.** Padrão recomendado e seguro é "Gerenciado automaticamente"; tamanho customizado só como opção avançada (SSD pequeno, controle de dump para diagnóstico).
- **Risco**: **Arriscado** para "sem pagefile"; Seguro para gerenciado automático.
- **Fonte**: [Introduction to the page file — Microsoft Learn](https://learn.microsoft.com/en-us/troubleshoot/windows-client/performance/introduction-to-the-page-file).

## 9. Resizable BAR / Above 4G Decoding

- **Não é configurável pelo Windows** — é 100% firmware (BIOS/UEFI), fora do alcance de qualquer app rodando no SO. Exige suporte simultâneo de BIOS + VBIOS da GPU + driver, e CSM/Legacy **desativado**.
- **Verificar (não configurar)**: painel NVIDIA → Ajuda → Informações do Sistema → campo "Resizable BAR"; ou GPU-Z como admin.
- **Impacto**: MUITO variável por jogo/GPU/driver — de nulo a ~15% em casos favoráveis publicados por AMD/NVIDIA em 2020-2021 (números não re-verificados nesta pesquisa com benchmarks recentes — ver notas de incerteza).
- **Recomendação de produto**: **detectar e informar** se está ativo (via WMI, `Win32_PNPAllocatedResource`/`Win32_DeviceMemoryAddress`, olhando o tamanho da janela de memória alocada à GPU), nunca tentar configurar — está fora do alcance do software.
- **Veredicto**: confirmado (quanto à natureza do recurso); apresentar só como card informativo, não como toggle.

## 10. Message Signaled Interrupts (MSI Mode) na GPU (`#msi-mode`)

- **Chave**: `HKLM\SYSTEM\CurrentControlSet\Control\Class\{4d36e968-e325-11ce-bfc1-08002be10318}\0000\Interrupt Management\MessageSignaledInterruptProperties`, valor `MSISupported`, REG_DWORD. Padrão antigo = `0` (Line-Based IRQ) ou ausente. Otimizado = `1` (MSI Mode habilitado).
- **Ganho**: substitui o mecanismo de interrupção legado compartilhado por interrupções diretas via barramento PCIe endereçadas por mensagens. Elimina conflitos de IRQ sharing e reduz o tempo de serviço de interrupções (ISR) e chamadas de procedimento diferido (DPC) no driver de vídeo.
- **Impacto**: Alto em sistemas com periféricos múltiplos, mouses de alta taxa e placas de som dedicadas.
- **Risco**: Baixo em GPUs modernas (NVIDIA GTX 900+ / RTX, AMD GCN+ / RDNA).
- **Veredicto**: confirmado (respaldado por `valleyofdoom/PC-Tuning` §11.38 e `valleyofdoom/AutoGpuAffinity`).

---

## Regras para a UI (derivadas deste domínio)

1. **Nunca oferecer "desativar pagefile"** — é o item mais perigoso deste domínio, confirmado oficialmente como capaz de causar travamentos.
2. Planos de energia devem ser identificados por **GUID, nunca por nome exibido** — nomes colidem com software de terceiros.
3. "Ultimate Performance" só no perfil avançado/desktop, nunca ativado automaticamente em notebook.
4. HAGS é toggle informativo com aviso de "resultado variável" — checar se DLSS Frame Generation está em uso antes de sugerir desativar.
5. Resizable BAR aparece só como detecção/informação, nunca como controle (está fora do alcance do Windows).
6. TRIM/Otimizar Unidades: nunca oferecer desfragmentação clássica forçada em SSD.

## Notas de incerteza

- Números percentuais de ganho do Resizable BAR (ex. "~15%") refletem conhecimento geral do lançamento 2020-2021, não benchmarks recentes (2025/2026) — validar antes de usar em material de marketing.
- Fórmula "RAM ÷ 8, limitado a 32 GB" para dimensionamento de pagefile é amplamente citada mas não encontrada literalmente na doc oficial atual — tratar como bem estabelecida, mas secundária.
- Windows 10 22H2 não foi testado ao vivo nesta pesquisa (só build 26200 disponível) — compatibilidade extrapolada de requisitos mínimos documentados.
- GUID do Ultimate Performance e sintaxe `-attributes`/`ATTRIB_HIDE`: sem página canônica única no learn.microsoft.com, mas convergência consistente de múltiplas fontes + confirmação de que não aparece no `powercfg /list` padrão (comportamento "oculto" esperado).
