# Arquitetura do app desktop em Go

> Todas as bibliotecas citadas foram verificadas ao vivo (GitHub/pkg.go.dev/docs oficiais) em agosto/2026. A única lib cogitada e descartada por estar morta foi `github.com/leoluk/perflib_exporter` (arquivada em 2023).

## Resumo das decisões

| Decisão | Escolha | Por quê |
|---|---|---|
| GUI | **Wails v2 (v2.14.0)** | Estável hoje; v3 ainda beta (v3.0.0-beta.8) |
| Registro/serviços | `golang.org/x/sys/windows` direto | API oficial Go, sem processo externo |
| WMI | `github.com/yusufpapurcu/wmi` | Único fork mantido do extinto `StackExchange/wmi` |
| Contadores de desempenho | WMI `Win32_PerfFormattedData_*` ou wrapper próprio de PDH | lib "óbvia" (perflib_exporter) está arquivada |
| Elevação | **(b) elevação sob demanda, em lote** | Sem serviço sempre-elevado = sem alvo permanente de escalonamento |
| Catálogo | **Híbrido**: código sempre local e compilado; metadados via manifesto assinado remoto | Nunca baixar/executar código de terceiros com privilégio elevado |
| Instalador | **Inno Setup** + assinatura + submissão ao winget | Maduro, leve, assinável, gratuito, menos associado a malware que NSIS |
| Assinatura de código | Avaliar Azure Artifact Signing (ex-Trusted Signing, ainda em preview); fallback certificado OV/EV tradicional | Elegibilidade regional do preview precisa ser confirmada |

## 1. Framework de interface gráfica

### Comparação verificada (agosto/2026)

| Critério | **Wails v2** | Wails v3 (beta) | Fyne v2 | Gio | walk (fork Tailscale) |
|---|---|---|---|---|---|
| Qualidade visual p/ leigo | Alta (HTML/CSS/JS) | Igual | Boa, "cara de app Go" sem customização | Depende de você desenhar tudo | Win32 clássico, funcional mas datado |
| Tamanho do .exe | Pequeno (~5-15 MB) | Igual | Médio (~15-25 MB) | Pequeno (~8-15 MB) | Muito pequeno (~5-8 MB) |
| Depende de WebView2 no Win10 | **Sim** | Sim | Não | Não | Não |
| Velocidade de desenvolvimento | Muito alta | Alta, tooling mudando | Média | Baixa | Baixa/média |
| CGO no Windows | **Não** | Não | **Sim** (OpenGL, exige gcc/mingw) | **Não** | Não |
| Maturidade hoje | Alta, produção há anos | **Beta há vários ciclos** | Alta | Média-alta, nicho | Alta (Tailscale usa em produção), comunidade pequena |

**Recomendação: Wails v2.** Interface web permite qualidade visual de SaaS moderno rapidamente (crítico para percepção de leigo), .exe continua pequeno (não embute engine de navegador), v3 fica só "a acompanhar" — não vale apostar o lançamento numa API ainda em beta.

**Ponto de atenção real**: Windows 10 saiu de suporte gratuito em out/2025 — parte do parque pode estar sem WebView2 Evergreen atualizado. **Embutir o runtime "Fixed Version" do WebView2** no instalador (Wails suporta essa estratégia) em vez de depender só do bootstrapper online — custa ~150 MB a mais, mas garante que o app abre mesmo offline/desatualizado.

Ícone de bandeja: nenhum framework GUI traz de fábrica — combinar com `energye/systray` (fork ativo, sem GTK, suporte Windows confirmado).

Backup se precisar de build 100% nativo sem WebView2: **Fyne** — mas exige toolchain C no build (a máquina de dev não tem gcc/mingw confirmado, custo extra de setup).

## 2. Acesso nativo ao Windows

