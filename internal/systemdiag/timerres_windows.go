//go:build windows

package systemdiag

import (
	"context"
	"syscall"
	"unsafe"
)

var (
	modNtdll                   = syscall.NewLazyDLL("ntdll.dll")
	procNtQueryTimerResolution = modNtdll.NewProc("NtQueryTimerResolution")
	procNtSetTimerResolution   = modNtdll.NewProc("NtSetTimerResolution")
)

// ObterTimerResolution consulta a resolução atual e limites do timer no Windows.
func ObterTimerResolution() (TimerResolutionInfo, error) {
	var minRes, maxRes, curRes uint32

	r, _, _ := procNtQueryTimerResolution.Call(
		uintptr(unsafe.Pointer(&minRes)),
		uintptr(unsafe.Pointer(&maxRes)),
		uintptr(unsafe.Pointer(&curRes)),
	)
	if r != 0 {
		return TimerResolutionInfo{
			MinResolutionMs:     15.625,
			MaxResolutionMs:     0.500,
			CurrentResolutionMs: 1.000,
			IsHighPrecision:     false,
		}, syscall.Errno(r)
	}

	// Os valores retornados pelo NTDLL estão em unidades de 100ns (1ms = 10.000 unidades)
	minMs := float64(minRes) / 10000.0
	maxMs := float64(maxRes) / 10000.0
	curMs := float64(curRes) / 10000.0

	return TimerResolutionInfo{
		MinResolutionMs:     minMs,
		MaxResolutionMs:     maxMs,
		CurrentResolutionMs: curMs,
		IsHighPrecision:     curMs <= 1.0,
		IsPersistent:        VerificarPersistenciaTimer(context.Background()),
	}, nil
}

// DefinirTimerResolution solicita uma resolução mais alta (ex: 0.5 ms = 5000 unidades de 100ns).
func DefinirTimerResolution(desiredMs float64, set bool) (float64, error) {
	if desiredMs <= 0 {
		desiredMs = 0.5
	}
	units := uint32(desiredMs * 10000.0)
	var curRes uint32
	var setVal uintptr
	if set {
		setVal = 1
	}

	r, _, _ := procNtSetTimerResolution.Call(
		uintptr(units),
		setVal,
		uintptr(unsafe.Pointer(&curRes)),
	)
	if r != 0 {
		return 0, syscall.Errno(r)
	}

	return float64(curRes) / 10000.0, nil
}
