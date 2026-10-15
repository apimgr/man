# casman implementation backlog

The repository already contains a substantial embedded man-page application,
including the web UI, search/browse/API routes, GraphQL/OpenAPI, CLI/TUI,
static data, export, sitemap/feeds, metrics, backup, GeoIP, Tor, TLS, and
scheduler components. These are implementation areas, not a claim that every
IDEA.md feature or every supported platform dataset is complete.

## Completed in this pass

- [x] Makefile uses `casjaysdev/go:latest`, the standard Go module/build cache
      mounts, `GOFLAGS=-buildvcs=false`, and `-trimpath` on build commands.
- [x] Production Docker builder uses the shared Go toolchain image and does not
      install build tools inline. Development Docker builder likewise relies on
      the prebuilt toolchain.
- [x] Align the committed container overlay directory with AI.md at
      `docker/rootfs/` and update both Dockerfiles and `.dockerignore` paths.
- [x] Remove the invented `casman.example.com` official-site default; the
      official site is now optional and can be supplied through `site.txt` or
      the build variable.
- [x] Container-based Actions jobs run as root so workspace and temporary
      coverage output remain writable.
- [x] HTTP server errors are returned through the normal startup lifecycle
      rather than terminating the process from a goroutine with `log.Fatalf`.
- [x] Build workflows already pass `main.BuildEpoch`; the executable now
      derives its RFC3339 build date from that epoch instead of accepting a
      separate build-date linker value.
- [x] GitHub stable, beta, daily, and container workflows already publish the
      configured artifacts, checksums, SBOMs, and provenance. Go container jobs
      in these workflows run with writable workspace permissions.
- [x] Docker image metadata is supplied by the Docker publishing workflow's
      labels and manifest annotations instead of Dockerfile `LABEL` blocks.

## Remaining IDEA.md product gaps

- [ ] Translation content and translated man-page search are not backed by
      stored datasets. `/api/v1/man/{lang}/...` currently returns the English
      source and sets `X-Content-Language`; it does not provide translations.
- [ ] The export catalog supports PDF, EPUB, and HTML. MOBI and the complete
      bulk-export formats/options from IDEA.md are not available.
- [ ] The current dataset cannot be claimed to contain every page, version, or
      platform listed in IDEA.md. The repository does not include a complete
      source manifest with per-platform/version provenance and license terms.
      Add and attribute redistributable source datasets before claiming full
      coverage; do not infer missing provenance from page content.
- [x] The existing server-rendered templates and progressive-enhancement script
      implement keyboard shortcuts, table-of-contents navigation, copy controls,
      theme persistence, responsive navigation, and touch swipe navigation.
- [ ] Popular/trending data currently reflects locally recorded views only;
      it does not represent a cross-deployment usage ranking.

## Remaining shared AI.md and release requirements

- [ ] Add equivalent CI and release workflows for the other required
      providers, plus the required Jenkins pipeline. The existing workflows are
      GitHub-specific.
- [ ] Add dependency update automation and required public-repository
      governance files (CODEOWNERS, security policy, issue/PR templates, branch
      protection documentation/configuration).
- [ ] Audit the embedded database contents against the required platform,
      version, language, and source-license inventory. The exact coverage gap
      is unknown until that inventory is created.
- [ ] Add integration coverage for Docker startup, health checks, HTTPS and
      Tor lifecycle, embedded database presence, API formats, feeds, sitemap,
      and CLI/server shutdown. Existing Go unit tests cover individual
      packages; no container integration run was performed in this pass.
- [ ] Confirm CI coverage remains at or above 60% after the current in-progress
      test additions; no build, test, lint, formatting, or workflow validation
      was run in this pass per task instructions.
