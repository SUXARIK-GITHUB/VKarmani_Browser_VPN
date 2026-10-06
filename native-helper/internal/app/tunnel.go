package app

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

type tunnelProcess struct {
	mu         sync.Mutex
	cmd        *exec.Cmd
	port       int
	serverID   string
	configPath string
}

func findSingBox() (string, string, error) {
	var candidates []string
	if p := strings.TrimSpace(os.Getenv("VKARMANI_SING_BOX_PATH")); p != "" {
		candidates = append(candidates, p)
	}
	if exe, err := os.Executable(); err == nil {
		name := "sing-box"
		if runtime.GOOS == "windows" {
			name = "sing-box.exe"
		}
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), name))
	}
	if p, err := exec.LookPath("sing-box"); err == nil {
		candidates = append(candidates, p)
	}
	for _, p := range candidates {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			out, err := exec.Command(p, "version").CombinedOutput()
			if err != nil {
				return "", "", fmt.Errorf("sing-box version failed: %w", err)
			}
			line := strings.Split(strings.TrimSpace(string(out)), "\n")[0]
			return p, line, nil
		}
	}
	return "", "", errors.New("sing-box binary not found; run installer")
}

func allocatePort() (int, error) {
	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port, nil
}

func resolveServerIPv4(host string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	addrs, err := net.DefaultResolver.LookupIP(ctx, "ip4", host)
	if err == nil && len(addrs) > 0 {
		return addrs[0].String(), nil
	}
	providers := []struct{ h, ip, path string }{{"cloudflare-dns.com", "1.1.1.1", "/dns-query?name="}, {"dns.google", "8.8.8.8", "/resolve?name="}}
	for _, p := range providers {
		ips, e := resolveDoH(host, p.h, p.ip, p.path)
		if e == nil && len(ips) > 0 {
			return ips[0], nil
		}
	}
	return "", fmt.Errorf("cannot resolve VLESS server %s", host)
}

func buildSingBoxConfig(s Server, serverIP string, port int) map[string]any {
	return map[string]any{
		"log": map[string]any{"level": "warn", "timestamp": true},
		"dns": map[string]any{
			"servers": []any{map[string]any{
				"type": "https", "tag": "remote-doh", "server": "1.1.1.1", "server_port": 443, "path": "/dns-query",
				"tls":    map[string]any{"enabled": true, "server_name": "cloudflare-dns.com"},
				"detour": "proxy",
			}},
			"final": "remote-doh", "strategy": "ipv4_only",
		},
		"inbounds": []any{map[string]any{"type": "mixed", "tag": "browser-in", "listen": "127.0.0.1", "listen_port": port}},
		"outbounds": []any{map[string]any{
			"type": "vless", "tag": "proxy", "server": serverIP, "server_port": s.Port, "uuid": s.UUID, "flow": s.Flow,
			"tls": map[string]any{"enabled": true, "server_name": s.SNI,
				"utls":    map[string]any{"enabled": true, "fingerprint": s.Fingerprint},
				"reality": map[string]any{"enabled": true, "public_key": s.PublicKey, "short_id": s.ShortID},
			},
		}},
		"route": map[string]any{"final": "proxy", "auto_detect_interface": true},
	}
}

func (t *tunnelProcess) Stop() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.cmd != nil && t.cmd.Process != nil {
		_ = t.cmd.Process.Kill()
		_, _ = t.cmd.Process.Wait()
	}
	if t.configPath != "" {
		_ = os.Remove(t.configPath)
	}
	t.cmd = nil
	t.port = 0
	t.serverID = ""
	t.configPath = ""
}

func (t *tunnelProcess) Connected() (bool, int, string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.cmd != nil && t.cmd.Process != nil, t.port, t.serverID
}

func probeThroughLocalProxy(port int) error {
	proxyAddr := fmt.Sprintf("127.0.0.1:%d", port)
	dial := socksDialContext(proxyAddr)
	probes := []struct {
		URL  string
		Want int
	}{
		{"https://www.google.com/generate_204", 204},
		{"https://www.cloudflare.com/cdn-cgi/trace", 200},
	}
	var failures []string
	for _, p := range probes {
		u, _ := url.Parse(p.URL)
		tr := &http.Transport{
			Proxy:               nil,
			TLSClientConfig:     &tls.Config{MinVersion: tls.VersionTLS12, ServerName: u.Hostname()},
			TLSHandshakeTimeout: 6 * time.Second,
			ForceAttemptHTTP2:   true,
			DialContext:         dial,
		}
		client := &http.Client{Timeout: 10 * time.Second, Transport: tr}
		req, _ := http.NewRequest(http.MethodGet, p.URL, nil)
		req.Header.Set("User-Agent", "VKarmani-Browser-VPN/0.3")
		resp, err := client.Do(req)
		if err != nil {
			failures = append(failures, u.Hostname())
			continue
		}
		_ = resp.Body.Close()
		if resp.StatusCode == p.Want {
			return nil
		}
		failures = append(failures, fmt.Sprintf("%s:%d", u.Hostname(), resp.StatusCode))
	}
	return fmt.Errorf("VLESS data-plane probe failed (%s)", strings.Join(failures, ", "))
}

