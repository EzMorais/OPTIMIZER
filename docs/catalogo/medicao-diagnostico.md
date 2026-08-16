# Catálogo técnico — Medição e Diagnóstico

> Como o app prova objetivamente o antes/depois e descobre o gargalo real de cada PC. Windows 11 24H2/25H2 (build 26100+), testado ao vivo numa máquina real build 26200, usuário **sem** privilégio de administrador, idioma **pt-BR** (achado crítico de engenharia — ver seção 8).

## 1. WinSAT / Windows Experience Index

- **Ler resultado em cache (sem admin)**: `Get-CimInstance Win32_WinSAT`. **Rodar avaliação nova (`winsat.exe`) exige admin** — confirmado ao vivo (`winsat /?` falha sem elevação).
- **Arquivos brutos**: `C:\Windows\Performance\WinSAT\DataStore\*.WinSAT.xml`.
- **Evidência real coletada** (não-admin): `CPUScore=9.2, D3DScore=9.9, DiskScore=7.35, GraphicsScore=9.7, MemoryScore=9.2, WinSPRLevel=7.35` — confirma que a nota final = mínimo dos subscores.
- **Veredicto**: confirmado_com_ressalva — motor funcional neste build, mas **sem página oficial atual documentando WinSAT** (3 URLs tentadas retornaram 404); escala 1-9.9 satura em hardware moderno (SSD NVMe/GPU atual tendem ao teto), diferenciando mal "bom" de "topo de linha".

## 2. Get-Counter (Performance Counters / PDH)

- **⚠️ ACHADO CRÍTICO DE ENGENHARIA, confirmado ao vivo**: paths de contador em inglês (`\Processor(_Total)\% Processor Time`) **falham silenciosamente** em Windows pt-BR — erro genérico "objeto especificado não foi encontrado", sem nenhuma pista de que o problema é idioma.
- Alguns *nomes de conjunto* ficam em inglês mesmo em pt-BR (`PhysicalDisk`, `LogicalDisk`), mas os *contadores individuais* dentro deles são sempre traduzidos (ex.: `\PhysicalDisk(_Total)\% tempo de disco`, não `% Disk Time`). **Não existe atalho — todos exigem tratamento.**
- O qualificador `(_Total)` **não é localizado** — funciona igual em qualquer idioma.
- **✅ Mitigação robusta confirmada empiricamente**: usar classes **WMI/CIM `Win32_PerfFormattedData_*`** em vez de `Get-Counter` por nome de texto — os nomes de propriedade permanecem em **inglês e estáveis independentemente do idioma**:
  ```
  Win32_PerfFormattedData_PerfOS_Memory      → AvailableMBytes, PagesPersec, PageFaultsPersec
  Win32_PerfFormattedData_PerfOS_Processor   → PercentProcessorTime, PercentInterruptTime, PercentDPCTime
  Win32_PerfFormattedData_PerfDisk_PhysicalDisk → PercentDiskTime, AvgDisksecPerTransfer
  ```
  Todos funcionam **sem admin**, mesmos valores que os contadores PDH equivalentes. **Para o binário Go: consumir CIM/WMI diretamente via COM** (`go-ole`, `github.com/StackExchange/wmi`), sem depender de texto localizado e sem precisar invocar `powershell.exe` como subprocesso.
