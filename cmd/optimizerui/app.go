package main

import (
	"context"
	"fmt"
	"os/exec"
	"sort"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"optimizer/internal/cpudiag"
	"optimizer/internal/diskdiag"
	"optimizer/internal/elevate"
	"optimizer/internal/engine"
	"optimizer/internal/history"
	"optimizer/internal/netdiag"
	"optimizer/internal/profiles"
	"optimizer/internal/restore"
	"optimizer/internal/systemdiag"
	"optimizer/internal/telemetry"
	"optimizer/internal/tweak"
	"optimizer/internal/tweaks"
	"optimizer/internal/winreg"
)

// App é a fronteira entre a interface (HTML/JS) e o motor. Toda decisão continua
// no Go compilado — o frontend só pede e mostra.
type App struct {
	ctx context.Context
	eng *engine.Engine

	// última medição de rede, para o botão "aplicar ajuste" saber o que aplicar
	ultimoMTU *medicaoMTU

	// Sessão de telemetria e benchmark
	benchMu           sync.Mutex
	benchCancel       context.CancelFunc
	benchCollector    *telemetry.Collector
	ultimoReportAntes map[string]telemetry.BenchmarkReport
	dnsRunner         netdiag.DNSRunner
}

type medicaoMTU struct {
	iface netdiag.Interface
	rep   netdiag.Report
	diag  netdiag.Diagnosis
}

func NewApp() *App { return &App{} }

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	caminho, err := history.DefaultPath()
	if err != nil {
		caminho = "history.jsonl"
	}
	store, err := history.Open(caminho)
	if err != nil {
		// Sem histórico não há desfazer confiável: melhor abrir avisando do que
		// deixar o usuário aplicar coisas sem rede de segurança.
		store, _ = history.Open("history.jsonl")
	}

	a.eng = &engine.Engine{
		Registry: tweaks.Build(winreg.NewLive()),
		History:  store,
		Elevated: elevate.IsElevated(),
	}

	a.benchCollector = telemetry.NewCollector(telemetry.NewWindowsLiveProvider())
	a.ultimoReportAntes = make(map[string]telemetry.BenchmarkReport)
	a.dnsRunner = netdiag.LiveDNSRunner{}
}

// ---------------------------------------------------------------- tipos da UI

type ItemUI struct {
	ID           string `json:"id"`
	Nome         string `json:"nome"`
	Descricao    string `json:"descricao"`
	Ressalva     string `json:"ressalva"`
	Categoria    string `json:"categoria"`
	Risco        string `json:"risco"`
	PrecisaAdmin bool   `json:"precisaAdmin"`
	Recomendado  bool   `json:"recomendado"`
	Estado       string `json:"estado"`
	Detalhe      string `json:"detalhe"`
	PodeDesfazer bool   `json:"podeDesfazer"`
	Base         string `json:"base"`
}

type DiagnosticoUI struct {
	Perfil                string   `json:"perfil"`
	Admin                 bool     `json:"admin"`
	Itens                 []ItemUI `json:"itens"`
	Total                 int      `json:"total"`
	Aplicados             int      `json:"aplicados"`
	RecomendadosPendentes int      `json:"recomendadosPendentes"`
	PendentesDesfazer     int      `json:"pendentesDesfazer"`
	CaminhoHistorico      string   `json:"caminhoHistorico"`
}

// CategoriaVisaoUI agrega os ajustes do catálogo para a visualização de
// cobertura. Ela não representa uma nota de saúde do Windows: mostra apenas
// o estado dos ajustes que o Optimizer conhece.
type CategoriaVisaoUI struct {
	Nome      string `json:"nome"`
	Total     int    `json:"total"`
	Aplicados int    `json:"aplicados"`
}

// VisaoGeralUI entrega números consolidados para o painel inicial.
type VisaoGeralUI struct {
	Perfil                string             `json:"perfil"`
	TotalAjustes          int                `json:"totalAjustes"`
	Aplicados             int                `json:"aplicados"`
	RecomendadosPendentes int                `json:"recomendadosPendentes"`
	PendentesDesfazer     int                `json:"pendentesDesfazer"`
	CoberturaPercentual   float64            `json:"coberturaPercentual"`
	Categorias            []CategoriaVisaoUI `json:"categorias"`
}

type ResultadoUI struct {
	ID          string `json:"id"`
	Nome        string `json:"nome"`
	Estado      string `json:"estado"` // ok | pulado | falhou
	Mensagem    string `json:"mensagem"`
	PrecisaSair bool   `json:"precisaSair"`
}

type PassoUI struct {
	Tamanho int    `json:"tamanho"`
	MTU     int    `json:"mtu"`
	Estado  string `json:"estado"`
	Comando string `json:"comando"`
	Passou  bool   `json:"passou"`
}

type MTUUI struct {
	Destino      string    `json:"destino"`
	Adaptador    string    `json:"adaptador"`
	TipoAdapt    string    `json:"tipoAdaptador"`
	MTUAtual     uint32    `json:"mtuAtual"`
	MTUCaminho   int       `json:"mtuCaminho"`
	Bloqueado    bool      `json:"bloqueado"`
	Veredito     string    `json:"veredito"`
	Resumo       string    `json:"resumo"`
	Explicacao   string    `json:"explicacao"`
	ComandoOK    string    `json:"comandoOk"`
	ComandoFalha string    `json:"comandoFalha"`
	Sugestao     uint32    `json:"sugestao"`
	PodeAplicar  bool      `json:"podeAplicar"`
	Tentativas   []PassoUI `json:"tentativas"`
	Erro         string    `json:"erro"`
}

