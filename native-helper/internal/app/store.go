package app

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
)

func userStateDir() (string, error) {
	if runtime.GOOS == "windows" {
		base := os.Getenv("LOCALAPPDATA")
		if base == "" {
			return "", errors.New("LOCALAPPDATA is not set")
		}
		return filepath.Join(base, "VKarmaniBrowserVPN"), nil
	}
	if d, err := os.UserConfigDir(); err == nil {
		return filepath.Join(d, "vkarmani-browser-vpn"), nil
	}
	return "", errors.New("cannot determine user config directory")
}

func statePath() (string, error) {
	dir, err := userStateDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return filepath.Join(dir, "state.bin"), nil
}

func loadState() (persistedState, error) {
	p, err := statePath()
	if err != nil {
		return persistedState{}, err
	}
	b, err := os.ReadFile(p)
	if errors.Is(err, os.ErrNotExist) {
		return persistedState{Version: 1}, nil
	}
	if err != nil {
		return persistedState{}, err
	}
	plain, err := unprotectBytes(b)
	if err != nil {
		return persistedState{}, err
	}
	var s persistedState
	if err := json.Unmarshal(plain, &s); err != nil {
		return persistedState{}, err
	}
	if s.Version == 0 {
		s.Version = 1
	}
	return s, nil
}

func saveState(s persistedState) error {
	p, err := statePath()
	if err != nil {
		return err
	}
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	cipher, err := protectBytes(b)
	if err != nil {
		return err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, cipher, 0o600); err != nil {
		return err
	}
	if err := os.Chmod(tmp, 0o600); err != nil && runtime.GOOS != "windows" {
		return err
	}
	return os.Rename(tmp, p)
}
