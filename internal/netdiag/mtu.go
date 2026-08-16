// Package netdiag mede a rede antes de mexer nela.
//
// O MTU é o caso exemplar da regra de ouro do produto: mexer em MTU "no chute"
// quase sempre piora a internet (ver docs/catalogo/rede.md, seção 5, e a lista
// de armadilhas). Por isso aqui não existe botão "otimizar MTU" — existe uma
// MEDIÇÃO, e só depois dela um ajuste com valor justificado.
//
// Como a medição funciona (é a mesma técnica do `ping -f -l` do Windows):
// manda-se um eco ICMP com o sinalizador "não fragmentar" (DF) ligado e um
// tamanho de dados fixo. Se algum equipamento no caminho precisar quebrar esse
// pacote para repassá-lo, ele não pode — e devolve "pacote precisa ser
// fragmentado" (ou simplesmente descarta). Achando por busca binária o maior
// tamanho que ainda passa inteiro, o MTU do caminho é esse tamanho + 28 bytes
// (20 do cabeçalho IPv4 + 8 do cabeçalho ICMP) — exatamente a conta que o
// `ping -f -l` assume.
package netdiag

import (
	"context"
	"fmt"
	"net/netip"
	"time"
)

// HeaderOverhead é o que o `ping -f -l N` NÃO conta no N: 20 bytes de cabeçalho
// IPv4 + 8 do cabeçalho ICMP. MTU = N + 28.
const HeaderOverhead = 28

// Limites da busca. 1472 é o maior payload que cabe num quadro Ethernet padrão
// (1500 - 28); acima disso só em rede com quadros jumbo, que não é o caso de
// nenhum caminho até a internet.
const (
	MinPayload      = 64
	MaxPayload      = 1472
	EthernetMTU     = 1500
	MinIPv4MTU      = 576
	MinWindowsIPMTU = 576
)

type ProbeStatus int

const (
	ProbeOK ProbeStatus = iota
	// ProbeTooBig: alguém no caminho (ou a própria pilha local) disse que o
	// pacote precisaria ser fragmentado.
	ProbeTooBig
	// ProbeTimeout: sem resposta. Muitos roteadores descartam calados em vez de
	// avisar — para a busca conta como "não passou".
	ProbeTimeout
	// ProbeUnreachable: destino inalcançável, rota inexistente etc.
	ProbeUnreachable
)

func (s ProbeStatus) String() string {
	switch s {
	case ProbeOK:
		return "passou"
	case ProbeTooBig:
		return "precisaria fragmentar"
	case ProbeTimeout:
		return "sem resposta"
	case ProbeUnreachable:
		return "destino inalcançável"
	}
	return "desconhecido"
}

// Prober envia um eco ICMP com DF ligado. Interface para o teste poder rodar
// sem tocar a rede real.
type Prober interface {
	Ping(ctx context.Context, dst netip.Addr, payload int, timeout time.Duration) (ProbeStatus, error)
}

// Step é uma tentativa da busca — a UI mostra isso como "o que a gente testou",
// junto do comando equivalente para o usuário conferir no Prompt de Comando.
type Step struct {
	Payload int
	MTU     int
	Status  ProbeStatus
	Command string
}

type Options struct {
	Timeout time.Duration
	Retries int
	Min     int
	Max     int
}

func (o Options) withDefaults() Options {
	if o.Timeout <= 0 {
		o.Timeout = 1200 * time.Millisecond
	}
	if o.Retries <= 0 {
		o.Retries = 2
	}
	if o.Min <= 0 {
		o.Min = MinPayload
	}
	if o.Max <= 0 {
		o.Max = MaxPayload
	}
	return o
}

// PingCommand devolve o comando que o usuário pode digitar no Prompt de Comando
// para conferir, na mão, exatamente o mesmo teste que o app fez.
func PingCommand(host string, payload int) string {
	return fmt.Sprintf("ping -n 1 -f -l %d %s", payload, host)
}

// Report é o resultado da medição.
type Report struct {
	Host string
	Addr netip.Addr

	// Blocked: o destino não respondeu nem no menor tamanho — não dá para medir
	// por aqui (muitos servidores e alguns provedores bloqueiam ICMP).
	Blocked bool

	// LargestOK / SmallestFail são os dois tamanhos que delimitam a resposta.
	LargestOK    int
	SmallestFail int
	// PathMTU é LargestOK + 28. 0 quando não foi possível medir.
	PathMTU int

	// PMTUDWorking diz se o equipamento que barrou o pacote AVISOU ("precisa
	// fragmentar") em vez de descartar calado. Essa distinção muda tudo no
	// diagnóstico: quando há aviso, o próprio Windows já reduz o tamanho para
	// aquele destino sozinho (Path MTU Discovery), e mexer no adaptador é
	// opcional. Quando o pacote some calado, o Windows não tem como descobrir —
	// é o caso em que ajustar o MTU resolve travamento de verdade.
	PMTUDWorking bool

	Steps []Step

	// CommandOK / CommandFail são os dois comandos decisivos, para conferência
	// manual: o maior que passa e o menor que já não passa.
	CommandOK   string
	CommandFail string
}

