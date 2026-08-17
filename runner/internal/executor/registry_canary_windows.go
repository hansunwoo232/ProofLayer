//go:build windows

package executor

import (
	"context"
	"fmt"
	"syscall"
	"unsafe"
)

const (
	hkeyCurrentUser = 0x80000001
	keyQueryValue   = 0x0001
	keySetValue     = 0x0002
	regSZ           = 1
	errorSuccess    = 0
	errorFileAbsent = 2
)

var (
	advapi32             = syscall.NewLazyDLL("advapi32.dll")
	procRegCreateKeyExW  = advapi32.NewProc("RegCreateKeyExW")
	procRegOpenKeyExW    = advapi32.NewProc("RegOpenKeyExW")
	procRegSetValueExW   = advapi32.NewProc("RegSetValueExW")
	procRegDeleteValueW  = advapi32.NewProc("RegDeleteValueW")
	procRegQueryValueExW = advapi32.NewProc("RegQueryValueExW")
	procRegCloseKey      = advapi32.NewProc("RegCloseKey")
)

func runRegistryCanary(ctx context.Context, valueName string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	handle, _, err := createRegistryKey(registryCanaryKeyPath)
	if err != nil {
		return err
	}
	defer closeRegistryKey(handle)

	name, err := syscall.UTF16PtrFromString(valueName)
	if err != nil {
		return fmt.Errorf("encode registry canary name: %w", err)
	}
	data, err := syscall.UTF16FromString(registryCanaryData)
	if err != nil {
		return fmt.Errorf("encode registry canary data: %w", err)
	}
	status, _, _ := procRegSetValueExW.Call(
		uintptr(handle),
		uintptr(unsafe.Pointer(name)),
		0,
		regSZ,
		uintptr(unsafe.Pointer(&data[0])),
		uintptr(len(data)*2),
	)
	if status != errorSuccess {
		return fmt.Errorf("set registry canary: %w", syscall.Errno(status))
	}
	return ctx.Err()
}

func cleanupRegistryCanary(ctx context.Context, valueName string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	handle, found, err := openRegistryKey(registryCanaryKeyPath, keySetValue|keyQueryValue)
	if err != nil || !found {
		return err
	}
	defer closeRegistryKey(handle)

	name, err := syscall.UTF16PtrFromString(valueName)
	if err != nil {
		return fmt.Errorf("encode registry canary name: %w", err)
	}
	status, _, _ := procRegDeleteValueW.Call(uintptr(handle), uintptr(unsafe.Pointer(name)))
	if status != errorSuccess && status != errorFileAbsent {
		return fmt.Errorf("remove registry canary: %w", syscall.Errno(status))
	}
	return ctx.Err()
}

func verifyRegistryCanaryAbsent(ctx context.Context, valueName string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	handle, found, err := openRegistryKey(registryCanaryKeyPath, keyQueryValue)
	if err != nil || !found {
		return err
	}
	defer closeRegistryKey(handle)

	name, err := syscall.UTF16PtrFromString(valueName)
	if err != nil {
		return fmt.Errorf("encode registry canary name: %w", err)
	}
	var size uint32
	status, _, _ := procRegQueryValueExW.Call(
		uintptr(handle),
		uintptr(unsafe.Pointer(name)),
		0,
		0,
		0,
		uintptr(unsafe.Pointer(&size)),
	)
	if status == errorFileAbsent {
		return ctx.Err()
	}
	if status != errorSuccess {
		return fmt.Errorf("verify registry canary absence: %w", syscall.Errno(status))
	}
	return ErrArtifactPresent
}

func createRegistryKey(path string) (syscall.Handle, uint32, error) {
	pathPointer, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 0, 0, fmt.Errorf("encode registry path: %w", err)
	}
	var handle syscall.Handle
	var disposition uint32
	status, _, _ := procRegCreateKeyExW.Call(
		hkeyCurrentUser,
		uintptr(unsafe.Pointer(pathPointer)),
		0,
		0,
		0,
		keySetValue|keyQueryValue,
		0,
		uintptr(unsafe.Pointer(&handle)),
		uintptr(unsafe.Pointer(&disposition)),
	)
	if status != errorSuccess {
		return 0, 0, fmt.Errorf("open registry canary path: %w", syscall.Errno(status))
	}
	return handle, disposition, nil
}

func openRegistryKey(path string, access uint32) (syscall.Handle, bool, error) {
	pathPointer, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 0, false, fmt.Errorf("encode registry path: %w", err)
	}
	var handle syscall.Handle
	status, _, _ := procRegOpenKeyExW.Call(
		hkeyCurrentUser,
		uintptr(unsafe.Pointer(pathPointer)),
		0,
		uintptr(access),
		uintptr(unsafe.Pointer(&handle)),
	)
	if status == errorFileAbsent {
		return 0, false, nil
	}
	if status != errorSuccess {
		return 0, false, fmt.Errorf("open registry canary path: %w", syscall.Errno(status))
	}
	return handle, true, nil
}

func closeRegistryKey(handle syscall.Handle) {
	_, _, _ = procRegCloseKey.Call(uintptr(handle))
}
