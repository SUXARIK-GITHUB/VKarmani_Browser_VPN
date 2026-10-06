package app

import (
	"errors"
	"fmt"
	"net"
	"sort"
	"strings"
	"sync"
	"time"
)

const Version = "0.4.0"

type App struct {
	mu        sync.Mutex
	state     persistedState
	bootstrap BootstrapConfig
	tunnel    tunnelProcess
}

func New() (*App, error) {
	cfg, err := loadBootstrap()
	if err != nil {
		return nil, err
	}
	s, err := loadState()
	if err != nil {
		return nil, err
	}
	return &App{state: s, bootstrap: cfg}, nil
}

func (a *App) Close() { a.tunnel.Stop() }

func (a *App) Handle(req Request) Response {
	resp := Response{ID: req.ID, OK: false}
	var err error
	switch req.Action {
	case "ping", "status":
		resp.Status = a.status()
		resp.OK = true
		return resp
	case "set_subscription":
		err = a.setSubscription(req.SubscriptionURL)
	case "refresh":
		err = a.refresh()
	case "probe_servers":
		err = a.probeServers()
	case "connect":
		err = a.connect(req.ServerID)
	case "connect_auto":
		err = a.connectAuto()
	case "disconnect":
		a.tunnel.Stop()
	case "clear_subscription":
		a.tunnel.Stop()
		err = a.clearSubscription()
	default:
		err = fmt.Errorf("unknown action: %s", req.Action)
	}
	if err != nil {
		resp.Error = userSafeError(err)
		resp.Status = a.status()
		return resp
	}
	resp.OK = true
	resp.Status = a.status()
	return resp
}

func userSafeError(err error) string {
	if err == nil {
		return ""
	}
	s := err.Error()
	// Never return a subscription URL or VLESS URI through native messaging.
	if strings.Contains(s, "vless://") {
		return "VLESS configuration error"
	}
	if strings.Contains(s, "https://sub.vkarmani.com/") {
		return "subscription request failed"
	}
	if len(s) > 500 {
		s = s[:500] + "…"
	}
	return s
}

func (a *App) status() *Status {
	a.mu.Lock()
	s := a.state
	a.mu.Unlock()
	connected, port, serverID := a.tunnel.Connected()
	serverPublic := make([]ServerPublic, 0, len(s.Servers))
	for _, srv := range s.Servers {
		serverPublic = append(serverPublic, ServerPublic{ID: srv.ID, Name: srv.Name, Host: srv.Host, Port: srv.Port, LatencyMS: srv.LatencyMS, Reachable: srv.Reachable})
	}
	path, ver, err := findSingBox()
	_ = path
	st := &Status{
		HelperVersion:      Version,
		SubscriptionSet:    s.SubscriptionURL != "",
		SubscriptionMasked: maskSubscription(s.SubscriptionURL),
		Servers:            serverPublic,
		SelectedID:         s.SelectedID,
		Connected:          connected,
		LocalProxyPort:     port,
		LastSource:         s.LastSource,
		LastError:          s.LastError,
		CacheAvailable:     len(s.LastGood) > 0,
		SingBoxFound:       err == nil,
		SingBoxVersion:     ver,
	}
	if connected && serverID != "" {
		st.SelectedID = serverID
	}
	if !s.LastSuccess.IsZero() {
		st.LastSuccess = s.LastSuccess.Format(time.RFC3339)
	}
	if !s.LastAttempt.IsZero() {
		st.LastAttempt = s.LastAttempt.Format(time.RFC3339)
	}
	return st
}

func maskSubscription(raw string) string {
	if raw == "" {
		return ""
	}
	idx := strings.Index(raw, "/")
	if len(raw) <= 12 {
		return "••••"
	}
	prefix := raw
	if i := strings.Index(raw, "://"); i >= 0 {
		start := i + 3
		if j := strings.Index(raw[start:], "/"); j >= 0 {
			prefix = raw[:start+j+1]
		}
	}
	_ = idx
	return prefix + "••••••••" + raw[maxInt(len(raw)-4, 0):]
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (a *App) setSubscription(raw string) error {
	u, err := validateSubscriptionURL(raw, a.bootstrap)
	if err != nil {
		return err
	}
	a.mu.Lock()
	old := a.state.SubscriptionURL
	a.state.SubscriptionURL = u.String()
	a.state.LastError = ""
	if err := saveState(a.state); err != nil {
		a.state.SubscriptionURL = old
		a.mu.Unlock()
		return err
	}
	a.mu.Unlock()
	if err := a.refresh(); err != nil {
		a.mu.Lock()
		a.state.SubscriptionURL = old
		_ = saveState(a.state)
		a.mu.Unlock()
		return err
	}
	return nil
}

func (a *App) clearSubscription() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.state = persistedState{Version: 1}
	return saveState(a.state)
}

