# Changelog

All notable changes to this project are documented here. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and versions follow [Semantic Versioning](https://semver.org/).

## [Unreleased]

## [0.3.1] - 2026-07-31

### Fixed

- Fresh HAR archives no longer require a captured `track_likes` request before authenticated streaming can work.
- Stream resolution retries every advertised SoundCloud transcoding and resolves search results by numeric track ID before falling back to `yt-dlp`.
- Track radio automatically skips unavailable recommendations instead of stopping on HTTP 404.

### Added

- `w` toggles the animated waveform and restores a compact progress bar when hidden.

## [0.3.0] - 2026-07-31

### Added

- Animated infinite SoundCloud-style waveform with deterministic per-track shape, progress coloring, and pause-aware motion.
- Endless track radio powered by SoundCloud recommendations, with automatic queue refills.

## [0.2.3] - 2026-07-31

### Fixed

- Search input now preserves spaces between words.
- Public search results now resolve through the SoundCloud API first, with cookie-free `yt-dlp` retry as a final fallback.

## [0.2.2] - 2026-07-31

### Fixed

- Removed the obsolete `yt-dlp --no-call-home` option that prevented stream resolution in current releases.
- Personal library tracks now resolve through SoundCloud media transcodings instead of failing public-page extraction with HTTP 404.

## [0.2.1] - 2026-07-31

### Fixed

- Pause now reliably suspends and resumes `ffplay` on Linux and macOS.

## [0.2.0] - 2026-07-31

### Fixed

- Personal mixes now accept SoundCloud container URNs and numeric or quoted track IDs.
- Personal mixes no longer fall back to public search and show `nothing found`.

### Added

- Persistent playback queue with automatic next-track playback.
- Shuffle and repeat-all/repeat-one modes.
- Runtime volume and mute controls.
- Redesigned responsive interface with navigation tabs, progress, queue position, and keyboard help.

## [0.1.0] - 2026-07-31

### Added

- Full-screen SoundCloud terminal interface.
- Public track search, profile browsing, sets, playlists, and public likes.
- Personal mixes, account likes, and listening history from a local HAR session.
- Playback, pause, stop, next, and previous controls through `ffplay`.
- Automatic local cookie and HAR discovery with a public mode fallback.
- Linux, macOS, and Windows release archives.

[Unreleased]: https://github.com/cons0leweb/soundcloud-cli/compare/v0.3.1...HEAD
[0.3.1]: https://github.com/cons0leweb/soundcloud-cli/compare/v0.3.0...v0.3.1
[0.3.0]: https://github.com/cons0leweb/soundcloud-cli/compare/v0.2.3...v0.3.0
[0.2.3]: https://github.com/cons0leweb/soundcloud-cli/compare/v0.2.2...v0.2.3
[0.2.2]: https://github.com/cons0leweb/soundcloud-cli/compare/v0.2.1...v0.2.2
[0.2.1]: https://github.com/cons0leweb/soundcloud-cli/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/cons0leweb/soundcloud-cli/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/cons0leweb/soundcloud-cli/releases/tag/v0.1.0