type EntradaUI struct {
	Quando    string `json:"quando"`
	Acao      string `json:"acao"`
	Item      string `json:"item"`
	Resultado string `json:"resultado"`
	Ok        bool   `json:"ok"`
}

// ------------------------------------------------------------------ bindings

// Diagnosticar lê o estado real da máquina. Não altera nada.
func (a *App) Diagnosticar(perfil string) DiagnosticoUI {
	p, ok := tweak.ParseProfile(perfil)
	if !ok {
		p = tweak.ProfilePersonal
	}

	out := DiagnosticoUI{Perfil: p.String(), Admin: a.eng.Elevated, CaminhoHistorico: a.eng.History.Path()}
	for _, s := range a.eng.Scan(a.ctx, p) {
		item := ItemUI{
			ID:           s.Meta.ID,
			Nome:         s.Meta.DisplayName,
			Descricao:    s.Meta.Description,
			Ressalva:     s.Meta.Caveat,
			PrecisaAdmin: s.Meta.RequiresAdmin,
			Recomendado:  s.Meta.RecommendedDefault,
			Detalhe:      s.Detail,
			PodeDesfazer: s.CanUndo,
			Base:         s.Meta.Evidence,
		}
		if t, ok := a.eng.Registry.Known(s.Meta.ID); ok {
			item.Categoria = tweak.CategoryLabel(t.Category())
			item.Risco = t.Risk().String()
		}
		switch {
		case s.Err != nil:
			item.Estado = "desconhecido"
		case s.State == tweak.StateApplied:
			item.Estado = "aplicado"
			out.Aplicados++
		case s.State == tweak.StatePartial:
			item.Estado = "parcial"
		default:
			item.Estado = "nao_aplicado"
			if s.Meta.RecommendedDefault {
				out.RecomendadosPendentes++
			}
		}
		out.Itens = append(out.Itens, item)
	}
	out.Total = len(out.Itens)

	if pend, err := a.eng.PendingIDs(); err == nil {
		out.PendentesDesfazer = len(pend)
	}
	return out
}

// ResumoVisao consolida o diagnóstico atual em dados prontos para gráficos e
// indicadores. O percentual é de cobertura do catálogo, não uma promessa de
// ganho de desempenho ou uma avaliação de saúde do sistema.
func (a *App) ResumoVisao(perfil string) VisaoGeralUI {
	diag := a.Diagnosticar(perfil)
	out := VisaoGeralUI{
		Perfil:                diag.Perfil,
		TotalAjustes:          diag.Total,
		Aplicados:             diag.Aplicados,
		RecomendadosPendentes: diag.RecomendadosPendentes,
		PendentesDesfazer:     diag.PendentesDesfazer,
		Categorias:            make([]CategoriaVisaoUI, 0),
	}
	if out.TotalAjustes > 0 {
		out.CoberturaPercentual = float64(out.Aplicados) / float64(out.TotalAjustes) * 100
	}

	porCategoria := make(map[string]*CategoriaVisaoUI)
	for _, item := range diag.Itens {
		nome := item.Categoria
		if nome == "" {
			nome = "Outros"
		}
		categoria := porCategoria[nome]
		if categoria == nil {
			categoria = &CategoriaVisaoUI{Nome: nome}
			porCategoria[nome] = categoria
		}
		categoria.Total++
		if item.Estado == "aplicado" {
			categoria.Aplicados++
		}
	}
	for _, categoria := range porCategoria {
		out.Categorias = append(out.Categorias, *categoria)
	}
	sort.Slice(out.Categorias, func(i, j int) bool {
		if out.Categorias[i].Total == out.Categorias[j].Total {
			return out.Categorias[i].Nome < out.Categorias[j].Nome
		}
		return out.Categorias[i].Total > out.Categorias[j].Total
	})
	return out
}

// Aplicar aplica os itens escolhidos. Com simular=true nada é gravado.
func (a *App) Aplicar(ids []string, simular bool, pontoDeRestauracao bool) []ResultadoUI {
	if len(ids) == 0 {
		return nil
	}
	var seq uint64
	var anyApplied bool
	if pontoDeRestauracao && !simular {
		a.eng.CreateRestorePoint = func(desc string) (uint64, error) {
			s, err := restore.Begin(desc)
			seq = s
			return s, err
		}
		defer func() {
			a.eng.CreateRestorePoint = nil
			if seq != 0 {
				if anyApplied {
					_ = restore.End(seq)
				} else {
					_ = restore.Cancel(seq)
				}
			}
		}()
	} else {
		a.eng.CreateRestorePoint = nil
	}
	rawResults := a.eng.Apply(a.ctx, ids, simular)
	for _, r := range rawResults {
		if r.Applied {
			anyApplied = true
			break
		}
	}
	return traduzir(rawResults)
}

// Desfazer volta os itens ao estado anterior gravado no histórico.
func (a *App) Desfazer(ids []string) []ResultadoUI {
	if len(ids) == 0 {
		return nil
	}
	return traduzir(a.eng.Revert(a.ctx, ids, false))
}

