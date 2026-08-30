//go:build windows

package console

import (
	"os/exec"
	"syscall"
)

const createNoWindow = 0x08000000

// HideWindow impede que comandos auxiliares do app abram janelas de console.
func HideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: createNoWindow}
}
