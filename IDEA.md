# casman - Universal Man Page Application

## Project Overview

**casman** is a complete, self-contained man page application. Unlike traditional `man` commands that rely on locally installed pages, casman embeds ALL known man pages from multiple operating systems and distributions into a single binary.

Think `man.cgi` but:
- All man pages built-in (no external dependencies)
- Multi-platform coverage (BSD, macOS, Linux, Void, etc.)
- Modern web interface with search
- API for programmatic access
- Single static binary deployment

## Core Features

### Man Page Database
- **Embedded pages**: All man pages compiled into the binary
- **Multi-platform**: BSD (FreeBSD, OpenBSD, NetBSD), macOS, Linux (various distros), Void Linux
- **All sections**: 1-9 (User commands, System calls, Library functions, etc.)
- **Versioned**: Track which OS version each page comes from
- **Searchable**: Full-text search across all pages

### Web Interface
- Browse by section, OS, or alphabetically
- Full-text search with highlighting
- Responsive design (mobile-friendly)
- Syntax highlighting for code examples
- Cross-references (see also) as clickable links
- Dark/light theme support

### API
- REST API for programmatic access
- Search endpoint with filters (OS, section, keyword)
- Get specific man page by name/section/OS
- List available pages with pagination
- GraphQL endpoint for complex queries

### CLI Client
- `casman-cli man <page>` - view man page
- `casman-cli search <term>` - search pages
- `casman-cli list` - list available pages
- TUI mode for interactive browsing

## Man Page Sources

**Strategy: Combined approach** - aggregate from both online archives AND official source repositories for maximum coverage.

### Online Archives
| Source | URL | Coverage |
|--------|-----|----------|
| **man7.org** | man7.org/linux/man-pages/ | Linux man-pages project |
| **manpages.ubuntu.com** | manpages.ubuntu.com | Ubuntu/Debian packages |
| **manpages.debian.org** | manpages.debian.org | Debian stable |
| **FreeBSD Man** | man.freebsd.org | FreeBSD all versions |
| **OpenBSD Man** | man.openbsd.org | OpenBSD current |
| **NetBSD Man** | man.netbsd.org | NetBSD releases |
| **die.net** | linux.die.net/man/ | Linux aggregated |
| **mankier.com** | mankier.com | Multi-distro |

### Official Repositories
| Source | Repository | Description |
|--------|------------|-------------|
| **man-pages** | kernel.org/doc/man-pages | Linux kernel/libc docs |
| **GNU coreutils** | git.savannah.gnu.org | Core utilities (ls, cp, mv) |
| **util-linux** | github.com/util-linux | System utilities |
| **systemd** | github.com/systemd/systemd | systemd man pages |
| **FreeBSD src** | github.com/freebsd/freebsd-src | FreeBSD base system |
| **OpenBSD src** | github.com/openbsd/src | OpenBSD base system |
| **NetBSD src** | github.com/NetBSD/src | NetBSD base system |
| **Apple OSS** | opensource.apple.com | Darwin/XNU pages |
| **Void Linux** | github.com/void-linux | Void-specific pages |
| **Alpine** | gitlab.alpinelinux.org | Alpine/musl pages |

### Coverage by Platform
| Platform | Sources |
|----------|---------|
| **Linux** | man-pages, GNU, util-linux, systemd, distro packages |
| **FreeBSD** | FreeBSD src, man.freebsd.org |
| **OpenBSD** | OpenBSD src, man.openbsd.org |
| **NetBSD** | NetBSD src, man.netbsd.org |
| **DragonFly** | DragonFly gitweb |
| **macOS** | Apple OSS, BSD heritage |
| **Void Linux** | Void packages |
| **Alpine** | Alpine packages |
| **POSIX** | IEEE Std 1003.1 (where available) |

## Data Model

### Man Page
```yaml
page:
  name: "ls"
  section: 1
  title: "list directory contents"
  os: "linux"
  distro: "debian"
  version: "bookworm"

  # Source
  source_format: "groff"          # groff, mdoc, markdown
  source_raw: "<original content>"
  source_url: "https://..."

  # Pre-rendered formats (all generated at build time)
  rendered:
    html: "<html with colorization>"
    txt: "plain text output"
    md: "# ls(1)\n\n**NAME**\n..."

  # Metadata
  see_also: ["dir(1)", "vdir(1)"]
  synopsis: "ls [OPTION]... [FILE]..."
  description_short: "list directory contents"
  updated_at: "2024-01-15"

  # Search indexing
  search_text: "normalized searchable text"
```