// DesfazerTudo reverte tudo que o app alterou e ainda não foi desfeito.
func (a *App) DesfazerTudo() []ResultadoUI {
	ids, err := a.eng.PendingIDs()
	if err != nil {
		return []ResultadoUI{{Estado: "falhou", Mensagem: err.Error()}}
	}
	if len(ids) == 0 {
		return nil
	}
	return traduzir(a.eng.Revert(a.ctx, ids, false))
}

func traduzir(rs []engine.Result) []ResultadoUI {
	out := make([]ResultadoUI, 0, len(rs))
	for _, r := range rs {
		u := ResultadoUI{ID: r.ID, Nome: r.Name, PrecisaSair: r.RestartNeeded}
		if u.Nome == "" {
			u.Nome = r.ID
		}
		switch {
		case r.Err != nil:
			u.Estado, u.Mensagem = "falhou", r.Err.Error()
		case r.Skipped:
			u.Estado, u.Mensagem = "pulado", r.Reason
		default:
			u.Estado, u.Mensagem = "ok", r.Detail
		}
		out = append(out, u)
	}
	return out
}

// MedirMTU mede o maior pacote que atravessa a conexão até o destino.
func (a *App) MedirMTU(destino string) MTUUI {
	if destino == "" {
		destino = "8.8.8.8"
	}
	res := MTUUI{Destino: destino}

	addr, err := netdiag.Resolve(destino)
	if err != nil {
		res.Erro = err.Error()
		return res
	}
	iface, err := netdiag.OutboundInterface(addr)
	if err != nil {
		res.Erro = err.Error()
		return res
	}
	res.Adaptador, res.TipoAdapt, res.MTUAtual = iface.Name, iface.KindLabel(), iface.MTU

	ctx, cancel := context.WithTimeout(a.ctx, 60*time.Second)
	defer cancel()

	rep, err := netdiag.ProbePathMTU(ctx, netdiag.LiveProber{}, destino, addr, netdiag.Options{})
	if err != nil {
		res.Erro = err.Error()
		return res
	}
	diag := netdiag.Diagnose(rep, iface)
	a.ultimoMTU = &medicaoMTU{iface: *iface, rep: rep, diag: diag}

	res.MTUCaminho = rep.PathMTU
	res.Bloqueado = rep.Blocked
	res.ComandoOK = rep.CommandOK
	res.ComandoFalha = rep.CommandFail
	res.Veredito = string(diag.Verdict)
	res.Resumo = diag.Summary
	res.Explicacao = diag.Explanation
	res.Sugestao = diag.SuggestedMTU
	res.PodeAplicar = diag.SuggestedMTU != 0
	for _, s := range rep.Steps {
		res.Tentativas = append(res.Tentativas, PassoUI{
			Tamanho: s.Payload, MTU: s.MTU, Estado: s.Status.String(),
			Comando: s.Command, Passou: s.Status == netdiag.ProbeOK,
		})
	}
	return res
}

// AplicarMTU grava o valor medido no adaptador, registrando no histórico para
// poder desfazer depois. Exige administrador.
func (a *App) AplicarMTU(simular bool) []ResultadoUI {
	m := a.ultimoMTU
	if m == nil {
		return []ResultadoUI{{Estado: "falhou", Mensagem: "Meça a conexão antes de aplicar qualquer ajuste."}}
	}
	tw, err := netdiag.NewMTUTweak(m.iface, m.rep, m.diag, netdiag.LiveMTU{})
	if err != nil {
		return []ResultadoUI{{Estado: "falhou", Mensagem: err.Error()}}
	}
	if !a.eng.Elevated && !simular {
		return []ResultadoUI{{Estado: "pulado", Nome: tw.Name(),
			Mensagem: "alterar o MTU precisa de permissão de administrador"}}
	}
	a.eng.Registry.Register(tw, tweak.Meta{
		Enabled: true, Profiles: tweak.ProfileBoth, RequiresAdmin: true,
		Evidence: "docs/catalogo/rede.md#5-mtu",
	})
	a.eng.CreateRestorePoint = nil
	return traduzir(a.eng.Apply(a.ctx, []string{tw.ID()}, simular))
}

// Historico devolve tudo que o app já alterou nesta máquina, do mais novo para
// o mais antigo.
func (a *App) Historico() []EntradaUI {
	entradas, err := a.eng.History.All()
	if err != nil {
		return nil
	}
	sort.SliceStable(entradas, func(i, j int) bool { return entradas[i].At.After(entradas[j].At) })

	out := make([]EntradaUI, 0, len(entradas))
	for _, e := range entradas {
		acao := "Aplicou"
		if e.Action == history.ActionRevert {
			acao = "Desfez"
		}
		if e.DryRun {
			acao += " (simulação)"
		}
		item := EntradaUI{
			Quando: e.At.Local().Format("02/01/2006 15:04"),
			Acao:   acao,
			Item:   e.TweakID,
			Ok:     e.Success,
		}
		item.Resultado = e.Detail
		if !e.Success {
			item.Resultado = e.Error
		}
		out = append(out, item)
	}
	return out
}

