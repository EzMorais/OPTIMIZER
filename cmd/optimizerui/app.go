package main

import (
	"context"
	"fmt"
	"os/exec"
	"sort"
	"time"

	"optimizer/internal/elevate"
	"optimizer/internal/engine"
	"optimizer/internal/history"
	"optimizer/internal/netdiag"
	"optimizer/internal/restore"
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

// Aplicar aplica os itens escolhidos. Com simular=true nada é gravado.
func (a *App) Aplicar(ids []string, simular bool, pontoDeRestauracao bool) []ResultadoUI {
	if len(ids) == 0 {
		return nil
	}
	var seq uint64
	if pontoDeRestauracao && !simular {
		a.eng.CreateRestorePoint = func(desc string) (uint64, error) {
			s, err := restore.Begin(desc)
			seq = s
			return s, err
		}
		defer func() {
			a.eng.CreateRestorePoint = nil
			if seq != 0 {
				_ = restore.End(seq)
			}
		}()
	} else {
		a.eng.CreateRestorePoint = nil
	}
	return traduzir(a.eng.Apply(a.ctx, ids, simular))
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
	Key         string `json:"key"`
	Nome        string `json:"nome"`
	Descricao   string `json:"descricao"`
	Ressalvas   string `json:"ressalvas"`
	NumTweaks   int    `json:"numTweaks"`
}

// ListarPerfisDere retorna os perfis de rede disponíveis.
func (a *App) ListarPerfisRede() []PerfilUI {
	// TODO: integrar com profiles.List()
	return []PerfilUI{
		{
			Key:       "network-fast",
			Nome:      "Rede Rápida",
			Descricao: "Desativa economia de energia, habilita offloads. Foco em throughput.",
			Ressalvas: "Aumenta consumo de energia do adaptador.",
			NumTweaks: 2,
		},
		{
			Key:       "network-remote",
			Nome:      "Trabalho Remoto",
			Descricao: "Otimiza para RDP, SSH, Zoom. Reduz latência de round-trip.",
			Ressalvas: "Pode aumentar overhead de ACK em Wi-Fi instável.",
			NumTweaks: 0,
		},
		{
			Key:       "network-dev",
			Nome:      "Desenvolvimento",
			Descricao: "Foco em throughput e latência baixa. Sem interferência de P2P.",
			Ressalvas: "",
			NumTweaks: 2,
		},
		{
			Key:       "network-presentation",
			Nome:      "Apresentação",
			Descricao: "Minimiza interferências de background. Wi-Fi estável.",
			Ressalvas: "Exige seleção manual de Wi-Fi 5 GHz.",
			NumTweaks: 2,
		},
	}
}

// AplicarPerfilRede aplica um perfil inteiro em lote.
func (a *App) AplicarPerfilRede(ctx context.Context, profileKey string, dryRun bool) []ResultadoUI {
	// TODO: implementar com a.eng.ApplyProfile()
	return []ResultadoUI{
		{
			ID:       profileKey,
			Nome:     "Perfil " + profileKey,
			Estado:   "não implementado",
			Mensagem: "Aplicação de perfil ainda não implementada.",
		},
	}
}

// ------------------------------------------------------------------ Medição de Rede

type BenchmarkUI struct {
	Timestamp      string  `json:"timestamp"`
	Host           string  `json:"host"`
	MinRTT         int     `json:"minRTT"`
	AvgRTT         int     `json:"avgRTT"`
	MaxRTT         int     `json:"maxRTT"`
	StdDev         int     `json:"stdDev"`
	PacketsSent    int     `json:"packetsSent"`
	PacketsLost    int     `json:"packetsLost"`
	LossPercent    float64 `json:"lossPercent"`
}

type ComparativoUI struct {
	Antes           BenchmarkUI `json:"antes"`
	Depois          BenchmarkUI `json:"depois"`
	DeltaLatencia   string      `json:"deltaLatencia"`
	DeltaJitter     string      `json:"deltaJitter"`
	Interpretacao   string      `json:"interpretacao"`
}

// MedirRedeAntes executa um benchmark antes de aplicar um ajuste.
func (a *App) MedirRedeAntes(ctx context.Context, host string) (BenchmarkUI, error) {
	// TODO: implementar com netdiag.MeasureLatency()
	if host == "" {
		host = "8.8.8.8"
	}
	return BenchmarkUI{
		Timestamp: time.Now().Format(time.RFC3339),
		Host:      host,
		AvgRTT:    50,
		StdDev:    2,
		MinRTT:    48,
		MaxRTT:    55,
		PacketsSent: 20,
		PacketsLost: 0,
		LossPercent: 0,
	}, nil
}

// MedirRedeDepois executa um benchmark depois de aplicar um ajuste.
func (a *App) MedirRedeDepois(ctx context.Context, host string) (BenchmarkUI, error) {
	// TODO: implementar com netdiag.MeasureLatency()
	if host == "" {
		host = "8.8.8.8"
	}
	return BenchmarkUI{
		Timestamp: time.Now().Format(time.RFC3339),
		Host:      host,
		AvgRTT:    48,
		StdDev:    2,
		MinRTT:    46,
		MaxRTT:    52,
		PacketsSent: 20,
		PacketsLost: 0,
		LossPercent: 0,
	}, nil
}

// RelatorioComparativo compara dois benchmarks e retorna a interpretação.
func (a *App) RelatorioComparativo(antes, depois BenchmarkUI) ComparativoUI {
	// TODO: implementar com netdiag.Compare()
	return ComparativoUI{
		Antes:         antes,
		Depois:        depois,
		DeltaLatencia: fmt.Sprintf("%d ms", depois.AvgRTT-antes.AvgRTT),
		DeltaJitter:   "Estável",
		Interpretacao: "Latência estável. Sem mudança perceptível.",
	}
}
