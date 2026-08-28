package console_test

import (
	"testing"

	"optimizer/internal/console"
)

func TestEnableUTF8NaoEntraEmPanico(t *testing.T) {
	// A chamada deve ser segura mesmo rodando em ambiente sem console real anexado
	console.EnableUTF8()
}
