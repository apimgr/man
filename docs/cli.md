# CLI Reference

## Installation

The CLI client is released as a separate binary:

```bash
# Download CLI
curl -LO https://github.com/casapps/casman/releases/latest/download/casman-cli-linux-amd64
chmod +x casman-cli-linux-amd64
mv casman-cli-linux-amd64 /usr/local/bin/casman-cli
```

## Usage

```bash
# Configure the server on first use. The valid address is saved to cli.yml
casman-cli --server https://your-server.example.com health

# Auto-detected GUI or TUI mode
casman-cli

# View man page
casman-cli man ls
casman-cli man 5 passwd

# Search
casman-cli search "file permission"
casman-cli --limit 10 search "file permission"

# Other commands
casman-cli whatis chmod
casman-cli apropos permission
casman-cli stats
```

## Commands

| Command | Description |
|---------|-------------|
| `man [SECTION] NAME` | View man page |
| `search QUERY` | Search man pages |
| `whatis NAME` | One-line description |
| `apropos QUERY` | Search descriptions |
| `stats` | Database statistics |
| `sections` | List sections |
| `platforms` | List platforms |
| `health` | Check server health |

## Global Options

| Option | Description |
|--------|-------------|
| `--server URL` | Server URL for this invocation; saves it only when the config has no valid server |
| `--token TOKEN` | API token |
| `--token-file FILE` | Read an API token from a file for this invocation |
| `--config PATH` | Select a config file or profile |
| `--debug` | Enable debug logging |
| `--lang CODE` | Select a supported output language or `auto` |
| `--format FORMAT` | `table`, `json`, `yaml`, `plain`, or `csv` output |
| `--limit NUMBER` | Maximum search results (default: `20`) |
| `--color auto\|yes\|no` | Color table headings; automatic mode respects `NO_COLOR`, `TERM=dumb`, and terminal detection |
| `--quiet[=BOOL]` | Suppress nonessential man headers and search summaries/snippets |
| `--verbose[=BOOL]` | Print the request server URL to stderr |
| `--pager` | Always use a pager |
| `--no-pager` | Never use a pager |
| `-h, --help` | Show help |
| `-v, --version` | Show version |

## Environment Variables

| Variable | Description |
|----------|-------------|
| `CASMAN_SERVER_PRIMARY` | Default server URL |
| `CASMAN_SERVER_API_VERSION` | API version prefix, such as `v1` |
| `CASMAN_SERVER_TIMEOUT` | Per-request timeout; accepts Go durations or integer seconds |
| `CASMAN_SERVER_RETRY` | Number of retries after a failed request |
| `CASMAN_SERVER_RETRY_DELAY` | Delay between retries |
| `CASMAN_DEBUG` | Enable debug logging |
| `CASMAN_LANG` | Output language or `auto` |
| `CASMAN_TOKEN` | Default API token |
| `CASMAN_TOKEN_FILE` | Read the API token from a file |
| `CASMAN_OUTPUT_FORMAT` | Output format (`table`, `json`, `yaml`, `plain`, `csv`) |
| `CASMAN_LIMIT` | Default maximum search results (positive integer; default `20`) |
| `CASMAN_OUTPUT_COLOR` | Color preference (`auto`, `yes`, `no`) |
| `CASMAN_OUTPUT_QUIET` | Suppress nonessential output (truthy/falsey values) |
| `CASMAN_OUTPUT_VERBOSE` | Print request diagnostics (truthy/falsey values) |
| `CASMAN_OUTPUT_PAGER` | Pager policy (`auto`, `always`, `never`) |
| `CASMAN_CACHE_ENABLED` | Enable or disable response caching |
| `CASMAN_CACHE_TTL` | Response cache lifetime |
| `CASMAN_CACHE_MAX_SIZE` | Maximum cache size in bytes, KB, MB, or GB |
| `CASMAN_LOGGING_LEVEL` | Log level (`debug`, `info`, `warn`, `error`) |
| `CASMAN_LOGGING_FILE` | Log file path |
| `CASMAN_LOGGING_MAX_SIZE` | Maximum log file size in bytes, KB, MB, or GB |
| `CASMAN_LOGGING_MAX_FILES` | Number of rotated log files to retain |
| `CASMAN_TUI_ENABLED` | Enable or disable automatic TUI mode |
| `CASMAN_TUI_THEME` | TUI theme (`dark`, `light`, `system`) |
| `CASMAN_TUI_MOUSE` | Enable or disable mouse input |
| `CASMAN_TUI_UNICODE` | Enable or disable Unicode UI characters |

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Command succeeded |
| `1` | General error or unhealthy server |
| `2` | Invalid command usage or flags |
| `65` | Server rejected input with `VALIDATION_FAILED` |
| `66` | Requested resource was not found |
| `69` | Server connection failed |
| `77` | Authentication or permission failure |
| `78` | CLI configuration error |

## Configuration

On Unix-like systems, the CLI uses `~/.config/casapps/casman/cli.yml`. On
Windows, it uses `%APPDATA%\\casapps\\casman\\cli.yml`. The default config
directory and file are created with user-only permissions where supported.
When absent, the CLI writes a complete config with the documented defaults.

