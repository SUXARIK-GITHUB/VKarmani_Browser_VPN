package app

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	uuidRe       = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	realityKeyRe = regexp.MustCompile(`^[A-Za-z0-9_-]{40,64}$`)
	shortIDRe    = regexp.MustCompile(`(?i)^[0-9a-f]{2,16}$`)
)

func validateSubscriptionURL(raw string, cfg BootstrapConfig) (*url.URL, error) {
	if len(raw) > 4096 {
		return nil, errors.New("subscription URL is too long")
	}
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return nil, errors.New("invalid subscription URL")
	}
	if u.Scheme != "https" || !strings.EqualFold(u.Hostname(), cfg.SubscriptionHost) {
		return nil, fmt.Errorf("subscription must be https://%s/...", cfg.SubscriptionHost)
	}
	if u.User != nil || u.Fragment != "" || (u.Port() != "" && u.Port() != "443") {
		return nil, errors.New("subscription URL contains unsupported authority/fragment")
	}
	if u.Path == "" || u.Path == "/" {
		return nil, errors.New("subscription URL must contain a user path")
	}
	return u, nil
}

func sameHostRedirect(host string) func(*http.Request, []*http.Request) error {
	return func(req *http.Request, via []*http.Request) error {
		if len(via) >= 3 {
			return errors.New("too many redirects")
		}
		if req.URL.Scheme != "https" || !strings.EqualFold(req.URL.Hostname(), host) {
			return errors.New("cross-host redirect blocked")
		}
		return nil
	}
}

func newTransport(serverName string, dialAddr string) *http.Transport {
	d := &net.Dialer{Timeout: 6 * time.Second, KeepAlive: 30 * time.Second}
	t := &http.Transport{
		Proxy:               nil,
		TLSClientConfig:     &tls.Config{MinVersion: tls.VersionTLS12, ServerName: serverName},
		TLSHandshakeTimeout: 8 * time.Second,
		IdleConnTimeout:     30 * time.Second,
		ForceAttemptHTTP2:   true,
	}
	if dialAddr != "" {
		t.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			return d.DialContext(ctx, network, net.JoinHostPort(dialAddr, "443"))
		}
	} else {
		t.DialContext = d.DialContext
	}
	return t
}

func fetchWithClient(client *http.Client, rawURL string, max int64) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "VKarmani-Browser-VPN/0.3")
	req.Header.Set("Accept", "text/plain, application/json;q=0.9, */*;q=0.1")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("subscription HTTP status %d", resp.StatusCode)
	}
	lr := io.LimitReader(resp.Body, max+1)
	b, err := io.ReadAll(lr)
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > max {
		return nil, errors.New("subscription response too large")
	}
	return b, nil
}

func fetchDirect(rawURL string, cfg BootstrapConfig) ([]byte, string, error) {
	u, _ := url.Parse(rawURL)
	client := &http.Client{
		Timeout:       15 * time.Second,
		Transport:     newTransport(u.Hostname(), ""),
		CheckRedirect: sameHostRedirect(u.Hostname()),
	}
	b, err := fetchWithClient(client, rawURL, cfg.MaxSubscriptionBytes)
	if err != nil {
		return nil, "", err
	}
	return b, "direct", nil
}

func fetchDirectIP(rawURL string, cfg BootstrapConfig, ip string, label string) ([]byte, string, error) {
	u, _ := url.Parse(rawURL)
	if net.ParseIP(ip) == nil {
		return nil, "", errors.New("invalid bootstrap IP")
	}
	client := &http.Client{
		Timeout:       15 * time.Second,
		Transport:     newTransport(u.Hostname(), ip),
		CheckRedirect: sameHostRedirect(u.Hostname()),
	}
	b, err := fetchWithClient(client, rawURL, cfg.MaxSubscriptionBytes)
	if err != nil {
		return nil, "", err
	}
	return b, label, nil
}

type dohResponse struct {
	Status int `json:"Status"`
	Answer []struct {
		Type int    `json:"type"`
		Data string `json:"data"`
	} `json:"Answer"`
}

func resolveDoH(host string, endpointHost, endpointIP, endpointPath string) ([]string, error) {
	q := "https://" + endpointHost + endpointPath + url.QueryEscape(host) + "&type=A"
	client := &http.Client{
		Timeout:   8 * time.Second,
		Transport: newTransport(endpointHost, endpointIP),
	}
	req, _ := http.NewRequest(http.MethodGet, q, nil)
	req.Header.Set("Accept", "application/dns-json")
	req.Header.Set("User-Agent", "VKarmani-Browser-VPN/0.3")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("DoH HTTP %d", resp.StatusCode)
	}
	var d dohResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 256<<10)).Decode(&d); err != nil {
		return nil, err
	}
	if d.Status != 0 {
		return nil, fmt.Errorf("DoH status %d", d.Status)
	}
	var ips []string
	for _, a := range d.Answer {
		if a.Type == 1 && net.ParseIP(a.Data) != nil && !strings.Contains(a.Data, ":") {
			ips = append(ips, a.Data)
		}
	}
	if len(ips) == 0 {
		return nil, errors.New("DoH returned no IPv4")
	}
	return ips, nil
}

