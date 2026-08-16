//go:build !windows

package netdiag

import "errors"

var errOnlyWindows = errors.New("netdiag: leitura/alteração de MTU disponível apenas no Windows")

func Interfaces() ([]Interface, error) { return nil, errOnlyWindows }
func GetMTU(uint64) (uint32, error)    { return 0, errOnlyWindows }
func SetMTU(uint64, uint32) error      { return errOnlyWindows }

type LiveMTU struct{}

var _ MTUController = LiveMTU{}

func (LiveMTU) Get(uint64) (uint32, error) { return 0, errOnlyWindows }
func (LiveMTU) Set(uint64, uint32) error   { return errOnlyWindows }