| Necessidade | Biblioteca verificada | Status |
|---|---|---|
| Registro | `golang.org/x/sys/windows/registry` | Oficial, `v0.47.0` (30/jun/2026) |
| Serviços | `golang.org/x/sys/windows/svc/mgr` | Mesmo módulo, ativo |
| WMI | `github.com/yusufpapurcu/wmi` + `github.com/go-ole/go-ole` | Fork mantido (original `StackExchange/wmi` morto desde 2021) |
| Contadores de desempenho | WMI `Win32_PerfFormattedData_PerfOS_*` (simples) ou wrapper próprio de PDH via `pdh.dll`/syscall (~150 linhas, tempo real) | `perflib_exporter` **arquivado desde jul/2023** — não usar |
| Medição de rede (MTU/ping) | `IcmpSendEcho` (`iphlpapi.dll`) via syscall próprio | **Implementado** em `internal/netdiag` — não exige administrador, ao contrário de socket ICMP cru |
| MTU do adaptador | `GetAdaptersAddresses` (leitura) + `SetIpInterfaceEntry` (escrita) | **Implementado**; escrita exige administrador e `SitePrefixLength = 0` para IPv4 |

### API direta vs. shell out (`powershell.exe`/`netsh.exe`)

**Regra**: API direta é a via padrão (cobre a maioria: efeitos visuais, itens de inicialização, Explorer, energia via `PowrProf.dll`). `netsh.exe`/`powercfg.exe` são exceção documentada caso a caso (ex.: `netsh int ip reset` reescreve estado de driver sem API pública; `powercfg /h off` redimensiona/apaga `hiberfil.sys` atomicamente). `powershell.exe` só como último recurso.

**Por que evitar PowerShell sempre que possível**:
1. **Custo de start**: ~150-400ms por invocação (carrega CLR/host) vs. microssegundos de chamada in-process — num catálogo com dezenas de itens, soma segundos perceptíveis.
2. **Sinal de confiança/AV**: processo comum invocando `powershell.exe` oculto é heurístico clássico de "LOLBin abuse" usado por malware fileless — risco real de o produto ser confundido com esse padrão.
3. **Testabilidade**: API Go permite mockar a fronteira; shell-out é mais difícil de testar sem tocar o sistema real.

## 3. Permissão de administrador

| Abordagem | Risco |
|---|---|
| (a) App inteiro sempre admin | Viola menor privilégio; toda a UI (inclusive renderização WebView2) roda elevada; prompt UAC a cada abertura, mesmo só para ver estado |
| **(b) Elevação sob demanda, em lote** | Mitiga atrito com lote único: usuário revisa mudanças sem privilégio, clica "aplicar" uma vez, processo elevado de curta duração aplica tudo e encerra — padrão usado por CCleaner, O&O ShutUp10 |
| (c) Serviço sempre elevado desde o boot | **Alvo clássico de escalonamento de privilégio local (LPE)** — endpoint sempre-elevado aceitando pedidos de qualquer processo comum sem verificar quem pede é ponte de baixo-privilégio para SYSTEM. Superfície de ataque permanente |

**Recomendação: (b)**, com lote único por sessão de aplicação. Só reconsiderar (c) se precisar de monitoramento/agendamento com usuário deslogado — nesse caso, serviço separado minimalista com todo o endurecimento abaixo.

### IPC seguro UI ↔ processo elevado (se necessário)

Bibliotecas: `github.com/microsoft/go-winio` (mantida pela Microsoft, usada por Docker/containerd) para named pipe.

1. **Pipe efêmero**: nome aleatório (GUID) por sessão, criado só quando necessário.
2. **SDDL restritivo**: negar `Everyone`/`ANONYMOUS LOGON`, permitir só o SID do usuário logado (via go-winio).
3. **Verificar identidade do processo do outro lado**: `GetNamedPipeClientProcessId` → confirmar caminho da imagem em disco **e** assinatura Authenticode via `WinVerifyTrust` (mesma API que o Windows usa) — impede outro executável fingir ser a UI.
4. **Nonce de sessão**: token aleatório passado só pelo processo pai (variável de ambiente/handle herdado) — cobre caso de duas instâncias legítimas simultâneas.
5. **Protocolo estreito**: só "aplique o Tweak de ID X", nunca comando arbitrário — revalidar no lado elevado que o ID existe no catálogo compilado.
6. **Timeout curto**: processo elevado aplica o lote, retorna, encerra — nunca fica escutando indefinidamente.