func (t *tunnelProcess) Start(s Server) (int, error) {
	t.Stop()
	path, _, err := findSingBox()
	if err != nil {
		return 0, err
	}
	serverIP, err := resolveServerIPv4(s.Host)
	if err != nil {
		return 0, err
	}
	port, err := allocatePort()
	if err != nil {
		return 0, err
	}
	cfg := buildSingBoxConfig(s, serverIP, port)
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return 0, err
	}
	dir, err := userStateDir()
	if err != nil {
		return 0, err
	}
	configPath := filepath.Join(dir, fmt.Sprintf("runtime-%d.json", os.Getpid()))
	if err = os.WriteFile(configPath, b, 0o600); err != nil {
		return 0, err
	}
	check := exec.Command(path, "check", "-c", configPath)
	if out, e := check.CombinedOutput(); e != nil {
		_ = os.Remove(configPath)
		return 0, fmt.Errorf("sing-box config check failed: %s", safeOneLine(string(out)))
	}
	devNull, e := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if e != nil {
		_ = os.Remove(configPath)
		return 0, e
	}
	cmd := exec.Command(path, "run", "-c", configPath)
	// Never inherit stdout/stderr: stdout is the Native Messaging protocol channel.
	// We intentionally do not persist browsing/error destinations to a local log.
	cmd.Stdout = devNull
	cmd.Stderr = devNull
	if err = cmd.Start(); err != nil {
		_ = devNull.Close()
		_ = os.Remove(configPath)
		return 0, err
	}
	_ = devNull.Close()
	deadline := time.Now().Add(7 * time.Second)
	for time.Now().Before(deadline) {
		c, e := net.DialTimeout("tcp4", fmt.Sprintf("127.0.0.1:%d", port), 250*time.Millisecond)
		if e == nil {
			c.Close()
			if probeErr := probeThroughLocalProxy(port); probeErr != nil {
				_ = cmd.Process.Kill()
				_, _ = cmd.Process.Wait()
				_ = os.Remove(configPath)
				return 0, probeErr
			}
			t.mu.Lock()
			t.cmd = cmd
			t.port = port
			t.serverID = s.ID
			_ = os.Remove(configPath)
			t.configPath = ""
			t.mu.Unlock()
			return port, nil
		}
		if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
			break
		}
		time.Sleep(150 * time.Millisecond)
	}
	_ = cmd.Process.Kill()
	_, _ = cmd.Process.Wait()
	_ = os.Remove(configPath)
	return 0, errors.New("sing-box did not open local proxy port")
}

func safeOneLine(s string) string {
	s = strings.ReplaceAll(strings.TrimSpace(s), "\n", " ")
	if len(s) > 300 {
		s = s[:300] + "…"
	}
	return s
}

func fetchViaTunnel(rawURL string, cfg BootstrapConfig, port int) ([]byte, string, error) {
	if port <= 0 {
		return nil, "", errors.New("tunnel not connected")
	}
	u, _ := urlParse(rawURL)
	tr := &http.Transport{Proxy: nil, TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12, ServerName: u.Hostname()}, TLSHandshakeTimeout: 8 * time.Second, ForceAttemptHTTP2: true}
	tr.DialContext = socksDialContext(fmt.Sprintf("127.0.0.1:%d", port))
	client := &http.Client{Timeout: 20 * time.Second, Transport: tr, CheckRedirect: sameHostRedirect(u.Hostname())}
	b, err := fetchWithClient(client, rawURL, cfg.MaxSubscriptionBytes)
	if err != nil {
		return nil, "", err
	}
	return b, "active-vpn", nil
}

// wrapper makes testing malformed URLs explicit without importing url in public call sites.
func urlParse(raw string) (*url.URL, error) { return url.Parse(raw) }
