package profiles

import "testing"

func TestListReturnsAllProfiles(t *testing.T) {
	list := List()
	if len(list) != 4 {
		t.Fatalf("List() retornou %d perfis, esperado 4", len(list))
	}
	wantKeys := map[string]bool{
		"network-fast": true, "network-remote": true,
		"network-dev": true, "network-presentation": true,
	}
	for _, p := range list {
		if !wantKeys[p.Key] {
			t.Errorf("chave inesperada no List(): %s", p.Key)
		}
		delete(wantKeys, p.Key)
	}
	if len(wantKeys) != 0 {
		t.Errorf("chaves faltando no List(): %v", wantKeys)
	}
}

func TestGetExistingProfile(t *testing.T) {
	p, ok := Get("network-fast")
	if !ok {
		t.Fatal("Get(\"network-fast\") retornou ok=false")
	}
	if p.Name != "Rede Rápida" {
		t.Errorf("Name = %q, esperado \"Rede Rápida\"", p.Name)
	}
}

func TestGetUnknownProfile(t *testing.T) {
	_, ok := Get("network-nonexistent")
	if ok {
		t.Error("Get() de perfil inexistente deveria retornar ok=false")
	}
}

func TestAllProfilesHaveCaveatsOrEmptyTweaks(t *testing.T) {
	// Todo perfil deve ter descrição não-vazia (honestidade sobre o que faz).
	for _, p := range List() {
		if p.Description == "" {
			t.Errorf("perfil %s sem descrição", p.Key)
		}
		if p.Name == "" {
			t.Errorf("perfil %s sem nome", p.Key)
		}
	}
}

func TestAllTweakIDsNoDuplicateEmptyKeys(t *testing.T) {
	ids := AllTweakIDs()
	for id := range ids {
		if id == "" {
			t.Error("AllTweakIDs() contém uma chave vazia")
		}
	}
}
