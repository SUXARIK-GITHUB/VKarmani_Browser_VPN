package app

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

var defaultBootstrap = BootstrapConfig{
	SubscriptionHost:      "sub.vkarmani.com",
	DirectIPs:             []string{"81.163.22.205", "80.66.81.68"},
	AllowedServerSuffixes: []string{".vkarmani.com"},
	AllowedServerPorts:    []int{443, 9443},
	MaxSubscriptionBytes:  1 << 20,
	PublicRelayPrefixes:   []string{"https://proxy.cors.dev/"},
}

func loadBootstrap() (BootstrapConfig, error) {
	cfg := defaultBootstrap
	exe, err := os.Executable()
	if err != nil {
		return cfg, nil
	}
	b, err := os.ReadFile(filepath.Join(filepath.Dir(exe), "bootstrap.json"))
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		return cfg, err
	}
	if cfg.SubscriptionHost == "" || cfg.MaxSubscriptionBytes < 1024 {
		return defaultBootstrap, errors.New("invalid bootstrap.json")
	}
	return cfg, nil
}
