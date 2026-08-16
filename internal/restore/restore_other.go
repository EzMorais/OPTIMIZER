//go:build !windows

package restore

import "errors"

var errOnlyWindows = errors.New("restore: ponto de restauração existe apenas no Windows")

func Begin(string) (uint64, error) { return 0, errOnlyWindows }
func End(uint64) error             { return errOnlyWindows }
func Cancel(uint64) error          { return errOnlyWindows }
