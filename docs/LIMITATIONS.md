# Current limitations (v0.3.0)

- Windows x64 is the packaged/installer target. Helper source itself is portable and Linux amd64 build is included for engineering tests, but Linux/macOS installers are not shipped in v0.3.0.
- Only the current VKarmani VLESS/REALITY/Vision TCP/raw profile is accepted. WebSocket/gRPC/Hysteria2/etc. are intentionally rejected rather than silently misconfigured.
- TCP latency is not a VLESS/REALITY E2E health test.
- First-run reachability is improved with a final `proxy.cors.dev` public bootstrap fallback. It is an external zero-account service, not VKarmani infrastructure, so availability cannot be guaranteed. If a region blocks both VKarmani subscription paths and the relay, first-run still needs another independent transport such as Snowflake/Tor.
- Public bootstrap has a privacy cost: when it is actually used, the relay operator can observe the full subscription URL and upstream response. For this reason it is tried only after all first-party paths and an already-active VPN tunnel fail.
- This build was not tested against a real production user subscription because no user credential was provided to the build environment.
- Publishing to Chrome Web Store / Edge Add-ons will likely assign store-specific extension IDs. The native host `allowed_origins` and installer must then include those production IDs.