## 4. Motor de otimização como dado

```go
// internal/tweak/tweak.go
type State int
const (
    StateUnknown State = iota
    StateApplied
    StateNotApplied
    StatePartial
)

type Risk int
const ( RiskLow Risk = iota; RiskMedium; RiskHigh )

type Snapshot map[string]any

type CheckResult struct {
    State       State
    Detail      string
    RawSnapshot Snapshot
}
type ApplyResult struct {
    Snapshot      Snapshot
    RestartNeeded bool
    Detail        string
}
type VerifyResult struct { Success bool; Detail string }

// Tweak é o contrato que toda otimização do catálogo implementa.
type Tweak interface {
    ID() string
    Name() string
    Description() string
    Category() Category
    Risk() Risk
    RequiresRestart() bool

    Check(ctx context.Context) (CheckResult, error)
    Apply(ctx context.Context, dryRun bool) (ApplyResult, error)
    Revert(ctx context.Context, snapshot Snapshot, dryRun bool) error
    Verify(ctx context.Context) (VerifyResult, error)
}
```

Implementado em `internal/tweak/tweak.go`. O padrão "otimização é um conjunto de valores de registro" — que cobre a maior parte do catálogo — foi escrito uma única vez em `internal/tweaks/regtweak`, e cada item do catálogo passou a ser só dados (chave, tipo, valor otimizado, valor padrão do Windows, texto e ressalva). Ver o anexo ao final deste documento.

### Catálogo: embutido vs. remoto — o meio-termo seguro

Baixar e **executar código** remoto com privilégio elevado é risco de RCE — inaceitável. Solução: separar **código** (sempre local, compilado, revisado, assinado) de **metadados** (manifesto remoto assinado, sem exigir nova versão para reordenar/reescrever textos/desativar item problemático):

```go
type TweakMeta struct {
    ID                 string
    Enabled            bool
    DisplayNamePtBR    string
    DescriptionPtBR    string
    RecommendedDefault bool
    SortOrder          int
    MinAppVersion      string
}

func LoadSignedManifest(payload, signature []byte, pubKey ed25519.PublicKey) ([]TweakMeta, error) {
    if !ed25519.Verify(pubKey, payload, signature) {
        return nil, errors.New("assinatura do manifesto remoto inválida")
    }
    var metas []TweakMeta
    json.Unmarshal(payload, &metas)
    return metas, nil
}
```

`Registry.Known(id)` ignora silenciosamente qualquer ID do manifesto que não corresponda a um `Tweak` já compilado — **nunca executa o que veio de fora**. Consequência: adicionar um tipo *novo* de otimização exige nova versão do .exe (preço pequeno pela segurança); reordenar/desativar remotamente não exige.

### Histórico e "desfazer tudo"

`HistoryEntry` (ID, TweakID, Action, AppliedAt, Snapshot, RestorePointID, Success) persistido como **JSONL append-only** (barato, auditável) — evoluível para `modernc.org/sqlite` (driver 100% Go, sem CGO, ativo) se precisar filtrar/consultar por período.

### Dry-run

`Apply(ctx, dryRun bool)`: com `dryRun=true`, roda toda validação e retorna o que faria sem escrever nada — serve tanto para o botão "simular" na UI quanto para testes automatizados sem tocar o Windows real.

## 5. Empacotamento

