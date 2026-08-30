package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"optimizer/internal/systemdiag"
)

func cmdTimerDaemon(args []string) error {
	fs := flag.NewFlagSet("timer-daemon", flag.ContinueOnError)
	res := fs.Float64("resolution", 0.5, "Resolução desejada em milissegundos")
	if err := fs.Parse(args); err != nil {
		return err
	}

	_, err := systemdiag.DefinirTimerResolution(*res, true)
	if err != nil {
		return fmt.Errorf("falha ao definir resolução do temporizador: %w", err)
	}

	// Mantém o processo vivo segurando a resolução
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig

	_, _ = systemdiag.DefinirTimerResolution(15.625, false)
	return nil
}
