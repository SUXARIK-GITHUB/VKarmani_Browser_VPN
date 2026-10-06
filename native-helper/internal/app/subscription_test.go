package app

import (
	"encoding/base64"
	"strings"
	"testing"
)

const testURI = "vless://123e4567-e89b-12d3-a456-426614174000@node.vkarmani.com:443?security=reality&type=tcp&flow=xtls-rprx-vision&sni=example.com&fp=chrome&pbk=AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA&sid=abcd#%F0%9F%87%AA%F0%9F%87%AA%20Test"

func TestParseRawAndBase64(t *testing.T) {
	for _, body := range []string{testURI, base64.StdEncoding.EncodeToString([]byte(testURI + "\n"))} {
		s, err := parseSubscription([]byte(body), defaultBootstrap)
		if err != nil {
			t.Fatal(err)
		}
		if len(s) != 1 || s[0].Name != "🇪🇪 Test" {
			t.Fatalf("unexpected server: %#v", s)
		}
	}
}

func TestRejectWrongHost(t *testing.T) {
	u := strings.Replace(testURI, "node.vkarmani.com", "evil.example", 1)
	if _, err := parseSubscription([]byte(u), defaultBootstrap); err == nil {
		t.Fatal("expected rejection")
	}
}

func TestMask(t *testing.T) {
	m := maskSubscription("https://sub.vkarmani.com/abcdefghijkl")
	if strings.Contains(m, "abcdefgh") {
		t.Fatalf("secret leaked: %s", m)
	}
}

func TestBuildPublicRelayURL(t *testing.T) {
	got, host, err := buildPublicRelayURL("https://proxy.cors.dev/", "https://sub.vkarmani.com/abcdefghijkl")
	if err != nil {
		t.Fatal(err)
	}
	if host != "proxy.cors.dev" {
		t.Fatalf("unexpected relay host: %s", host)
	}
	want := "https://proxy.cors.dev/https://sub.vkarmani.com/abcdefghijkl"
	if got != want {
		t.Fatalf("relay URL mismatch: got %q want %q", got, want)
	}
}

func TestRejectUnsafePublicRelayPrefix(t *testing.T) {
	bad := []string{
		"http://proxy.example/",
		"https://user:pass@proxy.example/",
		"https://proxy.example/?x=1",
		"https://proxy.example/#frag",
		"https://proxy.example:8443/",
	}
	for _, prefix := range bad {
		if _, _, err := buildPublicRelayURL(prefix, "https://sub.vkarmani.com/abcdefghijkl"); err == nil {
			t.Fatalf("expected rejection for %q", prefix)
		}
	}
}
