// Comando optimizerctl — ferramenta interna de diagnóstico e desenvolvimento.
//
// NÃO é o produto: é a mesma engrenagem que o app desktop vai usar (catálogo,
// motor, histórico, medição de rede), exposta em linha de comando para dar para
// testar tudo antes de existir interface gráfica. Não é distribuída ao usuário
// final. Ver docs/arquitetura-app-desktop.md, seção 6.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"text/tabwriter"
	"time"

	"optimizer/internal/console"
	"optimizer/internal/elevate"
	"optimizer/internal/engine"
	"optimizer/internal/history"
	"optimizer/internal/netdiag"
	"optimizer/internal/profiles"
	"optimizer/internal/restore"
	"optimizer/internal/tweak"
	"optimizer/internal/tweaks"
	"optimizer/internal/winreg"
)

const uso = `optimizerctl — ferramenta interna do Optimizer (não distribuída ao usuário final)

Uso:
  optimizerctl <comando> [opções]

Comandos:
  listar         Mostra o catálogo de otimizações e o que cada uma faz
  diagnosticar   Lê o estado atual de cada item nesta máquina (não altera nada)
  aplicar        Aplica otimizações:  aplicar <id...>  |  aplicar --recomendados
  desfazer       Desfaz o que foi aplicado:  desfazer <id...>  |  desfazer --tudo
  historico      Mostra tudo que o app já alterou nesta máquina
  mtu            Mede o MTU ideal desta conexão e explica o resultado
  perfil         Gerencia perfis nomeados (Rede Rápida, Trabalho Remoto, etc)
  rede           Realiza medições e diagnósticos de rede

Opções gerais:
  --perfil pessoal|trabalho   Qual aba usar (padrão: pessoal)
  --simular                   Faz todas as validações sem gravar nada
  --historico <arquivo>       Caminho do histórico (padrão: %LOCALAPPDATA%\Optimizer)

Exemplos:
  optimizerctl diagnosticar
  optimizerctl aplicar --recomendados --simular
  optimizerctl mtu --destino 1.1.1.1 --detalhado
  optimizerctl desfazer --tudo
`

func main() {
	console.EnableUTF8()
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "\nErro: %v\n", err)
		os.Exit(1)
	}
}

type globals struct {
	profile     tweak.Profile
	dryRun      bool
	historyPath string
	// resolve traduz as flags depois do Parse (o valor só existe a partir daí).
	resolve func() error
}

func run(args []string) error {
	if len(args) == 0 {
		fmt.Print(uso)
		return nil
	}
	cmd, rest := args[0], args[1:]

	switch cmd {
	case "-h", "--help", "help", "ajuda":
		fmt.Print(uso)
		return nil
	case "listar":
		return cmdListar(rest)
	case "diagnosticar":
		return cmdDiagnosticar(rest)
	case "aplicar":
		return cmdAplicar(rest)
	case "desfazer":
		return cmdDesfazer(rest)
	case "historico", "histórico":
		return cmdHistorico(rest)
	case "mtu":
		return cmdMTU(rest)
	case "perfil":
		return cmdPerfil(rest)
	case "rede":
		return cmdRede(rest)
	}
	return fmt.Errorf("comando desconhecido %q — rode `optimizerctl ajuda`", cmd)
}

func addGlobals(fs *flag.FlagSet) *globals {
	g := &globals{}
	perfil := fs.String("perfil", "pessoal", "aba do produto: pessoal ou trabalho")
	fs.BoolVar(&g.dryRun, "simular", false, "não grava nada, só mostra o que aconteceria")
	fs.StringVar(&g.historyPath, "historico", "", "caminho do arquivo de histórico")
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, uso)
		fmt.Fprintln(os.Stderr, "\nOpções deste comando:")
		fs.PrintDefaults()
	}
	// A resolução do perfil precisa acontecer depois do Parse; guardamos o ponteiro.
	g.resolve = func() error {
		p, ok := tweak.ParseProfile(*perfil)
		if !ok {
			return fmt.Errorf("perfil inválido %q — use pessoal ou trabalho", *perfil)
		}
		g.profile = p
		return nil
	}
	return g
}

