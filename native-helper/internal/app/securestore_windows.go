//go:build windows

package app

import (
	"fmt"
	"syscall"
	"unsafe"
)

type dataBlob struct {
	cbData uint32
	pbData *byte
}

var (
	crypt32            = syscall.NewLazyDLL("crypt32.dll")
	kernel32           = syscall.NewLazyDLL("kernel32.dll")
	procCryptProtect   = crypt32.NewProc("CryptProtectData")
	procCryptUnprotect = crypt32.NewProc("CryptUnprotectData")
	procLocalFree      = kernel32.NewProc("LocalFree")
)

func blobFromBytes(b []byte) dataBlob {
	if len(b) == 0 {
		return dataBlob{}
	}
	return dataBlob{cbData: uint32(len(b)), pbData: &b[0]}
}

func bytesFromBlob(blob dataBlob) []byte {
	if blob.cbData == 0 || blob.pbData == nil {
		return nil
	}
	return append([]byte(nil), unsafe.Slice(blob.pbData, blob.cbData)...)
}

func protectBytes(plain []byte) ([]byte, error) {
	in := blobFromBytes(plain)
	var out dataBlob
	const cryptProtectUIForbidden = 0x1
	r1, _, e := procCryptProtect.Call(
		uintptr(unsafe.Pointer(&in)),
		0,
		0,
		0,
		0,
		cryptProtectUIForbidden,
		uintptr(unsafe.Pointer(&out)),
	)
	if r1 == 0 {
		return nil, fmt.Errorf("CryptProtectData failed: %v", e)
	}
	defer procLocalFree.Call(uintptr(unsafe.Pointer(out.pbData)))
	return bytesFromBlob(out), nil
}

func unprotectBytes(cipher []byte) ([]byte, error) {
	in := blobFromBytes(cipher)
	var out dataBlob
	const cryptProtectUIForbidden = 0x1
	r1, _, e := procCryptUnprotect.Call(
		uintptr(unsafe.Pointer(&in)),
		0,
		0,
		0,
		0,
		cryptProtectUIForbidden,
		uintptr(unsafe.Pointer(&out)),
	)
	if r1 == 0 {
		return nil, fmt.Errorf("CryptUnprotectData failed: %v", e)
	}
	defer procLocalFree.Call(uintptr(unsafe.Pointer(out.pbData)))
	return bytesFromBlob(out), nil
}
