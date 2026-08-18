package tweaks_test

import (
	"context"
	"testing"

	"optimizer/internal/profiles"
	"optimizer/internal/tweak"
	"optimizer/internal/tweaks"
	"optimizer/internal/tweaks/regtweak"
	"optimizer/internal/winreg"
)

// Estes testes travam, em código, as regras de produto do catálogo — para uma
// otimização nova não entrar no app violando um princípio já decidido.

func TestIDsUnicosEOrdenados(t *testing.T) {
	reg := tweaks.Build(winreg.NewFake())
	visto := map[string]bool{}
	anterior := -1
	for _, e := range reg.Catalog() {
		if visto[e.Meta.ID] {
			t.Fatalf("ID duplicado no catálogo: %s", e.Meta.ID)
		}
		visto[e.Meta.ID] = true
		if e.Meta.SortOrder < anterior {
			t.Fatalf("catálogo fora de ordem em %s", e.Meta.ID)
		}
		anterior = e.Meta.SortOrder
	}
	if len(visto) == 0 {
		t.Fatal("catálogo vazio")
	}
}

func TestTodoItemDeMaquinaExigeAdministrador(t *testing.T) {
	reg := tweaks.Build(winreg.NewFake())
	for _, e := range reg.Catalog() {
		rt, ok := e.Tweak.(regtweak.Tweak)
		if !ok {
			continue
		}
		for _, v := range rt.Values {
			if v.Root == winreg.HKLM && !e.Meta.RequiresAdmin {
				t.Errorf("%s mexe em %s mas não está marcado como RequiresAdmin", e.Meta.ID, v.Human())
			}
		}
	}
}

func TestItemNaoRecomendadoPrecisaExplicarPorQue(t *testing.T) {
	// Princípio de produto: se não recomendamos por padrão, o usuário tem
	// direito de saber o motivo — é o oposto de marcar tudo para inflar a lista.
	reg := tweaks.Build(winreg.NewFake())
	for _, e := range reg.Catalog() {
		if !e.Meta.RecommendedDefault && e.Meta.Caveat == "" {
			t.Errorf("%s não é recomendado por padrão e não explica a ressalva", e.Meta.ID)
		}
	}
}

func TestTodoItemTemBaseTecnica(t *testing.T) {
	reg := tweaks.Build(winreg.NewFake())
	for _, e := range reg.Catalog() {
		if e.Meta.Evidence == "" {
			t.Errorf("%s não aponta a seção do catálogo técnico que o embasa", e.Meta.ID)
		}
		if e.Meta.Description == "" {
			t.Errorf("%s não tem explicação para o usuário", e.Meta.ID)
		}
	}
}

func TestSystemResponsivenessNuncaUsaZero(t *testing.T) {
	// Achado do catálogo técnico: o Windows força qualquer valor abaixo de 10
	// de volta para 20. Todo guia da internet manda usar 0 — que é justamente o
	// que não funciona. Este teste impede a regressão.
	reg := tweaks.Build(winreg.NewFake())
	e, ok := reg.Known("sistema.mmcss-system-responsiveness")
	if !ok {
		t.Skip("item não está no catálogo")
	}
	rt := e.(regtweak.Tweak)
	res, err := rt.Apply(context.Background(), true)
	if err != nil {
		t.Fatalf("Apply dry-run: %v", err)
	}
	if res.Detail == "" {
		t.Fatal("simulação sem descrição do que faria")
	}

	f := winreg.NewFake()
	reg2 := tweaks.Build(f)
	impl, _ := reg2.Known("sistema.mmcss-system-responsiveness")
	if _, err := impl.Apply(context.Background(), false); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	got, err := f.GetDWord(winreg.HKLM,
		`SOFTWARE\Microsoft\Windows NT\CurrentVersion\Multimedia\SystemProfile`, "SystemResponsiveness")
	if err != nil {
		t.Fatalf("lendo valor gravado: %v", err)
	}
	if got != 10 {
		t.Fatalf("SystemResponsiveness gravado = %d, tem que ser 10 (0 é grampeado para 20 pelo Windows)", got)
	}
}

func TestPerfilTrabalhoNaoTemItemExclusivoDePessoal(t *testing.T) {
	reg := tweaks.Build(winreg.NewFake())
	for _, e := range reg.ForProfile(tweak.ProfileWork) {
		if !e.Meta.Profiles.Has(tweak.ProfileWork) {
			t.Errorf("%s apareceu no perfil trabalho sem estar marcado para ele", e.Meta.ID)
		}
	}
}

func TestManifestoRemotoNaoInventaOtimizacao(t *testing.T) {
	// Propriedade de segurança central: metadado remoto pode reordenar,
	// reescrever texto e desativar item — nunca criar um tweak novo.
	reg := tweaks.Build(winreg.NewFake())
	antes := len(reg.Catalog())

	aplicados := reg.ApplyManifest([]tweak.Meta{
		{ID: "otimizacao.que.nao.existe", Enabled: true, DisplayName: "Malicioso"},
		{ID: "visual.transparency-effects", Enabled: false},
	})
	if aplicados != 1 {
		t.Fatalf("manifesto aplicou %d itens, queria 1 (o desconhecido tem que ser ignorado)", aplicados)
	}
	if _, ok := reg.Known("otimizacao.que.nao.existe"); ok {
		t.Fatal("o manifesto conseguiu registrar uma otimização que não existe no binário")
	}
	if len(reg.Catalog()) != antes-1 {
		t.Fatal("desativar um item pelo manifesto não surtiu efeito")
	}
}

func TestManifestoNaoRebaixaExigenciaDeAdministrador(t *testing.T) {
	reg := tweaks.Build(winreg.NewFake())
	const id = "rede.sem-upload-de-atualizacoes"
	reg.ApplyManifest([]tweak.Meta{{ID: id, Enabled: true, RequiresAdmin: false}})

	m, ok := reg.Meta(id)
	if !ok {
		t.Fatalf("item %s sumiu", id)
	}
	if !m.RequiresAdmin {
		t.Fatal("um manifesto conseguiu fazer o app achar que um tweak de máquina dispensa administrador")
	}
}

// TestPerfisDeRedeReferenciamTweaksExistentes garante que todo ID listado nos
// perfis de internal/profiles corresponda a um tweak registry-backed real no
// catálogo — um perfil apontando para um ID inexistente falharia silenciosamente
// em produção (engine.Apply pula IDs desconhecidos sem erro fatal).
func TestPerfisDeRedeReferenciamTweaksExistentes(t *testing.T) {
	reg := tweaks.Build(winreg.NewFake())
	for _, p := range profiles.List() {
		for _, id := range p.TweakIDs {
			if _, ok := reg.Known(id); !ok {
				t.Errorf("perfil %q referencia tweak %q, que não existe no catálogo registry-backed (pode ser um tweak WMI ainda não registrado — nesse caso, remover do perfil por enquanto)", p.Key, id)
			}
		}
	}
}
