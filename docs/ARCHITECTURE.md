# Architecture

```text
Chromium extension (MV3)
  -> Native Messaging
  -> VKarmani helper
      -> subscription fetch / validation / LKG cache
      -> sing-box child process
      -> 127.0.0.1:<random> mixed proxy
  <- local proxy port
  -> chrome.proxy fixed_servers (HTTP proxy)
  -> only browser traffic
  -> VLESS + REALITY + Vision selected node
```

Control plane and data plane are intentionally separated.

## Control plane

The helper owns subscription retrieval. The extension never fetches the subscription URL itself, avoiding browser CORS and giving us direct-IP/bootstrap control.

A candidate subscription is parsed and validated completely before replacing the current/LKG list.

## Data plane

The helper translates one selected `vless://` URI to a short-lived sing-box JSON config. The VLESS node hostname is resolved to an IPv4 address before sing-box starts, while REALITY `server_name`, `public_key`, `short_id`, `flow` and uTLS fingerprint are preserved.

sing-box exposes only a random loopback proxy. The browser extension points Chromium at it.

## Fail-closed

Unexpected helper/native-port loss does not clear `chrome.proxy`. The browser remains pointed at a dead loopback proxy rather than leaking traffic directly. Explicit user Disconnect restores the underlying browser proxy state first and only then stops sing-box.


## Subscription bootstrap fallback (v0.3.0)

Control-plane fetch order is deliberately privacy-first:

```text
direct HTTPS
  -> DoH resolved HTTPS
  -> brown-bear direct IP with origin TLS SNI
  -> TECH direct IP with origin TLS SNI
  -> active VKarmani VPN tunnel, when available
  -> proxy.cors.dev public GET relay (last online fallback)
  -> preserve Last Known Good on failure
```

The public relay never carries browser/data-plane traffic. Relay output is parsed and validated with the same VLESS/REALITY/Vision and allowed-host restrictions before state replacement.
