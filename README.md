# VKarmani Browser VPN v0.4.0

Расширение для Chrome / Edge / Brave / Opera и других Chromium-браузеров, которое принимает пользовательскую Remnawave subscription-ссылку и позволяет подключать **только браузерный трафик** через выбранный VLESS + REALITY + `xtls-rprx-vision` сервер.

> **Пользователю:** короткая установка находится в [`INSTALL.md`](INSTALL.md).
> **Владельцу репозитория:** публикация на GitHub описана в [`PUBLISH_GITHUB.md`](PUBLISH_GITHUB.md).

## Быстрый старт для пользователя

```text
1. Распаковать ONE-FOLDER ZIP
2. Запустить 00_INSTALL_VKARMANI.cmd
3. Открыть chrome://extensions (или страницу расширений своего Chromium-браузера)
4. Включить Developer mode -> Load unpacked
5. Выбрать папку расширения
6. Вставить Remnawave subscription URL
7. Подключиться
```

Другие VPN-клиенты заранее не требуются.


## Что уже реализовано

- Manifest V3 Chromium extension.
- Native Messaging helper без сторонних Go-зависимостей.
- Windows x64 helper binary уже собран в `dist/windows-amd64/`.
- Установка helper без admin через HKCU.
- Пользователю не нужен заранее установленный VPN-клиент или VPN-ядро: one-click установщик сам получает pinned sing-box 1.14.1, проверяет официальный SHA256 и устанавливает его локально.
- Subscription URL не хранится в `chrome.storage`; она хранится helper-ом локально.
- На Windows persisted state защищается DPAPI текущего пользователя.
- Поддержка raw и Base64 Remnawave subscription responses с `vless://` links.
- Строгая проверка текущего VKarmani профиля: VLESS + REALITY + TCP/raw + Vision, серверы `*.vkarmani.com`, порты 443/9443.
- Список серверов, ручной выбор и `Подключиться к лучшему`.
- Современный компактный popup: отдельная карточка состояния, чистый список локаций и технические детали убраны из основного UI.
- Для локаций используются 249 локально упакованных PNG-флагов ISO-стран; внешний CDN для картинок не нужен.
- Страна определяется по flag-emoji, ISO-коду, русскому/английскому названию и распространённым названиям городов.
- В пользовательском списке не показываются `hostname:port` и транспорт вроде `TCP`; остаются имя локации, страна и понятная задержка в мс.
- TCP latency probe используется только как быстрый критерий выбора и не выдаётся за полноценный VLESS health-check.
- Локальный mixed proxy sing-box поднимается на случайном `127.0.0.1` порту.
- Chrome/Chromium трафик переключается через `chrome.proxy`.
- WebRTC non-proxied UDP блокируется, browser network prediction отключается на время подключения.
- Fail-closed: если native helper/sing-box неожиданно падает, расширение **не переключает браузер автоматически на DIRECT**.
- Last Known Good server list сохраняется; неудачный refresh не удаляет рабочий кеш.
- Refresh цепочка: normal HTTPS -> DoH-resolved HTTPS -> direct brown-bear IP -> direct TECH IP -> current VPN tunnel (если уже подключён) -> бесплатный public bootstrap relay `proxy.cors.dev` только как последний online fallback.
- TLS verification никогда не отключается: direct-IP fetch всё равно проверяет сертификат/SNI `sub.vkarmani.com`.
- Автоматический refresh подписки раз в 60 минут.

## Windows x64: установка

Пользовательский пакет теперь собран как **одна папка расширения**. Никакие Hiddify, v2rayN, Xray, WARP или отдельный sing-box заранее не нужны.

1. Распаковать пользовательский ZIP.
2. Внутри единственной папки расширения дважды запустить:

```text
00_INSTALL_VKARMANI.cmd
```

Скрипт без прав администратора устанавливает Native Messaging helper в `%LOCALAPPDATA%`, автоматически получает pinned `sing-box 1.14.1` и принимает его **только** при точном совпадении официального release SHA256:

```text
5197f16d492d93202dc623622149a6ed040f8eca263128f91d603f2b901baa89
```

Сначала используется официальный GitHub release. Если GitHub из региона недоступен, установщик автоматически пробует SourceForge exact mirror и несколько GitHub download mirrors. Любой ответ с другим SHA256 отвергается. Credentials/subscription при скачивании ядра не передаются: загружается только публичный upstream archive sing-box.