### Section Definitions
| Section | Name | Description |
|---------|------|-------------|
| 1 | User Commands | Executable programs, shell commands |
| 2 | System Calls | Kernel system calls |
| 3 | Library Functions | C library functions |
| 4 | Devices | Device files, special files (/dev/*) |
| 5 | File Formats | Configuration file formats |
| 6 | Games | Games and screensavers |
| 7 | Miscellaneous | Conventions, protocols, standards |
| 8 | Admin Commands | System administration commands |
| 9 | Kernel | Kernel routines (Linux-specific) |
| n | New/Tcl | New documentation, Tcl/Tk commands |
| x | X Window System | X11/Xorg related documentation |

## URL Structure

### Web Routes
| Route | Description |
|-------|-------------|
| `/` | Homepage with search |
| `/man/{name}` | Show page (best match) |
| `/man/{section}/{name}` | Show page in section |
| `/man/{os}/{section}/{name}` | Show specific OS page |
| `/man/{lang}/{os}/{section}/{name}` | Full path with language |
| `/search?q={term}` | Search results |
| `/browse` | Browse all pages |
| `/browse/{section}` | Browse by section |
| `/browse/{os}` | Browse by OS |
| `/compare/{name}` | Compare page across platforms |
| `/compare/{section}/{name}` | Compare within section |
| `/whatis/{name}` | One-line description |
| `/apropos?q={term}` | Search descriptions |
| `/export/{type}/{id}` | Bulk export (section, platform) |
| `/sitemap.xml` | Sitemap index |
| `/robots.txt` | Crawler instructions |
| `/feed.xml` | Atom feed (all updates) |
| `/feed/{platform}.xml` | Platform-specific feed |
| `/feed/section/{num}.xml` | Section-specific feed |

### API Routes
| Route | Description |
|-------|-------------|
| `/api/v1/man/{name}` | Get page (JSON) |
| `/api/v1/man/{section}/{name}` | Get page in section |
| `/api/v1/man/{os}/{section}/{name}` | Get specific OS page |
| `/api/v1/man/{lang}/{os}/{section}/{name}` | Full path with language |
| `/api/v1/search?q={term}` | Full search with results |
| `/api/v1/autocomplete?q={term}` | Quick autocomplete (name matches only) |
| `/api/v1/suggest?q={term}` | "Did you mean..." suggestions |
| `/api/v1/sections` | List sections with counts |
| `/api/v1/platforms` | List platforms/OS with counts |
| `/api/v1/languages` | List available languages |
| `/api/v1/popular` | Popular/trending pages |
| `/api/v1/stats` | Database statistics |
| `/api/v1/compare/{name}` | Compare page across platforms |
| `/api/v1/compare/{section}/{name}` | Compare within section |
| `/api/v1/whatis/{name}` | One-line description |
| `/api/v1/apropos?q={term}` | Search descriptions |
| `/api/v1/tldr/{name}` | Get TLDR summary only |
| `/api/v1/export/formats` | List available export formats |

### Search API Parameters
| Parameter | Type | Description |
|-----------|------|-------------|
| `q` | string | Search query (required) |
| `section` | int | Filter by section (1-9) |
| `os` | string | Filter by OS (linux, freebsd, etc.) |
| `distro` | string | Filter by distro (debian, ubuntu, etc.) |
| `type` | string | Search type: `name`, `content`, `all` (default) |
| `page` | int | Page number (default: 1) |
| `limit` | int | Results per page (default: 25, max: 100) |
| `sort` | string | Sort: `relevance`, `name`, `section` |

### Search Response Format
```json
{
  "query": "ls",
  "total": 127,
  "page": 1,
  "limit": 25,
  "results": [
    {
      "name": "ls",
      "section": 1,
      "title": "list directory contents",
      "os": "linux",
      "distro": "debian",
      "snippet": "...list information about <mark>files</mark>...",
      "score": 0.95,
      "url": "/man/linux/1/ls"
    }
  ],
  "suggestions": ["lsof", "lsblk", "lscpu"],
  "filters": {
    "sections": [{"id": 1, "count": 45}, {"id": 8, "count": 32}],
    "platforms": [{"id": "linux", "count": 67}, {"id": "freebsd", "count": 28}]
  }
}

## Search Features

### Search Box
- **Prominent placement**: Center of homepage, sticky header on other pages
- **Autocomplete**: Live suggestions as you type (debounced, 150ms)
- **Recent searches**: Show user's recent searches (localStorage)
- **Popular searches**: Show trending/popular searches
- **Keyboard shortcut**: `/` or `Ctrl+K` to focus search

### Search Types
| Type | Trigger | Example | Behavior |
|------|---------|---------|----------|
| **Quick** | Just type | `ls` | Search names first, then content |
| **Name only** | `name:` prefix | `name:ls` | Only match page names |
| **Content** | `content:` prefix | `content:directory` | Full-text content search |
| **Section** | `section:` or number | `1 ls` or `section:1 ls` | Filter by section |
| **OS** | `os:` prefix | `os:freebsd ls` | Filter by OS/platform |
| **Combined** | Multiple | `os:linux section:1 copy` | All filters combined |

### Search Results
- **Grouped by relevance**: Exact name match → partial name → content match
- **Snippets**: Show matching context with highlighted terms
- **Metadata shown**: Section, OS, short description
- **Pagination**: 25 results per page, infinite scroll option
- **Sort options**: Relevance, name (A-Z), section, OS

### Autocomplete Dropdown
```
┌─────────────────────────────────────────────┐
│ 🔍 ls                                    ✕  │
├─────────────────────────────────────────────┤
│ 📄 ls(1)         - list directory contents  │
│ 📄 lsof(8)       - list open files          │
│ 📄 lsattr(1)     - list file attributes     │
│ 📄 lsblk(8)      - list block devices       │
│ 📄 lscpu(1)      - display CPU info         │
├─────────────────────────────────────────────┤
│ 🔎 Search all for "ls"                      │
└─────────────────────────────────────────────┘
```

## User Interface

### Global Header
```
┌─────────────────────────────────────────────────────────────────┐
│ 🔧 casman    [🔍 Search...          ]  [Section ▼] [OS ▼] [☀/🌙]│
└─────────────────────────────────────────────────────────────────┘
```

### Dropdown Menus

#### Section Dropdown
```
┌──────────────────────────┐
│ All Sections          ▼ │
├──────────────────────────┤
│ ○ All Sections           │
│ ─────────────────────────│
│ ○ 1 - User Commands      │
│ ○ 2 - System Calls       │
│ ○ 3 - Library Functions  │
│ ○ 4 - Special Files      │
│ ○ 5 - File Formats       │
│ ○ 6 - Games              │
│ ○ 7 - Miscellaneous      │
│ ○ 8 - System Admin       │
│ ○ 9 - Kernel             │
└──────────────────────────┘
```

#### OS/Platform Dropdown
```
┌──────────────────────────┐
│ All Platforms         ▼ │
├──────────────────────────┤
│ ○ All Platforms          │
│ ─────────────────────────│
│ ▸ Linux                  │
│   ├ Debian/Ubuntu        │
│   ├ RHEL/Fedora          │
│   ├ Arch                 │
│   ├ Alpine               │
│   └ Void                 │
│ ▸ BSD                    │
│   ├ FreeBSD              │
│   ├ OpenBSD              │
│   ├ NetBSD               │
│   └ DragonFly            │
│ ○ macOS                  │
│ ○ POSIX                  │
└──────────────────────────┘
```

#### Format Dropdown (on man page view)
```
┌──────────────────────────┐
│ View as: HTML         ▼ │
├──────────────────────────┤
│ ● HTML (formatted)       │
│ ○ Plain Text             │
│ ○ Markdown               │
│ ○ Raw Source             │
│ ─────────────────────────│
│ ⬇ Download as...         │
│   └ PDF                  │
└──────────────────────────┘
```

### Homepage Layout
```
┌─────────────────────────────────────────────────────────────────┐
│                         🔧 casman                                │
│                   Universal Man Pages                            │
│                                                                  │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │                     Search Man Pages                       │  │
│  │                                                            │  │
│  │  Search: [________________________]                        │  │
│  │                                                            │  │
│  │  Platform:        [Linux           ▼] (default)           │  │
│  │                                                            │  │
│  │  Section:         [ANY             ▼]                     │  │
│  │                                                            │  │
│  │                   [ Fetch man pages ]                      │  │
│  └───────────────────────────────────────────────────────────┘  │
│                                                                  │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Popular Pages              │  Browse by Section                 │
│  ─────────────              │  ──────────────────                │
│  • ls(1)                    │  📁 1 - User Commands (12,453)    │
│  • grep(1)                  │  📁 2 - System Calls (1,823)      │
│  • chmod(1)                 │  📁 3 - Library (8,921)           │
│  • ssh(1)                   │  📁 8 - Admin Commands (3,291)    │
│  • find(1)                  │  📁 x - X Window System (892)     │
│                             │                                    │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Statistics: 48,592 pages │ 11 sections │ 12 platforms          │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### Search Form HTML Structure
```html
<form action="/search" method="GET" class="search-form">
  <div class="form-group">
    <label for="q">Search:</label>
    <input type="text" id="q" name="q" placeholder="Enter man page name or keyword...">
  </div>

  <div class="form-group">
    <label for="platform">Platform:</label>
    <select id="platform" name="platform">
      <option value="any">Any Platform</option>
      <option value="linux" selected>Linux (default)</option>
      <option value="freebsd">FreeBSD</option>
      <option value="openbsd">OpenBSD</option>
      <option value="netbsd">NetBSD</option>
      <option value="dragonfly">DragonFly BSD</option>
      <option value="macos">macOS</option>
      <option value="void">Void Linux</option>
      <option value="alpine">Alpine Linux</option>
      <option value="posix">POSIX</option>
    </select>
  </div>

  <div class="form-group">
    <label for="section">Section:</label>
    <select id="section" name="section">
      <option value="any">ANY</option>
      <option value="1">1 - User Commands</option>
      <option value="2">2 - System Calls</option>
      <option value="3">3 - Library Functions</option>
      <option value="4">4 - Devices</option>
      <option value="5">5 - File Formats</option>
      <option value="6">6 - Games</option>
      <option value="7">7 - Miscellaneous</option>
      <option value="8">8 - Admin Commands</option>
      <option value="9">9 - Kernel</option>
      <option value="n">n - New/Tcl</option>
      <option value="x">x - X Window System</option>
    </select>
  </div>

  <input type="submit" value="Fetch man pages">
</form>
```

### Man Page View Layout
```
┌─────────────────────────────────────────────────────────────────┐
│ 🔧 casman    [🔍 Search...          ]  [Section ▼] [OS ▼] [☀/🌙]│
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ← Back    ls(1) - list directory contents    [View as ▼] [📋]  │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │ Also available: FreeBSD │ OpenBSD │ NetBSD │ macOS          ││
│  └─────────────────────────────────────────────────────────────┘│
│                                                                  │
│  ┌──────────────┐  ┌───────────────────────────────────────────┐│
│  │ Contents     │  │ NAME                                      ││
│  │ ──────────── │  │   ls - list directory contents            ││
│  │ • NAME       │  │                                           ││
│  │ • SYNOPSIS   │  │ SYNOPSIS                                  ││
│  │ • DESCRIPTION│  │   ls [OPTION]... [FILE]...                ││
│  │ • OPTIONS    │  │                                           ││
│  │ • EXAMPLES   │  │ DESCRIPTION                               ││
│  │ • SEE ALSO   │  │   List information about the FILEs...     ││
│  │              │  │                                           ││
│  └──────────────┘  │ OPTIONS                                   ││
│                    │   -a, --all                                ││
│                    │       do not ignore entries starting...    ││
│                    │                                           ││
│                    │   -l  use a long listing format           ││
│                    │                                           ││
│                    └───────────────────────────────────────────┘│
│                                                                  │
│  SEE ALSO: dir(1), vdir(1), chmod(1), chown(1)                  │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### Interactive Features

#### Table of Contents (Sidebar)
- Auto-generated from page sections
- Sticky sidebar on desktop
- Collapsible on mobile (hamburger menu)
- Highlight current section on scroll
- Click to jump to section

#### Cross-References
- All `SEE ALSO` entries are clickable links
- Hover preview (tooltip with short description)
- Links to same OS version when available, fallback to best match

#### Copy Features
- Copy button on code blocks
- Copy entire page (plain text)
- Copy link to specific section (`#SYNOPSIS`)

#### Keyboard Navigation
| Key | Action |
|-----|--------|
| `/` or `Ctrl+K` | Focus search |
| `Escape` | Close dropdown/modal |
| `↑` / `↓` | Navigate autocomplete |
| `Enter` | Select/submit |
| `j` / `k` | Scroll (vim-style) |
| `g` `g` | Go to top |
| `G` | Go to bottom |
| `[` / `]` | Previous/next section |
| `?` | Show keyboard shortcuts |

### Mobile Responsive
- Hamburger menu for navigation
- Full-width search
- Collapsible TOC
- Swipe gestures for navigation
- Touch-friendly dropdowns

### Theme Support
- Light mode (default)
- Dark mode
- Auto (follow system preference)
- Persist choice in localStorage

## Compare Feature

Compare the same man page across different platforms side-by-side.

### Compare URL Structure
| Route | Description |
|-------|-------------|
| `/compare/{name}` | Compare page across all available platforms |
| `/compare/{name}?platforms=linux,freebsd,macos` | Compare specific platforms |
| `/compare/{section}/{name}` | Compare within specific section |

### Compare Layout
```
┌─────────────────────────────────────────────────────────────────┐
│  Compare: ls(1)                              [Add Platform ▼]   │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐   │
│  │ Linux (GNU)     │ │ FreeBSD         │ │ macOS           │   │
│  │ ───────────────│ │ ───────────────│ │ ───────────────│   │
│  │ NAME            │ │ NAME            │ │ NAME            │   │
│  │  ls - list...   │ │  ls - list...   │ │  ls - list...   │   │
│  │                 │ │                 │ │                 │   │
│  │ SYNOPSIS        │ │ SYNOPSIS        │ │ SYNOPSIS        │   │
│  │  ls [OPTION]... │ │  ls [-ABCFGHILPRSTUWZabcd... │ │  ls [-@ABCFGHILOPRSTUWabcd... │   │
│  │                 │ │                 │ │                 │   │
│  │ ⚠ GNU-specific: │ │ ✓ BSD standard  │ │ ✓ BSD + Apple   │   │
│  │ --color=auto    │ │                 │ │ -@ (extended)   │   │
│  └─────────────────┘ └─────────────────┘ └─────────────────┘   │
│                                                                  │
│  Legend: 🟢 Common  🔵 Platform-specific  🔴 Missing            │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### Compare Features
- Side-by-side scrolling (synchronized or independent)
- Highlight differences (options that exist on one platform but not others)
- Show platform-specific flags/options
- Collapsible sections
- Export comparison as PDF/HTML

## TLDR Summaries

Auto-generated concise summaries shown above full man pages.

### TLDR Generation
- Extract SYNOPSIS for quick command format
- Pull key examples from EXAMPLES section
- Identify most common options (based on usage frequency data)
- Generate 3-5 practical one-liners

### TLDR Display
```
┌─────────────────────────────────────────────────────────────────┐
│  📋 Quick Summary                                    [Collapse] │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ls - list directory contents                                   │
│                                                                  │
│  Common usage:                                                   │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │ ls -la                 # List all files with details        ││
│  │ ls -lh                 # Human-readable sizes               ││
│  │ ls -lt                 # Sort by modification time          ││
│  │ ls -R                  # Recursive listing                  ││
│  │ ls *.txt               # List only .txt files               ││
│  └─────────────────────────────────────────────────────────────┘│
│                                                                  │
│  Key options: -a (all), -l (long), -h (human), -R (recursive)   │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
│                                                                  │
│  📖 Full Man Page                                               │
│  ───────────────                                                │
│  NAME                                                           │
│      ls - list directory contents                               │
│  ...                                                            │
```

### TLDR Data Model
```yaml
tldr:
  name: "ls"
  section: 1
  one_liner: "list directory contents"
  common_examples:
    - cmd: "ls -la"
      desc: "List all files with details"
    - cmd: "ls -lh"
      desc: "Human-readable sizes"
  key_options:
    - flag: "-a"
      desc: "all"
    - flag: "-l"
      desc: "long"
  generated_at: "2025-01-15"
  source: "auto"  # auto-generated vs manual
```

## Bookmarks (localStorage)

Save favorite man pages locally without requiring an account.

### Bookmark Features
- Add/remove bookmarks with one click (star icon)
- Bookmarks stored in localStorage (persists across sessions)
- Bookmark list accessible from header dropdown
- Export/import bookmarks as JSON
- Organize bookmarks with tags (optional)

### Bookmark Storage
```javascript
// localStorage structure
{
  "casman_bookmarks": [
    {
      "name": "ls",
      "section": "1",
      "platform": "linux",
      "added": "2025-01-15T10:30:00Z",
      "tags": ["files", "common"]
    },
    {
      "name": "grep",
      "section": "1",
      "platform": "linux",
      "added": "2025-01-14T15:20:00Z",
      "tags": ["search", "common"]
    }
  ]
}
```

### Bookmark UI
```
┌─────────────────────────────────────────────────────────────────┐
│  ⭐ Bookmarks (5)                                               │
├─────────────────────────────────────────────────────────────────┤
│  📄 ls(1) - Linux                              [✕]              │
│  📄 grep(1) - Linux                            [✕]              │
│  📄 ssh(1) - Linux                             [✕]              │
│  📄 zfs(8) - FreeBSD                           [✕]              │
│  📄 launchd(8) - macOS                         [✕]              │
├─────────────────────────────────────────────────────────────────┤
│  [Export JSON]  [Import]  [Clear All]                           │
└─────────────────────────────────────────────────────────────────┘
```

## Whatis / Apropos

Traditional man page utilities for quick lookups.

### Whatis
One-line descriptions for man pages (like `whatis` command).

| Route | Description |
|-------|-------------|
| `/whatis/{name}` | Show one-line description |
| `/api/v1/whatis/{name}` | JSON response |

```
$ curl casman.local/whatis/ls
ls(1) - list directory contents

$ curl casman.local/whatis/grep
grep(1) - print lines that match patterns
```

### Apropos
Search man page descriptions (like `apropos` command).

| Route | Description |
|-------|-------------|
| `/apropos?q={term}` | Search descriptions for term |
| `/api/v1/apropos?q={term}` | JSON response |

```
$ curl "casman.local/apropos?q=directory"
ls(1) - list directory contents
mkdir(1) - make directories
rmdir(1) - remove empty directories
cd(1) - change the working directory
pwd(1) - print name of current/working directory
tree(1) - list contents of directories in a tree-like format
```

### Apropos Web UI
```
┌─────────────────────────────────────────────────────────────────┐
│  Apropos: Search man page descriptions                          │
│                                                                  │
│  Search: [directory_____________] [Search]                      │
│                                                                  │
│  Results for "directory":                                       │
│  ─────────────────────────                                      │
│  • ls(1) - list directory contents                              │
│  • mkdir(1) - make directories                                  │
│  • rmdir(1) - remove empty directories                          │
│  • cd(1) - change the working directory                         │
│  • pwd(1) - print name of current/working directory             │
│  • tree(1) - list contents of directories in a tree-like format │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

## Translations (i18n)

Support localized man pages where available.

### Supported Locales
| Code | Language | Coverage |
|------|----------|----------|
| en | English | 100% (default) |
| de | German | ~40% |
| fr | French | ~35% |
| ja | Japanese | ~30% |
| zh | Chinese | ~25% |
| es | Spanish | ~20% |
| ru | Russian | ~15% |
| pt | Portuguese | ~10% |

### Translation URL Structure
| Route | Description |
|-------|-------------|
| `/man/{name}` | Default language (English) |
| `/man/{lang}/{name}` | Specific language |
| `/man/{lang}/{section}/{name}` | Language + section |
| `/man/{lang}/{platform}/{section}/{name}` | Full path |

### Translation Examples
```
/man/ls              → English (default)
/man/de/ls           → German translation
/man/ja/1/ls         → Japanese, section 1
/man/fr/linux/1/ls   → French, Linux, section 1
```

### Language Selector
```
┌─────────────────────────────────────────────────────────────────┐
│  ls(1) - list directory contents                                │
│                                                                  │
│  Language: [English ▼]                                          │
│            ├─ English (original)                                │
│            ├─ Deutsch (German)                                  │
│            ├─ Français (French)                                 │
│            ├─ 日本語 (Japanese)                                  │
│            └─ 中文 (Chinese)                                     │
│                                                                  │
│  ⚠️ Translation may be outdated. Last updated: 2024-06-15       │
│     English version updated: 2025-01-10                          │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### Translation Sources
- Linux: man-pages-l10n project
- FreeBSD: FreeBSD doc translations
- Community contributions

## Export (PDF/EPUB)

Download man pages in various formats for offline use.

### Export Formats
| Format | Extension | Use Case |
|--------|-----------|----------|
| PDF | .pdf | Print, archive |
| EPUB | .epub | E-readers (Kindle, Kobo) |
| MOBI | .mobi | Older Kindle devices |
| Single HTML | .html | Offline browser viewing |

### Export URL Structure
```
/man/ls.pdf           → Download PDF
/man/ls.epub          → Download EPUB
/man/1/ls.pdf         → Section 1, PDF
/export/section/1.pdf → All section 1 pages as single PDF
/export/platform/linux.epub → All Linux pages as EPUB
```

### Export Options Dialog
```
┌─────────────────────────────────────────────────────────────────┐
│  Export: ls(1)                                                  │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Format:  ○ PDF  ○ EPUB  ○ MOBI  ○ HTML                        │
│                                                                  │
│  Include: ☑ Table of contents                                  │
│           ☑ TLDR summary                                        │
│           ☐ SEE ALSO pages (bundle related pages)              │
│           ☐ All platforms (comparison bundle)                   │
│                                                                  │
│  Language: [English ▼]                                          │
│                                                                  │
│  [Download]  [Cancel]                                           │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### Bulk Export
- Export entire sections as single file
- Export by platform
- Export bookmarked pages as bundle
- Generate "man page book" with TOC

## History (localStorage)

Track recently viewed man pages for quick access.

### History Features
- Automatically track viewed pages
- Store last 50 pages (configurable)
- Clear history option
- No account required (localStorage)
- Disable tracking option (privacy)

### History Storage
```javascript
// localStorage structure
{
  "casman_history": [
    {
      "name": "ls",
      "section": "1",
      "platform": "linux",
      "viewed": "2025-01-15T10:30:00Z"
    },
    {
      "name": "grep",
      "section": "1",
      "platform": "linux",
      "viewed": "2025-01-15T10:25:00Z"
    }
  ],
  "casman_history_enabled": true,
  "casman_history_max": 50
}
```

### History UI
```
┌─────────────────────────────────────────────────────────────────┐
│  🕐 History                                         [Clear All] │
├─────────────────────────────────────────────────────────────────┤
│  Today                                                          │
│  • ls(1) - Linux                              10:30 AM          │
│  • grep(1) - Linux                            10:25 AM          │
│  • ssh(1) - Linux                             10:20 AM          │
│                                                                  │
│  Yesterday                                                       │
│  • chmod(1) - Linux                           3:45 PM           │
│  • zfs(8) - FreeBSD                           2:30 PM           │
├─────────────────────────────────────────────────────────────────┤
│  ☐ Disable history tracking                                     │
└─────────────────────────────────────────────────────────────────┘
```

## Print Stylesheet

Optimized CSS for printing man pages.

### Print Features
- Clean, readable layout without navigation
- Proper page breaks (avoid breaking code blocks)
- Monospace font for code/synopsis
- Headers with page name and section
- Page numbers in footer
- QR code with URL (optional)
- Hide interactive elements (buttons, dropdowns)

### Print CSS
```css
@media print {
  /* Hide navigation, search, buttons */
  header, nav, .search-form, .sidebar,
  .bookmark-btn, .copy-btn, footer {
    display: none !important;
  }

  /* Clean layout */
  body {
    font-size: 11pt;
    line-height: 1.4;
    color: #000;
    background: #fff;
  }

  /* Code blocks */
  pre, code, .synopsis {
    font-family: "Courier New", monospace;
    font-size: 10pt;
    page-break-inside: avoid;
  }

  /* Page breaks */
  h1, h2 { page-break-after: avoid; }
  pre, blockquote { page-break-inside: avoid; }

  /* Header on each page */
  .man-header {
    position: running(header);
    font-weight: bold;
  }

  @page {
    margin: 2cm;
    @top-center { content: element(header); }
    @bottom-center { content: counter(page); }
  }
}
```

### Print Preview
- "Print this page" button
- Print preview modal
- Options: include TLDR, include SEE ALSO descriptions

## Sitemap & SEO

Search engine optimization for discoverability.

### Sitemap Files
| File | Description |
|------|-------------|
| `/sitemap.xml` | Main sitemap index |
| `/sitemap-pages.xml` | All man pages |
| `/sitemap-sections.xml` | Section browse pages |
| `/sitemap-platforms.xml` | Platform browse pages |
| `/robots.txt` | Crawler instructions |

### Sitemap Structure
```xml
<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url>
    <loc>https://casman.example.com/man/ls</loc>
    <lastmod>2025-01-15</lastmod>
    <changefreq>monthly</changefreq>
    <priority>0.8</priority>
  </url>
  <!-- ... thousands more ... -->
