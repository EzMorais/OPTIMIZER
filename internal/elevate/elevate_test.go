package elevate_test

import (
	"testing"

	"optimizer/internal/elevate"
)

func TestIsElevatedRetornaBooleano(t *testing.T) {
	// A função deve retornar sem pânico
	_ = elevate.IsElevated()
}
