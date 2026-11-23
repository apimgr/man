# Translating casman

The page model stores a language code for each man page. `/api/v1/languages`
reports only the codes and row counts found in the active database; it does
not publish estimated coverage. The translated API route returns only an
exact stored match and returns 404 when that translation is unavailable.

The web interface currently uses English UI text. Translation source projects
named in the repository's IDEA.md include man-pages-l10n for Linux and the
FreeBSD documentation translations. Follow each source project's license and
contribution rules when preparing content.

## Contributing translated pages

1. Identify the original page, section, platform, and source version.
2. Preserve command names, option spelling, code samples, and cross-reference
   targets.
3. Include source attribution and the applicable license with contributed
   data.
4. Review rendered HTML, plain text, Markdown, and raw source where available.
5. Submit the contribution with the source and translation provenance.

Translations should be reviewed by a fluent maintainer before they are
presented as authoritative. casman does not currently provide a browser-based
translation editor.
