# Third-party notices

## sing-box

The Windows one-click setup automatically obtains the unmodified upstream sing-box 1.14.1 Windows amd64 release archive. The archive is accepted only when its SHA256 exactly matches the digest published for the upstream GitHub release. Download mirrors are transport fallbacks only and are not trust roots.

Upstream: SagerNet/sing-box
Version: 1.14.1
License: GPL-3.0-or-later
Pinned upstream release archive SHA256: `5197f16d492d93202dc623622149a6ed040f8eca263128f91d603f2b901baa89`

The executable remains an upstream component and is not presented as a VKarmani-derived sing-box build. See upstream source/license materials for the corresponding version before redistribution outside this installer model.

## Noto Color Emoji flag artwork

The PNG country flags in `extension/flags/` were rendered at build time from country-flag glyphs in Google Noto Color Emoji. The font file itself is **not** redistributed in this project or artifact.

Upstream: googlefonts/noto-emoji
Copyright: Google Inc. and contributors
Font license: SIL Open Font License 1.1

The flag PNGs are static local UI assets; no third-party flag CDN or remote image service is contacted by the extension.


## cors.dev public relay service

VKarmani Browser VPN v0.3.0 can optionally use the external `https://proxy.cors.dev/` service as a last-resort subscription bootstrap transport after first-party routes fail. No cors.dev source code is bundled in this project. Service availability, limits and terms are controlled by its operator.