// parseArgs aceita opções antes E depois dos IDs. O flag padrão da biblioteca
// para de interpretar opções no primeiro argumento solto, o que faria
// `aplicar <id> --historico x` tratar "--historico" como se fosse outro ID.
func parseArgs(fs *flag.FlagSet, g *globals, args []string) ([]string, error) {
	var soltos []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, err
		}
		resto := fs.Args()
		if len(resto) == 0 {
			break
		}
		soltos = append(soltos, resto[0])
		args = resto[1:]
	}
	if err := g.resolve(); err != nil {
		return nil, err
	}
	return soltos, nil
}

func openEngine(g *globals) (*engine.Engine, error) {
	path := g.historyPath
	if path == "" {
		p, err := history.DefaultPath()
		if err != nil {
			return nil, fmt.Errorf("descobrindo a pasta do histórico: %w", err)
		}
		path = p
	}
	store, err := history.Open(path)
	if err != nil {
		return nil, err
	}
	return &engine.Engine{
		Registry: tweaks.Build(winreg.NewLive()),
		History:  store,
		Elevated: elevate.IsElevated(),
	}, nil
}

func ctxWithSignal() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt)
}

func newTab() *tabwriter.Writer {
	return tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
}

// ---------------------------------------------------------------- listar

func cmdListar(args []string) error {
	fs := flag.NewFlagSet("listar", flag.ExitOnError)
	g := addGlobals(fs)
	detalhado := fs.Bool("detalhado", false, "mostra explicação e ressalva de cada item")
	if _, err := parseArgs(fs, g, args); err != nil {
		return err
	}

	reg := tweaks.Build(winreg.NewLive())
	entries := reg.ForProfile(g.profile)
	fmt.Printf("Catálogo — perfil %s (%d itens)\n\n", g.profile, len(entries))

	if !*detalhado {
		w := newTab()
		fmt.Fprintln(w, "ID\tNOME\tCATEGORIA\tRISCO\tADMIN\tRECOMENDADO")
		for _, e := range entries {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				e.Meta.ID, e.Meta.DisplayName, tweak.CategoryLabel(e.Tweak.Category()),
				e.Tweak.Risk(), simNao(e.Meta.RequiresAdmin), simNao(e.Meta.RecommendedDefault))
		}
		return w.Flush()
	}

	for _, e := range entries {
		fmt.Printf("• %s  [%s]\n", e.Meta.DisplayName, e.Meta.ID)
		fmt.Printf("  %s\n", e.Meta.Description)
		if e.Meta.Caveat != "" {
			fmt.Printf("  Ressalva honesta: %s\n", e.Meta.Caveat)
		}
		fmt.Printf("  Categoria: %s · Risco: %s · Precisa de administrador: %s · Recomendado por padrão: %s\n",
			tweak.CategoryLabel(e.Tweak.Category()), e.Tweak.Risk(),
			simNao(e.Meta.RequiresAdmin), simNao(e.Meta.RecommendedDefault))
		if e.Meta.Evidence != "" {
			fmt.Printf("  Base técnica: %s\n", e.Meta.Evidence)
		}
		fmt.Println()
	}
	return nil
}

func simNao(b bool) string {
	if b {
		return "sim"
	}
	return "não"
}

// ---------------------------------------------------------- diagnosticar

