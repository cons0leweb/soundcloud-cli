<div align="center">

# SoundCloud CLI

**SoundCloud в терминале — поиск, персональные миксы и воспроизведение без браузера.**

[![CI](https://github.com/cons0leweb/soundcloud-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/cons0leweb/soundcloud-cli/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/cons0leweb/soundcloud-cli?display_name=tag)](https://github.com/cons0leweb/soundcloud-cli/releases/latest)
[![Go](https://img.shields.io/github/go-mod/go-version/cons0leweb/soundcloud-cli)](go.mod)
[![License: MIT](https://img.shields.io/badge/license-MIT-2f2f2f.svg)](LICENSE)

</div>

SoundCloud CLI — быстрый полноэкранный TUI-клиент для публичного поиска, профилей, сетов и воспроизведения. С локальной сессией он также показывает персональные `Your Mix`, лайки и историю прослушивания.

> Неофициальный клиент, не связанный с SoundCloud. Используйте его в соответствии с правилами сервиса и правами авторов контента.

## Возможности

- поиск треков, исполнителей, профилей и жанров;
- просмотр треков пользователя, публичных лайков, сетов и плейлистов;
- персональные `Your Mix`, свои лайки и история прослушивания;
- раскрытие миксов и сетов в очередь треков;
- воспроизведение через headless `ffplay`;
- пауза, остановка, следующий и предыдущий трек;
- управление только с клавиатуры;
- публичный режим без аккаунта и локальный авторизованный режим.

## Установка

### Готовый бинарник

Скачайте архив для своей платформы на странице [Releases](https://github.com/cons0leweb/soundcloud-cli/releases/latest), распакуйте `soundcloud` и поместите его в каталог из `$PATH`.

### Из исходников

```bash
go install github.com/cons0leweb/soundcloud-cli/cmd/soundcloud@latest
```

Понадобятся:

- Go 1.23+ — только для сборки из исходников;
- [`yt-dlp`](https://github.com/yt-dlp/yt-dlp);
- FFmpeg с утилитой `ffplay`.

Ubuntu/Debian:

```bash
sudo apt install ffmpeg yt-dlp
```

macOS:

```bash
brew install ffmpeg yt-dlp
```

## Быстрый старт

Публичный поиск работает без настройки:

```bash
soundcloud
```

Начните вводить название трека, исполнителя или жанр и нажмите `Enter`.

### Варианты поиска

| Ввод | Результат |
|---|---|
| `deep house` | поиск треков |
| `@username` | треки профиля |
| `@username/sets` | сеты, плейлисты и миксы профиля |
| `@username/likes` | публичные лайки профиля |
| `https://soundcloud.com/...` | открыть ссылку на трек, профиль или сет |

## Авторизованный режим

Для доступа к своей медиатеке экспортируйте cookies SoundCloud в формате Netscape и передайте путь:

```bash
soundcloud --cookies ~/.config/soundcloud-cli/cookies.txt
```

Если рядом с бинарником есть `netscape.txt`, приложение подхватит его автоматически. Чтобы принудительно использовать публичный режим:

```bash
soundcloud --cookies=
```

Персональные миксы, лайки и история используют параметры авторизованного API из HAR. Приложение автоматически ищет самый свежий `soundcloud*.har` в `~/Downloads` и `~/Загрузки`, либо путь можно передать явно:

```bash
soundcloud --har ~/Downloads/soundcloud-session.har
```

Cookies и HAR читаются только локально. Они содержат доступ к сессии аккаунта: храните их с правами `0600`, никогда не добавляйте в Git и обновляйте после истечения сессии. Подробнее — в [политике безопасности](SECURITY.md).

## Управление

| Клавиша | Действие |
|---|---|
| `Enter` | поиск, воспроизведение или открытие сета |
| `↑` / `↓`, `j` / `k` | выбрать трек |
| `Space` | пауза / продолжить |
| `n` / `p` | следующий / предыдущий трек |
| `s` | остановить |
| `b` | вернуться из сета к предыдущему списку |
| `m` | персональные миксы |
| `l` | свои лайки |
| `h` | история прослушивания |
| `/` | вернуться к поиску |
| `Esc` | перейти между поиском и результатами |
| `q` / `Ctrl+C` | выйти |

## Параметры

```text
--limit N          число результатов: 1–100 (по умолчанию 20)
--cookies FILE     Netscape cookie file, auto или пустое значение
--har FILE         HAR с авторизованной сессией
--yt-dlp PATH      путь к yt-dlp
--ffplay PATH      путь к ffplay
--version          версия приложения
```

## Разработка

```bash
make check
make build
```

Архитектура разделяет ответственность по пакетам:

- `internal/soundcloud` — поиск, SoundCloud API и локальная сессия;
- `internal/player` — жизненный цикл `ffplay`;
- `internal/tui` — состояние, клавиши и рендеринг интерфейса.

## Лицензия

[MIT](LICENSE) © 2026 cons0leweb