- **Build**: `go build -ldflags "-H=windowsgui -s -w"` (sem console, sem símbolos debug).
- **Ícone/versão**: `github.com/tc-hib/go-winres` (ativo, v0.3.3) — gera `.syso` com ícone, `FileVersion`/`CompanyName`/`FileDescription` (reduz falso-positivo de AV) e manifesto (`requestedExecutionLevel="asInvoker"`, DPI-awareness).
- **Assinatura**: sem Authenticode, SmartScreen mostra "Editor desconhecido" — pior primeira impressão possível para leigo.
  - **Azure Artifact Signing** (ex-Trusted Signing): HSM gerenciado (FIPS 140-3 nível 3), cobrança por uso, reputação SmartScreen quase imediata — **ainda em preview público em ago/2026**, elegibilidade por país/empresa precisa ser confirmada no cadastro.
  - **Certificado OV/EV tradicional**: universal, independe de elegibilidade regional. OV mais barato mas reputação demora semanas/meses; EV dá confiança imediata, custa mais, exige token HSM.
  - Sempre assinar **com carimbo de tempo** (`signtool sign /tr <URL RFC3161>`) e **todos** os artefatos (instalador, exe principal, auxiliar elevado, updater).
- **Instalador**: **Inno Setup** (ativo em 2026, assinatura de instalador/uninstaller, Arm64, ~1,78MB overhead) preferido sobre NSIS (também ativo, mas historicamente mais associado a droppers de malware em heurísticas de AV). MSIX considerado mais adiante, não no v1 (empacotar componente elevado é mais burocrático). **winget** como canal adicional depois de estável — validação automática do winget contra Defender é uma segunda opinião de antivírus grátis a cada release.
- **Auto-update**: `github.com/minio/selfupdate` (multiplataforma, checksum + assinatura estilo Ed25519). Fluxo: manifesto de versão assinado → verificar assinatura antes de baixar → revalidar Authenticode via `WinVerifyTrust` antes de substituir o binário em uso → aplicar e reiniciar.

### Por que este tipo de app é sinalizado por engano, e como reduzir

Mexer em registro + elevar privilégio + instalar serviço + falar com servidor de update bate em vários heurísticos de PUA simultaneamente (padrão histórico de toda a categoria). Mitigação:
- Assinar tudo (já coberto).
- **Nunca usar packers/UPX** — comprimir/ofuscar o binário é sinal clássico de malware, aumenta detecções.
- Metadados de versão completos via go-winres — binário "anônimo" é sinal negativo.
- Submeter cada release ao VirusTotal antes de publicar; tratar detecção nova como bloqueador, não "falso positivo, ignora".
- Submeter proativamente ao canal de submissão da Microsoft Defender Security Intelligence quando aparecer detecção.
- Evitar o padrão de serviço sempre-elevado silencioso (seção 3) — histórico local auditável (seção 4) reduz heurística de AV e aumenta confiança do usuário.

## 6. Estrutura de repositório

```
optimizer/
  cmd/
    optimizerctl/         # [existe] CLI interna de diagnóstico/dev, não distribuída
    optimizerui/           # [placeholder] app desktop (Wails v2) — falta `wails init`
    optimizerhelper/        # [a fazer] processo auxiliar elevado, sob demanda
  internal/
    tweak/                     # [existe] contrato Tweak, Meta (catálogo como dado), Registry
    tweaks/                     # [existe] catálogo embutido
      regtweak/                  #   o padrão "tweak de registro", escrito uma vez
    winreg/                       # [existe] fronteira mockável: Live (Windows) + Fake (testes)
    engine/                        # [existe] lote: diagnosticar → aplicar → verificar → desfazer
    history/                        # [existe] Store JSONL append-only
    netdiag/                         # [existe] medição de MTU (ICMP DF) e ajuste do adaptador
    restore/                          # [existe] ponto de restauração (falta validar em VM)
    elevate/ console/                  # [existe] detecção de administrador; UTF-8 no terminal
    ipc/                                # [a fazer] UI <-> auxiliar elevado
    update/ manifest/ config/ logging/   # [a fazer]
  build/windows/{winres,installer}/       # [a fazer] go-winres + script .iss (Inno Setup)
  scripts/{sign.ps1,release.ps1}           # [a fazer]
  test/integration/                         # [a fazer] roda só em VM/CI descartável
  go.mod go.sum
```

## 7. Testar sem quebrar a máquina de desenvolvimento