</urlset>
```

### robots.txt
```
User-agent: *
Allow: /

# Sitemaps
Sitemap: https://casman.example.com/sitemap.xml

# Rate limiting hint
Crawl-delay: 1

# Exclude admin, API (except docs)
Disallow: /admin/
Disallow: /api/
Allow: /api/v1/openapi
```

### SEO Meta Tags
```html
<head>
  <title>ls(1) - list directory contents | casman</title>
  <meta name="description" content="Linux man page for ls - list directory contents. View synopsis, options, examples, and related commands.">
  <meta name="keywords" content="ls, man page, linux, directory, list files, command">

  <!-- Open Graph -->
  <meta property="og:title" content="ls(1) - list directory contents">
  <meta property="og:description" content="Linux man page for ls command">
  <meta property="og:type" content="article">
  <meta property="og:url" content="https://casman.example.com/man/linux/1/ls">

  <!-- Twitter -->
  <meta name="twitter:card" content="summary">
  <meta name="twitter:title" content="ls(1) - list directory contents">

  <!-- Canonical URL -->
  <link rel="canonical" href="https://casman.example.com/man/linux/1/ls">

  <!-- JSON-LD structured data -->
  <script type="application/ld+json">
  {
    "@context": "https://schema.org",
    "@type": "TechArticle",
    "name": "ls(1)",
    "headline": "ls - list directory contents",
    "description": "List information about the FILEs...",
    "operatingSystem": "Linux",
    "applicationCategory": "Command Line Tool"
  }
  </script>