// ProbePathMTU acha, por busca binária, o maior pacote que atravessa o caminho
// inteiro sem ser fragmentado.
func ProbePathMTU(ctx context.Context, p Prober, host string, addr netip.Addr, opts Options) (Report, error) {
	opts = opts.withDefaults()
	rep := Report{Host: host, Addr: addr}

	probe := func(payload int) (ProbeStatus, error) {
		var last ProbeStatus
		var err error
		// Repetir antes de concluir "não passou": uma perda solta de pacote não
		// pode virar um diagnóstico errado de MTU.
		for i := 0; i < opts.Retries; i++ {
			last, err = p.Ping(ctx, addr, payload, opts.Timeout)
			if err != nil {
				return last, err
			}
			if last == ProbeOK {
				break
			}
		}
		rep.Steps = append(rep.Steps, Step{
			Payload: payload,
			MTU:     payload + HeaderOverhead,
			Status:  last,
			Command: PingCommand(host, payload),
		})
		return last, nil
	}

	// 1) O destino responde a ping? Sem isso, nada abaixo faz sentido.
	base, err := probe(opts.Min)
	if err != nil {
		return rep, err
	}
	if base != ProbeOK {
		rep.Blocked = true
		return rep, nil
	}

	// 2) Caminho já aceita o quadro Ethernet cheio? Caso mais comum — resolve
	// em duas medições, sem fazer o usuário esperar a busca inteira.
	top, err := probe(opts.Max)
	if err != nil {
		return rep, err
	}
	if top == ProbeOK {
		rep.LargestOK = opts.Max
		rep.PathMTU = opts.Max + HeaderOverhead
		rep.CommandOK = PingCommand(host, opts.Max)
		return rep, nil
	}

	// 3) Busca binária entre o que passou e o que não passou.
	lo, hi := opts.Min, opts.Max // lo sempre passa, hi sempre falha
	for hi-lo > 1 {
		if err := ctx.Err(); err != nil {
			return rep, err
		}
		mid := lo + (hi-lo)/2
		st, err := probe(mid)
		if err != nil {
			return rep, err
		}
		if st == ProbeOK {
			lo = mid
		} else {
			hi = mid
		}
	}

	rep.LargestOK = lo
	rep.SmallestFail = hi
	rep.PathMTU = lo + HeaderOverhead
	rep.CommandOK = PingCommand(host, lo)
	rep.CommandFail = PingCommand(host, hi)
	rep.PMTUDWorking = rep.statusOf(hi) == ProbeTooBig
	return rep, nil
}

// statusOf devolve o resultado registrado para um tamanho já testado.
func (r Report) statusOf(payload int) ProbeStatus {
	for i := len(r.Steps) - 1; i >= 0; i-- {
		if r.Steps[i].Payload == payload {
			return r.Steps[i].Status
		}
	}
	return ProbeTimeout
}

type Verdict string

const (
	// VerdictIdeal: nada a fazer. É o resultado esperado na maioria das máquinas.
	VerdictIdeal Verdict = "ideal"
	// VerdictLower: o adaptador está acima do que o caminho aceita E o pacote
	// grande some calado — o Windows não tem como se ajustar sozinho. É o único
	// caso em que ajustar o MTU resolve travamento de verdade.
	VerdictLower Verdict = "reduzir"
	// VerdictLowerOptional: o adaptador está acima do que o caminho aceita, mas
	// a rede avisa corretamente — o Windows já se ajusta sozinho por destino.
	// Ajustar é opcional e o ganho costuma ser pequeno; dizer o contrário seria
	// vender problema inventado.
	VerdictLowerOptional Verdict = "reduzir_opcional"
	// VerdictCapped: o adaptador é o próprio teto da medição; não dá para saber
	// se o caminho aceitaria mais sem antes aumentar o adaptador.
	VerdictCapped Verdict = "limitado_pelo_adaptador"
	// VerdictBlocked: destino não responde a ping, medição impossível.
	VerdictBlocked Verdict = "sem_resposta"
)

// Diagnosis é o texto honesto que a UI mostra. Nenhum campo aqui promete
// "internet mais rápida": MTU errado causa fragmentação e perda, não lentidão
// genérica, e MTU certo não acelera nada além do que já estava certo.
type Diagnosis struct {
	Verdict     Verdict
	Summary     string
	Explanation string
	// SuggestedMTU é 0 quando não há nada a mudar.
	SuggestedMTU uint32
}