func cmdDiagnosticar(args []string) error {
	fs := flag.NewFlagSet("diagnosticar", flag.ExitOnError)
	g := addGlobals(fs)
	if _, err := parseArgs(fs, g, args); err != nil {
		return err
	}
	eng, err := openEngine(g)
	if err != nil {
		return err
	}
	ctx, cancel := ctxWithSignal()
	defer cancel()

	status := eng.Scan(ctx, g.profile)

	var aplicados, disponiveis, recomendadosPendentes, comErro int
	for _, s := range status {
		switch {
		case s.Err != nil:
			comErro++
		case s.State == tweak.StateApplied:
			aplicados++
		default:
			disponiveis++
			if s.Meta.RecommendedDefault {
				recomendadosPendentes++
			}
		}
	}

	fmt.Printf("Diagnóstico — perfil %s%s\n\n", g.profile, seNaoAdmin(eng.Elevated))
	w := newTab()
	fmt.Fprintln(w, "ESTADO\tNOME\tDETALHE")
	for _, s := range status {
		fmt.Fprintf(w, "%s\t%s\t%s\n", marcador(s), s.Meta.DisplayName, s.Detail)
	}
	if err := w.Flush(); err != nil {
		return err
	}

	fmt.Printf("\n%d de %d itens já estão como você quer.\n", aplicados, len(status))
	if recomendadosPendentes == 0 {
		fmt.Println("Nenhuma otimização recomendada pendente — está tudo certo por aqui.")
	} else {
		fmt.Printf("%d otimização(ões) recomendada(s) ainda não aplicada(s). Para simular: optimizerctl aplicar --recomendados --simular\n",
			recomendadosPendentes)
	}
	if comErro > 0 {
		fmt.Printf("%d item(ns) não puderam ser lidos nesta execução.\n", comErro)
	}
	return nil
}

func marcador(s engine.Status) string {
	if s.Err != nil {
		return "?"
	}
	switch s.State {
	case tweak.StateApplied:
		return "[x]"
	case tweak.StatePartial:
		return "[~]"
	default:
		return "[ ]"
	}
}

func seNaoAdmin(elevated bool) string {
	if elevated {
		return " (rodando como administrador)"
	}
	return " (sem privilégio de administrador — itens de máquina aparecem, mas não podem ser aplicados)"
}

// --------------------------------------------------------------- aplicar

func cmdAplicar(args []string) error {
	fs := flag.NewFlagSet("aplicar", flag.ExitOnError)
	g := addGlobals(fs)
	recomendados := fs.Bool("recomendados", false, "aplica os itens recomendados do perfil")
	pontoRestauracao := fs.Bool("ponto-de-restauracao", false,
		"cria ponto de restauração antes (exige administrador e Proteção do Sistema ligada)")
	ids, err := parseArgs(fs, g, args)
	if err != nil {
		return err
	}
	eng, err := openEngine(g)
	if err != nil {
		return err
	}
	if *recomendados {
		ids = append(ids, eng.Recommended(g.profile)...)
	}
	if len(ids) == 0 {
		return errors.New("diga o que aplicar: `optimizerctl aplicar <id...>` ou `--recomendados`")
	}
	// O ponto de restauração é aberto antes da primeira alteração real e
	// fechado depois do lote inteiro — nunca um ponto por item.
	var restoreSeq uint64
	if *pontoRestauracao && !g.dryRun {
		eng.CreateRestorePoint = func(desc string) (uint64, error) {
			seq, err := restore.Begin(desc)
			if err != nil {
				return 0, err
			}
			restoreSeq = seq
			fmt.Printf("Ponto de restauração criado (sequência %d).\n\n", seq)
			return seq, nil
		}
		defer func() {
			if restoreSeq != 0 {
				if err := restore.End(restoreSeq); err != nil {
					fmt.Fprintf(os.Stderr, "aviso: ponto de restauração não foi fechado: %v\n", err)
				}
			}
		}()
	}

	ctx, cancel := ctxWithSignal()
	defer cancel()

	fmt.Printf("Aplicando %d item(ns)%s\n\n", len(ids), seSimulacao(g.dryRun))
	return imprimirResultados(eng.Apply(ctx, ids, g.dryRun), g.dryRun)
}

// --------------------------------------------------------------- desfazer

