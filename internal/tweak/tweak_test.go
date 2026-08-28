package tweak_test

import (
	"context"
	"testing"

	"optimizer/internal/tweak"
)

type dummyTweak struct {
	id   string
	name string
	desc string
	cat  tweak.Category
	risk tweak.Risk
}

func (d dummyTweak) ID() string               { return d.id }
func (d dummyTweak) Name() string             { return d.name }
func (d dummyTweak) Description() string      { return d.desc }
func (d dummyTweak) Category() tweak.Category { return d.cat }
func (d dummyTweak) Risk() tweak.Risk         { return d.risk }
func (d dummyTweak) RequiresRestart() bool    { return false }
func (d dummyTweak) Check(ctx context.Context) (tweak.CheckResult, error) {
	return tweak.CheckResult{State: tweak.StateNotApplied}, nil
}
func (d dummyTweak) Apply(ctx context.Context, dryRun bool) (tweak.ApplyResult, error) {
	return tweak.ApplyResult{}, nil
}
func (d dummyTweak) Revert(ctx context.Context, snap tweak.Snapshot, dryRun bool) error {
	return nil
}
func (d dummyTweak) Verify(ctx context.Context) (tweak.VerifyResult, error) {
	return tweak.VerifyResult{Success: true}, nil
}

func TestRegistryBasic(t *testing.T) {
	reg := tweak.NewRegistry()

	t1 := dummyTweak{id: "test.t1", name: "T1", cat: tweak.CategoryVisualEffects, risk: tweak.RiskLow}
	t2 := dummyTweak{id: "test.t2", name: "T2", cat: tweak.CategoryGaming, risk: tweak.RiskMedium}

	reg.Register(t1, tweak.Meta{Enabled: true, Profiles: tweak.ProfilePersonal, SortOrder: 20})
	reg.Register(t2, tweak.Meta{Enabled: true, Profiles: tweak.ProfileWork, SortOrder: 10})

	// Test Catalog sorting
	catalog := reg.Catalog()
	if len(catalog) != 2 {
		t.Fatalf("esperado 2 itens, obteve %d", len(catalog))
	}
	if catalog[0].Meta.ID != "test.t2" || catalog[1].Meta.ID != "test.t1" {
		t.Errorf("ordenação por SortOrder incorreta: %v, %v", catalog[0].Meta.ID, catalog[1].Meta.ID)
	}

	// Test ForProfile
	personal := reg.ForProfile(tweak.ProfilePersonal)
	if len(personal) != 1 || personal[0].Meta.ID != "test.t1" {
		t.Errorf("ForProfile(Personal) incorreto: %v", personal)
	}

	work := reg.ForProfile(tweak.ProfileWork)
	if len(work) != 1 || work[0].Meta.ID != "test.t2" {
		t.Errorf("ForProfile(Work) incorreto: %v", work)
	}
}

func TestParseProfile(t *testing.T) {
	cases := []struct {
		in     string
		want   tweak.Profile
		wantOk bool
	}{
		{"pessoal", tweak.ProfilePersonal, true},
		{"personal", tweak.ProfilePersonal, true},
		{"trabalho", tweak.ProfileWork, true},
		{"work", tweak.ProfileWork, true},
		{"ambos", tweak.ProfileBoth, true},
		{"invalido", 0, false},
	}

	for _, tc := range cases {
		got, ok := tweak.ParseProfile(tc.in)
		if ok != tc.wantOk || got != tc.want {
			t.Errorf("ParseProfile(%q) = (%v, %v), esperado (%v, %v)", tc.in, got, ok, tc.want, tc.wantOk)
		}
	}
}

func TestCategoryLabel(t *testing.T) {
	if tweak.CategoryLabel(tweak.CategoryGaming) != "Jogos" {
		t.Errorf("CategoryLabel incorreto para Gaming")
	}
	if tweak.CategoryLabel(tweak.CategoryStorage) != "Armazenamento" {
		t.Errorf("CategoryLabel incorreto para Storage")
	}
}