- **Requer admin**: não, para leitura de contadores.
- **Veredicto**: confirmado_com_ressalva — API real e útil, mas a armadilha de localização é severa o suficiente para quebrar o produto para a maioria dos usuários brasileiros se implementada ingenuamente com paths em inglês.
- **Fonte**: [about Get-Counter](https://learn.microsoft.com/en-us/powershell/module/microsoft.powershell.diagnostics/get-counter).

## 3. Tempo de boot via Event Log (Diagnostics-Performance)

- **Comando**: `Get-WinEvent -LogName "Microsoft-Windows-Diagnostics-Performance/Operational" -FilterXPath "*[System[(EventID=100)]]"`.
- **Requer admin**: **sim**, para ler o log em si (ACL restrita, diferente da maioria dos logs Operational). Consultar só o *esquema* de eventos (`-ListProvider`) **não exige admin**.
- **Tabela de Event IDs confirmada ao vivo no manifesto ETW real (build 26100)**:

| ID | Categoria | Significado |
|---|---|---|
| **100** | Boot | Boot concluído — **Duração da Inicialização (ms)** + flag `IsDegradation`. Evento principal para medir boot. |
| 101-110 | Boot | App/driver/serviço/core/prefetch/GPO/dispositivo específico atrasou o boot |
| **200** | Shutdown | Desligamento concluído — Duração (ms) + `IsDegradation` |
| 201-203 | Shutdown | App/dispositivo/serviço atrasou o desligamento |
| **300** | Standby | Saída da suspensão — cobre duração de suspensão E retomada no mesmo evento |
| 301-310, 350-352 | Standby | App/driver/cache atrasou entrada/saída de suspensão; **350 = BIOS demorou >250ms para retomar** (viola requisito de logo Windows) |

- **Veredicto**: confirmado — é a única fonte nativa que já vem com "quem atrasou o boot" pré-calculado, sem página learn.microsoft.com dedicada e atual encontrada (extraído diretamente do manifesto ETW local).

## 4. DPC/ISR Latency

- **Proxy grosseiro (sem admin)**: `Get-Counter` ou `Win32_PerfFormattedData_PerfOS_Processor` (`PercentDPCTime`, `PercentInterruptTime`).
- **Ferramenta oficial completa**: WPR (gravar, `wpr.exe`) + WPA (analisar, `wpa.exe`), parte do Windows Performance Toolkit.
- **Achado de engenharia**: `wpr.exe` **já vem nativo** em `C:\Windows\System32` (confirmado, build 10.0.26100) — gravar um trace não exige instalar nada. `wpa.exe` (o analisador gráfico) **não vem nativo** — exige instalar o Windows ADK. `wpr -start` exige admin (sessão ETW kernel).
- **Recomendação**: não embutir WPA no produto — é ferramenta pesada de análise manual, não API programável. LatencyMon (terceiro) é o que a comunidade usa na prática por ser mais acessível ao leigo, mas não redistribuir.
- **Veredicto**: confirmado_com_ressalva — Microsoft não tem ferramenta gráfica simples e nativa para isso.

## 5. Hard Faults/sec (indicador de RAM insuficiente)

- **⚠️ Achado importante**: "Hard Faults/sec" **não é um contador PDH/perfmon clássico** — busca exaustiva em todos os contadores registrados não encontrou correspondência exata em nenhum idioma. É calculado pelo **Resource Monitor** (`resmon.exe`, nativo, sem admin) a partir de eventos ETW do gerenciador de memória.
- **Proxy PDH mais próximo**: `Pages Input/sec` (= `\Memória\Entrada de páginas/s`) — técnica historicamente recomendada, funciona sem admin.
- **Por processo, número exato**: só via `resmon.exe` → aba Memória → coluna "Falhas Rígidas/s", ou consumindo eventos ETW do provedor de memória diretamente (complexo).
- **Veredicto**: confirmado_com_ressalva — sinal direto de "PC precisa de mais RAM", mas **não dá para obter puro via PDH/WMI clássico** — decisão de arquitetura necessária (proxy vs. orientar para resmon vs. ETW).

## 6. SMART / Saúde do disco

- **Status básico (sem admin)**: `Get-PhysicalDisk | Select FriendlyName, MediaType, HealthStatus` — confirmado ao vivo, retornou "Healthy".
- **Contadores detalhados (temperatura, desgaste, horas ligado) — exige admin**: `Get-PhysicalDisk | Get-StorageReliabilityCounter` — confirmado ao vivo (falha sem elevação: "Acesso a um recurso CIM não estava disponível").
- **⚠️ `wmic diskdrive get status` está OBSOLETO, confirmado oficialmente**: *"Starting with Windows 11, version 24H2, WMIC is not preinstalled"* — é Feature on Demand ausente por padrão em instalação limpa do 24H2/25H2. **Nunca depender de `wmic.exe`** — usar sempre `Get-CimInstance Win32_DiskDrive` / classes CIM equivalentes.
- **Veredicto**: confirmado_com_ressalva (status básico) / **obsoleto** (wmic especificamente).
- **Fonte**: [Get-StorageReliabilityCounter](https://learn.microsoft.com/en-us/powershell/module/storage/get-storagereliabilitycounter), [Deprecated features in the Windows client](https://learn.microsoft.com/en-us/windows/whats-new/deprecated-features).

## 7. powercfg (bateria, energia, transições)

Requisito de admin **varia por subcomando** (testado ao vivo):

| Subcomando | Requer admin? | O que faz |
|---|---|---|
| `/batteryreport` | **Não** | Histórico e degradação de bateria; trata graciosamente desktop sem bateria |
| `/srumutil` | **Sim** (confirmado: erro 5/acesso negado sem admin) | Consumo de energia por app (banco SRUM) |
| `/systempowerreport` (`/spr`) | **Sim** (confirmado: mensagem explícita de elevação) | Transições de energia dos últimos 3 dias; substitui oficialmente o `/systemsleepdiagnostics` (preterido) |
| `/energy` | **Sim** (confirmado) | Observa 60s e lista problemas de eficiência energética — relatório mais rico do domínio |

- **Veredicto**: confirmado para todos.

## 8. Confiabilidade em processo standalone Go — a armadilha central desta pesquisa

1. **Hardcodar paths de contador em inglês quebra silenciosamente em qualquer Windows não-inglês** — confirmado ao vivo. Como o público é majoritariamente brasileiro com Windows em pt-BR, isso quebraria o produto para a maioria se implementado ingenuamente.
2. **Não existe atalho simples** — a localização é inconsistente (nome do *set* às vezes em inglês, contadores sempre traduzidos).
3. **Mitigação recomendada e testada**: classes `Win32_PerfFormattedData_*` via WMI/CIM — nomes de propriedade estáveis em inglês independente do idioma. Alternativa de nível mais baixo: API PDH nativa com `PdhLookupPerfIndexByName`/`PdhLookupPerfNameByIndex` (mapeamento via `HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Perflib`), via `golang.org/x/sys/windows` — mais robusto ainda, mas mais complexo.
4. **Erros de permissão têm formatos diferentes** — mensagem explícita ("requer privilégios de administrador"), erro genérico disfarçado ("dispositivo não está funcionando" = ERROR_ACCESS_DENIED disfarçado), erro de infraestrutura CIM. Um parser de "preciso pedir elevação" precisa tratar os três formatos, não só procurar a palavra "administrador".
5. **Custo de processo**: cada chamada via `powershell.exe -Command` tem overhead de inicialização não-trivial — para medições repetidas, favorece ainda mais WMI/CIM direto via COM em vez de spawnar processo PowerShell a cada métrica.
6. **`wmic` não deve ser dependência de jeito nenhum** — Feature on Demand ausente por padrão desde 24H2; funciona na máquina de dev (provavelmente atualizada, não instalada do zero) mas falha numa instalação limpa real.

---

## Regras para o motor de medição do produto

1. Usar `Win32_PerfFormattedData_*` (WMI/CIM) como via primária de coleta de métricas — nunca `Get-Counter` com paths em texto/inglês.
2. Nunca usar `wmic.exe` como dependência.
3. Hard Faults/sec: usar `Pages Input/sec` como proxy PDH, ou orientar para `resmon.exe` para número exato por processo.
4. Operações que exigem admin (WinSAT formal, srumutil, systempowerreport, energy, leitura do log de boot) devem ser claramente sinalizadas na UI antes de pedir elevação — o app deve funcionar em modo degradado sem admin.
5. Tratar erros de permissão com múltiplos padrões de detecção (mensagem explícita, erro genérico disfarçado, erro CIM).

## Notas de incerteza

- Significado exato de `WinSATAssessmentState` (`1`=provavelmente "válido") — inferido, não confirmado por documentação nesta sessão.
- `powercfg /systempowerreport` em Windows 10 22H2: não confirmado se existe nessa versão (o predecessor `/systemsleepdiagnostics` certamente existe) — usar como fallback se ausente.
- `Get-PhysicalDisk`/`Get-StorageReliabilityCounter` em Windows Home: não testado diretamente (máquina de teste era Pro) — alta confiança de que funciona (infraestrutura básica de Storage Management), não confirmação direta.
- Existência de "Windows Performance Analyzer (Preview)" na Microsoft Store como alternativa leve ao ADK completo — não confirmado nesta sessão.
