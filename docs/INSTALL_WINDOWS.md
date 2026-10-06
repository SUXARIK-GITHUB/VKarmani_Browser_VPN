# Windows x64 / Chromium — one-folder install

Пользователю выдаётся одна папка VKarmani Browser VPN. Отдельные Hiddify, v2rayN, Xray, WARP или sing-box не требуются.

## 1. One-click runtime setup

Дважды запустить в папке расширения:

```text
00_INSTALL_VKARMANI.cmd
```

Setup работает от обычного пользователя через HKCU/%LOCALAPPDATA%, поэтому elevation не требуется. Он проверяет SHA256 упакованного VKarmani helper/bootstrap, затем автоматически получает официальный `sing-box 1.14.1`. Архив принимается только при SHA256:

```text
5197f16d492d93202dc623622149a6ed040f8eca263128f91d603f2b901baa89
```

Порядок получения ядра: bundled ZIP (если он присутствует в runtime) -> GitHub official -> SourceForge exact mirror -> ghproxy.net -> ghfast.top -> gh-proxy.com. Зеркало не является trust root: байты всегда проверяются по официальному pinned digest, иначе источник отвергается.

Runtime устанавливается versioned в `%LOCALAPPDATA%\VKarmaniBrowserVPN\versions\0.4.0`; native host manifest указывает на эту версию. Старые versioned runtime автоматически не удаляются, чтобы не делать destructive cleanup при обновлении.

## 2. Load unpacked

После setup папка расширения и страница `chrome://extensions/`/совместимого браузера открываются автоматически, когда браузер найден. Включить Developer mode -> Load unpacked -> выбрать ту же папку.

Ожидаемый ID:

```text
bfcnangfjclakpliejpaamnalnjbmlgh
```

Это единственный шаг, который нельзя безопасно автоматизировать для обычного unpacked Chromium extension: само расширение не имеет права записать Native Messaging registration или самоустановиться как unpacked extension.

## 3. First use

Вставить обычную пользовательскую Remnawave subscription URL. Не использовать Remnawave admin API token.

## Diagnostics

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\setup\diagnose.ps1
```

Диагностика не печатает subscription URL, UUID или VLESS credentials.

## Uninstall

Сначала нажать «Отключиться», затем:

```text
UNINSTALL_VKARMANI.cmd
```

После этого удалить unpacked extension из браузера.
