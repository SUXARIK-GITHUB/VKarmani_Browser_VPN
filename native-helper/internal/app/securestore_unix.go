//go:build !windows

package app

func protectBytes(plain []byte) ([]byte, error)    { return plain, nil }
func unprotectBytes(cipher []byte) ([]byte, error) { return cipher, nil }