func fetchViaDoH(rawURL string, cfg BootstrapConfig) ([]byte, string, error) {
	u, _ := url.Parse(rawURL)
	providers := []struct {
		host, ip, path, label string
	}{
		{"cloudflare-dns.com", "1.1.1.1", "/dns-query?name=", "doh-cloudflare"},
		{"dns.google", "8.8.8.8", "/resolve?name=", "doh-google"},
	}
	var errs []string
	for _, p := range providers {
		ips, err := resolveDoH(u.Hostname(), p.host, p.ip, p.path)
		if err != nil {
			errs = append(errs, p.label+": "+err.Error())
			continue
		}
		for _, ip := range ips {
			b, source, err := fetchDirectIP(rawURL, cfg, ip, p.label+":"+ip)
			if err == nil {
				return b, source, nil
			}
			errs = append(errs, p.label+":"+ip+": "+err.Error())
		}
	}
	return nil, "", errors.New(strings.Join(errs, "; "))
}

func normalizePublicRelayPrefix(raw string) (string, error) {
	prefix := strings.TrimSpace(raw)
	if prefix == "" {
		return "", errors.New("empty public relay prefix")
	}
	u, err := url.Parse(prefix)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" {
		return "", errors.New("public relay must be an HTTPS URL")
	}
	if u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return "", errors.New("public relay URL must not contain credentials, query, or fragment")
	}
	if u.Port() != "" && u.Port() != "443" {
		return "", errors.New("public relay must use HTTPS port 443")
	}
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	return prefix, nil
}

func buildPublicRelayURL(prefix, target string) (string, string, error) {
	prefix, err := normalizePublicRelayPrefix(prefix)
	if err != nil {
		return "", "", err
	}
	u, err := url.Parse(prefix)
	if err != nil {
		return "", "", err
	}
	return prefix + target, u.Hostname(), nil
}

// fetchViaPublicRelays is deliberately the final online fallback. The full
// subscription URL is disclosed to the relay provider, so first-party routes
// and an already-active VPN tunnel are always attempted before this function.
// The returned body is still parsed through the normal strict VKarmani VLESS
// validation before it can replace Last Known Good state.
func fetchViaPublicRelays(rawURL string, cfg BootstrapConfig) ([]byte, string, error) {
	if _, err := validateSubscriptionURL(rawURL, cfg); err != nil {
		return nil, "", err
	}
	var errs []string
	for _, prefix := range cfg.PublicRelayPrefixes {
		relayURL, relayHost, err := buildPublicRelayURL(prefix, rawURL)
		if err != nil {
			errs = append(errs, "invalid relay config")
			continue
		}
		client := &http.Client{
			Timeout:       15 * time.Second,
			Transport:     newTransport(relayHost, ""),
			CheckRedirect: sameHostRedirect(relayHost),
		}
		b, err := fetchWithClient(client, relayURL, cfg.MaxSubscriptionBytes)
		if err == nil {
			return b, "public-relay:" + relayHost, nil
		}
		errs = append(errs, relayHost)
	}
	if len(errs) == 0 {
		return nil, "", errors.New("no public relay configured")
	}
	return nil, "", errors.New("public relay fetch failed: " + strings.Join(errs, ", "))
}

func decodeMaybeBase64(s string) []string {
	clean := strings.Map(func(r rune) rune {
		if r == '\r' || r == '\n' || r == '\t' || r == ' ' {
			return -1
		}
		return r
	}, s)
	if clean == "" {
		return nil
	}
	encs := []*base64.Encoding{
		base64.StdEncoding,
		base64.RawStdEncoding,
		base64.URLEncoding,
		base64.RawURLEncoding,
	}
	var out []string
	for _, enc := range encs {
		if b, err := enc.DecodeString(clean); err == nil && len(b) > 0 {
			out = append(out, string(b))
		}
	}
	return out
}

func collectVlessStrings(v any, out *[]string) {
	switch x := v.(type) {
	case string:
		if strings.HasPrefix(strings.TrimSpace(x), "vless://") {
			*out = append(*out, strings.TrimSpace(x))
		}
	case []any:
		for _, item := range x {
			collectVlessStrings(item, out)
		}
	case map[string]any:
		for _, item := range x {
			collectVlessStrings(item, out)
		}
	}
}

