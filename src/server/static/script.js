// casman - Universal Man Page Viewer

(function() {
    'use strict';

    // Initialize when DOM is ready
    document.addEventListener('DOMContentLoaded', init);

    function init() {
        initSearch();
        initBookmarks();
        initHistory();
        initKeyboardShortcuts();
        initThemeToggle();
        initHamburger();
        initBookmarksDropdown();
        initHistoryDropdown();
        initToC();
        initCopyButtons();
        initSeeAlsoPreview();
        initCompareDiff();
        initBookmarkTags();
        initI18nStaleWarning();
        initSwipe();
    }

    // -------------------------------------------------------------------
    // Swipe gestures (IDEA.md "Mobile Responsive — Swipe gestures for
    // navigation"). On the manpage view, a horizontal swipe of more than
    // 60 px while not over a code block scrolls to the previous (right
    // swipe) or next (left swipe) heading — mirroring the [/] keys.
    // -------------------------------------------------------------------
    function initSwipe() {
        const manpage = document.querySelector('.manpage');
        if (!manpage) return;
        // Don't bother attaching on devices without touch.
        if (!('ontouchstart' in window) && !navigator.maxTouchPoints) return;

        let startX = 0, startY = 0, started = false;
        manpage.addEventListener('touchstart', function(e) {
            if (e.touches.length !== 1) { started = false; return; }
            // Ignore swipes that begin inside a scrollable code block — they
            // belong to the horizontal pre.
            if (e.target.closest('pre')) { started = false; return; }
            const t = e.touches[0];
            startX = t.clientX;
            startY = t.clientY;
            started = true;
        }, { passive: true });

        manpage.addEventListener('touchend', function(e) {
            if (!started) return;
            started = false;
            const t = (e.changedTouches && e.changedTouches[0]) || null;
            if (!t) return;
            const dx = t.clientX - startX;
            const dy = t.clientY - startY;
            // Require dominantly horizontal motion >= 60 px.
            if (Math.abs(dx) < 60 || Math.abs(dx) <= Math.abs(dy)) return;
            jumpHeading(dx > 0 ? -1 : +1);
        }, { passive: true });
    }

    // -------------------------------------------------------------------
    // Compare page diff highlighting (IDEA.md "Compare Features").
    // For each option list across the rendered .compare-card cards,
    // tag the entry with .opt-common / .opt-specific based on which
    // platforms also list it. Platforms missing a given option get a
    // .opt-missing placeholder so the legend reads truthfully.
    // -------------------------------------------------------------------
    function initCompareDiff() {
        const grid = document.getElementById('compareGrid');
        if (!grid) return;
        const cards = Array.from(grid.querySelectorAll('.compare-card'));
        if (cards.length < 2) return;

        // Collect option sets per platform.
        const sets = cards.map(c => {
            const items = Array.from(c.querySelectorAll('.compare-options li'));
            return { card: c, opts: new Set(items.map(li => li.dataset.option || li.textContent.trim())), liByOpt: indexBy(items) };
        });
        // Union of all options.
        const all = new Set();
        sets.forEach(s => s.opts.forEach(o => all.add(o)));

        sets.forEach(({ opts, liByOpt }, i) => {
            // Mark each rendered option as common (in every set) or specific.
            opts.forEach(o => {
                const li = liByOpt[o];
                if (!li) return;
                const inAll = sets.every(s => s.opts.has(o));
                li.classList.add(inAll ? 'opt-common' : 'opt-specific');
            });
            // Add a 'missing' placeholder for any option present elsewhere
            // but absent from this card, so the diff is symmetric.
            const ul = sets[i].card.querySelector('.compare-options');
            if (!ul) return;
            all.forEach(o => {
                if (opts.has(o)) return;
                const li = document.createElement('li');
                li.className = 'opt-missing';
                li.dataset.option = o;
                li.textContent = o;
                ul.appendChild(li);
            });
        });
    }

    function indexBy(items) {
        const out = {};
        for (const li of items) {
            const k = li.dataset.option || li.textContent.trim();
            out[k] = li;
        }
        return out;
    }

    // -------------------------------------------------------------------
    // Bookmark tags (IDEA.md tags array). When you bookmark a page, an
    // inline editor under the star button accepts comma-separated tags
    // and persists them on the existing bookmark record.
    // -------------------------------------------------------------------
    function initBookmarkTags() {
        const star = document.querySelector('.bookmark-btn');
        if (!star) return;
        const wrap = document.createElement('div');
        wrap.className = 'bookmark-tags';
        wrap.style.display = 'none';
        const inp = document.createElement('input');
        inp.type = 'text';
        inp.placeholder = 'tags (comma separated)';
        inp.setAttribute('aria-label', 'Bookmark tags');
        const save = document.createElement('button');
        save.type = 'button';
        save.textContent = 'Save tags';
        save.className = 'btn btn-small';
        wrap.appendChild(inp);
        wrap.appendChild(save);
        star.parentElement.appendChild(wrap);

        function key() {
            return star.dataset.platform + '/' + star.dataset.section + '/' + star.dataset.name;
        }
        function render() {
            const map = getBookmarks();
            const b = map[key()];
            if (b) {
                wrap.style.display = '';
                inp.value = (b.tags || []).join(', ');
            } else {
                wrap.style.display = 'none';
            }
        }
        render();
        star.addEventListener('click', () => setTimeout(render, 0));
        save.addEventListener('click', function() {
            const map = getBookmarks();
            const k = key();
            if (!map[k]) return;
            map[k].tags = inp.value.split(',').map(s => s.trim()).filter(Boolean);
            localStorage.setItem('casman_bookmarks', JSON.stringify(map));
            save.textContent = '✓ Saved';
            setTimeout(() => { save.textContent = 'Save tags'; }, 1200);
        });
    }

    // -------------------------------------------------------------------
    // Translation "may be outdated" warning (IDEA.md i18n note). When the
    // page advertises X-Content-Language as non-English (handled by the
    // /api/v1/man/{lang}/.../ route) or the manpage element exposes
    // data-lang, surface a banner. Currently no translated content is
    // stored, but the affordance is in place per spec.
    // -------------------------------------------------------------------
    function initI18nStaleWarning() {
        const manpage = document.querySelector('.manpage[data-lang]');
        if (!manpage) return;
        const lang = manpage.getAttribute('data-lang');
        if (!lang || lang.toLowerCase() === 'en') return;
        const banner = document.createElement('aside');
        banner.className = 'i18n-stale-banner';
        banner.setAttribute('role', 'note');
        banner.textContent = 'Translation (' + lang + ') may be outdated. The English version is the source of truth.';
        manpage.insertBefore(banner, manpage.firstChild);
    }

    // -------------------------------------------------------------------
    // Mobile hamburger menu — toggles a `nav-open` class on <body> that
    // CSS uses to show the nav drawer below 768px.
    // -------------------------------------------------------------------
    function initHamburger() {
        const btn = document.getElementById('hamburgerBtn');
        const nav = document.getElementById('mainNav');
        if (!btn || !nav) return;
        function setOpen(open) {
            document.body.classList.toggle('nav-open', open);
            btn.setAttribute('aria-expanded', open ? 'true' : 'false');
            btn.setAttribute('aria-label', open ? 'Close menu' : 'Open menu');
        }
        btn.addEventListener('click', function() {
            setOpen(!document.body.classList.contains('nav-open'));
        });
        // Close drawer when a nav link is clicked.
        nav.addEventListener('click', function(e) {
            if (e.target.tagName === 'A') setOpen(false);
        });
    }

    // -------------------------------------------------------------------
    // Header dropdowns: bookmarks + history.
    // Generic toggle helper so both buttons share behavior — esc/outside-
    // click close, aria-expanded sync, single-instance dropdown so opening
    // one auto-closes the other.
    // -------------------------------------------------------------------
    function buildHeaderDropdown(triggerId, render) {
        const btn = document.getElementById(triggerId);
        if (!btn) return;
        const dropdown = document.createElement('div');
        dropdown.className = 'header-dropdown';
        dropdown.setAttribute('role', 'menu');
        dropdown.style.display = 'none';
        document.body.appendChild(dropdown);

        function position() {
            const rect = btn.getBoundingClientRect();
            dropdown.style.position = 'fixed';
            dropdown.style.top = (rect.bottom + 4) + 'px';
            // Right-anchor so it doesn't overflow viewport.
            const right = window.innerWidth - rect.right;
            dropdown.style.right = right + 'px';
        }
        function close() {
            dropdown.style.display = 'none';
            btn.setAttribute('aria-expanded', 'false');
        }
        function open() {
            // Close any other open dropdowns first.
            document.querySelectorAll('.header-dropdown').forEach(d => {
                if (d !== dropdown) d.style.display = 'none';
            });
            dropdown.innerHTML = render();
            position();
            dropdown.style.display = 'block';
            btn.setAttribute('aria-expanded', 'true');
        }
        btn.addEventListener('click', function(e) {
            e.stopPropagation();
            if (dropdown.style.display === 'block') close(); else open();
        });
        document.addEventListener('click', function(e) {
            if (dropdown.style.display === 'block' && !dropdown.contains(e.target) && e.target !== btn) {
                close();
            }
        });
        document.addEventListener('keydown', function(e) {
            if (e.key === 'Escape' && dropdown.style.display === 'block') close();
        });
        window.addEventListener('resize', function() {
            if (dropdown.style.display === 'block') position();
        });
        return { dropdown, close, refresh: function() { if (dropdown.style.display === 'block') dropdown.innerHTML = render(); } };
    }

    function initBookmarksDropdown() {
        const countEl = document.getElementById('bookmarksCount');
        function refreshCount() {
            if (!countEl) return;
            const n = Object.keys(getBookmarks()).length;
            countEl.textContent = String(n);
            countEl.style.display = n ? '' : 'none';
        }
        refreshCount();

        function renderList() {
            const list = Object.values(getBookmarks()).sort((a, b) => (b.added || 0) - (a.added || 0));
            let html = '<div class="dropdown-header">⭐ Bookmarks (' + list.length + ')</div>';
            if (list.length === 0) {
                html += '<div class="dropdown-empty">No bookmarks yet. Click the star on any man page.</div>';
            } else {
                html += '<ul class="dropdown-list">';
                for (const b of list) {
                    const url = '/man/' + encodeURIComponent(b.platform) + '/' + encodeURIComponent(b.section) + '/' + encodeURIComponent(b.name);
                    html += '<li><a href="' + url + '">' + escapeHTML(b.name) + '(' + escapeHTML(b.section) + ') &middot; ' + escapeHTML(b.platform) + '</a>'
                          + ' <button class="dropdown-x" data-bm-remove="' + escapeAttr(b.platform + '/' + b.section + '/' + b.name) + '" aria-label="Remove">✕</button></li>';
                }
                html += '</ul>';
            }
            html += '<div class="dropdown-actions">'
                  + '<button data-bm-action="export">Export JSON</button>'
                  + '<button data-bm-action="import">Import</button>'
                  + '<button data-bm-action="clear">Clear all</button>'
                  + '</div>';
            return html;
        }

        const ctrl = buildHeaderDropdown('bookmarksBtn', renderList);
        if (!ctrl) return;

        ctrl.dropdown.addEventListener('click', function(e) {
            const removeKey = e.target.getAttribute && e.target.getAttribute('data-bm-remove');
            if (removeKey) {
                const map = getBookmarks();
                delete map[removeKey];
                localStorage.setItem('casman_bookmarks', JSON.stringify(map));
                refreshCount();
                ctrl.refresh();
                return;
            }
            const action = e.target.getAttribute && e.target.getAttribute('data-bm-action');
            if (action === 'export') {
                const blob = new Blob([JSON.stringify(getBookmarks(), null, 2)], { type: 'application/json' });
                const a = document.createElement('a');
                a.href = URL.createObjectURL(blob);
                a.download = 'casman-bookmarks.json';
                a.click();
                URL.revokeObjectURL(a.href);
            } else if (action === 'import') {
                const inp = document.createElement('input');
                inp.type = 'file';
                inp.accept = 'application/json';
                inp.onchange = function() {
                    const f = inp.files && inp.files[0];
                    if (!f) return;
                    const reader = new FileReader();
                    reader.onload = function() {
                        try {
                            const parsed = JSON.parse(String(reader.result));
                            if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
                                const merged = Object.assign({}, getBookmarks(), parsed);
                                localStorage.setItem('casman_bookmarks', JSON.stringify(merged));
                                refreshCount();
                                ctrl.refresh();
                            }
                        } catch(_) { alert('Could not parse bookmarks JSON.'); }
                    };
                    reader.readAsText(f);
                };
                inp.click();
            } else if (action === 'clear') {
                if (confirm('Remove all bookmarks?')) {
                    localStorage.setItem('casman_bookmarks', '{}');
                    refreshCount();
                    ctrl.refresh();
                }
            }
        });
    }

    function initHistoryDropdown() {
        function renderList() {
            const list = getHistory();
            let html = '<div class="dropdown-header">🕐 History (' + list.length + ')</div>';
            if (list.length === 0) {
                html += '<div class="dropdown-empty">No history yet. Browse a man page to populate.</div>';
            } else {
                html += '<ul class="dropdown-list">';
                for (const h of list.slice(0, 25)) {
                    const url = '/man/' + encodeURIComponent(h.platform) + '/' + encodeURIComponent(h.section) + '/' + encodeURIComponent(h.name);
                    html += '<li><a href="' + url + '">' + escapeHTML(h.name) + '(' + escapeHTML(h.section) + ') &middot; ' + escapeHTML(h.platform) + '</a></li>';
                }
                html += '</ul>';
            }
            html += '<div class="dropdown-actions">'
                  + '<label><input type="checkbox" data-hist-action="toggle" ' + (isHistoryEnabled() ? 'checked' : '') + '> Track history</label>'
                  + '<button data-hist-action="clear">Clear all</button>'
                  + '</div>';
            return html;
        }

        const ctrl = buildHeaderDropdown('historyBtn', renderList);
        if (!ctrl) return;

        ctrl.dropdown.addEventListener('click', function(e) {
            const action = e.target.getAttribute && e.target.getAttribute('data-hist-action');
            if (action === 'clear') {
                if (confirm('Clear all history?')) {
                    clearHistory();
                    ctrl.refresh();
                }
            }
        });
        ctrl.dropdown.addEventListener('change', function(e) {
            if (e.target && e.target.getAttribute('data-hist-action') === 'toggle') {
                setHistoryEnabled(e.target.checked);
                ctrl.refresh();
            }
        });
    }

    // -------------------------------------------------------------------
    // Table-of-Contents sidebar built from the rendered manpage's H2
    // headings. Sticky on desktop; collapsible on narrow screens.
    // -------------------------------------------------------------------
    function initToC() {
        const manpage = document.querySelector('.manpage');
        if (!manpage) return;
        const headings = manpage.querySelectorAll('.manpage-content h1, .manpage-content h2, .synopsis h2');
        if (headings.length < 2) return;

        const sidebar = document.createElement('aside');
        sidebar.className = 'toc-sidebar';
        sidebar.setAttribute('aria-label', 'Table of contents');

        const toggle = document.createElement('button');
        toggle.className = 'toc-toggle';
        toggle.setAttribute('aria-expanded', 'true');
        toggle.setAttribute('aria-controls', 'tocList');
        toggle.textContent = 'Contents';

        const list = document.createElement('ol');
        list.id = 'tocList';
        list.className = 'toc-list';

        headings.forEach((h, i) => {
            if (!h.id) {
                h.id = 'sec-' + (h.textContent || ('s' + i)).toLowerCase()
                    .replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '') || ('sec-' + i);
            }
            const li = document.createElement('li');
            const a = document.createElement('a');
            a.href = '#' + h.id;
            a.textContent = h.textContent || ('Section ' + (i + 1));
            a.dataset.tocTarget = h.id;
            li.appendChild(a);
            list.appendChild(li);
        });

        sidebar.appendChild(toggle);
        sidebar.appendChild(list);
        manpage.classList.add('manpage-with-toc');
        manpage.insertBefore(sidebar, manpage.firstChild);

        toggle.addEventListener('click', function() {
            const open = list.style.display !== 'none';
            list.style.display = open ? 'none' : '';
            toggle.setAttribute('aria-expanded', open ? 'false' : 'true');
        });

        // Highlight currently scrolled-into-view section.
        const links = list.querySelectorAll('a');
        function update() {
            let active = null;
            const top = window.scrollY + 120;
            headings.forEach(h => {
                if (h.offsetTop <= top) active = h.id;
            });
            links.forEach(a => {
                a.classList.toggle('toc-active', a.dataset.tocTarget === active);
            });
        }
        window.addEventListener('scroll', update, { passive: true });
        update();
    }

    // -------------------------------------------------------------------
    // Copy buttons:
    //   * each <pre> gets a "Copy" button that copies its text content
    //   * each h2/h3 in the manpage gets a "🔗" button that copies the
    //     heading-anchored URL (#id) per IDEA.md "Copy link to section"
    //   * the manpage header gets a "Copy page" button that copies the
    //     whole rendered text content
    // -------------------------------------------------------------------
    function initCopyButtons() {
        const manpage = document.querySelector('.manpage');
        if (!manpage) return;

        manpage.querySelectorAll('pre').forEach(pre => {
            if (pre.querySelector('.copy-btn')) return;
            const btn = document.createElement('button');
            btn.type = 'button';
            btn.className = 'copy-btn copy-pre';
            btn.textContent = 'Copy';
            btn.setAttribute('aria-label', 'Copy code');
            btn.addEventListener('click', () => copyText(pre.innerText, btn));
            pre.appendChild(btn);
        });

        manpage.querySelectorAll('.manpage-content h2[id], .manpage-content h3[id]').forEach(h => {
            if (h.querySelector('.copy-link')) return;
            const a = document.createElement('button');
            a.type = 'button';
            a.className = 'copy-link';
            a.textContent = '🔗';
            a.setAttribute('aria-label', 'Copy link to ' + (h.textContent || 'section'));
            a.addEventListener('click', function(e) {
                e.preventDefault();
                const url = location.origin + location.pathname + '#' + h.id;
                copyText(url, a);
            });
            h.appendChild(a);
        });

        const actions = manpage.querySelector('.actions');
        if (actions && !actions.querySelector('.copy-page')) {
            const btn = document.createElement('button');
            btn.type = 'button';
            btn.className = 'btn btn-small copy-page';
            btn.textContent = 'Copy page';
            btn.addEventListener('click', () => {
                const content = manpage.querySelector('.manpage-content');
                copyText(content ? content.innerText : manpage.innerText, btn);
            });
            actions.appendChild(btn);
        }
    }

    function copyText(text, btn) {
        const restore = btn.textContent;
        function ok() {
            btn.textContent = '✓ Copied';
            setTimeout(() => { btn.textContent = restore; }, 1500);
        }
        if (navigator.clipboard && navigator.clipboard.writeText) {
            navigator.clipboard.writeText(text).then(ok).catch(() => fallback(text, btn, restore));
        } else {
            fallback(text, btn, restore);
        }
    }

    function fallback(text, btn, restore) {
        const ta = document.createElement('textarea');
        ta.value = text;
        ta.style.position = 'fixed';
        ta.style.opacity = '0';
        document.body.appendChild(ta);
        ta.select();
        try { document.execCommand('copy'); btn.textContent = '✓ Copied'; }
        catch(_) { btn.textContent = '✗ Failed'; }
        document.body.removeChild(ta);
        setTimeout(() => { btn.textContent = restore; }, 1500);
    }

    // -------------------------------------------------------------------
    // Cross-reference hover preview: hovering a SEE ALSO link fetches
    // /api/v1/whatis/{name} and shows a tooltip with the one-line
    // description. Per IDEA.md "Cross-references — Hover preview".
    // -------------------------------------------------------------------
    function initSeeAlsoPreview() {
        const seeAlso = document.querySelector('.see-also');
        if (!seeAlso) return;
        const cache = new Map();
        let tip = null;
        let hideTimer = null;

        function ensureTip() {
            if (tip) return tip;
            tip = document.createElement('div');
            tip.className = 'whatis-tooltip';
            tip.setAttribute('role', 'tooltip');
            tip.style.display = 'none';
            document.body.appendChild(tip);
            return tip;
        }

        function showTip(text, target) {
            const t = ensureTip();
            t.textContent = text;
            const rect = target.getBoundingClientRect();
            t.style.position = 'fixed';
            t.style.left = rect.left + 'px';
            t.style.top = (rect.bottom + 6) + 'px';
            t.style.display = 'block';
        }
        function hideTip() {
            if (tip) tip.style.display = 'none';
        }

        seeAlso.addEventListener('mouseover', function(e) {
            const a = e.target.closest('a');
            if (!a || !a.href) return;
            // SEE ALSO links look like /man/{platform}/{section}/{name} or /man/{section}/{name}
            const parts = a.getAttribute('href').split('/');
            const name = parts[parts.length - 1];
            if (!name) return;
            clearTimeout(hideTimer);
            if (cache.has(name)) {
                showTip(cache.get(name), a);
                return;
            }
            fetch('/api/v1/whatis/' + encodeURIComponent(name), { headers: { Accept: 'application/json' } })
                .then(r => r.ok ? r.json() : null)
                .then(data => {
                    if (!data || !data.length) return;
                    const first = data[0];
                    const text = (first.name || name) + '(' + (first.section || '?') + ') — ' + (first.title || '');
                    cache.set(name, text);
                    if (a.matches(':hover')) showTip(text, a);
                })
                .catch(() => {});
        });
        seeAlso.addEventListener('mouseout', function(e) {
            const a = e.target.closest('a');
            if (!a) return;
            hideTimer = setTimeout(hideTip, 200);
        });
    }

    function escapeHTML(s) {
        return String(s).replace(/[&<>"']/g, c => ({
            '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;'
        }[c]));
    }
    function escapeAttr(s) { return escapeHTML(s); }

    // -------------------------------------------------------------------
    // Recent searches (LRU, max 10) — surfaced in the focus-time dropdown.
    // -------------------------------------------------------------------
    function getRecentSearches() {
        try {
            const raw = JSON.parse(localStorage.getItem('casman_recent_searches') || '[]');
            return Array.isArray(raw) ? raw.slice(0, 10) : [];
        } catch(_) { return []; }
    }
    function addRecentSearch(q) {
        const list = getRecentSearches().filter(x => x !== q);
        list.unshift(q);
        try { localStorage.setItem('casman_recent_searches', JSON.stringify(list.slice(0, 10))); } catch(_) {}
    }

    // Theme toggle (light / dark / auto) per IDEA.md.
    function initThemeToggle() {
        const btn = document.getElementById('themeToggle');
        if (!btn) return;
        const order = ['auto', 'light', 'dark'];
        const labels = { auto: '🌓 Auto', light: '☀ Light', dark: '🌙 Dark' };
        function apply(t) {
            document.body.setAttribute('data-theme', t);
            btn.textContent = labels[t] || labels.auto;
            try { localStorage.setItem('casman_theme', t); } catch(_) {}
        }
        let current = (function(){
            try { return localStorage.getItem('casman_theme') || 'auto'; }
            catch(_) { return 'auto'; }
        })();
        apply(current);
        btn.addEventListener('click', function() {
            current = order[(order.indexOf(current) + 1) % order.length];
            apply(current);
        });
    }

    // Search functionality. Per IDEA.md "Search Box" + "Autocomplete
    // Dropdown": focus shows recent searches (localStorage) and popular
    // pages (/api/v1/popular); typing fetches autocomplete; the dropdown
    // supports arrow-key navigation + Enter to select; an explicit
    // "Search all for X" row routes to /search?q=...; the form submit
    // records the query into the recent-searches LRU.
    function initSearch() {
        const searchInput = document.querySelector('.search-box input, .search-input');
        if (!searchInput) return;
        const form = searchInput.closest('form');

        let autocompleteTimeout;
        let popularCache = null;

        searchInput.addEventListener('input', function() {
            clearTimeout(autocompleteTimeout);
            const query = this.value.trim();
            if (query.length === 0) {
                showSearchOnFocus(searchInput);
                return;
            }
            if (query.length < 2) {
                hideAutocomplete();
                return;
            }
            autocompleteTimeout = setTimeout(() => fetchAutocomplete(query, searchInput), 150);
        });

        searchInput.addEventListener('focus', function() {
            if (!this.value.trim()) showSearchOnFocus(searchInput);
        });

        searchInput.addEventListener('keydown', function(e) {
            const dropdown = document.querySelector('.autocomplete-dropdown');
            if (e.key === 'Escape') { hideAutocomplete(); return; }
            if (!dropdown || dropdown.style.display === 'none') return;
            const items = Array.from(dropdown.querySelectorAll('.autocomplete-item'));
            if (items.length === 0) return;
            const cur = items.findIndex(i => i.classList.contains('autocomplete-active'));

            if (e.key === 'ArrowDown') {
                e.preventDefault();
                const next = (cur < 0 ? 0 : (cur + 1) % items.length);
                items.forEach(i => i.classList.remove('autocomplete-active'));
                items[next].classList.add('autocomplete-active');
                items[next].scrollIntoView({ block: 'nearest' });
            } else if (e.key === 'ArrowUp') {
                e.preventDefault();
                const next = (cur <= 0 ? items.length - 1 : cur - 1);
                items.forEach(i => i.classList.remove('autocomplete-active'));
                items[next].classList.add('autocomplete-active');
                items[next].scrollIntoView({ block: 'nearest' });
            } else if (e.key === 'Enter' && cur >= 0) {
                e.preventDefault();
                items[cur].click();
            }
        });

        document.addEventListener('click', function(e) {
            if (!e.target.closest('.search-box, .search-form')) {
                hideAutocomplete();
            }
        });

        if (form) {
            form.addEventListener('submit', function() {
                const q = (searchInput.value || '').trim();
                if (q) addRecentSearch(q);
            });
        }

        function showSearchOnFocus(input) {
            const recent = getRecentSearches();
            if (popularCache) {
                renderFocusDropdown(input, recent, popularCache);
            } else {
                fetch('/api/v1/popular')
                    .then(r => r.ok ? r.json() : [])
                    .then(data => {
                        popularCache = Array.isArray(data) ? data.slice(0, 8) : [];
                        renderFocusDropdown(input, recent, popularCache);
                    })
                    .catch(() => renderFocusDropdown(input, recent, []));
            }
        }
    }

    function renderFocusDropdown(input, recent, popular) {
        if (!recent.length && !popular.length) {
            hideAutocomplete();
            return;
        }
        const items = [];
        if (recent.length) {
            items.push('<div class="autocomplete-section">Recent searches</div>');
            for (const q of recent) {
                items.push('<a class="autocomplete-item" data-recent-query="' + escapeAttr(q) + '" href="/search?q=' + encodeURIComponent(q) + '">'
                          + '<span class="autocomplete-name">' + escapeHTML(q) + '</span></a>');
            }
        }
        if (popular.length) {
            items.push('<div class="autocomplete-section">Popular pages</div>');
            for (const p of popular) {
                const url = '/man/' + encodeURIComponent(p.platform || '') + '/' + encodeURIComponent(p.section || '') + '/' + encodeURIComponent(p.name || '');
                items.push('<a class="autocomplete-item" href="' + url + '">'
                          + '<span class="autocomplete-name">' + escapeHTML(p.name || '') + '(' + escapeHTML(p.section || '') + ')</span>'
                          + '<span class="autocomplete-platform">' + escapeHTML(p.platform || '') + '</span></a>');
            }
        }
        renderAutocompleteList(input, items.join(''));
    }

    function fetchAutocomplete(query, input) {
        fetch('/api/v1/autocomplete?q=' + encodeURIComponent(query))
            .then(response => response.json())
            .then(data => {
                if (data.suggestions && data.suggestions.length > 0) {
                    showAutocomplete(data.suggestions, query, input);
                } else {
                    showSearchAllOnly(query, input);
                }
            })
            .catch(() => showSearchAllOnly(query, input));
    }

    function showSearchAllOnly(query, input) {
        const html = '<a class="autocomplete-item autocomplete-search-all" href="/search?q=' + encodeURIComponent(query) + '">'
                   + '🔎 Search all for "<strong>' + escapeHTML(query) + '</strong>"</a>';
        renderAutocompleteList(input || document.querySelector('.search-input, .search-box input'), html);
    }

    function renderAutocompleteList(input, html) {
        let dropdown = document.querySelector('.autocomplete-dropdown');
        if (!dropdown) {
            dropdown = document.createElement('div');
            dropdown.className = 'autocomplete-dropdown';
            dropdown.setAttribute('role', 'listbox');
            const searchBox = (input && input.closest('.search-box, .search-form')) || document.body;
            searchBox.style.position = 'relative';
            searchBox.appendChild(dropdown);
        }
        dropdown.innerHTML = html;
        dropdown.style.display = 'block';
    }

    // showAutocomplete renders the suggestions list and appends a
    // "Search all for X" affordance per IDEA.md.
    function showAutocomplete(suggestions, query, input) {
        const items = suggestions.map(s =>
            '<a href="' + s.url + '" class="autocomplete-item">'
            + '<span class="autocomplete-name">' + escapeHTML(s.name) + '(' + escapeHTML(s.section) + ')</span>'
            + '<span class="autocomplete-platform">' + escapeHTML(s.platform) + '</span></a>'
        );
        if (query) {
            items.push('<a class="autocomplete-item autocomplete-search-all" href="/search?q=' + encodeURIComponent(query) + '">'
                       + '🔎 Search all for "<strong>' + escapeHTML(query) + '</strong>"</a>');
        }
        renderAutocompleteList(input || document.querySelector('.search-input, .search-box input'), items.join(''));
    }

    function hideAutocomplete() {
        const dropdown = document.querySelector('.autocomplete-dropdown');
        if (dropdown) {
            dropdown.style.display = 'none';
        }
    }

    // Bookmark functionality
    function initBookmarks() {
        const bookmarkBtn = document.querySelector('.bookmark-btn');
        if (!bookmarkBtn) return;

        bookmarkBtn.addEventListener('click', function() {
            const name = this.dataset.name;
            const section = this.dataset.section;
            const platform = this.dataset.platform;

            toggleBookmark(name, section, platform, this);
        });
    }

    function toggleBookmark(name, section, platform, btn) {
        const bookmarks = getBookmarks();
        const key = `${platform}/${section}/${name}`;

        if (bookmarks[key]) {
            delete bookmarks[key];
            btn.textContent = '☆ Bookmark';
            btn.classList.remove('bookmarked');
        } else {
            bookmarks[key] = { name, section, platform, added: Date.now() };
            btn.textContent = '★ Bookmarked';
            btn.classList.add('bookmarked');
        }

        localStorage.setItem('casman_bookmarks', JSON.stringify(bookmarks));
    }

    function getBookmarks() {
        try {
            return JSON.parse(localStorage.getItem('casman_bookmarks') || '{}');
        } catch {
            return {};
        }
    }

    function isBookmarked(name, section, platform) {
        const bookmarks = getBookmarks();
        return !!bookmarks[`${platform}/${section}/${name}`];
    }

    // History functionality
    function initHistory() {
        // Track page view if on man page
        const manpageEl = document.querySelector('.manpage');
        if (manpageEl) {
            const name = manpageEl.dataset.name;
            const section = manpageEl.dataset.section;
            const platform = manpageEl.dataset.platform;
            const title = document.querySelector('.manpage h1')?.textContent || name;

            if (name && section && platform && isHistoryEnabled()) {
                addToHistory(name, section, platform, title);
            }
        }
    }

    function addToHistory(name, section, platform, title) {
        const history = getHistory();
        const key = `${platform}/${section}/${name}`;

        // Remove if already exists (to move to front)
        const filtered = history.filter(h => h.key !== key);

        // Add to front
        filtered.unshift({
            key,
            name,
            section,
            platform,
            title,
            viewed: Date.now()
        });

        // Keep only last 50
        const maxHistory = getHistoryMax();
        const trimmed = filtered.slice(0, maxHistory);

        localStorage.setItem('casman_history', JSON.stringify(trimmed));
    }

    function getHistory() {
        try {
            return JSON.parse(localStorage.getItem('casman_history') || '[]');
        } catch {
            return [];
        }
    }

    function clearHistory() {
        localStorage.setItem('casman_history', '[]');
    }

    function isHistoryEnabled() {
        return localStorage.getItem('casman_history_enabled') !== 'false';
    }

    function setHistoryEnabled(enabled) {
        localStorage.setItem('casman_history_enabled', enabled ? 'true' : 'false');
    }

    function getHistoryMax() {
        const max = parseInt(localStorage.getItem('casman_history_max') || '50', 10);
        return isNaN(max) ? 50 : max;
    }

    function setHistoryMax(max) {
        localStorage.setItem('casman_history_max', String(Math.max(1, Math.min(200, max))));
    }

    // Keyboard shortcuts. Per IDEA.md "Keyboard Navigation".
    //   /  or Ctrl+K  → focus search
    //   h            → home
    //   b            → browse
    //   j / k        → scroll down / up (vim style)
    //   gg / G       → top / bottom
    //   [ / ]        → previous / next H2 in manpage
    //   ?            → keyboard help modal
    //   Esc          → close modals
    function initKeyboardShortcuts() {
        let lastG = 0;
        const inInput = el => el && (el.tagName === 'INPUT' || el.tagName === 'TEXTAREA' || el.isContentEditable);

        document.addEventListener('keydown', function(e) {
            // Ctrl+K / Cmd+K: focus search even from inside inputs.
            if (e.key === 'k' && (e.ctrlKey || e.metaKey)) {
                e.preventDefault();
                const si = document.querySelector('.search-box input, .search-input');
                if (si) { si.focus(); si.select(); }
                return;
            }
            if (inInput(e.target)) return;

            switch (e.key) {
                case '/': {
                    e.preventDefault();
                    const si = document.querySelector('.search-box input, .search-input');
                    if (si) si.focus();
                    return;
                }
                case 'h':
                    if (!e.ctrlKey && !e.metaKey && !e.altKey) {
                        window.location.href = '/';
                    }
                    return;
                case 'b':
                    if (!e.ctrlKey && !e.metaKey && !e.altKey) {
                        window.location.href = '/browse';
                    }
                    return;
                case '?':
                    showKeyboardHelp();
                    return;
                case 'Escape':
                    closeModals();
                    return;
                case 'j':
                    e.preventDefault();
                    window.scrollBy({ top: window.innerHeight * 0.1, behavior: 'smooth' });
                    return;
                case 'k':
                    e.preventDefault();
                    window.scrollBy({ top: -window.innerHeight * 0.1, behavior: 'smooth' });
                    return;
                case 'g': {
                    const now = Date.now();
                    if (now - lastG < 400) {
                        e.preventDefault();
                        window.scrollTo({ top: 0, behavior: 'smooth' });
                        lastG = 0;
                    } else {
                        lastG = now;
                    }
                    return;
                }
                case 'G':
                    e.preventDefault();
                    window.scrollTo({ top: document.documentElement.scrollHeight, behavior: 'smooth' });
                    return;
                case '[':
                    e.preventDefault();
                    jumpHeading(-1);
                    return;
                case ']':
                    e.preventDefault();
                    jumpHeading(+1);
                    return;
            }
        });
    }

    function jumpHeading(dir) {
        const headings = Array.from(document.querySelectorAll('.manpage h1, .manpage h2, .synopsis h2, .manpage-content h2, .manpage-content h3'));
        if (!headings.length) return;
        const top = window.scrollY + 80;
        let target = null;
        if (dir > 0) {
            target = headings.find(h => h.offsetTop > top + 4);
        } else {
            for (let i = headings.length - 1; i >= 0; i--) {
                if (headings[i].offsetTop < top - 4) { target = headings[i]; break; }
            }
        }
        if (target) target.scrollIntoView({ behavior: 'smooth', block: 'start' });
    }

    function showKeyboardHelp() {
        let modal = document.querySelector('.keyboard-help-modal');
        if (modal) {
            modal.style.display = 'flex';
            return;
        }

        modal = document.createElement('div');
        modal.className = 'keyboard-help-modal';
        modal.innerHTML = `
            <div class="modal-content">
                <h2>Keyboard Shortcuts</h2>
                <dl>
                    <dt><kbd>/</kbd> or <kbd>Ctrl</kbd>+<kbd>K</kbd></dt>
                    <dd>Focus search</dd>
                    <dt><kbd>h</kbd></dt>
                    <dd>Go to home</dd>
                    <dt><kbd>b</kbd></dt>
                    <dd>Go to browse</dd>
                    <dt><kbd>j</kbd> / <kbd>k</kbd></dt>
                    <dd>Scroll down / up</dd>
                    <dt><kbd>g</kbd> <kbd>g</kbd> / <kbd>G</kbd></dt>
                    <dd>Top / bottom of page</dd>
                    <dt><kbd>[</kbd> / <kbd>]</kbd></dt>
                    <dd>Previous / next section</dd>
                    <dt><kbd>?</kbd></dt>
                    <dd>Show this help</dd>
                    <dt><kbd>Esc</kbd></dt>
                    <dd>Close modal</dd>
                </dl>
                <button class="btn" onclick="this.closest('.keyboard-help-modal').style.display='none'">Close</button>
            </div>
        `;
        document.body.appendChild(modal);
    }

    function closeModals() {
        document.querySelectorAll('.keyboard-help-modal').forEach(m => {
            m.style.display = 'none';
        });
    }

    // Add styles for autocomplete and modal
    const style = document.createElement('style');
    style.textContent = `
        .autocomplete-dropdown {
            position: absolute;
            top: 100%;
            left: 0;
            right: 0;
            background: var(--bg-secondary);
            border: 1px solid var(--border-color);
            border-radius: 4px;
            margin-top: 4px;
            max-height: 300px;
            overflow-y: auto;
            z-index: 1000;
            display: none;
        }
        .autocomplete-item {
            display: flex;
            justify-content: space-between;
            padding: 0.5rem 1rem;
            color: var(--text-color);
            border-bottom: 1px solid var(--border-color);
        }
        .autocomplete-item:last-child {
            border-bottom: none;
        }
        .autocomplete-item:hover {
            background: var(--bg-tertiary);
        }
        .autocomplete-platform {
            color: var(--text-muted);
            font-size: 0.875rem;
        }
        .keyboard-help-modal {
            position: fixed;
            top: 0;
            left: 0;
            right: 0;
            bottom: 0;
            background: rgba(0,0,0,0.8);
            display: flex;
            justify-content: center;
            align-items: center;
            z-index: 2000;
        }
        .modal-content {
            background: var(--bg-secondary);
            padding: 2rem;
            border-radius: 8px;
            max-width: 400px;
        }
        .modal-content h2 {
            margin-bottom: 1rem;
        }
        .modal-content dl {
            display: grid;
            grid-template-columns: auto 1fr;
            gap: 0.5rem 1rem;
            margin-bottom: 1rem;
        }
        .modal-content kbd {
            background: var(--code-bg);
            padding: 0.25rem 0.5rem;
            border-radius: 4px;
            font-family: monospace;
        }
    `;
    document.head.appendChild(style);

    // Expose for external use
    window.casman = {
        getBookmarks,
        isBookmarked,
        toggleBookmark,
        getHistory,
        clearHistory,
        isHistoryEnabled,
        setHistoryEnabled,
        getHistoryMax,
        setHistoryMax
    };
})();
