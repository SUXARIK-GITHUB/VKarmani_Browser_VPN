# Security model

1. User secret is the Remnawave subscription URL. It is never stored in Chrome sync/local storage.
2. Windows persisted state is protected with DPAPI (`CryptProtectData`) under the current Windows user.
3. Native Messaging manifest restricts caller to the fixed extension ID.
4. The helper independently checks the caller origin when Chrome supplies it.
5. Subscription URL is restricted to `https://sub.vkarmani.com/...`, no credentials, no fragment, port 443 only.
6. Redirects are allowed only to the same HTTPS hostname.
7. Subscription response is limited to 1 MiB.
8. Current release accepts only VLESS + REALITY + Vision + TCP/raw and the configured VKarmani server suffix/ports.
9. TLS verification is never disabled during direct-IP fallback. The TCP destination changes, but TLS `ServerName` remains `sub.vkarmani.com`.
10. No shared/bootstrap VLESS credential is embedded in the extension/helper.
11. Third-party public bootstrap is attempted only after direct HTTPS, DoH/direct-IP paths and an already-active VPN tunnel fail. The configured v0.3.0 relay is `https://proxy.cors.dev/`.
12. The public relay necessarily sees the full subscription URL; this is an explicit privacy trade-off for zero-account first-run reachability. No browser traffic is sent through that relay.
13. Invalid/empty/unparseable relay responses cannot replace Last Known Good.
14. Runtime sing-box configuration is user-local, mode 0600 on Unix, and is deleted immediately after a successful start/E2E gate; sing-box stdout/stderr are discarded instead of persisting browsing/error destinations.
15. Extension blocks non-proxied WebRTC UDP while connected.
16. `chrome.proxy` is cleared only on explicit Disconnect; unexpected helper failure stays fail-closed.
17. Country artwork is packaged locally in the extension; opening the popup does not contact a third-party flag/image CDN.

## Threat boundary

A process already running as the same local user can potentially inspect process/files/memory. DPAPI protects data at rest from other users/offline copying; it does not defend against a fully compromised logged-in user session.


## Public bootstrap dependency

`proxy.cors.dev` is an external service outside VKarmani control. It can be unavailable, rate-limit requests, change its terms, observe the requested subscription URL, or return modified/invalid content. The helper therefore treats it only as a last-resort control-plane transport and still applies the same strict subscription validation before committing state. It is never used as the browser VPN/data-plane.
