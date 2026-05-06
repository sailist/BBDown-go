# Commit Convention

This project follows [Conventional Commits](https://www.conventionalcommits.org/).

## Format

```
<type>[(scope)]: <description>
```

## Types

| Type       | Description                                              |
|------------|----------------------------------------------------------|
| `feat`     | A new feature                                            |
| `fix`      | A bug fix                                                |
| `docs`     | Documentation only changes                               |
| `style`    | Changes that do not affect the meaning of the code       |
| `refactor` | A code change that neither fixes a bug nor adds a feature|
| `perf`     | A code change that improves performance                  |
| `test`     | Adding missing tests or correcting existing tests        |
| `build`    | Changes that affect the build system or dependencies     |
| `ci`       | Changes to CI configuration files and scripts            |
| `chore`    | Other changes that don't modify src or test files        |
| `revert`   | Reverts a previous commit                                |

## Scope Mapping

Scopes are optional but recommended. Common scopes map to project modules:

- `api` — Bilibili API client
- `fetcher` — media stream fetcher
- `parser` — URL / BV / EP parser
- `download` — download engine
- `muxer` — video/audio muxer
- `login` — login & credential manager
- `danmaku` — danmaku downloader
- `cli` — command-line interface
- `config` — configuration handling
- `server` — web server mode

## Description Rules

- Use the imperative mood: "add" not "added" or "adds"
- Don't capitalize the first letter
- No trailing period
- Keep the first line under 72 characters
