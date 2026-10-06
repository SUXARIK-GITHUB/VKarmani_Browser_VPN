# Controlled acceptance test plan

Use a dedicated test Remnawave user, not an administrator token.

1. On a clean Windows x64 workstation with no Hiddify/v2rayN/Xray/sing-box installed, extract the one-folder user package and run `00_INSTALL_VKARMANI.cmd`; verify no elevation is requested and setup completes automatically.
2. Verify the setup opens the extension folder/browser extensions page, then Load unpacked from the same folder. Verify extension ID and `setup\diagnose.ps1` output.
3. Open the popup before adding a subscription. Confirm the VKarmani site and Telegram-bot links are visible and open the expected destinations.
4. Insert the test subscription URL; confirm no URL appears in extension logs or `chrome.storage`.
5. Confirm server count/names match the test user's current subscription.
6. Confirm the server list does **not** expose raw `hostname:port` or transport labels such as `TCP` in the normal UI.
7. Verify country artwork and country detection with representative names:
   - a leading country flag emoji, for example `🇪🇪 Estonia`;
   - an English country name, for example `Sweden`;
   - a Russian country name, for example `Нидерланды`;
   - a known city hint, for example `Tallinn` / `Таллин`;
   - an explicit uppercase ISO token, for example `EE`;
   - an unknown/custom name must fall back to the neutral VKarmani location placeholder instead of guessing a wrong country.
8. Disconnect the workstation from the internet, reopen the popup, and confirm country images still render from packaged local assets; the extension must not require a third-party flag CDN.
9. With a subscription containing enough nodes to overflow the popup, verify the locations list scrolls without pushing the main connect/disconnect controls out of the usable popup area.
10. Connect one known-good node from the list and confirm the selected row becomes `Активен`.
11. Verify the large connection card shows the selected server/country and the status changes to `Подключено`.
12. Verify browser public IP changes while a non-browser application keeps the original route.
13. Verify DNS/web browsing and WebSocket sites.
14. Run a WebRTC leak test: no non-proxied public address should be exposed.
15. Kill `sing-box.exe` while connected: browser must fail closed, not go DIRECT.
16. Click Disconnect: browser must return to underlying/direct connectivity.
17. Reconnect using `Подключиться` (automatic best server) and verify the helper-selected node is shown correctly.
18. Block DNS for `sub.vkarmani.com`: refresh should succeed via DoH/direct-IP if those paths are reachable.
19. Block brown-bear path: verify TECH direct fallback if reachable.
20. After one successful subscription fetch, block all subscription paths: cached server list must remain and connection must still be possible.
21. While connected from cache, allow subscription only through VPN and verify refresh-via-active-tunnel.
22. Feed malformed/HTML/oversized subscription response in staging: LKG must remain unchanged.
23. Reboot browser/Windows and verify desired connection recovery behavior.
24. Repeat connect / disconnect / refresh at least five times in Chrome and one additional Chromium browser (Opera, Edge, or Brave) and confirm no `Illegal invocation` / `ChromeSetting` error returns.


## PUBLIC BOOTSTRAP RELAY acceptance (v0.4.0)

Use a dedicated non-production Remnawave test user. Do not paste the test subscription into logs or screenshots.

1. Normal network: add the subscription and verify `last_source` is direct/DoH/direct-IP, not public relay.
2. With the subscription origin routes intentionally unavailable from the test workstation, add a fresh subscription and verify the helper can fall through to `public-relay:proxy.cors.dev`, populate a non-empty validated server list, and connect.
3. After successful bootstrap, restore normal networking, refresh again and verify the helper returns to first-party/active-VPN paths rather than preferring the public relay.
4. Block `proxy.cors.dev` too and verify failed refresh preserves the existing Last Known Good list.
5. Feed an invalid/HTML response through a controlled relay test harness and verify it cannot replace Last Known Good.
6. Confirm browser traffic never uses `proxy.cors.dev`; only subscription GET is allowed to reach it.

## ONE-FOLDER / CORE BOOTSTRAP acceptance (v0.4.0)

1. Clean Windows x64 VM: confirm `sing-box.exe`, Hiddify, v2rayN and Xray are absent before setup.
2. Run only `00_INSTALL_VKARMANI.cmd`; do not manually download or install any VPN software.
3. Confirm runtime is installed under `%LOCALAPPDATA%\VKarmaniBrowserVPN\versions\0.4.0` and the user is not prompted for administrator rights.
4. Confirm downloaded sing-box archive was accepted only after exact SHA256 `5197f16d492d93202dc623622149a6ed040f8eca263128f91d603f2b901baa89`; corrupt a controlled test download and confirm setup rejects it.
5. Test with GitHub blocked but SourceForge/one mirror reachable; setup must fall through automatically with no manual URL entry.
6. Test with first mirror returning HTML/403/partial bytes; hash mismatch/failure must move to the next source and never install those bytes.
7. Block every configured core-download source: setup must fail closed and must not register a manifest pointing to a missing/unverified runtime.
8. After successful setup, block all core-download sources and reboot; already-installed VKarmani must continue to work because normal operation uses the local runtime.
9. Verify Chrome, Edge and Brave NativeMessagingHosts HKCU entries. Verify Opera using the Chrome-compatible registration and, where supported, the additional Opera key.
10. Run uninstall and verify registry entries/runtime are removed while the unpacked source folder remains untouched.