func cmdDesfazer(args []string) error {
	fs := flag.NewFlagSet("desfazer", flag.ExitOnError)
	g := addGlobals(fs)
	tudo := fs.Bool("tudo", false, "desfaz tudo que este app alterou e ainda não foi desfeito")
	ids, err := parseArgs(fs, g, args)
	if err != nil {
		return err
	}
	eng, err := openEngine(g)
	if err != nil {
		return err
	}
	if *tudo {
		pend, err := eng.PendingIDs()
		if err != nil {
			return err
		}
		ids = append(ids, pend...)
	}
	if len(ids) == 0 {
		fmt.Println("Não há nada para desfazer.")
		return nil
	}

	ctx, cancel := ctxWithSignal()
	defer cancel()

	fmt.Printf("Desfazendo %d item(ns)%s\n\n", len(ids), seSimulacao(g.dryRun))
	return imprimirResultados(eng.Revert(ctx, ids, g.dryRun), g.dryRun)
}

func seSimulacao(dryRun bool) string {
	if dryRun {
		return " — SIMULAÇÃO, nada será gravado"
	}
	return ""
}

func imprimirResultados(results []engine.Result, dryRun bool) error {
	var ok, pulados, falhas int
	precisaReiniciar := false
	for _, r := range results {
		nome := r.Name
		if nome == "" {
			nome = r.ID
		}
		switch {
		case r.Err != nil:
			falhas++
			fmt.Printf("  falhou   %s\n           %v\n", nome, r.Err)
		case r.Skipped:
			pulados++
			fmt.Printf("  pulado   %s — %s\n", nome, r.Reason)
		default:
			ok++
			fmt.Printf("  ok       %s — %s\n", nome, r.Detail)
			if r.RestartNeeded {
				precisaReiniciar = true
			}
		}
	}

	fmt.Printf("\n%d aplicado(s), %d pulado(s), %d falha(s).\n", ok, pulados, falhas)
	if precisaReiniciar && !dryRun {
		fmt.Println("Alguns ajustes só aparecem depois que você sair e entrar de novo na sua conta do Windows.")
	}
	if falhas > 0 {
		return errors.New("nem tudo foi aplicado")
	}
	return nil
}

// -------------------------------------------------------------- historico