func (a *App) refresh() error {
	a.mu.Lock()
	raw := a.state.SubscriptionURL
	a.state.LastAttempt = time.Now()
	a.mu.Unlock()
	if raw == "" {
		return errors.New("subscription is not configured")
	}
	var body []byte
	var source string
	var err error
	var errs []string
	if body, source, err = fetchDirect(raw, a.bootstrap); err != nil {
		errs = append(errs, "direct")
	}
	if err != nil {
		if b, s, e := fetchViaDoH(raw, a.bootstrap); e == nil {
			body, source, err = b, s, nil
		} else {
			errs = append(errs, "doh")
		}
	}
	if err != nil {
		for i, ip := range a.bootstrap.DirectIPs {
			label := fmt.Sprintf("direct-ip-%d", i+1)
			if b, s, e := fetchDirectIP(raw, a.bootstrap, ip, label); e == nil {
				body, source, err = b, s, nil
				break
			} else {
				errs = append(errs, label)
			}
		}
	}
	if err != nil {
		connected, port, _ := a.tunnel.Connected()
		if connected {
			if b, s, e := fetchViaTunnel(raw, a.bootstrap, port); e == nil {
				body, source, err = b, s, nil
			} else {
				errs = append(errs, "active-vpn")
			}
		}
	}
	if err != nil && len(a.bootstrap.PublicRelayPrefixes) > 0 {
		if b, s, e := fetchViaPublicRelays(raw, a.bootstrap); e == nil {
			body, source, err = b, s, nil
		} else {
			errs = append(errs, "public-relay")
		}
	}
	if err != nil {
		a.mu.Lock()
		a.state.LastError = "subscription refresh failed via " + strings.Join(errs, ", ")
		_ = saveState(a.state)
		a.mu.Unlock()
		return errors.New("subscription refresh failed; last known good list was preserved")
	}
	servers, err := parseSubscription(body, a.bootstrap)
	if err != nil {
		a.mu.Lock()
		a.state.LastError = "subscription received but validation failed"
		_ = saveState(a.state)
		a.mu.Unlock()
		return err
	}
	// Probe is best-effort and never invalidates a valid subscription.
	servers = probeServerList(servers)
	a.mu.Lock()
	defer a.mu.Unlock()
	a.state.Servers = servers
	a.state.LastGood = append([]Server(nil), servers...)
	a.state.LastSuccess = time.Now()
	a.state.LastSource = source
	a.state.LastError = ""
	if a.state.SelectedID != "" && !containsServer(servers, a.state.SelectedID) {
		a.state.SelectedID = ""
	}
	return saveState(a.state)
}

func containsServer(servers []Server, id string) bool {
	for _, s := range servers {
		if s.ID == id {
			return true
		}
	}
	return false
}

func probeServerList(in []Server) []Server {
	out := append([]Server(nil), in...)
	type result struct {
		i, ms int
		ok    bool
	}
	jobs := make(chan int)
	results := make(chan result, len(out))
	var wg sync.WaitGroup
	workers := 6
	if len(out) < workers {
		workers = len(out)
	}
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				start := time.Now()
				targetHost := out[i].Host
				if ip, resolveErr := resolveServerIPv4(out[i].Host); resolveErr == nil {
					targetHost = ip
				}
				c, e := net.DialTimeout("tcp4", fmt.Sprintf("%s:%d", targetHost, out[i].Port), 1200*time.Millisecond)
				if e == nil {
					c.Close()
					results <- result{i: i, ms: int(time.Since(start).Milliseconds()), ok: true}
				} else {
					results <- result{i: i, ok: false}
				}
			}
		}()
	}
	go func() {
		for i := range out {
			jobs <- i
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()
	for r := range results {
		out[r.i].Reachable = r.ok
		if r.ok {
			out[r.i].LatencyMS = r.ms
		} else {
			out[r.i].LatencyMS = 0
		}
	}
	return out
}

func (a *App) probeServers() error {
	a.mu.Lock()
	servers := append([]Server(nil), a.state.Servers...)
	a.mu.Unlock()
	if len(servers) == 0 {
		return errors.New("no cached servers")
	}
	servers = probeServerList(servers)
	a.mu.Lock()
	a.state.Servers = servers
	if len(a.state.LastGood) > 0 {
		a.state.LastGood = append([]Server(nil), servers...)
	}
	err := saveState(a.state)
	a.mu.Unlock()
	return err
}

func (a *App) serverByID(id string) (Server, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, s := range a.state.Servers {
		if s.ID == id {
			return s, true
		}
	}
	return Server{}, false
}

func (a *App) connect(id string) error {
	s, ok := a.serverByID(id)
	if !ok {
		return errors.New("server not found in current subscription")
	}
	port, err := a.tunnel.Start(s)
	_ = port
	if err != nil {
		return err
	}
	a.mu.Lock()
	a.state.SelectedID = s.ID
	err = saveState(a.state)
	a.mu.Unlock()
	return err
}

func (a *App) connectAuto() error {
	a.mu.Lock()
	servers := append([]Server(nil), a.state.Servers...)
	a.mu.Unlock()
	if len(servers) == 0 {
		return errors.New("no servers available")
	}
	sort.SliceStable(servers, func(i, j int) bool {
		if servers[i].Reachable != servers[j].Reachable {
			return servers[i].Reachable
		}
		if servers[i].Reachable && servers[i].LatencyMS != servers[j].LatencyMS {
			return servers[i].LatencyMS < servers[j].LatencyMS
		}
		return servers[i].Name < servers[j].Name
	})
	return a.connect(servers[0].ID)
}
