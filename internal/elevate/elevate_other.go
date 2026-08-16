//go:build !windows

package elevate

// IsElevated fora do Windows é sempre falso: o conceito não se aplica.
func IsElevated() bool { return false }