// AbrirHistorico abre o arquivo do histórico no Bloco de Notas — o usuário pode
// conferir com os próprios olhos tudo que foi feito.
func (a *App) AbrirHistorico() string {
	caminho := a.eng.History.Path()
	if err := exec.Command("notepad.exe", caminho).Start(); err != nil {
		return caminho
	}
	return ""
}

// ReiniciarComoAdmin reabre o app elevado (elevação sob demanda, nunca serviço
// permanente elevado).
func (a *App) ReiniciarComoAdmin() string {
	if a.eng.Elevated {
		return "O app já está rodando como administrador."
	}
	if err := elevate.Relaunch(); err != nil {
		return fmt.Sprintf("Não foi possível reabrir como administrador: %v", err)
	}
	return ""
}

// Sair fecha o app (usado depois de pedir elevação).
func (a *App) Sair() { encerrar(a.ctx) }

// ------------------------------------------------------------------ Perfis de Rede

type PerfilUI struct {
	Key       string `json:"key"`
	Nome      string `json:"nome"`
	Descricao string `json:"descricao"`
	Ressalvas string `json:"ressalvas"`
	NumTweaks int    `json:"numTweaks"`
}

// ListarPerfisRede retorna os perfis de rede disponíveis.
func (a *App) ListarPerfisRede() []PerfilUI {
	out := make([]PerfilUI, 0, len(profiles.List()))
	for _, p := range profiles.List() {
		out = append(out, PerfilUI{
			Key:       p.Key,
			Nome:      p.Name,
			Descricao: p.Description,
			Ressalvas: p.Caveats,
			NumTweaks: len(p.TweakIDs),
		})
	}
	return out
}

// AplicarPerfilRede aplica um perfil inteiro em lote.
func (a *App) AplicarPerfilRede(profileKey string, dryRun bool) []ResultadoUI {
	p, ok := profiles.Get(profileKey)
	if !ok {
		return []ResultadoUI{{
			ID:       profileKey,
			Estado:   "falhou",
			Mensagem: fmt.Sprintf("Perfil %q desconhecido.", profileKey),
		}}
	}

	if len(p.TweakIDs) == 0 {
		return []ResultadoUI{{
			ID:       profileKey,
			Nome:     p.Name,
			Estado:   "pulado",
			Mensagem: "Este perfil ainda não tem tweaks automatizados — apenas recomendações manuais.",
		}}
	}

	results := a.eng.Apply(a.ctx, p.TweakIDs, dryRun)
	out := make([]ResultadoUI, 0, len(results))
	for _, r := range results {
		estado := "ok"
		msg := r.Detail
		switch {
		case r.Err != nil:
			estado = "falhou"
			msg = r.Err.Error()
		case r.Skipped:
			estado = "pulado"
			msg = r.Reason
		}
		out = append(out, ResultadoUI{
			ID:          r.ID,
			Nome:        r.Name,
			Estado:      estado,
			Mensagem:    msg,
			PrecisaSair: r.RestartNeeded,
		})
	}
	return out
}

// ------------------------------------------------------------------ Medição de Rede

type BenchmarkUI struct {
	Timestamp   string  `json:"timestamp"`
	Host        string  `json:"host"`
	MinRTT      int     `json:"minRTT"`
	AvgRTT      int     `json:"avgRTT"`
	MaxRTT      int     `json:"maxRTT"`
	StdDev      int     `json:"stdDev"`
	PacketsSent int     `json:"packetsSent"`
	PacketsLost int     `json:"packetsLost"`
	LossPercent float64 `json:"lossPercent"`
	Erro        string  `json:"erro,omitempty"`
}

type ComparativoUI struct {
	Antes         BenchmarkUI `json:"antes"`
	Depois        BenchmarkUI `json:"depois"`
	DeltaLatencia string      `json:"deltaLatencia"`
	DeltaJitter   string      `json:"deltaJitter"`
	Interpretacao string      `json:"interpretacao"`
}

type DNSAtualUI struct {
	Interface  string   `json:"interface"`
	Servidores []string `json:"servidores"`
	Erro       string   `json:"erro,omitempty"`
}

type ResultadoDNSUI struct {
	Ok       bool   `json:"ok"`
	Mensagem string `json:"mensagem"`
}

// medirRede é a implementação comum de Antes/Depois — só muda o rótulo do momento.
// Segue o mesmo padrão de MedirMTU: nunca devolve um error Go cru para o
// frontend (isso vira uma promise rejeitada não tratada no JS); o erro vem
// embutido no campo Erro da própria struct.
func (a *App) medirRede(host string) BenchmarkUI {
	if host == "" {
		host = "8.8.8.8"
	}
	report, err := netdiag.MeasureLatency(a.ctx, host, 20)
	if err != nil {
		return BenchmarkUI{Host: host, Erro: err.Error()}
	}
	lossPercent := 0.0
	if report.PacketsSent > 0 {
		lossPercent = float64(report.PacketsLost) / float64(report.PacketsSent) * 100
	}
	return BenchmarkUI{
		Timestamp:   time.Now().Format(time.RFC3339),
		Host:        report.Host,
		MinRTT:      report.MinRTT,
		AvgRTT:      report.AvgRTT,
		MaxRTT:      report.MaxRTT,
		StdDev:      report.StdDev,
		PacketsSent: report.PacketsSent,
		PacketsLost: report.PacketsLost,
		LossPercent: lossPercent,
	}
}