3. После успешного setup установщик открывает папку расширения и менеджер расширений найденного Chromium-браузера. Включить **Developer mode -> Load unpacked** и выбрать эту же папку. Chromium не разрешает обычному unpacked extension самостоятельно зарегистрировать Native Messaging host, поэтому этот единственный запуск `.cmd` технически неизбежен.
4. Открыть popup, вставить пользовательскую Remnawave subscription URL и подключиться.

Ожидаемый extension ID:

```text
bfcnangfjclakpliejpaamnalnjbmlgh
```

Диагностика и удаление находятся в той же папке: `setup\diagnose.ps1` и `UNINSTALL_VKARMANI.cmd`.

## Работа при проблемной доступности подписки

Helper не зависит от браузерного `fetch`. При refresh он пробует последовательно:

```text
1. обычный HTTPS по системному DNS
2. DNS-over-HTTPS bootstrap через 1.1.1.1 / 8.8.8.8 + HTTPS
3. 81.163.22.205 (brown-bear), но TLS SNI/Host остаётся sub.vkarmani.com
4. 80.66.81.68 (TECH), но TLS SNI/Host остаётся sub.vkarmani.com
5. если VPN уже поднят — refresh оригинального URL через текущий туннель
6. только если всё выше недоступно — `https://proxy.cors.dev/` как публичный bootstrap relay
7. при полном неуспехе — сохраняется Last Known Good
```

`cors.dev` выбран потому, что его публичный GET/HEAD режим не требует аккаунта, API-ключа или оплаты и допускает текстовые ответы до 1 MiB — тот же предел, который уже применяется helper-ом к подписке. Public relay не используется для обычного браузерного трафика и не становится частью VPN data-plane.

Никаких изменений Remnawave, TECH, brown-bear или VPN-нод для этого клиента не требуется.

## Важная граница безопасности public bootstrap

При использовании `proxy.cors.dev` полный пользовательский subscription URL проходит через сторонний relay и поэтому становится виден его оператору. Именно поэтому этот путь стоит **последним**: сначала helper использует наши direct/DoH/IP пути и уже поднятый VPN. Полученный через relay ответ всё равно не принимается автоматически: он проходит прежний строгий парсинг VLESS/REALITY/Vision, ограничение `*.vkarmani.com`, разрешённые порты и transactional Last Known Good update.

Если сеть блокирует одновременно наши прямые пути **и** `proxy.cors.dev`, абсолютной гарантии первого запуска всё равно нет. В таком случае следующим независимым слоем остаётся Snowflake/Tor bootstrap.

## Security

- Не вставлять Remnawave admin API key. Нужна **пользовательская subscription URL**.
- Subscription URL и VLESS URI не пишутся в extension logs.
- Browser storage содержит только состояние UI/выбранный opaque server ID, не credentials.
- Windows state helper-а зашифрован DPAPI.
- Runtime sing-box config создаётся только для запуска, после успешного старта и E2E-проверки сразу удаляется; stdout/stderr sing-box не пишутся в persistent browsing log.
- Native host разрешает только фиксированный extension ID.
- Subscription fetch принимает только `https://sub.vkarmani.com/...`.
- Redirect на другой host блокируется.
- Ответ ограничен 1 MiB.
- Неуспешный refresh не заменяет Last Known Good.

Подробнее: `docs/SECURITY.md`, `docs/ARCHITECTURE.md`, `docs/LIMITATIONS.md`.

## Проверки, выполненные при сборке

```text
Go unit tests: PASS
Go vet: PASS (см. BUILD_REPORT.txt)
Windows amd64 cross-build: PASS
Linux amd64 build: PASS
manifest JSON parse: PASS
extension source syntax checks: PASS
archive integrity: PASS
```

Реальный VLESS/REALITY E2E не выполнялся: в проект не передавалась пользовательская subscription URL и в среде сборки нет доступа к вашим production credentials. Перед массовой раздачей нужен controlled acceptance на отдельном тестовом пользователе.


## Extension identity

`extension_public_key.txt` contains only the public key used to keep the unpacked extension ID stable. The private generation key was discarded and is not included.
