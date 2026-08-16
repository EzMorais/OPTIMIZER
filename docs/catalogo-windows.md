# Catálogo técnico de otimizações do Windows — índice

> Pesquisa completa em 6 domínios, cada um com verificação cética (confirmado / confirmado_com_ressalva / obsoleto / placebo / perigoso), chaves/comandos exatos, e testes empíricos ao vivo numa máquina Windows 11 build 26200 real. Cobertura: Windows 11 24H2/25H2 (build 26100+) e Windows 10 22H2.

## Domínios

| Domínio | Arquivo | Destaques |
|---|---|---|
| **Registro** | [`catalogo/registro.md`](./catalogo/registro.md) | Responsividade de UI, MMCSS, prioridade de CPU, memória, telemetria, hibernação, GameDVR, NTFS, mitigações de CPU (Spectre/Meltdown) |
| **Rede / Internet** | [`catalogo/rede.md`](./catalogo/rede.md) | TCP autotuning, Nagle/TcpAckFrequency, MTU, QoS, DNS, adaptador/Wi-Fi, Delivery Optimization |
| **Serviços e inicialização** | [`catalogo/servicos-inicializacao.md`](./catalogo/servicos-inicializacao.md) | O que é seguro desativar vs. nunca tocar, SysMain por tipo de disco, startup apps, bloatware, tarefas de telemetria, exclusões do Defender |
| **Hardware / Energia / GPU** | [`catalogo/hardware-energia-gpu.md`](./catalogo/hardware-energia-gpu.md) | Planos de energia, core parking, PCIe ASPM, HAGS, pagefile, Resizable BAR |
| **Limpeza e manutenção** | [`catalogo/limpeza-manutencao.md`](./catalogo/limpeza-manutencao.md) | Disk Cleanup, WinSxS, hibernação, por que não desfragmentar SSD, sfc/DISM/chkdsk, mito do registro |
| **Medição e diagnóstico** | [`catalogo/medicao-diagnostico.md`](./catalogo/medicao-diagnostico.md) | WinSAT, Get-Counter (⚠️ armadilha de localização pt-BR), Event Log de boot, DPC/ISR, hard faults, SMART, powercfg |

## Achados críticos que atravessam todos os domínios

1. **Armadilha de localização (Get-Counter)**: paths de contador em inglês falham silenciosamente em Windows pt-BR. Usar sempre classes WMI/CIM `Win32_PerfFormattedData_*` (nomes de propriedade estáveis em inglês independente do idioma) — nunca `Get-Counter` com texto localizado. Ver [`medicao-diagnostico.md`](./catalogo/medicao-diagnostico.md#8-confiabilidade-em-processo-standalone-go--a-armadilha-central-desta-pesquisa).
2. **`wmic.exe` está obsoleto e ausente por padrão desde o Windows 11 24H2** — nunca depender dele. Usar `Get-CimInstance`.
3. **`Get-Service WinDefend` não é confiável para saber se o Defender está ativo** em builds recentes — usar `Get-MpComputerStatus`.
4. **Nomes de serviços/tarefas mudam entre builds** (ex.: "Microsoft Compatibility Appraiser" virou "...Exp"; `ProgramDataUpdater` sumiu). O motor de recomendação deve **consultar o estado real da máquina em tempo de execução**, nunca aplicar uma lista fixa.
5. **`SystemResponsiveness=0`** (o valor mais recomendado por guias de gaming) é automaticamente grampeado para `20` pelo próprio Windows — o valor correto é `10`.
6. **Nunca oferecer "desativar pagefile"** — confirmado oficialmente como capaz de causar travamentos, sem ganho de performance compensatório.
7. **Mitigações de CPU (Spectre/Meltdown) nunca entram no catálogo padrão nem avançado sem confirmação explícita adicional** — risco de segurança bem estabelecido, ganho de performance incerto e mal documentado para hardware atual.

## Síntese acionável

[`catalogo/top-15-e-armadilhas.md`](./catalogo/top-15-e-armadilhas.md) — as 15 otimizações de melhor ganho-real/risco (o coração do produto) e o texto pronto de UI para as "armadilhas" que o produto se recusa a fazer, cruzados por duas pesquisas independentes.

## Uso deste catálogo no produto

- Cada item marcado `confirmado` ou `confirmado_com_ressalva` é candidato ao catálogo de toggles do app (ver [`decisoes.md`](./decisoes.md)).
- Itens `obsoleto` e `placebo` alimentam a lista de "armadilhas que o produto se recusa a fazer" — parte do diferencial de confiança contra CCleaner/IObit.
- Itens `perigoso` (mitigações de CPU, exclusões de Defender em pastas genéricas) nunca entram no perfil padrão; se oferecidos, exigem confirmação explícita adicional ("avançado" + aviso).