// Diagnose cruza a medição com o MTU configurado no adaptador de saída.
func Diagnose(rep Report, iface *Interface) Diagnosis {
	if rep.Blocked {
		return Diagnosis{
			Verdict: VerdictBlocked,
			Summary: fmt.Sprintf("Não deu para medir: %s não respondeu ao ping.", rep.Host),
			Explanation: "Muitos servidores e alguns provedores bloqueiam ping por padrão. " +
				"Isso não é problema no seu computador — só significa que a medição precisa de outro destino. " +
				"Tente de novo com outro endereço (por exemplo 1.1.1.1).",
		}
	}

	if iface == nil {
		return Diagnosis{
			Verdict: VerdictIdeal,
			Summary: fmt.Sprintf("O caminho até %s aceita pacotes de até %d bytes (MTU %d).", rep.Host, rep.LargestOK, rep.PathMTU),
			Explanation: "Não foi possível identificar qual adaptador de rede está sendo usado para chegar nesse destino, " +
				"então não há o que comparar nem ajustar.",
		}
	}

	configured := int(iface.MTU)
	switch {
	case rep.PathMTU < configured && rep.PMTUDWorking:
		return Diagnosis{
			Verdict: VerdictLowerOptional,
			Summary: fmt.Sprintf("O adaptador %q está em MTU %d e o caminho até %s aceita %d — mas a rede avisa "+
				"corretamente quando o pacote é grande demais, e o Windows já se ajusta sozinho.",
				iface.Name, configured, rep.Host, rep.PathMTU),
			Explanation: "Isso é normal e não é defeito: quando um equipamento no caminho devolve o aviso \"precisa " +
				"fragmentar\", o Windows passa a mandar pacotes menores para aquele destino automaticamente. " +
				"Baixar o MTU do adaptador aqui é opcional — ajuda em programas ou aparelhos antigos que lidam mal com " +
				"esse aviso, e economiza o primeiro pacote perdido de cada conexão nova. " +
				"Não espere ganho de velocidade: quem promete isso está vendendo problema inventado.",
			SuggestedMTU: uint32(rep.PathMTU),
		}

	case rep.PathMTU < configured:
		return Diagnosis{
			Verdict: VerdictLower,
			Summary: fmt.Sprintf("Seu adaptador %q está em MTU %d, o caminho até %s só aceita %d — e os pacotes grandes "+
				"estão sumindo sem aviso nenhum.",
				iface.Name, configured, rep.Host, rep.PathMTU),
			Explanation: "Esse é o caso que dá problema de verdade. Como nada no caminho avisa que o pacote era grande " +
				"demais, o Windows não tem como descobrir sozinho e continua mandando do mesmo tamanho — os pacotes somem " +
				"calados. O sintoma não é \"internet lenta\" em geral: são sites que travam no meio do carregamento, " +
				"download que para no fim, VPN instável, upload de arquivo grande que falha. " +
				"Ajustar o adaptador para o valor medido faz o Windows já mandar do tamanho certo desde o começo.",
			SuggestedMTU: uint32(rep.PathMTU),
		}

	case rep.PathMTU == configured && configured >= EthernetMTU:
		return Diagnosis{
			Verdict: VerdictIdeal,
			Summary: fmt.Sprintf("MTU já está certo: adaptador em %d e o caminho até %s aceita %d.",
				configured, rep.Host, rep.PathMTU),
			Explanation: "Não há nada a ajustar aqui. Qualquer app que ofereça um \"MTU otimizado\" diferente disso " +
				"estaria mexendo por mexer.",
		}

	case rep.PathMTU == configured:
		return Diagnosis{
			Verdict: VerdictCapped,
			Summary: fmt.Sprintf("O adaptador %q está em MTU %d e todos os pacotes desse tamanho passaram.", iface.Name, configured),
			Explanation: "O teste não consegue enxergar acima do valor que o próprio adaptador permite: o Windows já recusa " +
				"o pacote antes de ele sair da máquina. Se esse valor for menor que 1500, ele pode ter sido definido de " +
				"propósito (VPN, conexão PPPoE, driver do fabricante) — nesses casos, aumentar costuma piorar. " +
				"Só vale investigar se você não reconhece o motivo desse valor.",
		}

	default: // rep.PathMTU > configured — não deveria acontecer, o adaptador é o teto
		return Diagnosis{
			Verdict: VerdictCapped,
			Summary: fmt.Sprintf("Medição inconsistente: caminho aceitou %d, mas o adaptador %q está em %d.",
				rep.PathMTU, iface.Name, configured),
			Explanation: "Isso costuma indicar que o tráfego saiu por outro adaptador (VPN, máquina virtual, segunda placa). " +
				"Nada foi alterado.",
		}
	}
}