func cmdHistorico(args []string) error {
	fs := flag.NewFlagSet("historico", flag.ExitOnError)
	g := addGlobals(fs)
	limite := fs.Int("limite", 30, "quantas entradas mostrar (0 = todas)")
	if _, err := parseArgs(fs, g, args); err != nil {
		return err
	}
	eng, err := openEngine(g)
	if err != nil {
		return err
	}

	entradas, err := eng.History.All()
	if err != nil {
		return err
	}
	fmt.Printf("Histórico: %s\n\n", eng.History.Path())
	if len(entradas) == 0 {
		fmt.Println("Este app ainda não alterou nada nesta máquina.")
		return nil
	}
	if *limite > 0 && len(entradas) > *limite {
		entradas = entradas[len(entradas)-*limite:]
	}

	w := newTab()
	fmt.Fprintln(w, "QUANDO\tAÇÃO\tITEM\tRESULTADO")
	for _, e := range entradas {
		resultado := "ok"
		if !e.Success {
			resultado = "falhou: " + e.Error
		}
		acao := string(e.Action)
		if e.DryRun {
			acao += " (simulação)"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", e.At.Local().Format("02/01/2006 15:04"), acao, e.TweakID, resultado)
	}
	if err := w.Flush(); err != nil {
		return err
	}

	pend, err := eng.PendingIDs()
	if err == nil {
		fmt.Printf("\n%d alteração(ões) podem ser desfeitas agora (`optimizerctl desfazer --tudo`).\n", len(pend))
	}
	return nil
}

// -------------------------------------------------------------------- mtu

func cmdMTU(args []string) error {
	fs := flag.NewFlagSet("mtu", flag.ExitOnError)
	g := addGlobals(fs)
	destino := fs.String("destino", "8.8.8.8", "para onde medir (um servidor que responda ping)")
	detalhado := fs.Bool("detalhado", false, "mostra cada tentativa da medição")
	aplicar := fs.Bool("aplicar", false, "ajusta o MTU do adaptador para o valor medido")
	timeout := fs.Duration("tempo-limite", 1200*time.Millisecond, "espera máxima por resposta de cada teste")
	if _, err := parseArgs(fs, g, args); err != nil {
		return err
	}
	ctx, cancel := ctxWithSignal()
	defer cancel()

	addr, err := netdiag.Resolve(*destino)
	if err != nil {
		return err
	}

	ifaces, err := netdiag.Interfaces()
	if err != nil {
		return err
	}
	fmt.Println("Adaptadores de rede ativos:")
	w := newTab()
	fmt.Fprintln(w, "  ADAPTADOR\tTIPO\tMTU\tENDEREÇO")
	for _, i := range ifaces {
		if !i.Up || i.Type == netdiag.IfTypeLoopback {
			continue
		}
		ip := "—"
		if len(i.IPs) > 0 {
			ip = i.IPs[0].String()
		}
		fmt.Fprintf(w, "  %s\t%s\t%d\t%s\n", i.Name, i.KindLabel(), i.MTU, ip)
	}
	if err := w.Flush(); err != nil {
		return err
	}

	saida, err := netdiag.OutboundInterface(addr)
	if err != nil {
		fmt.Printf("\nAviso: %v\n", err)
	}

	fmt.Printf("\nMedindo o maior pacote que chega inteiro até %s", *destino)
	if saida != nil {
		fmt.Printf(" (saindo por %q)", saida.Name)
	}
	fmt.Println("...")
	fmt.Println("Método: eco ICMP com \"não fragmentar\" ligado — o mesmo que `ping -f -l <tamanho>` faz.")

	rep, err := netdiag.ProbePathMTU(ctx, netdiag.LiveProber{}, *destino, addr,
		netdiag.Options{Timeout: *timeout})
	if err != nil {
		return err
	}

	if *detalhado {
		fmt.Printf("\n%d tentativa(s):\n", len(rep.Steps))
		tw := newTab()
		fmt.Fprintln(tw, "  TAMANHO\tMTU EQUIVALENTE\tRESULTADO\tCOMANDO EQUIVALENTE")
		for _, s := range rep.Steps {
			fmt.Fprintf(tw, "  %d\t%d\t%s\t%s\n", s.Payload, s.MTU, s.Status, s.Command)
		}
		if err := tw.Flush(); err != nil {
			return err
		}
	}

	diag := netdiag.Diagnose(rep, saida)

	fmt.Println("\n" + strings.Repeat("─", 72))
	fmt.Printf("%s\n\n%s\n", diag.Summary, quebrar(diag.Explanation, 72))

	if rep.CommandOK != "" {
		fmt.Println("\nConfira você mesmo no Prompt de Comando:")
		fmt.Printf("  %-34s → responde normalmente\n", rep.CommandOK)
		if rep.CommandFail != "" {
			fmt.Printf("  %-34s → \"Pacote precisa ser fragmentado, mas o sinalizador DF está definido.\"\n", rep.CommandFail)
		}
		fmt.Println("  (o número depois do -l é o tamanho dos dados; o MTU é esse número + 28)")
	}

	if diag.SuggestedMTU == 0 || saida == nil {
		fmt.Println("\nNada a ajustar.")
		return nil
	}

	fmt.Printf("\nAjuste sugerido: MTU de %q → %d\n", saida.Name, diag.SuggestedMTU)
	if diag.Verdict == netdiag.VerdictLowerOptional {
		fmt.Println("Este ajuste é OPCIONAL: a medição mostrou que o Windows já está se adaptando sozinho neste caminho.")
	}
	if saida.LikelyTunnel() {
		fmt.Println("Atenção: este adaptador é de VPN/PPPoE. O valor menor pode ser proposital — confirme antes de mudar.")
	}
	if !*aplicar {
		fmt.Println("Para aplicar (com desfazer registrado no histórico): optimizerctl mtu --aplicar")
		return nil
	}

	mtuTweak, err := netdiag.NewMTUTweak(*saida, rep, diag, netdiag.LiveMTU{})
	if err != nil {
		return err
	}
	eng, err := openEngine(g)
	if err != nil {
		return err
	}
	if !eng.Elevated && !g.dryRun {
		return errors.New("alterar o MTU exige rodar como administrador")
	}
	eng.Registry.Register(mtuTweak, tweak.Meta{
		Enabled:       true,
		Profiles:      tweak.ProfileBoth,
		RequiresAdmin: true,
		Evidence:      "docs/catalogo/rede.md#5-mtu",
	})

	fmt.Println()
	return imprimirResultados(eng.Apply(ctx, []string{mtuTweak.ID()}, g.dryRun), g.dryRun)
}

// quebrar reflui o texto para caber na largura do terminal.
func quebrar(s string, largura int) string {
	palavras := strings.Fields(s)
	if len(palavras) == 0 {
		return ""
	}
	var b strings.Builder
	linha := 0
	for i, p := range palavras {
		if linha > 0 && linha+1+len(p) > largura {
			b.WriteString("\n")
			linha = 0
		} else if i > 0 {
			b.WriteString(" ")
			linha++
		}
		b.WriteString(p)
		linha += len(p)
	}
	return b.String()
}

func cmdPerfil(args []string) error {
	fs := flag.NewFlagSet("perfil", flag.ContinueOnError)
	g := addGlobals(fs)
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, `Uso: optimizerctl perfil <subcomando> [argumentos]

Subcomandos:
  listar                  Lista todos os perfis disponíveis (JOGO, CODING e perfis de rede)
  aplicar <jogo|coding>   Aplica um perfil inteiro com transação atômica
  verificar <jogo|coding> Verifica a integridade dos ajustes do perfil
  restaurar <jogo|coding> Restaura o estado anterior ao perfil ativo

Exemplos:
  optimizerctl perfil listar
  optimizerctl perfil aplicar jogo --simular
  optimizerctl perfil verificar coding
  optimizerctl perfil restaurar jogo
`)
	}
	soltos, err := parseArgs(fs, g, args)
	if err != nil {
		return err
	}

	if len(soltos) == 0 {
		return cmdPerfilListar()
	}

	subcmd := soltos[0]
	switch subcmd {
	case "listar":
		return cmdPerfilListar()
	case "aplicar":
		if len(soltos) < 2 {
			return fmt.Errorf("aplicar requer um nome de perfil (ex: jogo, coding, network-fast)")
		}
		return cmdPerfilAplicar(g, soltos[1])
	case "verificar":
		if len(soltos) < 2 {
			return fmt.Errorf("verificar requer um nome de perfil (ex: jogo, coding)")
		}
		return cmdPerfilVerificar(g, soltos[1])
	case "restaurar":
		if len(soltos) < 2 {
			return fmt.Errorf("restaurar requer o nome do perfil ativo a reverter (ex: jogo, coding)")
		}
		return cmdPerfilRestaurar(g, soltos[1])
	default:
		return fmt.Errorf("subcomando desconhecido: %s", subcmd)
	}
}

