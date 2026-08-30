package main

import (
	"context"
	"embed"
	"syscall"
	"unsafe"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

var (
	modKernel32       = syscall.NewLazyDLL("kernel32.dll")
	procCreateMutexW  = modKernel32.NewProc("CreateMutexW")
	procGetLastError  = modKernel32.NewProc("GetLastError")
	modUser32         = syscall.NewLazyDLL("user32.dll")
	procFindWindowW   = modUser32.NewProc("FindWindowW")
	procSetForeground = modUser32.NewProc("SetForegroundWindow")
)

const (
	ERROR_ALREADY_EXISTS = 183
)

func encerrar(ctx context.Context) {
	runtime.Quit(ctx)
}

func garantirInstanciaUnica() uintptr {
	mutexName, _ := syscall.UTF16PtrFromString("Local\\OptimizerApp_SingleInstance_Mutex")
	hMutex, _, _ := procCreateMutexW.Call(0, 1, uintptr(unsafe.Pointer(mutexName)))

	lastErr, _, _ := procGetLastError.Call()
	if lastErr == ERROR_ALREADY_EXISTS {
		// Traz a janela existente para o primeiro plano
		wName, _ := syscall.UTF16PtrFromString("Optimizer")
		hwnd, _, _ := procFindWindowW.Call(0, uintptr(unsafe.Pointer(wName)))
		if hwnd != 0 {
			procSetForeground.Call(hwnd)
		}
		return 0
	}
	return hMutex
}

func main() {
	hMutex := garantirInstanciaUnica()
	if hMutex == 0 {
		// Já existe uma instância/aba aberta — encerra a duplicata imediatamente
		return
	}

	app := NewApp()

	err := wails.Run(&options.App{
		Title:     "Optimizer",
		Width:     1280,
		Height:    820,
		MinWidth:  1024,
		MinHeight: 700,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 10, G: 14, B: 23, A: 255},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			BackdropType:         windows.Mica,
			Theme:                windows.Dark,
		},
	})

	if err != nil {
		println("Erro ao iniciar aplicação:", err.Error())
	}
}
