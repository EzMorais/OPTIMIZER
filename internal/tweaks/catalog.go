// Package tweaks monta o catálogo padrão embutido no binário.
//
// Todo item aqui tem lastro no catálogo técnico em docs/catalogo/ — chave,
// tipo, valor padrão do Windows e veredito. RecommendedDefault só é true onde
// as duas pesquisas independentes convergiram para ganho real com risco baixo;
// o resto fica disponível, mas desmarcado, com a ressalva escrita no Caveat.
// É o oposto do padrão da categoria (marcar tudo para inflar "problemas").
package tweaks

import (
	"strconv"

	"optimizer/internal/tweak"
	"optimizer/internal/tweaks/regtweak"
	"optimizer/internal/winreg"
)

type entry struct {
	spec   regtweak.Spec
	values []regtweak.Value
	meta   tweak.Meta
}

// Build monta o registro do catálogo sobre um backend de registro (real ou fake).
func Build(b winreg.Backend) *tweak.Registry {
	r := tweak.NewRegistry()
	for _, e := range entries() {
		impl := regtweak.Tweak{Spec: e.spec, Backend: b, Values: e.values}
		m := e.meta
		m.Enabled = true
		m.DisplayName = e.spec.DisplayName
		m.Description = e.spec.Explanation
		r.Register(impl, m)
	}
	return r
}

// atMost devolve um predicado "o valor de string atual já é numericamente
// menor ou igual ao alvo" — usado para nunca piorar um ajuste que o usuário já
// deixou melhor do que o nosso alvo.
func atMost(limit int) func(any) bool {
	return func(current any) bool {
		s, ok := current.(string)
		if !ok {
			return false
		}
		n, err := strconv.Atoi(s)
		return err == nil && n <= limit
	}
}

const (
	pathDesktop       = `Control Panel\Desktop`
	pathWindowMetrics = `Control Panel\Desktop\WindowMetrics`
	pathMouse         = `Control Panel\Mouse`
	pathPersonalize   = `Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`
	pathExplorerAdv   = `Software\Microsoft\Windows\CurrentVersion\Explorer\Advanced`
	pathGameConfig    = `System\GameConfigStore`
	pathMMCSS         = `SOFTWARE\Microsoft\Windows NT\CurrentVersion\Multimedia\SystemProfile`
	pathDeliveryOpt   = `SOFTWARE\Policies\Microsoft\Windows\DeliveryOptimization`
	pathGameDVRPolicy = `SOFTWARE\Policies\Microsoft\Windows\GameDVR`
)

