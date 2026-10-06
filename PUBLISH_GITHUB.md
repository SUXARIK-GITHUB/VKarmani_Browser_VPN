# Публикация VKarmani Browser VPN на GitHub

Этот файл описывает безопасный способ сделать репозиторий публичным и отдельно выложить пользовательский ZIP через GitHub Releases.

## Рекомендуемое имя и описание

Repository name:

```text
vkarmani-browser-vpn
```

Description:

```text
Browser-only VKarmani VPN extension for Chromium browsers with Remnawave VLESS/REALITY subscriptions.
```

Suggested topics:

```text
vpn chromium chrome-extension opera edge brave vless reality remnawave sing-box
```

## 1. Создайте пустой публичный репозиторий

На GitHub нажмите **New repository** и укажите:

- имя: `vkarmani-browser-vpn`;
- visibility: **Public**;
- не создавайте новый README, `.gitignore` или LICENSE в форме GitHub — они уже есть/управляются локальным проектом, а LICENSE для собственного кода нужно выбрать отдельно осознанно.

## 2. Отправьте этот Git-ready архив

Распакуйте архив и откройте PowerShell/Terminal в его корне:

```bash
git init
git add .
git commit -m "VKarmani Browser VPN v0.4.0"
git branch -M main
git remote add origin https://github.com/YOUR_USERNAME/vkarmani-browser-vpn.git
git push -u origin main
```

Замените `YOUR_USERNAME` на свой GitHub username.

## 3. Если репозиторий уже создан как Private

Откройте репозиторий -> **Settings** -> **General** -> **Danger Zone** -> **Change repository visibility** -> **Public** и подтвердите изменение.

Перед этим обязательно убедитесь, что в истории Git нет реальных subscription URL, API-токенов, приватных ключей, `.env`, пользовательских state-файлов или других секретов. Простого удаления файла в последнем commit недостаточно, если секрет уже попал в Git history.

## 4. Сделайте нормальную публичную загрузку для пользователей

Исходники и Git-архив лучше хранить в repository, а пользовательский ONE-FOLDER ZIP публиковать как **GitHub Release asset**:

1. Откройте **Releases** -> **Draft a new release**.
2. Создайте tag `v0.4.0`.
3. Release title: `VKarmani Browser VPN v0.4.0`.
4. Загрузите файлы:
   - `VKarmani_Browser_VPN_v0.4.0-ONE-FOLDER.zip`;
   - соответствующий `.SHA256.txt`.
5. Добавьте короткую инструкцию установки из `INSTALL.md`.
6. Опубликуйте release.

После этого пользователям можно давать ссылку на страницу Releases или на latest release.

## 5. Что не публиковать

Не коммитьте и не прикладывайте к Releases:

- реальные Remnawave subscription URL;
- Remnawave admin/API tokens;
- `.env` с секретами;
- приватные ключи и сертификаты;
- пользовательские DPAPI/state-файлы;
- runtime config с UUID/VLESS credentials;
- production backup/дампы/логи с секретами.

`extension_public_key.txt` содержит только публичную часть ключа, используемую для стабильного extension ID. Приватного generation key в проекте нет.

## 6. LICENSE

В проекте есть `THIRD_PARTY_NOTICES.md` для сторонних компонентов. Отдельную лицензию на собственный код VKarmani здесь намеренно не выбирали автоматически. Если хотите разрешить другим людям копировать/изменять/распространять ваш собственный код, сначала выберите подходящую лицензию и только потом добавьте `LICENSE`.
