# Security policy

## Session files

SoundCloud CLI can read Netscape cookie files and browser HAR archives. Both may contain active account credentials.

- Never commit or publish cookie or HAR files.
- Store them outside the repository with permissions `0600`.
- Revoke the SoundCloud session if a file is exposed.
- Prefer the public mode when personal sections are not needed.

The application reads these files locally and does not upload them. Authentication data is sent only to SoundCloud endpoints and to `yt-dlp` for SoundCloud extraction.

## Reporting a vulnerability

Please report vulnerabilities privately through [GitHub Security Advisories](https://github.com/cons0leweb/soundcloud-cli/security/advisories/new). Do not include active cookies, OAuth tokens, HAR files, or personal listening data in a public issue.