func cmdPerfilListar() error {
	fmt.Println("=== Perfis de Uso do Computador (private-optimizer) ===")
	fmt.Println()
	wUso := newTab()
	fmt.Fprintln(wUso, "CHAVE\tNOME\tTWEAKS\tOBJETIVO")
	for _, p := range profiles.ListarPerfisUso() {
		fmt.Fprintf(wUso, "%s\t%s\t%d\t%s\n", p.Key, p.Nome, len(p.TweakIDs), p.Objetivo)
	}
	_ = wUso.Flush()

	fmt.Println()
	fmt.Println("=== Perfis Granulares de Rede ===")
	fmt.Println()
	w := newTab()
	fmt.Fprintln(w, "CHAVE\tNOME\tTWEAKS\tDESCRIÇÃO")
	for _, p := range profiles.List() {
		fmt.Fprintf(w, "%s\t%s\t%d\t%s\n", p.Key, p.Name, len(p.TweakIDs), p.Description)
	}
	_ = w.Flush()

	fmt.Println("\nPara aplicar: optimizerctl perfil aplicar <chave> [--simular]")
	return nil
}

func cmdPerfilAplicar(g *globals, profileKey string) error {
	// 1. Verifica se é Perfil de Uso (JOGO / CODING)
	if up, ok := profiles.ObterPerfilUso(profileKey); ok {
		eng, err := openEngine(g)
		if err != nil {
			return err
		}
		ctx, cancel := ctxWithSignal()
		defer cancel()

		fmt.Printf("Aplicando Perfil de Uso %q (%s) — %d item(ns)%s\n", up.Nome, up.Key, len(up.TweakIDs), seSimulacao(g.dryRun))
		if up.Ressalvas != "" {
			fmt.Printf("Ressalva: %s\n", up.Ressalvas)
		}
		fmt.Println()

		batchID := fmt.Sprintf("batch-%s-%d", up.Key, time.Now().Unix())
		origin := "perfil-" + up.Key
		return imprimirResultados(eng.ApplyBatch(ctx, up.TweakIDs, g.dryRun, origin, batchID), g.dryRun)
	}

	// 2. Fallback para Perfis de Rede
	p, ok := profiles.Get(profileKey)
	if !ok {
		return fmt.Errorf("perfil desconhecido: %q — rode `optimizerctl perfil listar`", profileKey)
	}

	eng, err := openEngine(g)
	if err != nil {
		return err
	}

	ctx, cancel := ctxWithSignal()
	defer cancel()

	fmt.Printf("Aplicando perfil de rede %q (%s) — %d item(ns)%s\n", p.Name, p.Key, len(p.TweakIDs), seSimulacao(g.dryRun))
	if p.Caveats != "" {
		fmt.Printf("Ressalva: %s\n", p.Caveats)
	}
	fmt.Println()

	return imprimirResultados(eng.Apply(ctx, p.TweakIDs, g.dryRun), g.dryRun)
}

