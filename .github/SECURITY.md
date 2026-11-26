# Security Policy

## Reporting a source-code vulnerability

Report vulnerabilities in the casman source code privately through [GitHub's
private vulnerability reporting for this repository](https://github.com/apimgr/man/security/advisories/new).
Do not open a public issue, discussion, or bug report for an unpatched
vulnerability. The security mailbox listed in `security.txt` is a secondary
contact, not the primary source-code reporting channel.

This channel is for vulnerabilities in the repository and its distributed
software. Reports about a deployed casman instance belong to that instance's
coordinated disclosure process at [`/server/security`](/server/security); its
current security contacts are published at `/.well-known/security.txt`, and
the instance form is reached through its current one-shot security contact
URL at `/server/contact?security_id=...`. Never send a deployed-instance
report to the repository channel unless the source code itself is affected.

Coordinate disclosure with the maintainers. The default instance policy uses
a 90-day coordinated disclosure period, subject to an earlier fix release or
a mutually agreed extension; read the live
`/server/security/policy` before testing a service.

## Supported versions

The current default branch is the supported source version. Operators should
upgrade to the latest release before reporting an issue that may already be
fixed.
