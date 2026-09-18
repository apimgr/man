# TODO

## Makefile Convention Gaps

- `Makefile` line 44: uses `golang:alpine` — must use `casjaysdev/go:latest`.
- `Makefile` line 38: `GO_DOCKER` invocation is missing
  `-e GOFLAGS=-buildvcs=false`.
- `Makefile` lines 64, 74, 87, 100: `go build` calls missing `-buildvcs=false`.
- `Makefile` lines 123, 128, 134 (`local` target): `go build` calls missing
  `-buildvcs=false`.
- `Makefile` line 208: `go test` missing `-buildvcs=false`.
- `Makefile` line 19: `LDFLAGS` missing `-trimpath`.
- `Makefile` lines 227, 237, 248 (`dev` target): raw `docker run` commands
  use `golang:alpine` instead of `casjaysdev/go:latest`.

Found by go-lint during the admin-UI-strip/CI-fix pass; pre-existing and
out of scope for that pass — needs a dedicated Makefile convention pass.

## CLI/Runtime Convention Gaps

- `src/main.go`: missing `--color` flag with `auto`/`yes`/`no` values.
- `src/server/server.go` line 733: `log.Fatalf()` should use `os.Exit()`
  with a sysexits exit code instead.

Found by go-lint during the admin-UI-strip/CI-fix pass; pre-existing and
out of scope for that pass.
