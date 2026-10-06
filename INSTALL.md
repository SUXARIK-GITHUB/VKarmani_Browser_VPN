# Установка VKarmani Browser VPN

Короткая пользовательская инструкция для Windows 10/11 и Chromium-браузеров (Chrome, Edge, Brave, Opera и совместимые).

## Установка

1. Скачайте пользовательский архив **VKarmani Browser VPN ONE-FOLDER** и распакуйте его в обычную папку.
2. В распакованной папке дважды запустите:

   ```text
   00_INSTALL_VKARMANI.cmd
   ```

3. Дождитесь сообщения об успешной установке runtime/helper.
4. Откройте страницу расширений браузера:
   - Chrome: `chrome://extensions`
   - Edge: `edge://extensions`
   - Brave: `brave://extensions`
   - Opera: `opera://extensions`
5. Включите **Режим разработчика / Developer mode**.
6. Нажмите **Загрузить распакованное / Load unpacked**.
7. Выберите ту же папку VKarmani Browser VPN, где лежит `manifest.json`.
8. Откройте расширение, вставьте свою пользовательскую Remnawave subscription-ссылку и нажмите подключение.

Другие VPN-клиенты (Hiddify, v2rayN, Xray, WARP и т. п.) заранее не требуются.

## Если расширение не подключается

Запустите диагностику из папки проекта:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\setup\diagnose.ps1
```

Диагностика не должна выводить subscription URL, UUID или VLESS credentials.

## Удаление

1. Сначала нажмите **Отключиться** в расширении.
2. Запустите `UNINSTALL_VKARMANI.cmd`.
3. Удалите unpacked extension со страницы расширений браузера.

Подробности находятся в `docs/INSTALL_WINDOWS.md`, `docs/SECURITY.md` и `docs/LIMITATIONS.md`.
