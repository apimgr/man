# Security

## Service model

casman serves a read-only public man page collection. It does not provide user
accounts, an admin web UI, page editing, or browser-based server configuration.
Visitors can manage browser-only display and consent preferences at
`/server/preferences`; those preferences do not create an account or change
server settings. Configure the server through `server.yml` and the command
line. Do not expose private configuration, backup, or data directories through
the web server.

## Network protection

### Bearer credentials

The server generates its global `adm_` operator token and encryption key into
`{config_dir}/secrets.yml` on first startup. The file is written with mode
`0600`; the operator token is printed once to the first-start console and is
masked on later starts. Keep the configuration directory private. The token
is not stored in the database. `GET /api/v1/server/auth/check` reports whether
a supplied Bearer credential is an operator or owner token without returning
the credential.

Anonymous owners receive `own_` primary tokens, and may create linked tokens
for integrations. Owner tokens grant access only to that owner's resources;
the primary token also manages its linked tokens. The browser stores its owner
credential in an HttpOnly, Secure, SameSite=Strict cookie for token-management
forms. API and MCP authentication uses the Bearer header and ignores cookies.
See [API Reference](api.md) for token routes and the MCP transport.

- Use the built-in TLS configuration or place casman behind a trusted TLS
  reverse proxy. TLS settings and ACME options are documented in
  [Configuration](configuration.md).
- The server applies request rate limits and security middleware. Keep the
  service and its embedded dependencies updated. Responses include a restrictive
  Content Security Policy, cross-origin protections, and two-year HSTS when the
  request is served over TLS.
- `/metrics` is disabled unless enabled in configuration. Set its token before
  exposing the endpoint outside a trusted monitoring network.
- CORS uses one policy across routes. Wildcard origins never allow credentials;
  configure exact origins when browser credentials are required.

## Reporting vulnerabilities

Source-code vulnerabilities belong in [GitHub private vulnerability
reporting](https://github.com/apimgr/man/security/advisories/new). This is the
repository channel; it is distinct from reporting a problem with a particular
deployed service instance.

For a deployed instance, `/.well-known/security.txt` lists the configured
source-report URL, the instance's coordinated-disclosure form, and the
configured security mailbox in preference order. The dynamic overview at
`/server/security` shows those channels and the policy link. The instance form
is available from its security contact URL
(`/server/contact?security_id=…`); an expired or invalid ID returns the
ordinary abuse, DMCA, and privacy form. `/server/contact` is not linked
elsewhere, is not indexed, and returns `404` when disabled by configuration.

The form uses a simple CAPTCHA and a two-step submit/review/confirm flow. The
first submit never delivers a message. Confirmed non-security reports are
mailed to the configured role recipient, with the submitter's address kept in
the message body rather than a mail header. Security reports are encrypted
before database storage; maintainer email contains only encrypted report
bytes, and the acknowledgment contains only a tracking ID and status link.
The status link reveals triage state only, and its token can be used once per
UTC day. Invalid security IDs are recorded in `security.log` with a short hash
of the supplied value; the ID itself is not logged.

The public security pages are `/server/security/policy` and
`/server/security/thanks`. Reports about source-code vulnerabilities should
use GitHub private vulnerability reporting instead of public issues.

## Data and privacy

Bookmarks, browsing history, display preferences, and consent choices are
browser-side features; they do not require an account. The preferences page can
reset its site cookies and browser storage. Server request logs and operational
data are managed by the deployment. Review the deployment's log retention and
backup settings before making the service available to users.