</head>
```

### URL Structure for SEO
- Human-readable URLs: `/man/linux/1/ls` not `/man?id=12345`
- Consistent canonical URLs
- Proper redirects (301) for alternate paths
- Language alternates with hreflang

## RSS/Atom Feeds

Subscribe to updates for new and modified man pages.

### Feed URLs
| Feed | URL | Description |
|------|-----|-------------|
| All updates | `/feed.xml` | All new/updated pages |
| By platform | `/feed/linux.xml` | Linux updates only |
| By section | `/feed/section/1.xml` | Section 1 updates |
| Combined | `/feed/linux/1.xml` | Linux section 1 |

### Feed Content
```xml
<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>casman - Man Page Updates</title>
  <subtitle>New and updated Unix/Linux man pages</subtitle>
  <link href="https://casman.example.com/feed.xml" rel="self"/>
  <link href="https://casman.example.com/"/>
  <updated>2025-01-15T10:30:00Z</updated>
  <id>https://casman.example.com/</id>

  <entry>
    <title>ls(1) updated - Linux</title>
    <link href="https://casman.example.com/man/linux/1/ls"/>
    <id>https://casman.example.com/man/linux/1/ls#2025-01-15</id>
    <updated>2025-01-15T10:30:00Z</updated>
    <summary>list directory contents - Updated options documentation</summary>
    <category term="linux"/>
    <category term="section-1"/>
    <author>
      <name>man-pages project</name>
    </author>
  </entry>

  <entry>
    <title>zfs(8) added - FreeBSD</title>
    <link href="https://casman.example.com/man/freebsd/8/zfs"/>
    <id>https://casman.example.com/man/freebsd/8/zfs#2025-01-14</id>
    <updated>2025-01-14T15:00:00Z</updated>
    <summary>ZFS file system - New page added from FreeBSD 14</summary>
    <category term="freebsd"/>
    <category term="section-8"/>
  </entry>
