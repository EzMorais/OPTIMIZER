package netdiag_test

import (
	"context"
	"net/netip"
	"testing"
	"time"

	"optimizer/internal/netdiag"
)

// fakeRede simula um caminho de rede com um MTU real conhecido: pacotes acima
// dele "precisam fragmentar", como um roteador PPPoE faria.
type fakeRede struct {
	mtuReal  int
	silencio bool // simula quem descarta calado em vez de responder
	semPing  bool // simula destino que bloqueia ICMP
	chamadas int
}

func (f *fakeRede) Ping(_ context.Context, _ netip.Addr, payload int, _ time.Duration) (netdiag.ProbeStatus, error) {
	f.chamadas++
	if f.semPing {
		return netdiag.ProbeTimeout, nil
	}
	if payload+netdiag.HeaderOverhead <= f.mtuReal {
		return netdiag.ProbeOK, nil
	}
	if f.silencio {
		return netdiag.ProbeTimeout, nil
	}
	return netdiag.ProbeTooBig, nil
}

var destino = netip.MustParseAddr("8.8.8.8")

func TestAchaMTUdePPPoE(t *testing.T) {
	rede := &fakeRede{mtuReal: 1492} // valor clássico de PPPoE
	rep, err := netdiag.ProbePathMTU(context.Background(), rede, "8.8.8.8", destino, netdiag.Options{})
	if err != nil {
		t.Fatalf("medição: %v", err)
	}
	if rep.PathMTU != 1492 {
		t.Fatalf("MTU medido = %d, queria 1492", rep.PathMTU)
	}
	if rep.SmallestFail != rep.LargestOK+1 {
		t.Fatalf("os limites não fecham: passou %d, falhou %d", rep.LargestOK, rep.SmallestFail)
	}
	if rep.CommandOK == "" || rep.CommandFail == "" {
		t.Fatal("faltou o comando de ping para o usuário conferir na mão")
	}
	// Busca binária em ~1400 possibilidades tem que caber em poucas tentativas.
	if len(rep.Steps) > 14 {
		t.Fatalf("%d tentativas — a busca binária regrediu", len(rep.Steps))
	}
}

func TestCaminhoLimpoResolveEmDuasMedicoes(t *testing.T) {
	rede := &fakeRede{mtuReal: 1500}
	rep, err := netdiag.ProbePathMTU(context.Background(), rede, "8.8.8.8", destino, netdiag.Options{})
	if err != nil {
		t.Fatalf("medição: %v", err)
	}
	if rep.PathMTU != netdiag.EthernetMTU {
		t.Fatalf("MTU medido = %d, queria 1500", rep.PathMTU)
	}
	if len(rep.Steps) != 2 {
		t.Fatalf("%d tentativas para o caso comum, queria 2", len(rep.Steps))
	}
}

func TestRoteadorQueDescartaCaladoAindaEMedido(t *testing.T) {
	rede := &fakeRede{mtuReal: 1400, silencio: true}
	rep, err := netdiag.ProbePathMTU(context.Background(), rede, "8.8.8.8", destino, netdiag.Options{})
	if err != nil {
		t.Fatalf("medição: %v", err)
	}
	if rep.PathMTU != 1400 {
		t.Fatalf("MTU medido = %d, queria 1400", rep.PathMTU)
	}
}

func TestDestinoQueBloqueiaPingNaoViraDiagnostico(t *testing.T) {
	rede := &fakeRede{semPing: true}
	rep, err := netdiag.ProbePathMTU(context.Background(), rede, "8.8.8.8", destino, netdiag.Options{})
	if err != nil {
		t.Fatalf("medição: %v", err)
	}
	if !rep.Blocked {
		t.Fatal("destino sem resposta deveria marcar Blocked")
	}
	if rep.PathMTU != 0 {
		t.Fatalf("inventou um MTU (%d) sem ter medido nada", rep.PathMTU)
	}

	d := netdiag.Diagnose(rep, &netdiag.Interface{Name: "Ethernet", MTU: 1500})
	if d.Verdict != netdiag.VerdictBlocked || d.SuggestedMTU != 0 {
		t.Fatalf("veredito = %v, sugestão = %d — não pode sugerir mudança sem medição", d.Verdict, d.SuggestedMTU)
	}
}