func entries() []entry {
	return []entry{
		{
			spec: regtweak.Spec{
				TweakID:     "visual.menu-show-delay",
				DisplayName: "Abrir menus na hora",
				Explanation: "De fábrica, o Windows espera cerca de 0,4 segundo antes de abrir um submenu quando você " +
					"passa o mouse. Este ajuste reduz essa espera para quase zero.",
				Cat:            tweak.CategoryVisualEffects,
				RiskLevel:      tweak.RiskLow,
				NeedsRestart:   true,
				AppliedText:    "Os menus já abrem na hora.",
				NotAppliedText: "Os menus ainda esperam um tempo antes de abrir.",
			},
			values: []regtweak.Value{
				withSatisfied(regtweak.String(winreg.HKCU, pathDesktop, "MenuShowDelay", "10", "400"), atMost(10)),
			},
			meta: tweak.Meta{
				RecommendedDefault: true,
				Profiles:           tweak.ProfileBoth,
				Caveat:             "Vale para os menus clássicos do Windows; menus de aplicativos novos (estilo Windows 11) não mudam. O efeito completo aparece depois de sair e entrar de novo na sua conta.",
				Evidence:           "docs/catalogo/registro.md#menushowdelay",
				SortOrder:          10,
			},
		},
		{
			spec: regtweak.Spec{
				TweakID:     "jogos.gamedvr-usuario",
				DisplayName: "Desligar a gravação de jogos em segundo plano",
				Explanation: "O Windows mantém um gravador de jogos rodando por trás mesmo quando você não está gravando nada. " +
					"Desligar libera CPU e placa de vídeo durante o jogo.",
				Cat:            tweak.CategoryGaming,
				RiskLevel:      tweak.RiskLow,
				NeedsRestart:   false,
				AppliedText:    "A gravação em segundo plano está desligada.",
				NotAppliedText: "A gravação em segundo plano está ligada.",
			},
			values: []regtweak.Value{
				regtweak.DWord(winreg.HKCU, pathGameConfig, "GameDVR_Enabled", 0, 1),
			},
			meta: tweak.Meta{
				RecommendedDefault: true,
				Profiles:           tweak.ProfileBoth,
				Caveat:             "Você continua conseguindo gravar apertando o atalho da Barra de Jogos — o que sai é só a gravação automática por trás.",
				Evidence:           "docs/catalogo/registro.md#xbox-game-bar--game-dvr",
				SortOrder:          20,
			},
		},
		{
			spec: regtweak.Spec{
				TweakID:     "visual.transparency-effects",
				DisplayName: "Efeitos de transparência",
				Explanation: "Desliga o efeito de vidro fosco da barra de tarefas e dos menus.",
				Cat:         tweak.CategoryVisualEffects,
				RiskLevel:   tweak.RiskLow,

				AppliedText:    "A transparência está desligada.",
				NotAppliedText: "A transparência está ligada.",
			},
			values: []regtweak.Value{
				regtweak.DWord(winreg.HKCU, pathPersonalize, "EnableTransparency", 0, 1),
			},
			meta: tweak.Meta{
				RecommendedDefault: false,
				Profiles:           tweak.ProfileBoth,
				Caveat:             "Em qualquer PC comprado nos últimos 8 a 10 anos, o ganho de velocidade é praticamente zero — a diferença aparece em máquina antiga, com vídeo integrado fraco. Deixamos disponível por gosto pessoal, não como promessa de desempenho.",
				Evidence:           "docs/catalogo/registro.md#animações--4-chaves-independentes-toggles-separáveis",
				SortOrder:          30,
			},
		},
		{
			spec: regtweak.Spec{
				TweakID:        "visual.min-animate",
				DisplayName:    "Animação de minimizar e maximizar janelas",
				Explanation:    "Desliga a animação que o Windows faz ao minimizar ou maximizar uma janela.",
				Cat:            tweak.CategoryVisualEffects,
				RiskLevel:      tweak.RiskLow,
				NeedsRestart:   true,
				AppliedText:    "A animação de janelas está desligada.",
				NotAppliedText: "A animação de janelas está ligada.",
			},
			values: []regtweak.Value{
				regtweak.String(winreg.HKCU, pathWindowMetrics, "MinAnimate", "0", "1"),
			},
			meta: tweak.Meta{
				RecommendedDefault: false,
				Profiles:           tweak.ProfileBoth,
				Caveat:             "É puramente estético. O ganho de desempenho real é desprezível em hardware moderno — a sensação de rapidez vem de a janela aparecer sem o deslizamento.",
				Evidence:           "docs/catalogo/registro.md#animações--4-chaves-independentes-toggles-separáveis",
				SortOrder:          40,
			},
		},
		{
			spec: regtweak.Spec{
				TweakID:        "visual.taskbar-animations",
				DisplayName:    "Animações da barra de tarefas",
				Explanation:    "Desliga as animações dos botões da barra de tarefas.",
				Cat:            tweak.CategoryVisualEffects,
				RiskLevel:      tweak.RiskLow,
				NeedsRestart:   true,
				AppliedText:    "As animações da barra de tarefas estão desligadas.",
				NotAppliedText: "As animações da barra de tarefas estão ligadas.",
			},
			values: []regtweak.Value{
				regtweak.DWord(winreg.HKCU, pathExplorerAdv, "TaskbarAnimations", 0, 1),
			},
			meta: tweak.Meta{
				RecommendedDefault: false,
				Profiles:           tweak.ProfileBoth,
				Caveat:             "No Windows 11 a barra de tarefas foi reescrita e não está confirmado que esta chave ainda controla todas as animações dela — no Windows 10 o efeito é completo.",
				Evidence:           "docs/catalogo/registro.md#animações--4-chaves-independentes-toggles-separáveis",
				SortOrder:          50,
			},
		},
		{
			spec: regtweak.Spec{
				TweakID:     "entrada.mouse-sem-aceleracao",
				DisplayName: "Mouse sem aceleração (movimento 1:1)",
				Explanation: "Desliga o \"Aprimorar precisão do ponteiro\". Com isso, a mesma distância de mouse na mesa " +
					"sempre resulta na mesma distância na tela, independente da velocidade do movimento.",
				Cat:            tweak.CategoryInput,
				RiskLevel:      tweak.RiskLow,
				NeedsRestart:   true,
				AppliedText:    "O mouse está no modo 1:1, sem aceleração.",
				NotAppliedText: "A aceleração do ponteiro está ligada.",
			},
			values: []regtweak.Value{
				regtweak.String(winreg.HKCU, pathMouse, "MouseSpeed", "0", "1"),
				regtweak.String(winreg.HKCU, pathMouse, "MouseThreshold1", "0", "6"),
				regtweak.String(winreg.HKCU, pathMouse, "MouseThreshold2", "0", "10"),
			},
			meta: tweak.Meta{
				RecommendedDefault: false,
				Profiles:           tweak.ProfilePersonal,
				Caveat:             "Não é otimização de velocidade do PC: é preferência. Jogadores costumam preferir 1:1; quem usa o mouse para trabalho de escritório muitas vezes prefere a aceleração ligada. Experimente e desfaça se não gostar.",
				Evidence:           "docs/catalogo/registro.md#latência-de-entrada-mouse-e-teclado",
				SortOrder:          60,
			},
		},
		{
			spec: regtweak.Spec{
				TweakID:     "rede.sem-upload-de-atualizacoes",
				DisplayName: "Não usar seu upload para distribuir atualizações a outros PCs",
				Explanation: "Por padrão o Windows envia pedaços das atualizações que já baixou para outros computadores. " +
					"Este ajuste desliga esse envio e faz o download vir só dos servidores da Microsoft.",
				Cat:            tweak.CategoryNetwork,
				RiskLevel:      tweak.RiskLow,
				NeedsRestart:   false,
				AppliedText:    "Seu upload não é mais usado para distribuir atualizações.",
				NotAppliedText: "O Windows pode usar seu upload para enviar atualizações a outros PCs.",
			},
			values: []regtweak.Value{
				regtweak.DWord(winreg.HKLM, pathDeliveryOpt, "DODownloadMode", 0, 1),
			},
			meta: tweak.Meta{
				RecommendedDefault: false,
				Profiles:           tweak.ProfileBoth,
				RequiresAdmin:      true,
				Caveat:             "Vale a pena em internet com franquia limitada, 4G/5G ou conexão compartilhada. Em banda larga folgada, a diferença no dia a dia é pequena — e desligar faz cada PC da casa baixar a mesma atualização de novo.",
				Evidence:           "docs/catalogo/rede.md#10-delivery-optimization",
				SortOrder:          70,
			},
		},
		{
			spec: regtweak.Spec{
				TweakID:     "sistema.mmcss-system-responsiveness",
				DisplayName: "Reservar menos CPU para tarefas de multimídia",
				Explanation: "O Windows reserva uma fatia de CPU para áudio e vídeo não engasgarem. Este ajuste reduz essa " +
					"reserva de 20% para 10%, deixando mais CPU para o programa em primeiro plano.",
				Cat:            tweak.CategorySystem,
				RiskLevel:      tweak.RiskMedium,
				NeedsRestart:   true,
				AppliedText:    "A reserva de multimídia está no mínimo aceito pelo Windows (10).",
				NotAppliedText: "A reserva de multimídia não está no valor mínimo aceito pelo Windows.",
			},
			values: []regtweak.Value{
				regtweak.DWord(winreg.HKLM, pathMMCSS, "SystemResponsiveness", 10, 20),
			},
			meta: tweak.Meta{
				RecommendedDefault: false,
				Profiles:           tweak.ProfilePersonal,
				RequiresAdmin:      true,
				Caveat: "Quase todo guia da internet manda colocar 0 aqui. A documentação da própria Microsoft diz que valores abaixo de 10 são forçados de volta para 20 — ou seja, colocar 0 faz o contrário do que promete. " +
					"Por isso usamos 10, o menor valor que o Windows realmente aceita. O ganho é pequeno e pode causar engasgo em áudio durante carga pesada.",
				Evidence:  "docs/catalogo/registro.md#systemresponsiveness",
				SortOrder: 80,
			},
		},
		{
			spec: regtweak.Spec{
				TweakID:     "rede.network-throttling-off",
				DisplayName: "Tirar o limite de pacotes de rede durante multimídia",
				Explanation: "Quando há áudio ou vídeo tocando, o Windows limita a rede a cerca de 10 mil pacotes por segundo. " +
					"Este ajuste remove esse limite.",
				Cat:            tweak.CategoryNetwork,
				RiskLevel:      tweak.RiskMedium,
				NeedsRestart:   true,
				AppliedText:    "O limite de pacotes durante multimídia está desligado.",
				NotAppliedText: "O limite de pacotes durante multimídia está ativo.",
			},
			values: []regtweak.Value{
				regtweak.DWord(winreg.HKLM, pathMMCSS, "NetworkThrottlingIndex", 0xFFFFFFFF, 10),
			},
			meta: tweak.Meta{
				RecommendedDefault: false,
				Profiles:           tweak.ProfilePersonal,
				RequiresAdmin:      true,
				Caveat: "Só faz diferença perceptível em rede gigabit com tráfego pesado ao mesmo tempo que áudio/vídeo (produção de mídia, transferências grandes durante chamada). " +
					"Em uso comum, o efeito é nenhum. Este mecanismo saiu da documentação ativa da Microsoft — está documentado só em material arquivado.",
				Evidence:  "docs/catalogo/registro.md#networkthrottlingindex",
				SortOrder: 90,
			},
		},
		{
			spec: regtweak.Spec{
				TweakID:     "jogos.gamedvr-politica-maquina",
				DisplayName: "Bloquear a gravação de jogos para todos os usuários do PC",
				Explanation: "Aplica, para o computador inteiro, a política que impede a gravação de jogos em segundo plano — " +
					"inclusive para contas criadas depois.",
				Cat:            tweak.CategoryGaming,
				RiskLevel:      tweak.RiskLow,
				NeedsRestart:   true,
				AppliedText:    "A gravação de jogos está bloqueada para o computador inteiro.",
				NotAppliedText: "Não há política de máquina bloqueando a gravação de jogos.",
			},
			values: []regtweak.Value{
				regtweak.DWord(winreg.HKLM, pathGameDVRPolicy, "AllowGameDVR", 0, 1),
			},
			meta: tweak.Meta{
				RecommendedDefault: false,
				Profiles:           tweak.ProfileWork,
				RequiresAdmin:      true,
				Caveat:             "Pensado para PC de trabalho ou compartilhado. Em PC pessoal de quem joga, prefira o ajuste por usuário — este aqui vale para todo mundo que usa a máquina.",
				Evidence:           "docs/catalogo/registro.md#xbox-game-bar--game-dvr",
				SortOrder:          100,
			},
		},
	}
}

func withSatisfied(v regtweak.Value, f func(any) bool) regtweak.Value {
	v.AlreadyOptimized = f
	return v
}