func extractVlessCandidates(text string) []string {
	var out []string
	for _, line := range strings.Fields(text) {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "vless://") {
			out = append(out, line)
		}
	}
	var obj any
	if json.Unmarshal([]byte(text), &obj) == nil {
		collectVlessStrings(obj, &out)
	}
	if len(out) == 0 {
		for _, dec := range decodeMaybeBase64(text) {
			for _, line := range strings.Fields(dec) {
				if strings.HasPrefix(strings.TrimSpace(line), "vless://") {
					out = append(out, strings.TrimSpace(line))
				}
			}
		}
	}
	return out
}

func allowedPort(port int, allowed []int) bool {
	if len(allowed) == 0 {
		return port >= 1 && port <= 65535
	}
	for _, p := range allowed {
		if p == port {
			return true
		}
	}
	return false
}

func allowedHost(host string, suffixes []string) bool {
	if net.ParseIP(host) != nil {
		return false
	}
	h := strings.ToLower(strings.TrimSuffix(host, "."))
	for _, s := range suffixes {
		s = strings.ToLower(strings.TrimSpace(s))
		if s != "" && strings.HasSuffix(h, s) && len(h) > len(s) {
			return true
		}
	}
	return false
}

func parseVless(raw string, cfg BootstrapConfig) (Server, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "vless" || u.User == nil {
		return Server{}, errors.New("invalid VLESS URI")
	}
	uuid := u.User.Username()
	if !uuidRe.MatchString(uuid) {
		return Server{}, errors.New("invalid VLESS UUID")
	}
	host := strings.ToLower(strings.TrimSuffix(u.Hostname(), "."))
	if !allowedHost(host, cfg.AllowedServerSuffixes) {
		return Server{}, fmt.Errorf("server host outside allowed suffixes: %s", host)
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil || !allowedPort(port, cfg.AllowedServerPorts) {
		return Server{}, fmt.Errorf("unsupported server port: %s", u.Port())
	}
	q := u.Query()
	security := strings.ToLower(q.Get("security"))
	if security != "reality" {
		return Server{}, fmt.Errorf("unsupported security: %s", security)
	}
	flow := q.Get("flow")
	if flow != "xtls-rprx-vision" {
		return Server{}, fmt.Errorf("unsupported flow: %s", flow)
	}
	network := strings.ToLower(q.Get("type"))
	if network == "" {
		network = "tcp"
	}
	if network != "tcp" && network != "raw" {
		return Server{}, fmt.Errorf("unsupported transport: %s", network)
	}
	if enc := strings.ToLower(q.Get("encryption")); enc != "" && enc != "none" {
		return Server{}, fmt.Errorf("unsupported VLESS encryption: %s", enc)
	}
	for _, key := range []string{"allowInsecure", "insecure"} {
		v := strings.ToLower(strings.TrimSpace(q.Get(key)))
		if v == "1" || v == "true" || v == "yes" {
			return Server{}, errors.New("insecure TLS is not allowed")
		}
	}
	sni := strings.TrimSpace(q.Get("sni"))
	pbk := strings.TrimSpace(q.Get("pbk"))
	sid := strings.TrimSpace(q.Get("sid"))
	fp := strings.TrimSpace(q.Get("fp"))
	if sni == "" || strings.ContainsAny(sni, "/:@") || len(sni) > 253 {
		return Server{}, errors.New("invalid Reality SNI")
	}
	if !realityKeyRe.MatchString(pbk) {
		return Server{}, errors.New("invalid Reality public key")
	}
	if !shortIDRe.MatchString(sid) || len(sid)%2 != 0 {
		return Server{}, errors.New("invalid Reality short id")
	}
	if fp == "" {
		fp = "chrome"
	}
	name, _ := url.QueryUnescape(u.Fragment)
	name = strings.TrimSpace(name)
	if name == "" {
		name = fmt.Sprintf("%s:%d", host, port)
	}
	sum := sha256.Sum256([]byte(raw))
	id := hex.EncodeToString(sum[:8])
	return Server{
		ID: id, Name: name, URI: raw, Host: host, Port: port, UUID: uuid,
		Flow: flow, Security: security, Network: network, SNI: sni,
		Fingerprint: fp, PublicKey: pbk, ShortID: sid,
	}, nil
}

func parseSubscription(body []byte, cfg BootstrapConfig) ([]Server, error) {
	candidates := extractVlessCandidates(string(body))
	if len(candidates) == 0 {
		return nil, errors.New("no VLESS links found in subscription")
	}
	seen := map[string]bool{}
	servers := make([]Server, 0, len(candidates))
	var rejected int
	for _, raw := range candidates {
		s, err := parseVless(raw, cfg)
		if err != nil {
			rejected++
			continue
		}
		if !seen[s.ID] {
			seen[s.ID] = true
			servers = append(servers, s)
		}
		if len(servers) >= 256 {
			break
		}
	}
	if len(servers) == 0 {
		return nil, fmt.Errorf("all VLESS links rejected (%d)", rejected)
	}
	sort.SliceStable(servers, func(i, j int) bool { return servers[i].Name < servers[j].Name })
	return servers, nil
}
