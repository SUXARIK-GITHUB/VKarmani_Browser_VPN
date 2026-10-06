# Git import

Архив подготовлен как корень нового Git-репозитория. Основная документация:

- `README.md` — описание проекта, возможности и технические границы;
- `INSTALL.md` — короткая инструкция для пользователя;
- `PUBLISH_GITHUB.md` — как сделать репозиторий публичным и выложить пользовательский ZIP через Releases;
- `docs/` — архитектура, безопасность, ограничения и тест-план.

После распаковки:

```bash
git init
git add .
git commit -m "VKarmani Browser VPN v0.4.0"
git branch -M main
```

После создания **пустого** удалённого репозитория:

```bash
git remote add origin https://github.com/YOUR_USERNAME/vkarmani-browser-vpn.git
git push -u origin main
```

Не добавляйте реальные Remnawave subscription URL, API-токены, `.env`, приватные ключи или пользовательские state-файлы. Базовые маски для таких файлов уже внесены в `.gitignore`.
