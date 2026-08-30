//go:build !windows

package systemdiag

// ObterTimerResolution retorna valores simulados para plataformas não-Windows.
func ObterTimerResolution() (TimerResolutionInfo, error) {
	return TimerResolutionInfo{
		MinResolutionMs:     15.625,
		MaxResolutionMs:     0.500,
		CurrentResolutionMs: 1.000,
		IsHighPrecision:     true,
	}, nil
}

// DefinirTimerResolution simula alteração em plataformas não-Windows.
func DefinirTimerResolution(desiredMs float64, set bool) (float64, error) {
	if !set {
		return 15.625, nil
	}
	return desiredMs, nil
}