func cmdPerfilVerificar(g *globals, profileKey string) error {
	up, ok := profiles.ObterPerfilUso(profileKey)
	if !ok {
		return fmt.Errorf("perfil de uso desconhecido: %q", profileKey)
	}

	eng, err := openEngine(g)
	if err != nil {
		return err
	}

	ctx, cancel := ctxWithSignal()
	defer cancel()

	fmt.Printf("Verificando integridade do perfil %q (%s):\n\n", up.Nome, up.Key)
	w := newTab()
	fmt.Fprintln(w, "ESTADO\tTWEAK\tDETALHE")
	for _, id := range up.TweakIDs {
		t, ok := eng.Registry.Known(id)
		if !ok {
			continue
		}
		meta, _ := eng.Registry.Meta(id)
		st, _ := t.Check(ctx)
		statusMark := "[ ]"
		if st.State == tweak.StateApplied {
			statusMark = "[x]"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", statusMark, meta.DisplayName, st.Detail)
	}
	return w.Flush()
}

func cmdPerfilRestaurar(g *globals, profileKey string) error {
	eng, err := openEngine(g)
	if err != nil {
		return err
	}

	ctx, cancel := ctxWithSignal()
	defer cancel()

	batchID, entries, _ := eng.History.ActiveBatchForOrigin("perfil-" + profileKey)
	if batchID == "" || len(entries) == 0 {
		fmt.Printf("Nenhuma alteração ativa do perfil %q para restaurar.\n", profileKey)
		return nil
	}

	fmt.Printf("Restaurando %d item(ns) do lote %s...\n\n", len(entries), batchID)
	return imprimirResultados(eng.RevertBatch(ctx, batchID, g.dryRun), g.dryRun)
}

func cmdRede(args []string) error {
	fs := flag.NewFlagSet("rede", flag.ContinueOnError)
	g := addGlobals(fs)
	destino := fs.String("destino", "8.8.8.8", "host para medir")
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, `Uso: optimizerctl rede <subcomando> [opções]

Subcomandos:
  medir            Mede latência de ping a um host
  diag             Diagnóstico completo de rede (ping + performance counters)

Opções:
  --destino <host> Host para medir (padrão: 8.8.8.8)

Exemplos:
  optimizerctl rede medir
  optimizerctl rede medir --destino 1.1.1.1
`)
	}
	soltos, err := parseArgs(fs, g, args)
	if err != nil {
		return err
	}

	if len(soltos) == 0 {
		fs.Usage()
		return nil
	}

	subcmd := soltos[0]
	switch subcmd {
	case "medir":
		return cmdRedeMedir(*destino)
	case "diag":
		return cmdRedeDiag(*destino)
	default:
		return fmt.Errorf("subcomando desconhecido: %s", subcmd)
	}
}