// MedirRedeAntes executa um benchmark antes de aplicar um ajuste.
func (a *App) MedirRedeAntes(host string) BenchmarkUI {
	return a.medirRede(host)
}

// MedirRedeDepois executa um benchmark depois de aplicar um ajuste.
func (a *App) MedirRedeDepois(host string) BenchmarkUI {
	return a.medirRede(host)
}

// RelatorioComparativo compara dois benchmarks e retorna a interpretação honesta.
func (a *App) RelatorioComparativo(antes, depois BenchmarkUI) ComparativoUI {
	before := netdiag.Benchmark{
		Host: antes.Host,
		Latency: netdiag.LatencyReport{
			Host: antes.Host, MinRTT: antes.MinRTT, AvgRTT: antes.AvgRTT,
			MaxRTT: antes.MaxRTT, StdDev: antes.StdDev,
			PacketsSent: antes.PacketsSent, PacketsLost: antes.PacketsLost,
		},
		Jitter: antes.StdDev,
		Loss:   antes.LossPercent,
	}
	after := netdiag.Benchmark{
		Host: depois.Host,
		Latency: netdiag.LatencyReport{
			Host: depois.Host, MinRTT: depois.MinRTT, AvgRTT: depois.AvgRTT,
			MaxRTT: depois.MaxRTT, StdDev: depois.StdDev,
			PacketsSent: depois.PacketsSent, PacketsLost: depois.PacketsLost,
		},
		Jitter: depois.StdDev,
		Loss:   depois.LossPercent,
	}

	delta := netdiag.Compare(before, after)

	deltaJitter := "Estável"
	switch {
	case delta.JitterDeltaPercent > 10:
		deltaJitter = fmt.Sprintf("+%.1f%% (piorou)", delta.JitterDeltaPercent)
	case delta.JitterDeltaPercent < -10:
		deltaJitter = fmt.Sprintf("%.1f%% (melhorou)", delta.JitterDeltaPercent)
	}

	return ComparativoUI{
		Antes:         antes,
		Depois:        depois,
		DeltaLatencia: fmt.Sprintf("%+d ms (%.1f%%)", delta.LatencyAbsoluteDelta, delta.LatencyDeltaPercent),
		DeltaJitter:   deltaJitter,
		Interpretacao: delta.Interpretation,
	}
}

// ---------------------------------------------------------------- Novos Módulos

// ListarInicializacao lista programas configurados para iniciar com o Windows.
func (a *App) ListarInicializacao() []systemdiag.StartupItem {
	b := winreg.NewLive()
	itens, err := systemdiag.ListarStartupItems(b)
	if err != nil {
		return []systemdiag.StartupItem{}
	}
	return itens
}

// AlternarInicializacao ativa ou desativa um programa de inicialização.
func (a *App) AlternarInicializacao(id string, ativar bool) string {
	b := winreg.NewLive()
	err := systemdiag.AlternarStartup(b, id, ativar)
	if err != nil {
		return err.Error()
	}
	return ""
}

// AuditarReparo executa auditoria somente leitura de integridade do sistema (DISM/SFC).
func (a *App) AuditarReparo() systemdiag.RepairAuditReport {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	return systemdiag.ExecutarAuditoriaIntegridade(ctx)
}

// ListarPlanosEnergia lista os esquemas de energia do Windows.
func (a *App) ListarPlanosEnergia() []systemdiag.PowerPlan {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	planos, err := systemdiag.ListarPlanosEnergia(ctx)
	if err != nil {
		return []systemdiag.PowerPlan{}
	}
	return planos
}

// AtivarPlanoEnergia define o plano de energia ativo pelo GUID.
func (a *App) AtivarPlanoEnergia(guid string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := systemdiag.AtivarPlanoEnergia(ctx, guid)
	if err != nil {
		return err.Error()
	}
	return ""
}

// ListarDiscos audita e lista os volumes e tipos de mídia instalados.
func (a *App) ListarDiscos() []diskdiag.DriveInfo {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	drives, err := diskdiag.ListarUnidades(ctx)
	if err != nil {
		return []diskdiag.DriveInfo{}
	}
	return drives
}

// ExecutarTRIM executa TRIM seguro na unidade selecionada.
func (a *App) ExecutarTRIM(drive string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	out, err := diskdiag.ExecutarTRIM(ctx, drive)
	if err != nil {
		return fmt.Sprintf("Erro ao executar TRIM: %v\n%s", err, out)
	}
	return out
}

// ExecutarChkdsk executa verificação online não destrutiva (chkdsk /scan).
func (a *App) ExecutarChkdsk(drive string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	out, err := diskdiag.ExecutarChkdskScan(ctx, drive)
	if err != nil {
		return fmt.Sprintf("Verificação concluída com observações:\n%s", out)
	}
	return out
}

// BenchmarkDNS executa comparação de tempo de resposta dos principais provedores DNS globais.
func (a *App) BenchmarkDNS() []netdiag.DNSProvider {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	return netdiag.BenchmarkDNS(ctx, nil)
}

// ObterDNSAtual identifica o resolvedor usado pela interface da rota IPv4
// padrão. A operação é somente leitura.
func (a *App) ObterDNSAtual() DNSAtualUI {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	atual, err := netdiag.LerDNSAtual(ctx, a.obterDNSRunner())
	if err != nil {
		return DNSAtualUI{Erro: err.Error()}
	}
	return DNSAtualUI{Interface: atual.InterfaceAlias, Servidores: atual.Servidores}
}