</feed>
```

### Feed Discovery
```html
<head>
  <!-- Auto-discovery -->
  <link rel="alternate" type="application/atom+xml"
        title="casman Updates" href="/feed.xml">
  <link rel="alternate" type="application/atom+xml"
        title="Linux Updates" href="/feed/linux.xml">
</head>
```

### Feed Options
- Filter by platform, section, or both
- "New pages only" vs "All updates"
- JSON Feed format alternative (`/feed.json`)

## Page Rendering

**Strategy: Combined toolchain** - use both mandoc and groff for parsing, pre-render all formats at build time, store in standardized internal format.

### Input Formats
- **groff/troff**: Traditional man page format (Linux standard)
- **mdoc**: BSD mandoc format (BSD/macOS standard)
- **Markdown**: Some modern pages (converted to internal format)

### Rendering Pipeline
```
Source (groff/mdoc/md)
    │
    ├─► mandoc parser (primary - handles both formats well)
    │
    ├─► groff parser (fallback - for complex groff macros)
    │
    └─► Standardized Internal Format (AST/JSON)
            │
            ├─► HTML (full colorization, syntax highlighting)
            ├─► Plain text (.txt)
            ├─► Markdown (.md - pretty formatted)
            └─► [Extensible for future formats]
```

### Output Formats (via URL extension)
| Extension | Format | Content-Type | Use Case |
|-----------|--------|--------------|----------|
| (none) | HTML | text/html | Web browser |
| `.html` | HTML | text/html | Explicit HTML request |
| `.txt` | Plain text | text/plain | Terminal, curl |
| `.md` | Markdown | text/markdown | Documentation, copy-paste |
| `.json` | JSON (metadata + content) | application/json | API integration |
| `.raw` | Original source | text/plain | View raw groff/mdoc |

### URL Examples
```
/man/ls              → HTML (default)
/man/ls.txt          → Plain text
/man/ls.md           → Markdown
/man/ls.json         → JSON with metadata
/man/ls.raw          → Original groff source
/man/1/ls.txt        → Section 1, plain text
/man/freebsd/1/ls.md → FreeBSD section 1, markdown
```

### HTML Rendering Features
- Full colorization (section headers, options, examples)
- Syntax highlighting for code blocks
- Clickable cross-references (SEE ALSO → links)
- Collapsible sections (optional)
- Copy-to-clipboard for commands
- Responsive tables

### Build-Time Pre-Rendering
All formats are pre-rendered during build:
1. Parse source with mandoc/groff
2. Convert to internal AST
3. Generate HTML, TXT, MD from AST
4. Compress and embed all formats
5. No runtime rendering needed (fast serving)

## Build Process

### Man Page Collection
1. Download/clone man page sources
2. Parse groff/mdoc into structured data
3. Render to HTML and plain text
4. Store in embedded database (SQLite compiled in)
5. Index for full-text search

### Embedding Strategy
- Use Go `embed` for static assets
- Compress man page content (gzip)
- SQLite database embedded as binary blob
- Build-time generation of search index

## Admin Features

- View database statistics
- See popular searches/pages
- Manual page refresh (future updates)
- Cache management

## Future Considerations

- **Updates**: Mechanism to update embedded pages
- **User contributions**: Allow corrections/additions
- **API rate limiting**: For public deployments
- **Caching**: CDN-friendly responses

## Non-Features (Out of Scope)

- User accounts (read-only public service)
- Page editing through web UI
- Custom page uploads
- Real-time page fetching from external sources
