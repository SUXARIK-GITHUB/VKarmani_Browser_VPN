package app

import "time"

type Request struct {
	ID              string `json:"id,omitempty"`
	Action          string `json:"action"`
	SubscriptionURL string `json:"subscription_url,omitempty"`
	ServerID        string `json:"server_id,omitempty"`
}

type Response struct {
	ID     string  `json:"id,omitempty"`
	OK     bool    `json:"ok"`
	Error  string  `json:"error,omitempty"`
	Status *Status `json:"status,omitempty"`
}

type Status struct {
	HelperVersion      string         `json:"helper_version"`
	SubscriptionSet    bool           `json:"subscription_set"`
	SubscriptionMasked string         `json:"subscription_masked,omitempty"`
	Servers            []ServerPublic `json:"servers"`
	SelectedID         string         `json:"selected_id,omitempty"`
	Connected          bool           `json:"connected"`
	LocalProxyPort     int            `json:"local_proxy_port,omitempty"`
	LastSuccess        string         `json:"last_success,omitempty"`
	LastAttempt        string         `json:"last_attempt,omitempty"`
	LastSource         string         `json:"last_source,omitempty"`
	LastError          string         `json:"last_error,omitempty"`
	CacheAvailable     bool           `json:"cache_available"`
	SingBoxFound       bool           `json:"sing_box_found"`
	SingBoxVersion     string         `json:"sing_box_version,omitempty"`
}

type ServerPublic struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Host      string `json:"host"`
	Port      int    `json:"port"`
	LatencyMS int    `json:"latency_ms,omitempty"`
	Reachable bool   `json:"reachable"`
}

type Server struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	URI         string `json:"uri"`
	Host        string `json:"host"`
	Port        int    `json:"port"`
	UUID        string `json:"uuid"`
	Flow        string `json:"flow"`
	Security    string `json:"security"`
	Network     string `json:"network"`
	SNI         string `json:"sni"`
	Fingerprint string `json:"fingerprint"`
	PublicKey   string `json:"public_key"`
	ShortID     string `json:"short_id"`
	LatencyMS   int    `json:"latency_ms,omitempty"`
	Reachable   bool   `json:"reachable"`
}

type persistedState struct {
	Version         int       `json:"version"`
	SubscriptionURL string    `json:"subscription_url"`
	Servers         []Server  `json:"servers"`
	LastGood        []Server  `json:"last_good"`
	SelectedID      string    `json:"selected_id,omitempty"`
	LastSuccess     time.Time `json:"last_success,omitempty"`
	LastAttempt     time.Time `json:"last_attempt,omitempty"`
	LastSource      string    `json:"last_source,omitempty"`
	LastError       string    `json:"last_error,omitempty"`
}

type BootstrapConfig struct {
	SubscriptionHost      string   `json:"subscription_host"`
	DirectIPs             []string `json:"direct_ips"`
	AllowedServerSuffixes []string `json:"allowed_server_suffixes"`
	AllowedServerPorts    []int    `json:"allowed_server_ports"`
	MaxSubscriptionBytes  int64    `json:"max_subscription_bytes"`
	PublicRelayPrefixes   []string `json:"public_relay_prefixes,omitempty"`
}