// AplicarDNS configura os resolvedores escolhidos na interface da rota IPv4
// padrão. Exige o app elevado porque o Windows protege essa configuração.
func (a *App) AplicarDNS(servidores []string) ResultadoDNSUI {
	if a.eng == nil || !a.eng.Elevated {
		return ResultadoDNSUI{Mensagem: "Alterar o DNS exige permissão de administrador."}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	atual, err := netdiag.LerDNSAtual(ctx, a.obterDNSRunner())
	if err != nil {
		return ResultadoDNSUI{Mensagem: err.Error()}
	}
	if err := netdiag.ConfigurarDNS(ctx, a.obterDNSRunner(), atual, servidores); err != nil {
		return ResultadoDNSUI{Mensagem: err.Error()}
	}
	return ResultadoDNSUI{Ok: true, Mensagem: "DNS configurado para " + atual.InterfaceAlias + "."}
}

func (a *App) obterDNSRunner() netdiag.DNSRunner {
	if a.dnsRunner != nil {
		return a.dnsRunner
	}
	return netdiag.LiveDNSRunner{}
}

// ResultadoLimpezaPnp resume os nós removidos.
type ResultadoLimpezaPnp struct {
	Removidos int      `json:"removidos"`
	Erros     []string `json:"erros"`
	Mensagem  string   `json:"mensagem"`
}

// ListarDispositivosFantasmas identifica dispositivos PnP desconectados ou órfãos no sistema.
func (a *App) ListarDispositivosFantasmas() []systemdiag.PnpDevice {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	devs, err := systemdiag.ListarDispositivosFantasmas(ctx)
	if err != nil {
		return []systemdiag.PnpDevice{}
	}
	return devs
}

// LimparDispositivosFantasmas remove os dispositivos informados usando pnputil.
func (a *App) LimparDispositivosFantasmas(instanceIDs []string) ResultadoLimpezaPnp {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	removidos, errs, err := systemdiag.LimparDispositivosFantasmas(ctx, instanceIDs)
	msg := fmt.Sprintf("%d dispositivo(s) fantasma(s) removido(s) com sucesso.", removidos)
	if len(errs) > 0 {
		msg += fmt.Sprintf(" (%d falha(s))", len(errs))
	}
	if err != nil {
		msg = err.Error()
	}
	return ResultadoLimpezaPnp{
		Removidos: removidos,
		Erros:     errs,
		Mensagem:  msg,
	}
}

// ---------------------------------------------------------------- Perfis de Uso (JOGO / CODING)

// ListarPerfisUso lista os perfis de uso fechados (JOGO e CODING).
func (a *App) ListarPerfisUso() []profiles.UseProfile {
	return profiles.ListarPerfisUso()
}

// ObterPerfilAtivo identifica qual perfil de uso está atualmente ativo.
func (a *App) ObterPerfilAtivo() string {
	if a.eng == nil || a.eng.History == nil {
		return ""
	}
	if _, entries, _ := a.eng.History.ActiveBatchForOrigin("perfil-jogo"); len(entries) > 0 {
		return "jogo"
	}
	if _, entries, _ := a.eng.History.ActiveBatchForOrigin("perfil-coding"); len(entries) > 0 {
		return "coding"
	}
	return ""
}

// AplicarPerfilUso aplica o perfil JOGO ou CODING transacionalmente.
func (a *App) AplicarPerfilUso(key string, dryRun bool) []ResultadoUI {
	p, ok := profiles.ObterPerfilUso(key)
	if !ok {
		return []ResultadoUI{{Nome: key, Estado: "falhou", Mensagem: "Perfil de uso desconhecido"}}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// 1. Reverte o perfil anterior se houver algum ativo
	ativo := a.ObterPerfilAtivo()
	if ativo != "" && ativo != key && !dryRun {
		batchID, _, _ := a.eng.History.ActiveBatchForOrigin("perfil-" + ativo)
		if batchID != "" {
			_ = a.eng.RevertBatch(ctx, batchID, false)
		}
	}

	// 2. Aplica o novo lote com ID de transação e origem
	batchID := fmt.Sprintf("batch-%s-%d", key, time.Now().Unix())
	origin := "perfil-" + key
	rawResults := a.eng.ApplyBatch(ctx, p.TweakIDs, dryRun, origin, batchID)

	// 3. Ajustes de energia e suspensão
	if !dryRun {
		if p.PowerPlanGUID != "" {
			_ = systemdiag.AtivarPlanoEnergia(ctx, p.PowerPlanGUID)
		}
		if p.DisableSleepAC {
			_ = exec.CommandContext(ctx, "powercfg", "/change", "standby-timeout-ac", "0").Run()
		}
		// Pausa ou retomada de serviços
		for _, s := range p.ServicesToPause {
			_ = exec.CommandContext(ctx, "net", "stop", s, "/y").Run()
		}
		for _, s := range p.ServicesToEnsure {
			_ = exec.CommandContext(ctx, "net", "start", s).Run()
		}
	}

	var results []ResultadoUI
	for _, r := range rawResults {
		estado := "ok"
		if r.Skipped {
			estado = "pulado"
		} else if r.Err != nil {
			estado = "falhou"
		}
		msg := r.Detail
		if r.Reason != "" {
			msg = r.Reason
		}
		if r.Err != nil {
			msg = r.Err.Error()
		}
		results = append(results, ResultadoUI{
			ID:          r.ID,
			Nome:        r.Name,
			Estado:      estado,
			Mensagem:    msg,
			PrecisaSair: r.RestartNeeded,
		})
	}
	return results
}

// RestaurarPerfilUso restaura o estado anterior ao perfil ativo.
func (a *App) RestaurarPerfilUso(key string) []ResultadoUI {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	batchID, entries, _ := a.eng.History.ActiveBatchForOrigin("perfil-" + key)
	if batchID == "" || len(entries) == 0 {
		return []ResultadoUI{{Nome: key, Estado: "pulado", Mensagem: "Nenhum lote ativo deste perfil para restaurar"}}
	}

	rawResults := a.eng.RevertBatch(ctx, batchID, false)

	// Retoma serviços e restaura plano equilibrado se for jogo
	if key == "jogo" {
		_ = exec.CommandContext(ctx, "net", "start", "WSearch").Run()
		_ = exec.CommandContext(ctx, "net", "start", "SysMain").Run()
		_ = systemdiag.AtivarPlanoEnergia(ctx, "381b4222-f694-41f0-9685-ff5bb260df2e")
	}

	var results []ResultadoUI
	for _, r := range rawResults {
		estado := "ok"
		if r.Skipped {
			estado = "pulado"
		} else if r.Err != nil {
			estado = "falhou"
		}
		msg := r.Detail
		if r.Reason != "" {
			msg = r.Reason
		}
		if r.Err != nil {
			msg = r.Err.Error()
		}
		results = append(results, ResultadoUI{
			ID:          r.ID,
			Nome:        r.Name,
			Estado:      estado,
			Mensagem:    msg,
			PrecisaSair: r.RestartNeeded,
		})
	}
	return results
}

// ObterMetricasCPU lê as métricas de uso e processos de CPU em tempo real.
func (a *App) ObterMetricasCPU() cpudiag.CPUInfo {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	prober := cpudiag.NewLiveProber()
	info, err := prober.GetCPUInfo(ctx)
	if err != nil {
		return cpudiag.CPUInfo{Interpretation: "Não foi possível ler dados de CPU no momento."}
	}
	return info
}

// ---------------------------------------------------------------- Telemetria & Benchmark Obrigatório de Perfil

type BenchmarkProgressUI struct {
	Stage            string   `json:"stage"`
	Current          int      `json:"current"`
	Total            int      `json:"total"`
	Percent          float64  `json:"percent"`
	CPUUsage         float64  `json:"cpuUsage"`
	CPUTemp          *float64 `json:"cpuTemp,omitempty"`
	GPUUsage         float64  `json:"gpuUsage"`
	GPUTemp          *float64 `json:"gpuTemp,omitempty"`
	RAMUsedMB        float64  `json:"ramUsedMb"`
	ThermalThrottled bool     `json:"thermalThrottled"`
}

type ResultadoAplicacaoPerfil struct {
	BatchID     string                    `json:"batchId"`
	Resultados  []ResultadoUI             `json:"resultados"`
	ReportAntes telemetry.BenchmarkReport `json:"reportAntes"`
}

func (a *App) emitEvent(name string, data ...any) {
	if a.ctx == nil || a.ctx.Value("frontend") == nil {
		return
	}
	defer func() {
		_ = recover()
	}()
	runtime.EventsEmit(a.ctx, name, data...)
}

// IniciarBenchmarkBase executa o benchmark observacional de 60s antes de aplicar o perfil.
func (a *App) IniciarBenchmarkBase(perfilKey string, segundos int) (telemetry.BenchmarkReport, error) {
	if segundos <= 0 {
		segundos = 60
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(segundos+15)*time.Second)
	a.benchMu.Lock()
	a.benchCancel = cancel
	collector := a.benchCollector
	if collector == nil {
		collector = telemetry.NewCollector(telemetry.NewWindowsLiveProvider())
		a.benchCollector = collector
	}
	a.benchMu.Unlock()

	defer func() {
		a.benchMu.Lock()
		a.benchCancel = nil
		a.benchMu.Unlock()
	}()

	report, err := collector.RunBenchmark(ctx, "before", time.Duration(segundos)*time.Second, 1*time.Second, func(curr, total int, last telemetry.MetricSample) {
		pct := float64(curr) / float64(total) * 100.0
		progress := BenchmarkProgressUI{
			Stage:            "before",
			Current:          curr,
			Total:            total,
			Percent:          pct,
			CPUUsage:         last.CPUUsagePercent,
			CPUTemp:          last.CPUTempCelsius,
			GPUUsage:         last.GPUUsagePercent,
			GPUTemp:          last.GPUTempCelsius,
			RAMUsedMB:        last.RAMUsedMB,
			ThermalThrottled: last.ThermalThrottling,
		}
		a.emitEvent("benchmark:progress", progress)
	})

	if err == nil {
		a.benchMu.Lock()
		if a.ultimoReportAntes == nil {
			a.ultimoReportAntes = make(map[string]telemetry.BenchmarkReport)
		}
		a.ultimoReportAntes[perfilKey] = report
		a.benchMu.Unlock()
	}

	return report, err
}

// CancelarBenchmark interrompe imediatamente a coleta de telemetria em andamento.
func (a *App) CancelarBenchmark() {
	a.benchMu.Lock()
	defer a.benchMu.Unlock()
	if a.benchCancel != nil {
		a.benchCancel()
		a.benchCancel = nil
	}
}

// AplicarPerfilComBenchmark aplica o perfil registrando o benchmark prévio e gerando o lote.
func (a *App) AplicarPerfilComBenchmark(key string, dryRun bool, reportAntes telemetry.BenchmarkReport) ResultadoAplicacaoPerfil {
	p, ok := profiles.ObterPerfilUso(key)
	if !ok {
		return ResultadoAplicacaoPerfil{
			Resultados: []ResultadoUI{{Nome: key, Estado: "falhou", Mensagem: "Perfil de uso desconhecido"}},
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	ativo := a.ObterPerfilAtivo()
	if ativo != "" && ativo != key && !dryRun {
		batchID, _, _ := a.eng.History.ActiveBatchForOrigin("perfil-" + ativo)
		if batchID != "" {
			_ = a.eng.RevertBatch(ctx, batchID, false)
		}
	}

	batchID := fmt.Sprintf("batch-%s-%d", key, time.Now().Unix())
	origin := "perfil-" + key
	rawResults := a.eng.ApplyBatch(ctx, p.TweakIDs, dryRun, origin, batchID)

	if !dryRun {
		if p.PowerPlanGUID != "" {
			_ = systemdiag.AtivarPlanoEnergia(ctx, p.PowerPlanGUID)
		}
		if p.DisableSleepAC {
			_ = exec.CommandContext(ctx, "powercfg", "/change", "standby-timeout-ac", "0").Run()
		}
		for _, s := range p.ServicesToPause {
			_ = exec.CommandContext(ctx, "net", "stop", s, "/y").Run()
		}
		for _, s := range p.ServicesToEnsure {
			_ = exec.CommandContext(ctx, "net", "start", s).Run()
		}

		// Registra o relatório prévio no histórico junto ao lote
		if a.eng.History != nil {
			_, _ = a.eng.History.SaveBenchmarkRecord(origin, batchID, &reportAntes, nil)
		}
	}

	var results []ResultadoUI
	for _, r := range rawResults {
		estado := "ok"
		if r.Skipped {
			estado = "pulado"
		} else if r.Err != nil {
			estado = "falhou"
		}
		msg := r.Detail
		if r.Reason != "" {
			msg = r.Reason
		}
		if r.Err != nil {
			msg = r.Err.Error()
		}
		results = append(results, ResultadoUI{
			ID:          r.ID,
			Nome:        r.Name,
			Estado:      estado,
			Mensagem:    msg,
			PrecisaSair: r.RestartNeeded,
		})
	}

	return ResultadoAplicacaoPerfil{
		BatchID:     batchID,
		Resultados:  results,
		ReportAntes: reportAntes,
	}
}

// IniciarBenchmarkPos executa a coleta pós-aplicação e devolve o comparativo completo.
func (a *App) IniciarBenchmarkPos(perfilKey string, batchID string, segundos int) (telemetry.BenchmarkComparison, error) {
	if segundos <= 0 {
		segundos = 60
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(segundos+15)*time.Second)
	a.benchMu.Lock()
	a.benchCancel = cancel
	collector := a.benchCollector
	if collector == nil {
		collector = telemetry.NewCollector(telemetry.NewWindowsLiveProvider())
		a.benchCollector = collector
	}
	a.benchMu.Unlock()

	defer func() {
		a.benchMu.Lock()
		a.benchCancel = nil
		a.benchMu.Unlock()
	}()

	reportPos, err := collector.RunBenchmark(ctx, "after", time.Duration(segundos)*time.Second, 1*time.Second, func(curr, total int, last telemetry.MetricSample) {
		pct := float64(curr) / float64(total) * 100.0
		progress := BenchmarkProgressUI{
			Stage:            "after",
			Current:          curr,
			Total:            total,
			Percent:          pct,
			CPUUsage:         last.CPUUsagePercent,
			CPUTemp:          last.CPUTempCelsius,
			GPUUsage:         last.GPUUsagePercent,
			GPUTemp:          last.GPUTempCelsius,
			RAMUsedMB:        last.RAMUsedMB,
			ThermalThrottled: last.ThermalThrottling,
		}
		a.emitEvent("benchmark:progress", progress)
	})

	a.benchMu.Lock()
	reportAntes := a.ultimoReportAntes[perfilKey]
	a.benchMu.Unlock()

	// Se não estava em memória, tenta recuperar do histórico
	if reportAntes.SampleCount == 0 && a.eng.History != nil {
		bAntes, _, _ := a.eng.History.GetBatchBenchmark(batchID)
		if bAntes != nil {
			reportAntes = *bAntes
		}
	}

	comp := telemetry.CompareBenchmarks(perfilKey, batchID, reportAntes, reportPos)

	// Atualiza histórico com ambos relatórios
	if a.eng.History != nil && err == nil {
		origin := "perfil-" + perfilKey
		_, _ = a.eng.History.SaveBenchmarkRecord(origin, batchID, &reportAntes, &reportPos)
	}

	return comp, err
}