1. **Abstrair a fronteira** (`RegistryBackend`, `ServiceBackend`, `WMIClient`) — implementação real via x/sys/wmi, fake em memória para testes. Cobre a lógica de `Check`/`Apply`/`Revert`/`dryRun` sem tocar o Windows, roda até em não-Windows.
2. **HKEY_CURRENT_USER isolado**: testes que tocam registro de verdade usam subárvore `Software\OptimizerTests\<guid>`, nunca `HKEY_LOCAL_MACHINE`.
3. **`RegOverridePredefKey`** (advapi32.dll): redireciona, por processo, o que `HKEY_CURRENT_USER`/`HKEY_LOCAL_MACHINE` resolve para uma chave de teste — código de produção "funciona" contra área descartável sem `if` especial. Vale para o processo inteiro — um binário de teste por cenário, não em paralelo no mesmo processo.
4. **Windows Sandbox**: nativo no Windows 11 Pro (recurso opcional) — ambiente completo descartável, reseta ao fechar. Ideal para testar fluxo completo manual (instalador → aplicar → reiniciar → conferir).
5. **Hyper-V com checkpoints**: manter VMs "padrão-ouro" (Win10 22H2, Win11 24H2/25H2), checkpoint antes de cada bateria de integração, reverter depois.
6. **CI descartável**: `windows-latest` do GitHub Actions já são VMs efêmeras por job — rodar testes de integração com o truque do item 3, reservando a fidelidade de build exato para o pool Hyper-V (periódico, não a cada commit).
7. **Ponto de restauração (`SRSetRestorePointW`)** só testado em VM, nunca contra o System Restore real da máquina de dev.
8. **Fixtures gravadas para WMI**: capturar uma vez (em VM) resultados reais em JSON, reproduzir em testes rápidos via `WMIClient` fake.

---

## Anexo — como o motor ficou depois de implementado

O contrato desta seção 4 está em `internal/tweak/`. O que mudou da proposta original para o código real, e por quê:

**1. `Tweak` não fala com o registro direto.** Um `Tweak` recebe um `winreg.Backend` — `Live` no Windows, `Fake` em memória nos testes (seção 7, item 1). Consequência prática: `go test ./...` exercita `Check`/`Apply`/`Revert`/`dryRun` inteiros sem tocar no Windows da máquina de desenvolvimento.

**2. Um tweak pode controlar vários valores.** "Desligar aceleração do mouse" são 3 valores independentes. Por isso `regtweak.Tweak` recebe uma lista de valores, e `StatePartial` deixou de ser teórico: é o estado real de "2 de 3 ajustes aplicados".

**3. O snapshot guarda a ausência.** Se o valor não existia antes, desfazer é **apagar** — gravar o valor padrão no lugar deixaria a máquina num estado que ela nunca teve. O snapshot é `{exists, dword|str}` por valor e sobrevive à ida e volta pelo histórico em JSON.

**4. `AlreadyOptimized`: nunca piorar.** Se o usuário já tem `MenuShowDelay = 0` e nosso alvo é `10`, o item conta como aplicado. Um "otimizador" que aumenta o atraso de menu de 0 para 10 e chama isso de otimização é exatamente o que este produto não pode ser.

**5. O estado mostrado sempre carrega o número medido.** `Check` devolve `"... Medimos agora: MenuShowDelay = 100 (alvo 10)"`. Isso nasceu de um erro real detectado em teste: o texto dizia "o Windows está no padrão (0,4 s)" numa máquina cujo valor era 100. Afirmar o padrão sem ter lido é a versão pequena do mesmo pecado do "encontramos 3.847 problemas".

**6. O manifesto remoto é blindado por teste.** `Registry.ApplyManifest` ignora ID desconhecido (nunca cria otimização nova) e **não deixa o manifesto rebaixar `RequiresAdmin`** — senão um manifesto hostil faria o app achar que um tweak de máquina dispensa UAC. Ambos os casos têm teste.

**7. Ordem do lote no motor** (`internal/engine`): checar → pular o que já está aplicado → ponto de restauração **uma vez por lote** → aplicar → verificar → gravar no histórico. Falha ao criar o ponto de restauração aborta o lote inteiro sem alterar nada.
