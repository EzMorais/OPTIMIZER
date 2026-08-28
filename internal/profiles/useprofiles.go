// Package profiles define os perfis de uso fechados JOGO, CODING e NVIDIA.
package profiles

// UseProfile representa um perfil completo de uso do computador.
type UseProfile struct {
	Key              string   `json:"key"`
	Nome             string   `json:"nome"`
	Objetivo         string   `json:"objetivo"`
	Descricao        string   `json:"descricao"`
	Categorias       []string `json:"categorias"`
	Ressalvas        string   `json:"ressalvas"`
	TweakIDs         []string `json:"tweakIds"`
	PowerPlanGUID    string   `json:"powerPlanGuid,omitempty"`
	DisableSleepAC   bool     `json:"disableSleepAc"`
	ServicesToPause  []string `json:"servicesToPause,omitempty"`
	ServicesToEnsure []string `json:"servicesToEnsure,omitempty"`
}

// ListarPerfisUso devolve as definições oficiais dos perfis JOGO, CODING e NVIDIA.
func ListarPerfisUso() []UseProfile {
	return []UseProfile{
		{
			Key:        "jogo",
			Nome:       "JOGO — Performance & Baixa Latência",
			Objetivo:   "Taxa máxima de quadros (FPS), resposta instantânea de entrada e redução de jitter na rede.",
			Descricao:  "Ativa o plano de Alto Desempenho, prioridade de primeiro plano, MMCSS Games & Low Latency, otimizações de jogos em janela, HAGS, taxa de repetição do teclado e desativa gravação em segundo plano.",
			Categorias: []string{"Energia", "Jogos", "Visual", "Rede", "Entrada", "Hardware"},
			Ressalvas:  "Pausa temporariamente a indexação de busca (WSearch) e pré-carregamento (SysMain) para poupar CPU e disco durante a jogatina.",
			TweakIDs: []string{
				"jogos.game-mode",
				"jogos.gamedvr-usuario",
				"jogos.windowed-optimizations",
				"jogos.mmcss-low-latency",
				"jogos.dxgkrnl-latency-tolerance",
				"visual.hags",
				"visual.menu-show-delay",
				"visual.anim-controles",
				"visual.rastros-mouse",
				"entrada.mouse-sem-aceleracao",
				"entrada.sticky-keys-off",
				"entrada.keyboard-repeat-rate",
				"sistema.mmcss-system-responsiveness",
				"sistema.disable-paging-executive",
				"energia.power-throttling-off",
				"energia.coalescing-timer-zero",
				"hardware.usb-power-savings-off",
				"rede.network-throttling-off",
				"rede.tcp-timed-wait-delay",
				"rede.max-user-port",
				"jogos.nvidia-powermizer-performance",
				"jogos.nvidia-shader-cache-size",
				"jogos.nvidia-d3pc-low-latency",
				"privacidade.nvidia-telemetry-off",
				"privacidade.content-delivery-suggestions-off",
			},
			PowerPlanGUID:   "8c5e7fda-e8bf-4a96-9a85-a6e23a8c635c", // Alto Desempenho
			DisableSleepAC:  true,
			ServicesToPause: []string{"WSearch", "SysMain"},
		},
		{
			Key:        "nvidia",
			Nome:       "NVIDIA — Ultra Latência & Clocks",
			Objetivo:   "Desempenho máximo para GPUs GeForce: PowerMizer fixo, Cache de Shaders 10GB, HAGS, D3PC Ultra e sem telemetria.",
			Descricao:  "Otimiza o pipeline do driver NVIDIA, fixa clocks estáveis em jogos pesados e elimina micro-travamentos por compilação de shaders.",
			Categorias: []string{"GPU", "NVIDIA", "Jogos", "Latência"},
			Ressalvas:  "Exclusivo para placas de vídeo NVIDIA GeForce. Em laptops fora da tomada, mantém a GPU em maior consumo.",
			TweakIDs: []string{
				"jogos.nvidia-powermizer-performance",
				"jogos.nvidia-shader-cache-size",
				"jogos.nvidia-d3pc-low-latency",
				"privacidade.nvidia-telemetry-off",
				"visual.hags",
				"jogos.game-mode",
				"jogos.windowed-optimizations",
				"jogos.mmcss-low-latency",
				"jogos.dxgkrnl-latency-tolerance",
				"entrada.keyboard-repeat-rate",
				"sistema.mmcss-system-responsiveness",
			},
			PowerPlanGUID:  "8c5e7fda-e8bf-4a96-9a85-a6e23a8c635c", // Alto Desempenho
			DisableSleepAC: true,
		},
		{
			Key:        "coding",
			Nome:       "CODING — Compilação & Estabilidade",
			Objetivo:   "Máximo throughput para IDEs, compilação de código, suporte a caminhos longos e estabilidade para Docker/WSL.",
			Descricao:  "Mantém suspensão na tomada desativada, habilita Long Paths no sistema de arquivos, desativa carimbo de último acesso NTFS, fixa kernel na RAM e garante serviços essenciais ativos.",
			Categorias: []string{"Sistema", "Armazenamento", "Rede", "Produtividade", "Privacidade"},
			Ressalvas:  "Nunca desativa serviços de virtualização, Docker ou Hyper-V. Retoma e preserva a indexação do Windows Search para buscas rápidas no explorador.",
			TweakIDs: []string{
				"sistema.long-paths",
				"sistema.disable-paging-executive",
				"sistema.process-termination-timeouts",
				"sistema.storage-sense-on",
				"armazenamento.trim-ntfs",
				"armazenamento.ntfs-last-access",
				"energia.power-throttling-off",
				"rede.max-user-port",
				"rede.tcp-timed-wait-delay",
				"rede.thumbs-rede-off",
				"visual.menu-show-delay",
				"entrada.keyboard-repeat-rate",
				"privacidade.content-delivery-suggestions-off",
				"privacidade.activity-feed-timeline-off",
			},
			DisableSleepAC:   true,
			ServicesToEnsure: []string{"WSearch", "SysMain", "wslservice", "com.docker.service", "vmms"},
		},
	}
}

// ObterPerfilUso busca um perfil pela chave ("jogo", "nvidia" ou "coding").
func ObterPerfilUso(key string) (UseProfile, bool) {
	for _, p := range ListarPerfisUso() {
		if p.Key == key {
			return p, true
		}
	}
	return UseProfile{}, false
}
