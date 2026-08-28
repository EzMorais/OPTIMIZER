// Package tweaks monta o catálogo padrão embutido no binário.
//
// Todo item aqui tem lastro no catálogo técnico em docs/catalogo/ — chave,
// tipo, valor padrão do Windows e veredito. RecommendedDefault só é true onde
// as pesquisas convergiram para ganho real com risco baixo; o resto fica
// disponível, mas desmarcado, com a ressalva escrita no Caveat.
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
	pathDesktop             = `Control Panel\Desktop`
	pathWindowMetrics       = `Control Panel\Desktop\WindowMetrics`
	pathMouse               = `Control Panel\Mouse`
	pathKeyboard            = `Control Panel\Keyboard`
	pathPersonalize         = `Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`
	pathExplorerAdv         = `Software\Microsoft\Windows\CurrentVersion\Explorer\Advanced`
	pathGameConfig          = `System\GameConfigStore`
	pathMMCSS               = `SOFTWARE\Microsoft\Windows NT\CurrentVersion\Multimedia\SystemProfile`
	pathMMCSSLowLatency     = `SOFTWARE\Microsoft\Windows NT\CurrentVersion\Multimedia\SystemProfile\Tasks\Low Latency`
	pathDeliveryOpt         = `SOFTWARE\Policies\Microsoft\Windows\DeliveryOptimization`
	pathGameDVRPolicy       = `SOFTWARE\Policies\Microsoft\Windows\GameDVR`
	pathTcpip               = `SYSTEM\CurrentControlSet\Services\Tcpip\Parameters`
	pathGraphicsDrivers     = `SYSTEM\CurrentControlSet\Control\GraphicsDrivers`
	pathGameBar             = `Software\Microsoft\GameBar`
	pathFileSystem          = `SYSTEM\CurrentControlSet\Control\FileSystem`
	pathSearchPolicy        = `SOFTWARE\Policies\Microsoft\Windows\Windows Search`
	pathExplorerPolicy      = `Software\Policies\Microsoft\Windows\Explorer`
	pathAccessibilitySticky = `Control Panel\Accessibility\StickyKeys`
	pathAccessibilityToggle = `Control Panel\Accessibility\ToggleKeys`
	pathAccessibilityFilter = `Control Panel\Accessibility\Keyboard Response`
	pathBackgroundAccess    = `Software\Microsoft\Windows\CurrentVersion\BackgroundAccessApplications`
	pathStorageSense        = `Software\Microsoft\Windows\CurrentVersion\StorageSense\Parameters\StoragePolicy`
	pathPowerThrottling     = `SYSTEM\CurrentControlSet\Control\Power\PowerThrottling`
	pathPowerSession        = `SYSTEM\CurrentControlSet\Control\Session Manager\Power`
	pathMemMgmt             = `SYSTEM\CurrentControlSet\Control\Session Manager\Memory Management`
	pathSessionManager      = `SYSTEM\CurrentControlSet\Control`
	pathMouClass            = `SYSTEM\CurrentControlSet\Services\mouclass\Parameters`
	pathKbdClass            = `SYSTEM\CurrentControlSet\Services\kbdclass\Parameters`
	pathDXGKrnl             = `SYSTEM\CurrentControlSet\Services\DXGKrnl`
	pathGamesLegacy         = `Software\Microsoft\Games`
	pathUsb                 = `SYSTEM\CurrentControlSet\Services\USB`
	pathSettingSync         = `Software\Microsoft\Windows\CurrentVersion\SettingSync\Groups\Personalization`
	pathContentDelivery     = `Software\Microsoft\Windows\CurrentVersion\ContentDeliveryManager`
	pathAdvInfo             = `Software\Microsoft\Windows\CurrentVersion\AdvertisingInfo`
	pathErrorReporting      = `SOFTWARE\Policies\Microsoft\Windows\Windows Error Reporting`
	pathNvidiaClassPrimary  = `SYSTEM\CurrentControlSet\Control\Class\{4d36e968-e325-11ce-bfc1-08002be10318}\0000`
	pathNvidiaNvTweakGlobal = `SOFTWARE\NVIDIA Corporation\Global\NVTweak`
	pathNvidiaNvCplOptIn    = `SOFTWARE\NVIDIA Corporation\NvControlPanel2\Client`
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
				TweakID:        "armazenamento.trim-ntfs",
				DisplayName:    "Ativar TRIM para SSDs (manutenção de velocidade e vida útil)",
				Explanation:    "Garante que o comando TRIM esteja habilitado no sistema de arquivos, informando ao SSD quais blocos de dados não estão mais em uso para manter alta velocidade de gravação contínua.",
				Cat:            tweak.CategoryStorage,
				RiskLevel:      tweak.RiskLow,
				NeedsRestart:   false,
				AppliedText:    "O TRIM está ativado para SSDs.",
				NotAppliedText: "O TRIM está desativado no sistema de arquivos.",
			},
			values: []regtweak.Value{
				regtweak.DWord(winreg.HKLM, pathFileSystem, "DisableDeleteNotify", 0, 0),
			},
			meta: tweak.Meta{
				RecommendedDefault: true,
				Profiles:           tweak.ProfileBoth,
				RequiresAdmin:      true,
				Caveat:             "O padrão de fábrica do Windows é ativado (0). Se algum software desativou (1), este ajuste restaura a saúde e desempenho do seu SSD sem nenhum risco.",
				Evidence:           "docs/catalogo/hardware-energia-gpu.md#7-trim-e-otimizar-unidades",
				SortOrder:          15,
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
				TweakID:        "jogos.game-mode",
				DisplayName:    "Modo de Jogo do Windows",
				Explanation:    "Garante que o Modo de Jogo esteja ativo para priorizar recursos de CPU e suprimir notificações e tarefas pesadas de fundo durante os jogos.",
				Cat:            tweak.CategoryGaming,
				RiskLevel:      tweak.RiskLow,
				NeedsRestart:   false,
				AppliedText:    "O Modo de Jogo está ativado.",
				NotAppliedText: "O Modo de Jogo está desativado.",
			},
			values: []regtweak.Value{
				regtweak.DWord(winreg.HKCU, pathGameBar, "AutoGameModeEnabled", 1, 1),
			},
			meta: tweak.Meta{
				RecommendedDefault: false,
				Profiles:           tweak.ProfileBoth,
				Caveat:             "O Windows já ativa o Modo de Jogo de fábrica na maioria das instalações. Este ajuste assegura que ele não foi desativado acidentalmente por outros otimizadores.",
				Evidence:           "docs/catalogo/hardware-energia-gpu.md#6-modo-de-jogo-game-mode",
				SortOrder:          22,
			},
		},
		{
			spec: regtweak.Spec{
				TweakID:        "visual.hags",
				DisplayName:    "Agendamento de GPU acelerado por hardware (HAGS)",
				Explanation:    "Permite que a placa de vídeo gerencie sua própria memória de vídeo diretamente, reduzindo latência em jogos e habilitando tecnologias como DLSS Frame Generation.",
				Cat:            tweak.CategoryVisualEffects,
				RiskLevel:      tweak.RiskLow,
				NeedsRestart:   true,
				AppliedText:    "O agendamento de GPU acelerado por hardware está ativado.",
				NotAppliedText: "O agendamento de GPU acelerado por hardware não está forçado como ativo.",
			},
			values: []regtweak.Value{
				regtweak.DWord(winreg.HKLM, pathGraphicsDrivers, "HwSchMode", 2, 2),
			},
			meta: tweak.Meta{
				RecommendedDefault: false,
				Profiles:           tweak.ProfileBoth,
				RequiresAdmin:      true,
				Caveat:             "Em placas modernas (RTX 40-series+), é pré-requisito para Frame Generation. Em softwares de edição profissional (ex: Premiere/CUDA), o ganho é variável. Exige reiniciar o PC para surtir efeito.",
				Evidence:           "docs/catalogo/hardware-energia-gpu.md#5-hardware-accelerated-gpu-scheduling-hags",
				SortOrder:          24,
			},
		},
		{
			spec: regtweak.Spec{
				TweakID:        "visual.anim-controles",
				DisplayName:    "Desativar animações de controles e elementos dentro das janelas",
				Explanation:    "Remove animações e transições internas de caixas de diálogo, listas e controles de janelas para respostas visuais instantâneas.",
				Cat:            tweak.CategoryVisualEffects,
				RiskLevel:      tweak.RiskLow,
				NeedsRestart:   true,
				AppliedText:    "As animações internas de controles estão desativadas.",
				NotAppliedText: "As animações internas de controles estão ativas.",
			},
			values: []regtweak.Value{
				regtweak.String(winreg.HKCU, pathWindowMetrics, "MinAnimate", "0", "1"),
			},
			meta: tweak.Meta{
				RecommendedDefault: false,
				Profiles:           tweak.ProfilePersonal,
				Caveat:             "Ajuste estético/responsividade. Elimina atrasos de animação gráfica em menus e controles clássicos.",
				Evidence:           "https://support.microsoft.com/en-us/accessibility/windows/make-it-easier-to-focus-on-tasks",
				SortOrder:          25,
			},
		},
		{
			spec: regtweak.Spec{
				TweakID:        "visual.fade-menus",
				DisplayName:    "Desativar transição suave e rolagem lenta de menus (Smooth Scroll)",
				Explanation:    "Desativa o efeito de esmaecimento (fade/slide) e rolagem suave de menus e caixas de ferramentas.",
				Cat:            tweak.CategoryVisualEffects,
				RiskLevel:      tweak.RiskLow,
				NeedsRestart:   true,
				AppliedText:    "A rolagem e transição suave de menus está desligada.",
				NotAppliedText: "A transição suave de menus está ativa.",
			},
			values: []regtweak.Value{
				regtweak.DWord(winreg.HKCU, pathDesktop, "SmoothScroll", 0, 1),
			},
			meta: tweak.Meta{
				RecommendedDefault: false,
				Profiles:           tweak.ProfileBoth,
				Caveat:             "Melhora a sensação de agilidade imediata ao navegar em listas e menus suspensos.",
				Evidence:           "https://support.microsoft.com/en-us/accessibility/windows/make-it-easier-to-focus-on-tasks",
				SortOrder:          26,
			},
		},
		{
			spec: regtweak.Spec{
				TweakID:        "visual.sombras-janelas",
				DisplayName:    "Desativar sombras sob janelas e ícones",
				Explanation:    "Remove o efeito de sombra projetada sob as janelas e rótulos de ícones da área de trabalho, poupando composição gráfica.",
				Cat:            tweak.CategoryVisualEffects,
				RiskLevel:      tweak.RiskLow,
				NeedsRestart:   true,
				AppliedText:    "As sombras sob janelas e ícones estão desativadas.",
				NotAppliedText: "As sombras sob janelas estão ativas.",
			},
			values: []regtweak.Value{
				regtweak.DWord(winreg.HKCU, pathExplorerAdv, "ListviewShadow", 0, 1),
			},
			meta: tweak.Meta{
				RecommendedDefault: false,
				Profiles:           tweak.ProfileBoth,
				Caveat:             "Dá um aspecto visual limpo e plano (flat) à interface, reduzindo ligeiramente o uso do compositor DWM.",
				Evidence:           "https://support.microsoft.com/en-us/accessibility/windows/make-it-easier-to-focus-on-tasks",
				SortOrder:          27,
			},
		},
		{
			spec: regtweak.Spec{
				TweakID:        "visual.rastros-mouse",
				DisplayName:    "Garantir rastros do ponteiro do mouse desativados",
				Explanation:    "Garante que o rastro visual do ponteiro do mouse esteja completamente desligado para evitar interferência visual em jogos e precisão gráfica.",
				Cat:            tweak.CategoryVisualEffects,
				RiskLevel:      tweak.RiskLow,
				NeedsRestart:   false,
				AppliedText:    "O rastro do ponteiro do mouse está desligado.",
				NotAppliedText: "O rastro do ponteiro do mouse está ativo.",
			},
			values: []regtweak.Value{
				regtweak.String(winreg.HKCU, pathMouse, "MouseTrails", "0", "0"),
			},
			meta: tweak.Meta{
				RecommendedDefault: true,
				Profiles:           tweak.ProfileBoth,
				Caveat:             "Rastros do mouse são recursos de acessibilidade para telas antigas ou baixa visão; em jogos causam atraso e artefatos de renderização.",
				Evidence:           "https://support.microsoft.com/en-us/windows/hardware/input-devices/change-mouse-settings",
				SortOrder:          28,
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
				TweakID:        "sistema.search-sem-bing",
				DisplayName:    "Pesquisa do Menu Iniciar sem resultados da web (Bing)",
				Explanation:    "Faz a pesquisa do Menu Iniciar buscar somente programas e arquivos locais no computador, sem gastar conexão nem exibir sugestões e notícias da web.",
				Cat:            tweak.CategorySystem,
				RiskLevel:      tweak.RiskLow,
				NeedsRestart:   true,
				AppliedText:    "A pesquisa do Menu Iniciar está restrita a arquivos locais.",
				NotAppliedText: "A pesquisa do Menu Iniciar pode incluir resultados e sugestões da web.",
			},
			values: []regtweak.Value{
				regtweak.DWord(winreg.HKLM, pathSearchPolicy, "DisableWebSearch", 1, 0),
				regtweak.DWord(winreg.HKLM, pathSearchPolicy, "ConnectedSearchUseWeb", 0, 1),
			},
			meta: tweak.Meta{
				RecommendedDefault: false,
				Profiles:           tweak.ProfileBoth,
				RequiresAdmin:      true,
				Caveat:             "Acelera a resposta da busca do Menu Iniciar e protege sua privacidade ao não enviar o que você digita para a internet. Se você costuma pesquisar na web direto pelo Menu Iniciar, mantenha como está.",
				Evidence:           "docs/catalogo/registro.md#indexação-e-busca-windows-search",
				SortOrder:          35,
			},
		},
		{
			spec: regtweak.Spec{
				TweakID:        "jogos.windowed-optimizations",
				DisplayName:    "Ativar otimizações para jogos em modo janela (Windows 11)",
				Explanation:    "Habilita o modelo moderno de apresentação de flip para jogos em janela e sem borda (DX10/DX11), reduzindo latência de frame e permitindo Auto HDR e taxa de atualização variável.",
				Cat:            tweak.CategoryGaming,
				RiskLevel:      tweak.RiskLow,
				NeedsRestart:   false,
				AppliedText:    "As otimizações para jogos em janela estão ativadas.",
				NotAppliedText: "As otimizações para jogos em janela estão desativadas.",
			},
			values: []regtweak.Value{
				regtweak.DWord(winreg.HKCU, pathGameConfig, "GameDVR_DXGIHonorFSEWindowsCompatible", 1, 0),
				regtweak.DWord(winreg.HKCU, pathGameConfig, "GameDVR_DSEBehavior", 2, 0),
			},
			meta: tweak.Meta{
				RecommendedDefault: true,
				Profiles:           tweak.ProfilePersonal,
				Caveat:             "Recurso nativo oficial do Windows 11. Reduz o input lag ao jogar em modo janela sem borda ao equiparar a latência ao modo tela cheia exclusiva.",
				Evidence:           "https://support.microsoft.com/en-us/windows/hardware/display-graphics/optimizations-for-windowed-games-in-windows-11",
				SortOrder:          36,
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
				TweakID:        "sistema.background-apps-off",
				DisplayName:    "Desativar execução de aplicativos padrão em segundo plano",
				Explanation:    "Impede que aplicativos UWP empacotados continuem rodando silenciosamente e consumindo ciclos de CPU e RAM quando fechados.",
				Cat:            tweak.CategorySystem,
				RiskLevel:      tweak.RiskLow,
				NeedsRestart:   false,
				AppliedText:    "Aplicativos em segundo plano estão desativados.",
				NotAppliedText: "Aplicativos em segundo plano têm permissão para rodar.",
			},
			values: []regtweak.Value{
				regtweak.DWord(winreg.HKCU, pathBackgroundAccess, "GlobalUserDisabled", 1, 0),
			},
			meta: tweak.Meta{
				RecommendedDefault: false,
				Profiles:           tweak.ProfileBoth,
				Caveat:             "Pode pausar notificações ao vivo de aplicativos como Email ou Calculadora quando eles não estiverem abertos na tela.",
				Evidence:           "https://support.microsoft.com/ro-ro/windows/experience/performance-optimization/tips-to-improve-pc-performance-in-windows",
				SortOrder:          45,
			},
		},
		{
			spec: regtweak.Spec{
				TweakID:        "sistema.storage-sense-on",
				DisplayName:    "Ativar Sensor de Armazenamento (Storage Sense automático)",
				Explanation:    "Ativa a rotina oficial do Windows para liberar automaticamente espaço em disco excluindo arquivos temporários desnecessários.",
				Cat:            tweak.CategoryStorage,
				RiskLevel:      tweak.RiskLow,
				NeedsRestart:   false,
				AppliedText:    "O Sensor de Armazenamento está ativado.",
				NotAppliedText: "O Sensor de Armazenamento está desativado.",
			},
			values: []regtweak.Value{
				regtweak.DWord(winreg.HKCU, pathStorageSense, "01", 1, 0),
			},
			meta: tweak.Meta{
				RecommendedDefault: true,
				Profiles:           tweak.ProfileBoth,
				Caveat:             "Recurso oficial e altamente recomendado pela Microsoft para prevenir acúmulo de arquivos temporários em SSDs.",
				Evidence:           "https://learn.microsoft.com/en-us/windows/configuration/storage/storage-sense",
				SortOrder:          46,
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
				TweakID:        "entrada.sticky-keys-off",
				DisplayName:    "Desativar atalho de Teclas de Aderência (Sticky Keys ao pressionar Shift 5x)",
				Explanation:    "Desativa o popup que interrompe jogos em tela cheia quando a tecla Shift é pressionada cinco vezes consecutivas.",
				Cat:            tweak.CategoryInput,
				RiskLevel:      tweak.RiskLow,
				NeedsRestart:   false,
				AppliedText:    "O atalho de Teclas de Aderência está desativado.",
				NotAppliedText: "O atalho de Teclas de Aderência está ativo (padrão 5x Shift).",
			},
			values: []regtweak.Value{
				regtweak.String(winreg.HKCU, pathAccessibilitySticky, "Flags", "504", "506"),
			},
			meta: tweak.Meta{
				RecommendedDefault: false,
				Profiles:           tweak.ProfilePersonal,
				Caveat:             "Aviso de Acessibilidade: Desative apenas se você não utiliza o recurso de teclas de aderência para assistência motora ao digitar.",
				Evidence:           "https://support.microsoft.com/en-us/accessibility/windows/make-your-mouse-keyboard-and-other-input-devices-easier-to-use",
				SortOrder:          55,
			},
		},
		{
			spec: regtweak.Spec{
				TweakID:        "entrada.toggle-keys-off",
				DisplayName:    "Desativar atalho de Teclas de Alternância (Toggle Keys ao segurar NumLock)",
				Explanation:    "Desativa o bip e aviso sonoro ao segurar a tecla NumLock por 5 segundos.",
				Cat:            tweak.CategoryInput,
				RiskLevel:      tweak.RiskLow,
				NeedsRestart:   false,
				AppliedText:    "O atalho de Teclas de Alternância está desativado.",
				NotAppliedText: "O atalho de Teclas de Alternância está ativo.",
			},
			values: []regtweak.Value{
				regtweak.String(winreg.HKCU, pathAccessibilityToggle, "Flags", "50", "58"),
			},
			meta: tweak.Meta{
				RecommendedDefault: false,
				Profiles:           tweak.ProfilePersonal,
				Caveat:             "Recurso de acessibilidade para feedback sonoro. Seguro para desativar em máquinas de uso comum e jogos.",
				Evidence:           "https://support.microsoft.com/en-us/accessibility/windows/make-your-mouse-keyboard-and-other-input-devices-easier-to-use",
				SortOrder:          56,
			},
		},
		{
			spec: regtweak.Spec{
				TweakID:        "entrada.filter-keys-off",
				DisplayName:    "Desativar atalho de Teclas de Filtragem (Filter Keys ao segurar Shift)",
				Explanation:    "Impede o travamento de repetição de teclas ao segurar a tecla Shift direita por 8 segundos durante jogos ou digitação rápida.",
				Cat:            tweak.CategoryInput,
				RiskLevel:      tweak.RiskLow,
				NeedsRestart:   false,
				AppliedText:    "O atalho de Teclas de Filtragem está desativado.",
				NotAppliedText: "O atalho de Teclas de Filtragem está ativo.",
			},
			values: []regtweak.Value{
				regtweak.String(winreg.HKCU, pathAccessibilityFilter, "Flags", "120", "122"),
			},
			meta: tweak.Meta{
				RecommendedDefault: false,
				Profiles:           tweak.ProfilePersonal,
				Caveat:             "Aviso de Acessibilidade: Indicado para evitar que o teclado seja travado acidentalmente ao segurar teclas em jogos competitivos.",
				Evidence:           "https://support.microsoft.com/en-us/accessibility/windows/make-your-mouse-keyboard-and-other-input-devices-easier-to-use",
				SortOrder:          57,
			},
		},
		{
			spec: regtweak.Spec{
				TweakID:        "sistema.long-paths",
				DisplayName:    "Habilitar suporte a caminhos de arquivos longos (Long Paths 260+ caracteres)",
				Explanation:    "Remove o limite clássico de 260 caracteres do Windows para nomes e caminhos de arquivos, essencial para NodeJS, Git, Rust, Go e compilação de código.",
				Cat:            tweak.CategorySystem,
				RiskLevel:      tweak.RiskLow,
				NeedsRestart:   false,
				AppliedText:    "O suporte a caminhos longos (Long Paths) está habilitado.",
				NotAppliedText: "O limite padrão de 260 caracteres para caminhos de arquivo está ativo.",
			},
			values: []regtweak.Value{
				regtweak.DWord(winreg.HKLM, pathFileSystem, "LongPathsEnabled", 1, 0),
			},
			meta: tweak.Meta{
				RecommendedDefault: true,
				Profiles:           tweak.ProfileBoth,
				RequiresAdmin:      true,
				Caveat:             "Oficial da Microsoft desde o Windows 10. Permite que ferramentas de desenvolvimento lidem com pastas profundas em node_modules e subprojetos sem erros de I/O.",
				Evidence:           "https://learn.microsoft.com/en-us/windows/win32/fileio/maximum-file-path-limitation",
				SortOrder:          58,
			},
		},
		{
			spec: regtweak.Spec{
				TweakID:        "armazenamento.ntfs-last-access",
				DisplayName:    "Desativar atualização de carimbo de último acesso NTFS",
				Explanation:    "Evita que o Windows grave atualizações de data/hora de último acesso toda vez que um arquivo de código ou diretório for lido, acelerando compilações e buscas.",
				Cat:            tweak.CategoryStorage,
				RiskLevel:      tweak.RiskLow,
				NeedsRestart:   false,
				AppliedText:    "A atualização de carimbo de último acesso NTFS está desativada.",
				NotAppliedText: "A gravação de carimbo de último acesso NTFS está ativa.",
			},
			values: []regtweak.Value{
				regtweak.DWord(winreg.HKLM, pathFileSystem, "NtfsDisableLastAccessUpdate", 1, 0),
			},
			meta: tweak.Meta{
				RecommendedDefault: true,
				Profiles:           tweak.ProfileBoth,
				RequiresAdmin:      true,
				Caveat:             "Reduz operações de I/O de escrita redundantes no disco durante varreduras de antivírus e indexações pesadas de código.",
				Evidence:           "https://learn.microsoft.com/en-us/windows-server/administration/performance-tuning/subsystems/software-defined-datacenter/file-server",
				SortOrder:          59,
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
				TweakID:        "rede.thumbs-rede-off",
				DisplayName:    "Não criar arquivos thumbs.db em pastas de rede",
				Explanation:    "Impede o Windows de criar e travar arquivos ocultos de miniaturas em compartilhamentos de rede, facilitando operações de renomear e excluir pastas remotas.",
				Cat:            tweak.CategoryNetwork,
				RiskLevel:      tweak.RiskLow,
				NeedsRestart:   true,
				AppliedText:    "A criação de thumbs.db em rede está desativada.",
				NotAppliedText: "O Windows pode criar thumbs.db em pastas de rede.",
			},
			values: []regtweak.Value{
				regtweak.DWord(winreg.HKCU, pathExplorerPolicy, "DisableThumbsDBOnNetworkFolders", 1, 0),
			},
			meta: tweak.Meta{
				RecommendedDefault: false,
				Profiles:           tweak.ProfileWork,
				Caveat:             "Recomendado especialmente em ambientes corporativos e de trabalho com pastas em rede (NAS/compartilhamentos). Não afeta o cache de miniaturas em discos locais.",
				Evidence:           "docs/catalogo/registro.md#cache-de-miniaturas",
				SortOrder:          75,
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
		{
			spec: regtweak.Spec{
				TweakID:     "rede.tcp-timed-wait-delay",
				DisplayName: "Reduzir o tempo de espera de conexões encerradas (TIME_WAIT)",
				Explanation: "Quando uma conexão TCP é encerrada, o Windows mantém um registro dela por 4 minutos por padrão. " +
					"Este ajuste reduz esse tempo para 30 segundos, liberando mais portas para novas conexões.",
				Cat:            tweak.CategoryNetwork,
				RiskLevel:      tweak.RiskMedium,
				NeedsRestart:   true,
				AppliedText:    "O tempo de TIME_WAIT está reduzido para 30 segundos.",
				NotAppliedText: "O tempo de TIME_WAIT está no padrão (240 segundos).",
			},
			values: []regtweak.Value{
				regtweak.DWord(winreg.HKLM, pathTcpip, "TcpTimedWaitDelay", 30, 240),
			},
			meta: tweak.Meta{
				RecommendedDefault: false,
				Profiles:           tweak.ProfileBoth,
				RequiresAdmin:      true,
				Caveat: "Válido para uso intenso de conexões efêmeras (ex.: muitas requisições HTTP rápidas, teste de carga). " +
					"Em uso comum, porta alguma diferença. Em rede instável ou com muita perda de pacotes, pode causar ressincronização de conexão — deixar como está nesse caso.",
				Evidence:  "docs/catalogo/rede.md — notas adicionais",
				SortOrder: 110,
			},
		},
		{
			spec: regtweak.Spec{
				TweakID:     "rede.max-user-port",
				DisplayName: "Aumentar o número de portas efêmeras disponíveis",
				Explanation: "O Windows usa portas efêmeras (temporárias) para conexões de saída. O padrão é 5000 portas (de 49152 a 54151). " +
					"Este ajuste estende até 65535, disponibilizando mais portas para aplicações com muitas conexões simultâneas.",
				Cat:            tweak.CategoryNetwork,
				RiskLevel:      tweak.RiskLow,
				NeedsRestart:   true,
				AppliedText:    "O número máximo de portas efêmeras foi estendido.",
				NotAppliedText: "O número máximo de portas efêmeras está no padrão.",
			},
			values: []regtweak.Value{
				regtweak.DWord(winreg.HKLM, pathTcpip, "MaxUserPort", 65534, 5000),
			},
			meta: tweak.Meta{
				RecommendedDefault: false,
				Profiles:           tweak.ProfileBoth,
				RequiresAdmin:      true,
				Caveat: "Só importa se seus aplicativos costumam abrir muitas conexões simultâneas (acima de 5000). " +
					"Não causa problema em aumentar, mas também não traz ganho se você não estiver chegando no limite.",
				Evidence:  "docs/catalogo/rede.md — notas adicionais",
				SortOrder: 120,
			},
		},
		{
			spec: regtweak.Spec{
				TweakID:        "energia.power-throttling-off",
				DisplayName:    "Desativar Power Throttling para processos de fundo",
				Explanation:    "Impede o Windows de limitar a frequência e os recursos de threads em segundo plano de jogos, renderizadores e ferramentas de compilação.",
				Cat:            tweak.CategoryPower,
				RiskLevel:      tweak.RiskLow,
				NeedsRestart:   true,
				AppliedText:    "O Power Throttling está desativado no sistema.",
				NotAppliedText: "O Power Throttling do Windows está no modo padrão.",
			},
			values: []regtweak.Value{
				regtweak.DWord(winreg.HKLM, pathPowerThrottling, "PowerThrottlingOff", 1, 0),
			},
			meta: tweak.Meta{
				RecommendedDefault: false,
				Profiles:           tweak.ProfileBoth,
				RequiresAdmin:      true,
				Caveat:             "Garante desempenho máximo constante para ferramentas de compilação e jogos pesados. Em laptops na bateria, pode consumir carga ligeiramente mais rápido.",
				Evidence:           "docs/catalogo/hardware-energia-gpu.md#power-throttling",
				SortOrder:          130,
			},
		},
		{
			spec: regtweak.Spec{
				TweakID:        "energia.coalescing-timer-zero",
				DisplayName:    "Desativar agrupamento de temporizadores (Coalescing Timer)",
				Explanation:    "Zera o intervalo de agrupamento forçado de timers do agendador, eliminando micro-atrasos e jitter de temporização em jogos e áudio de baixa latência.",
				Cat:            tweak.CategoryPower,
				RiskLevel:      tweak.RiskLow,
				NeedsRestart:   true,
				AppliedText:    "O intervalo de Coalescing Timer está zerado (sem atraso).",
				NotAppliedText: "O Coalescing Timer do Windows está no modo padrão.",
			},
			values: []regtweak.Value{
				regtweak.DWord(winreg.HKLM, pathPowerSession, "CoalescingTimerInterval", 0, 0),
			},
			meta: tweak.Meta{
				RecommendedDefault: false,
				Profiles:           tweak.ProfilePersonal,
				RequiresAdmin:      true,
				Caveat:             "Melhora a precisão e consistência de tempo de resposta em setups gamers e estações de áudio.",
				Evidence:           "docs/catalogo/hardware-energia-gpu.md#coalescing-timers",
				SortOrder:          135,
			},
		},
		{
			spec: regtweak.Spec{
				TweakID:        "sistema.disable-paging-executive",
				DisplayName:    "Manter Kernel e Drivers na Memória RAM (Disable Paging Executive)",
				Explanation:    "Impede o Windows de paginar o código essencial do kernel e dos drivers para o arquivo de paginação no disco, garantindo tempos de resposta instantâneos.",
				Cat:            tweak.CategorySystem,
				RiskLevel:      tweak.RiskLow,
				NeedsRestart:   true,
				AppliedText:    "O kernel e os drivers estão fixados na memória RAM.",
				NotAppliedText: "O Windows pode paginar partes do kernel para o disco.",
			},
			values: []regtweak.Value{
				regtweak.DWord(winreg.HKLM, pathMemMgmt, "DisablePagingExecutive", 1, 0),
			},
			meta: tweak.Meta{
				RecommendedDefault: true,
				Profiles:           tweak.ProfileBoth,
				RequiresAdmin:      true,
				Caveat:             "Altamente recomendado para sistemas com 8GB ou mais de memória RAM para evitar I/O desnecessário.",
				Evidence:           "docs/catalogo/registro.md#disablepagingexecutive",
				SortOrder:          140,
			},
		},
		{
			spec: regtweak.Spec{
				TweakID:        "sistema.process-termination-timeouts",
				DisplayName:    "Acelerar encerramento de processos e desligamento",
				Explanation:    "Reduz os tempos de espera para fechar aplicativos travados de 5-20 segundos para 1-2 segundos, agilizando o uso diário e o desligamento do PC.",
				Cat:            tweak.CategorySystem,
				RiskLevel:      tweak.RiskLow,
				NeedsRestart:   false,
				AppliedText:    "Os tempos de espera para finalização de processos estão otimizados.",
				NotAppliedText: "Os tempos de espera para finalização de processos estão no padrão longo.",
			},
			values: []regtweak.Value{
				regtweak.String(winreg.HKCU, pathDesktop, "HungAppTimeout", "1000", "5000"),
				regtweak.String(winreg.HKCU, pathDesktop, "WaitToKillAppTimeout", "2000", "20000"),
				regtweak.String(winreg.HKCU, pathDesktop, "AutoEndTasks", "1", "0"),
				regtweak.String(winreg.HKLM, pathSessionManager, "WaitToKillServiceTimeout", "2000", "5000"),
			},
			meta: tweak.Meta{
				RecommendedDefault: true,
				Profiles:           tweak.ProfileBoth,
				RequiresAdmin:      true,
				Caveat:             "Permite fechar janelas que não respondem com mais rapidez e desliga o computador sem atrasos desnecessários.",
				Evidence:           "docs/catalogo/registro.md#hungapptimeout",
				SortOrder:          145,
			},
		},
		{
			spec: regtweak.Spec{
				TweakID:        "entrada.keyboard-repeat-rate",
				DisplayName:    "Taxa de repetição máxima do teclado e atraso mínimo",
				Explanation:    "Configura o teclado para iniciar a repetição imediatamente (Delay 0 / 250ms) e na velocidade máxima de 30 caracteres por segundo.",
				Cat:            tweak.CategoryInput,
				RiskLevel:      tweak.RiskLow,
				NeedsRestart:   false,
				AppliedText:    "O teclado está configurado para repetição instantânea e velocidade máxima.",
				NotAppliedText: "O teclado está no padrão de repetição do Windows.",
			},
			values: []regtweak.Value{
				regtweak.String(winreg.HKCU, pathKeyboard, "KeyboardDelay", "0", "1"),
				regtweak.String(winreg.HKCU, pathKeyboard, "KeyboardSpeed", "31", "31"),
			},
			meta: tweak.Meta{
				RecommendedDefault: true,
				Profiles:           tweak.ProfileBoth,
				Caveat:             "Torna comandos de movimentação em jogos (strafe/tap) e navegação em código extremamente ágeis.",
				Evidence:           "docs/catalogo/registro.md#latência-de-entrada-mouse-e-teclado",
				SortOrder:          150,
			},
		},
		{
			spec: regtweak.Spec{
				TweakID:        "entrada.mouse-keyboard-queue-size",
				DisplayName:    "Reduzir tamanho da fila de eventos de mouse e teclado",
				Explanation:    "Reduz o buffer interno de fila de eventos de entrada (mouclass/kbdclass) de 100 para 24 itens, minimizando latência de entrada.",
				Cat:            tweak.CategoryInput,
				RiskLevel:      tweak.RiskLow,
				NeedsRestart:   true,
				AppliedText:    "O tamanho da fila de eventos de entrada está otimizado (24).",
				NotAppliedText: "O tamanho da fila de eventos de entrada está no padrão (100).",
			},
			values: []regtweak.Value{
				regtweak.DWord(winreg.HKLM, pathMouClass, "MouseDataQueueSize", 24, 100),
				regtweak.DWord(winreg.HKLM, pathKbdClass, "KeyboardDataQueueSize", 24, 100),
			},
			meta: tweak.Meta{
				RecommendedDefault: false,
				Profiles:           tweak.ProfilePersonal,
				RequiresAdmin:      true,
				Caveat:             "Projetado para mouses e teclados gamer de 1000Hz+ para diminuir o atraso de buffer.",
				Evidence:           "docs/catalogo/registro.md#latência-de-entrada-mouse-e-teclado",
				SortOrder:          155,
			},
		},
		{
			spec: regtweak.Spec{
				TweakID:        "jogos.mmcss-low-latency",
				DisplayName:    "Prioridade de agendamento multimídia para Baixa Latência",
				Explanation:    "Configura a classe de agendamento Low Latency do MMCSS com prioridade de GPU e sensibilidade a latência para áudio e jogos.",
				Cat:            tweak.CategoryGaming,
				RiskLevel:      tweak.RiskLow,
				NeedsRestart:   true,
				AppliedText:    "O perfil MMCSS Low Latency está configurado para máxima prioridade.",
				NotAppliedText: "O perfil MMCSS Low Latency está no padrão do Windows.",
			},
			values: []regtweak.Value{
				regtweak.DWord(winreg.HKLM, pathMMCSSLowLatency, "GPU Priority", 8, 8),
				regtweak.DWord(winreg.HKLM, pathMMCSSLowLatency, "Priority", 2, 2),
				regtweak.String(winreg.HKLM, pathMMCSSLowLatency, "Scheduling Category", "Medium", "Medium"),
				regtweak.String(winreg.HKLM, pathMMCSSLowLatency, "SFIO Priority", "High", "Normal"),
				regtweak.String(winreg.HKLM, pathMMCSSLowLatency, "Latency Sensitive", "True", "True"),
			},
			meta: tweak.Meta{
				RecommendedDefault: true,
				Profiles:           tweak.ProfilePersonal,
				RequiresAdmin:      true,
				Caveat:             "Reduz atrasos de processamento em pipelines de áudio e jogos em tempo real.",
				Evidence:           "docs/catalogo/registro.md#mmcss",
				SortOrder:          160,
			},
		},
		{
			spec: regtweak.Spec{
				TweakID:        "jogos.dxgkrnl-latency-tolerance",
				DisplayName:    "Otimizar tolerância de latência do DXGKrnl",
				Explanation:    "Ajusta os temporizadores de tolerância de latência do subsistema de vídeo DirectX para reduzir o buffer na apresentação de quadros.",
				Cat:            tweak.CategoryGaming,
				RiskLevel:      tweak.RiskLow,
				NeedsRestart:   true,
				AppliedText:    "A tolerância de latência do DXGKrnl está configurada para resposta rápida.",
				NotAppliedText: "A tolerância de latência do DXGKrnl está no padrão.",
			},
			values: []regtweak.Value{
				regtweak.DWord(winreg.HKLM, pathDXGKrnl, "MonitorLatencyTolerance", 1, 0),
				regtweak.DWord(winreg.HKLM, pathDXGKrnl, "MonitorRefreshLatencyTolerance", 1, 0),
			},
			meta: tweak.Meta{
				RecommendedDefault: false,
				Profiles:           tweak.ProfilePersonal,
				RequiresAdmin:      true,
				Caveat:             "Reduz a latência de exibição do quadro entre a GPU e a tela em jogos 3D.",
				Evidence:           "docs/catalogo/hardware-energia-gpu.md#dxgkrnl-latency",
				SortOrder:          165,
			},
		},
		{
			spec: regtweak.Spec{
				TweakID:        "jogos.legacy-multimedia-fps",
				DisplayName:    "Ativar renderização fluida e contagem de FPS em jogos legados",
				Explanation:    "Habilita as diretrizes de fluidez gráfica e modo de alta taxa de quadros do subsistema clássico de jogos da Microsoft.",
				Cat:            tweak.CategoryGaming,
				RiskLevel:      tweak.RiskLow,
				NeedsRestart:   false,
				AppliedText:    "As flags de fluidez para jogos clássicos estão ativadas.",
				NotAppliedText: "As flags de fluidez para jogos clássicos estão desativadas.",
			},
			values: []regtweak.Value{
				regtweak.DWord(winreg.HKCU, pathGamesLegacy, "FpsAll", 1, 0),
				regtweak.DWord(winreg.HKCU, pathGamesLegacy, "GameFluidity", 1, 0),
			},
			meta: tweak.Meta{
				RecommendedDefault: false,
				Profiles:           tweak.ProfilePersonal,
				Caveat:             "Melhora a resposta em títulos DirectX mais antigos e executáveis legados de jogos.",
				Evidence:           "docs/catalogo/registro.md#jogos-legados",
				SortOrder:          170,
			},
		},
		{
			spec: regtweak.Spec{
				TweakID:        "hardware.usb-power-savings-off",
				DisplayName:    "Desativar suspensão seletiva e economia de energia USB",
				Explanation:    "Impede o Windows de desligar portas USB ou colocá-las em suspensão seletiva (D3), evitando desconexões e micro-travamentos no mouse.",
				Cat:            tweak.CategorySystem,
				RiskLevel:      tweak.RiskLow,
				NeedsRestart:   true,
				AppliedText:    "A suspensão seletiva USB está desativada.",
				NotAppliedText: "A suspensão seletiva USB pode ser ativada pelo Windows.",
			},
			values: []regtweak.Value{
				regtweak.DWord(winreg.HKLM, pathUsb, "DisableSelectiveSuspend", 1, 0),
			},
			meta: tweak.Meta{
				RecommendedDefault: false,
				Profiles:           tweak.ProfilePersonal,
				RequiresAdmin:      true,
				Caveat:             "Elimina congelamentos temporários em mouses e interfaces de áudio USB causados por transições de energia.",
				Evidence:           "docs/catalogo/hardware-energia-gpu.md#usb-power",
				SortOrder:          175,
			},
		},
		{
			spec: regtweak.Spec{
				TweakID:        "privacidade.cloud-setting-sync-off",
				DisplayName:    "Desativar sincronização de configurações na nuvem",
				Explanation:    "Impede o Windows de sincronizar automaticamente papéis de parede, temas e histórico de personalização entre múltiplos computadores.",
				Cat:            tweak.CategoryPrivacyNonSecurity,
				RiskLevel:      tweak.RiskLow,
				NeedsRestart:   false,
				AppliedText:    "A sincronização de configurações na nuvem está desligada.",
				NotAppliedText: "A sincronização de configurações na nuvem está ativa.",
			},
			values: []regtweak.Value{
				regtweak.DWord(winreg.HKCU, pathSettingSync, "Enabled", 0, 1),
			},
			meta: tweak.Meta{
				RecommendedDefault: false,
				Profiles:           tweak.ProfileBoth,
				Caveat:             "Economiza tráfego de rede e evita que alterações estéticas em outro dispositivo sobreponham seu computador.",
				Evidence:           "docs/catalogo/registro.md#cloud-sync",
				SortOrder:          180,
			},
		},
		{
			spec: regtweak.Spec{
				TweakID:        "privacidade.content-delivery-suggestions-off",
				DisplayName:    "Bloquear instalação automática de aplicativos patrocinados (OEM/Store)",
				Explanation:    "Impede o Windows de baixar silenciosamente aplicativos patrocinados, anúncios de parceiros e sugestões automáticas no Menu Iniciar.",
				Cat:            tweak.CategoryPrivacyNonSecurity,
				RiskLevel:      tweak.RiskLow,
				NeedsRestart:   false,
				AppliedText:    "O download automático de aplicativos patrocinados está bloqueado.",
				NotAppliedText: "O Windows pode instalar aplicativos e sugestões patrocinadas.",
			},
			values: []regtweak.Value{
				regtweak.DWord(winreg.HKCU, pathContentDelivery, "PreInstalledAppsEnabled", 0, 1),
				regtweak.DWord(winreg.HKCU, pathContentDelivery, "SilentInstalledAppsEnabled", 0, 1),
				regtweak.DWord(winreg.HKCU, pathContentDelivery, "SubscribedContentEnabled", 0, 1),
			},
			meta: tweak.Meta{
				RecommendedDefault: true,
				Profiles:           tweak.ProfileBoth,
				Caveat:             "Evita bloatware indesejado no Menu Iniciar após grandes atualizações do Windows.",
				Evidence:           "docs/catalogo/registro.md#contentdeliverymanager",
				SortOrder:          185,
			},
		},
		{
			spec: regtweak.Spec{
				TweakID:        "privacidade.activity-feed-timeline-off",
				DisplayName:    "Desativar ID de publicidade e rastreamento de atividade",
				Explanation:    "Desliga o identificador exclusivo de publicidade do Windows usado para veicular anúncios direcionados com base no seu perfil de uso.",
				Cat:            tweak.CategoryPrivacyNonSecurity,
				RiskLevel:      tweak.RiskLow,
				NeedsRestart:   false,
				AppliedText:    "O ID de publicidade e rastreamento de atividade está desativado.",
				NotAppliedText: "O ID de publicidade está habilitado.",
			},
			values: []regtweak.Value{
				regtweak.DWord(winreg.HKCU, pathAdvInfo, "Enabled", 0, 1),
			},
			meta: tweak.Meta{
				RecommendedDefault: true,
				Profiles:           tweak.ProfileBoth,
				Caveat:             "Protege a privacidade ao impedir que aplicativos monitorem seu comportamento para segmentação comercial.",
				Evidence:           "docs/catalogo/registro.md#advertising-info",
				SortOrder:          190,
			},
		},
		{
			spec: regtweak.Spec{
				TweakID:        "privacidade.error-reporting-telemetry-off",
				DisplayName:    "Desativar fila de relatórios de erros do Windows (WER)",
				Explanation:    "Desativa o envio automático de relatórios de travamentos de programas para os servidores da Microsoft, economizando dados e CPU.",
				Cat:            tweak.CategoryPrivacyNonSecurity,
				RiskLevel:      tweak.RiskLow,
				NeedsRestart:   false,
				AppliedText:    "A fila de relatórios de erros está desativada.",
				NotAppliedText: "A fila de relatórios de erros está ativa.",
			},
			values: []regtweak.Value{
				regtweak.DWord(winreg.HKLM, pathErrorReporting, "Disabled", 1, 0),
				regtweak.DWord(winreg.HKLM, pathErrorReporting, "DoReport", 0, 1),
			},
			meta: tweak.Meta{
				RecommendedDefault: false,
				Profiles:           tweak.ProfileBoth,
				RequiresAdmin:      true,
				Caveat:             "Impede que falhas de aplicativos abram processos pesados de coleta e envio de dados em segundo plano.",
				Evidence:           "docs/catalogo/registro.md#windows-error-reporting",
				SortOrder:          195,
			},
		},
		{
			spec: regtweak.Spec{
				TweakID:        "jogos.nvidia-powermizer-performance",
				DisplayName:    "NVIDIA PowerMizer em Desempenho Máximo (Modo de Energia GPU)",
				Explanation:    "Fixa o perfil de energia da GPU NVIDIA para Desempenho Máximo (PowerMizerLevel=1), impedindo que o clock da GPU caia durante trocas de cena ou carregamento de jogos.",
				Cat:            tweak.CategoryGaming,
				RiskLevel:      tweak.RiskLow,
				NeedsRestart:   false,
				AppliedText:    "O PowerMizer da GPU NVIDIA está configurado em Desempenho Máximo.",
				NotAppliedText: "O PowerMizer da GPU NVIDIA está no modo adaptativo/padrão.",
			},
			values: []regtweak.Value{
				regtweak.DWord(winreg.HKLM, pathNvidiaClassPrimary, "PowerMizerEnable", 1, 0),
				regtweak.DWord(winreg.HKLM, pathNvidiaClassPrimary, "PowerMizerLevel", 1, 0),
				regtweak.DWord(winreg.HKLM, pathNvidiaClassPrimary, "PowerMizerLevelAC", 1, 0),
			},
			meta: tweak.Meta{
				RecommendedDefault: true,
				Profiles:           tweak.ProfilePersonal,
				RequiresAdmin:      true,
				Caveat:             "Mantém os clocks de núcleo e memória estáveis em jogos pesados. Em laptops desconectados da tomada, consome mais bateria.",
				Evidence:           "docs/catalogo/hardware-energia-gpu.md#nvidia-powermizer",
				SortOrder:          200,
			},
		},
		{
			spec: regtweak.Spec{
				TweakID:        "jogos.nvidia-shader-cache-size",
				DisplayName:    "Aumentar Cache de Shaders NVIDIA (10 GB para evitar stutters)",
				Explanation:    "Expande o limite de cache de shaders compilados da placa NVIDIA para 10 GB, eliminando travamentos causados por recompilação constante em jogos DirectX 12 e Vulkan.",
				Cat:            tweak.CategoryGaming,
				RiskLevel:      tweak.RiskLow,
				NeedsRestart:   false,
				AppliedText:    "O Cache de Shaders NVIDIA está configurado para 10 GB.",
				NotAppliedText: "O Cache de Shaders NVIDIA está no limite padrão do driver.",
			},
			values: []regtweak.Value{
				regtweak.DWord(winreg.HKLM, pathNvidiaNvTweakGlobal, "ShaderCacheSize", 10240, 4096),
			},
			meta: tweak.Meta{
				RecommendedDefault: true,
				Profiles:           tweak.ProfileBoth,
				RequiresAdmin:      true,
				Caveat:             "Reduz quedas bruscas de FPS ao entrar em novas áreas em jogos modernos (Unreal Engine / DX12).",
				Evidence:           "docs/catalogo/hardware-energia-gpu.md#nvidia-shader-cache",
				SortOrder:          205,
			},
		},
		{
			spec: regtweak.Spec{
				TweakID:        "jogos.nvidia-d3pc-low-latency",
				DisplayName:    "Pipeline de Baixa Latência D3PC NVIDIA",
				Explanation:    "Configura a latência D3PC no driver de vídeo NVIDIA para 0 (modo ultrarrápido), reduzindo o atraso de sincronização de quadros da GPU.",
				Cat:            tweak.CategoryGaming,
				RiskLevel:      tweak.RiskLow,
				NeedsRestart:   true,
				AppliedText:    "O pipeline D3PC está configurado em latência ultrarrápida (0).",
				NotAppliedText: "O pipeline D3PC está no padrão de fábrica (1).",
			},
			values: []regtweak.Value{
				regtweak.DWord(winreg.HKLM, pathNvidiaClassPrimary, "D3PCLatency", 0, 1),
			},
			meta: tweak.Meta{
				RecommendedDefault: false,
				Profiles:           tweak.ProfilePersonal,
				RequiresAdmin:      true,
				Caveat:             "Otimização recomendada para jogos competitivos com alta taxa de quadros (144Hz+).",
				Evidence:           "docs/catalogo/hardware-energia-gpu.md#nvidia-low-latency",
				SortOrder:          210,
			},
		},
		{
			spec: regtweak.Spec{
				TweakID:        "privacidade.nvidia-telemetry-off",
				DisplayName:    "Desativar telemetria do Painel de Controle NVIDIA",
				Explanation:    "Desativa a coleta e o envio em segundo plano de telemetria de uso e diagnóstico do driver e painel NVIDIA.",
				Cat:            tweak.CategoryPrivacyNonSecurity,
				RiskLevel:      tweak.RiskLow,
				NeedsRestart:   false,
				AppliedText:    "A telemetria do Painel de Controle NVIDIA está desativada.",
				NotAppliedText: "A telemetria da NVIDIA está ativa.",
			},
			values: []regtweak.Value{
				regtweak.DWord(winreg.HKLM, pathNvidiaNvCplOptIn, "OptIn", 0, 1),
			},
			meta: tweak.Meta{
				RecommendedDefault: true,
				Profiles:           tweak.ProfileBoth,
				RequiresAdmin:      true,
				Caveat:             "Evita tráfego de rede e consumo em segundo plano de processos auxiliares do driver de vídeo.",
				Evidence:           "docs/catalogo/registro.md#nvidia-telemetry",
				SortOrder:          215,
			},
		},
	}
}

func withSatisfied(v regtweak.Value, f func(any) bool) regtweak.Value {
	v.AlreadyOptimized = f
	return v
}

