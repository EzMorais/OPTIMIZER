// Package profiles define os perfis de otimização nomeados (ex: "Rede Rápida", "Trabalho Remoto").
package profiles

// NetworkProfile é um conjunto de tweaks recomendado para um cenário específico.
type NetworkProfile struct {
	Key         string   `json:"key"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	TweakIDs    []string `json:"tweak_ids"`
	Caveats     string   `json:"caveats,omitempty"`
}

// NetworkProfiles mapeia chaves de perfil para suas definições.
var NetworkProfiles = map[string]NetworkProfile{
	"network-fast": {
		Key:  "network-fast",
		Name: "Rede Rápida",
		Description: "Desativa economia de energia do adaptador, habilita offloads (RSS/RSC) e " +
			"reduz espera de conexões encerradas. Foco em throughput máximo.",
		TweakIDs: []string{
			// Registry-backed (Fase 1)
			"rede.max-user-port",
			"rede.tcp-timed-wait-delay",
			// Custom tweaks via WMI (Fase 2, TODO)
			// "rede.nic-power-allow-disable",
			// "rede.rss-enable",
			// "rede.rsc-enable",
		},
		Caveats: "Aumenta consumo de energia do adaptador. Recomendado para trabalho intermitente com conexões intensas.",
	},
	"network-remote": {
		Key:  "network-remote",
		Name: "Trabalho Remoto",
		Description: "Otimiza para chamadas de vídeo, compartilhamento e conexões remotas: " +
			"reduz o tempo de espera de conexões TCP encerradas e desativa criação de thumbs.db em rede.",
		TweakIDs: []string{
			"rede.tcp-timed-wait-delay",
			"rede.thumbs-rede-off",
		},
		Caveats: "Adequado para uso com VPN, chamadas no Teams/Zoom e trabalho com servidores remotos.",
	},
	"network-dev": {
		Key:  "network-dev",
		Name: "Desenvolvimento",
		Description: "Para desenvolvimento com Git, Docker, APIs e transferências de arquivo: " +
			"foca em throughput e portas efêmeras, sem interferência de P2P em background.",
		TweakIDs: []string{
			"rede.sem-upload-de-atualizacoes",
			"rede.max-user-port",
		},
		Caveats: "Aumenta o número de portas para requisições concorrentes e corta upload de updates.",
	},
	"network-presentation": {
		Key:  "network-presentation",
		Name: "Apresentação",
		Description: "Para apresentações, reuniões online, streaming: minimiza interferências " +
			"de background (atualizações, downloads) e favorece a rede sem fio estável.",
		TweakIDs: []string{
			"rede.sem-upload-de-atualizacoes",
			"rede.network-throttling-off",
		},
		Caveats: "Exige seleção manual de Wi-Fi 5 GHz. Desativar este perfil restaura downloads em background.",
	},
}

// List retorna todos os perfis disponíveis, ordenados.
func List() []NetworkProfile {
	keys := []string{"network-fast", "network-remote", "network-dev", "network-presentation"}
	out := make([]NetworkProfile, 0, len(keys))
	for _, k := range keys {
		if p, ok := NetworkProfiles[k]; ok {
			out = append(out, p)
		}
	}
	return out
}

// Get retorna um perfil por chave, se existir.
func Get(key string) (NetworkProfile, bool) {
	p, ok := NetworkProfiles[key]
	return p, ok
}

// AllTweakIDs retorna a lista de todos os IDs de tweaks mencionados em qualquer perfil.
// Usado para validar que todos os IDs existem no catálogo antes de aplicar um perfil.
func AllTweakIDs() map[string]bool {
	seen := make(map[string]bool)
	for _, p := range NetworkProfiles {
		for _, id := range p.TweakIDs {
			seen[id] = true
		}
	}
	return seen
}