func cmdRedeMedir(destino string) error {
	ctx, cancel := ctxWithSignal()
	defer cancel()

	fmt.Printf("Medindo latência até %s (20 pacotes)...\n\n", destino)

	report, err := netdiag.MeasureLatency(ctx, destino, 20)
	if err != nil {
		return fmt.Errorf("erro na medição: %w", err)
	}

	fmt.Printf("Host:              %s\n", report.Host)
	fmt.Printf("Latência mínima:   %d ms\n", report.MinRTT)
	fmt.Printf("Latência média:    %d ms\n", report.AvgRTT)
	fmt.Printf("Latência máxima:   %d ms\n", report.MaxRTT)
	fmt.Printf("Jitter (stddev):   %d ms\n", report.StdDev)
	fmt.Printf("Pacotes enviados:  %d\n", report.PacketsSent)
	fmt.Printf("Pacotes perdidos:  %d\n", report.PacketsLost)
	if report.PacketsSent > 0 {
		lossPercent := float64(report.PacketsLost) / float64(report.PacketsSent) * 100
		fmt.Printf("Perda:             %.1f%%\n", lossPercent)
	}
	fmt.Println("\nNota: esta é uma medição isolada. Para comparar antes/depois de um ajuste, meça duas vezes e use a interpretação honesta (diferenças <5% costumam ser ruído).")

	return nil
}

func cmdRedeDiag(destino string) error {
	ctx, cancel := ctxWithSignal()
	defer cancel()

	fmt.Printf("Diagnóstico completo de rede — %s\n\n", destino)

	report, err := netdiag.MeasureLatency(ctx, destino, 20)
	if err != nil {
		return fmt.Errorf("erro na medição de latência: %w", err)
	}

	fmt.Println("== Latência ==")
	fmt.Printf("Média: %d ms · Min: %d ms · Max: %d ms · Jitter: %d ms\n\n",
		report.AvgRTT, report.MinRTT, report.MaxRTT, report.StdDev)

	fmt.Println("== Interpretação ==")
	switch {
	case report.PacketsLost > 0:
		lossPercent := float64(report.PacketsLost) / float64(report.PacketsSent) * 100
		fmt.Printf("Perda de %.1f%% detectada — pode indicar rede instável, Wi-Fi fraco ou congestionamento.\n", lossPercent)
	case report.StdDev > report.AvgRTT/2:
		fmt.Println("Jitter alto em relação à latência média — conexão pode estar instável para chamadas/streaming.")
	default:
		fmt.Println("Latência e jitter dentro do esperado para uso comum.")
	}

	fmt.Println("\nPara aplicar um perfil de otimização: optimizerctl perfil listar")
	return nil
}
