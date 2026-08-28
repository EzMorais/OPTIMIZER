package diskdiag

import (
	"context"
	"testing"
)

func TestListarUnidadesFallback(t *testing.T) {
	drives, err := ListarUnidades(context.Background())
	if err != nil {
		t.Fatalf("ListarUnidades falhou: %v", err)
	}

	if len(drives) == 0 {
		t.Fatal("esperado pelo menos uma unidade listada")
	}

	c := drives[0]
	if c.Letter == "" {
		t.Error("letra da unidade não pode ser vazia")
	}
	if c.MediaType != "SSD" && c.MediaType != "HDD" && c.MediaType != "Desconhecido" {
		t.Errorf("MediaType inesperado: %s", c.MediaType)
	}
}