The `display.mode` setting accepts `auto`, `gui`, or `tui`. Automatic mode
prefers the optional native GUI on a supported local display, then the TUI on
an interactive terminal. The standard static CLI build does not contain GUI
code and uses the TUI when a terminal is available. A forced mode reports an
error when its required display or terminal is unavailable. Build the optional
desktop client separately with `-tags gui`; its runtime graphics loader makes
it platform specific and it is not the static CLI artifact.
The standard Makefile targets build the server and static client; optional GUI
artifacts use the `gui` build tag in a platform-appropriate Docker build.
Custom config files are secured as files, while their parent directories are
left under the caller's control. The CLI preserves other YAML sections when it
updates a server or token value.

Server selection is `--server`, `CASMAN_SERVER_PRIMARY`, `server.primary` in
the config file, then the compiled official site when one is configured. If no
server is available, the CLI prints setup instructions. Token selection is
`--token`, `--token-file`, `CASMAN_TOKEN`, `CASMAN_TOKEN_FILE`,
`auth.token_file`, then `auth.token`.

`--config` accepts an absolute path or a profile name relative to the default
config directory. A profile with no extension resolves to `.yml` unless an
existing `.yml` or `.yaml` file is found. Use `--config prod.yaml` to select a
specific YAML file.

The CLI's `server` section supports `primary`, `api_version` (default `v1`;
match the server's configured API version),
`timeout` (default `30s`), `retry` (default `3`), and `retry_delay` (default
`1s`). The `auth` section supports `token` and `token_file`; relative token
file paths are resolved beside the selected config file. `--token` takes
precedence over `--token-file`; explicit CLI flags take precedence over
environment and config values. `CASMAN_TOKEN` takes precedence over
`CASMAN_TOKEN_FILE` and `auth.token_file`, which take precedence over
`auth.token`.

The `output` section supports `format` (default `table`), `color` (default
`auto`), `pager` (default `auto`), `quiet` (default `false`), and `verbose`
(default `false`). Paging is enabled in automatic mode only for an interactive
terminal and when `TERM` is not `dumb`. Format, pager, quiet, and verbose use
flag > environment > config precedence. Color uses flag > `CASMAN_OUTPUT_COLOR`
> config > terminal auto-detection; any nonempty `NO_COLOR` or `TERM=dumb`
disables automatic color, while an explicit `yes` forces it. Color is applied to table headings;
JSON, YAML, CSV, and plain output remain free of ANSI escapes.
`defaults.output` is also accepted as a fallback for the documented flag-default
setting when `output.format` is unset.

Search results use `defaults.limit` (default `20`). `--limit NUMBER` overrides
it for the command, and `CASMAN_LIMIT` overrides the config value. The CLI
passes the selected limit to the server search API.

The `tui` section supports `enabled` (default `true`), `theme` (`dark`,
`light`, or `system`), `mouse` (default `true`), and `unicode` (default
`true`). The matching environment variables are `CASMAN_TUI_ENABLED`,
`CASMAN_TUI_THEME`, `CASMAN_TUI_MOUSE`, and `CASMAN_TUI_UNICODE`. Setting
`tui.enabled: false` disables automatic TUI launch.

The `cache` section supports `enabled` (default `true`), `ttl` (default
`5m`), and `max_size` (default `100MB`). Successful GET responses are cached
under the user's cache directory, separated by server URL and token, and
evicted oldest-first when the configured size cap is exceeded. Cache files
use mode `0600` and the directory uses mode `0700` on Unix. Health and token
checks bypass the response cache.

The `logging` section supports `level` (default `warn`), `file` (default
`~/.local/log/casapps/casman/cli.log` on Unix-like systems), `max_size`
(default `10MB`), and `max_files` (default `5`). Relative log paths resolve
beside the selected config file. Logs are plain text, directories use mode
`0700`, files use mode `0600`, and rotation retains the configured number of
numbered backups.

`--debug`, `CASMAN_DEBUG`, and the top-level `debug` setting enable debug
logging. `--lang`, `CASMAN_LANG`, and `defaults.lang` select a supported
language; `auto` uses `LANG` and falls back to English. The resolved language
is sent to the API with `Accept-Language`.

## Interactive UI Mode

Run `casman-cli` without command arguments to launch the detected interface.
The optional GUI build opens its native window on a supported local display;
the static build opens the TUI in an interactive terminal. If no server is
configured, the selected interface first opens its setup form. Enter a server
URL and optionally an owner token; the CLI checks server reachability and
validates the token before saving initial values. A failed check leaves the
config unchanged.

In a non-interactive shell, configure a server explicitly with `--server` or
`CASMAN_SERVER_PRIMARY`. If the CLI starts without a command and no server is
configured, it exits with usage status `2` and names `server.primary` in the
selected `cli.yml` as the persistent configuration option. A command that
requires a server also prints setup instructions when neither is available.

The interactive menu provides man-page lookup, search, whatis, apropos,
statistics, sections, platforms, and health views. Select a result to open its
man page; `Esc` returns to the command menu and `q` returns to the menu or quits
from the menu.