func TestDiagnosticoNaoMexeQuandoJaEstaCerto(t *testing.T) {
	rep := netdiag.Report{Host: "8.8.8.8", PathMTU: 1500, LargestOK: 1472}
	d := netdiag.Diagnose(rep, &netdiag.Interface{Name: "Ethernet", MTU: 1500})
	if d.Verdict != netdiag.VerdictIdeal {
		t.Fatalf("veredito = %v, queria ideal", d.Verdict)
	}
	if d.SuggestedMTU != 0 {
		t.Fatalf("sugeriu mudar para %d num caso em que não há nada a fazer", d.SuggestedMTU)
	}
}

func TestPacoteQueSomeCaladoEProblemaDeVerdade(t *testing.T) {
	rep := netdiag.Report{Host: "8.8.8.8", PathMTU: 1492, LargestOK: 1464, PMTUDWorking: false}
	d := netdiag.Diagnose(rep, &netdiag.Interface{Name: "Wi-Fi", MTU: 1500})
	if d.Verdict != netdiag.VerdictLower {
		t.Fatalf("veredito = %v, queria reduzir", d.Verdict)
	}
	if d.SuggestedMTU != 1492 {
		t.Fatalf("sugeriu %d, queria 1492", d.SuggestedMTU)
	}
}

func TestRedeQueAvisaCorretamenteViraAjusteOpcional(t *testing.T) {
	// Se o caminho devolve "precisa fragmentar", o Windows já se ajusta sozinho
	// por destino. Vender isso como problema urgente seria o padrão escuro da
	// categoria — o produto tem que dizer que é opcional.
	rep := netdiag.Report{Host: "8.8.8.8", PathMTU: 1480, LargestOK: 1452, PMTUDWorking: true}
	d := netdiag.Diagnose(rep, &netdiag.Interface{Name: "Ethernet", MTU: 1500})
	if d.Verdict != netdiag.VerdictLowerOptional {
		t.Fatalf("veredito = %v, queria reduzir_opcional", d.Verdict)
	}
	if d.SuggestedMTU != 1480 {
		t.Fatalf("sugeriu %d, queria 1480", d.SuggestedMTU)
	}
}

func TestMedicaoDistingueAvisoDeDescarteSilencioso(t *testing.T) {
	comAviso := &fakeRede{mtuReal: 1480}
	rep, err := netdiag.ProbePathMTU(context.Background(), comAviso, "8.8.8.8", destino, netdiag.Options{})
	if err != nil {
		t.Fatalf("medição: %v", err)
	}
	if !rep.PMTUDWorking {
		t.Fatal("a rede respondeu \"precisa fragmentar\" e a medição não registrou isso")
	}

	buracoNegro := &fakeRede{mtuReal: 1480, silencio: true}
	rep, err = netdiag.ProbePathMTU(context.Background(), buracoNegro, "8.8.8.8", destino, netdiag.Options{})
	if err != nil {
		t.Fatalf("medição: %v", err)
	}
	if rep.PMTUDWorking {
		t.Fatal("os pacotes sumiram calados e a medição achou que havia aviso")
	}
}

func TestAdaptadorLimitandoAMedicaoEHonestoSobreIsso(t *testing.T) {
	rep := netdiag.Report{Host: "8.8.8.8", PathMTU: 1400, LargestOK: 1372}
	d := netdiag.Diagnose(rep, &netdiag.Interface{Name: "VPN", MTU: 1400, Type: netdiag.IfTypeTunnel})
	if d.Verdict != netdiag.VerdictCapped {
		t.Fatalf("veredito = %v, queria limitado_pelo_adaptador", d.Verdict)
	}
	if d.SuggestedMTU != 0 {
		t.Fatal("não dá para sugerir aumento com base numa medição que o próprio adaptador limitou")
	}
}

func TestComandoDePingEDoJeitoQueOUsuarioDigita(t *testing.T) {
	got := netdiag.PingCommand("8.8.8.8", 1472)
	want := "ping -n 1 -f -l 1472 8.8.8.8"
	if got != want {
		t.Fatalf("comando = %q, queria %q", got, want)
	}
}
