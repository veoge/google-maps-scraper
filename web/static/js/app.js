/* Results explorer for the Google Maps Scraper web UI.
 *
 * The scraper writes one CSV per job. The server re-reads that file and serves
 * it as JSON from /results, which is what lets this page show the data in a
 * table instead of making people download a file to see whether the scrape
 * found anything useful.
 */
(function () {
    'use strict';

    var LEAFLET_CSS = 'https://cdnjs.cloudflare.com/ajax/libs/leaflet/1.9.4/leaflet.min.css';
    var LEAFLET_JS = 'https://cdnjs.cloudflare.com/ajax/libs/leaflet/1.9.4/leaflet.min.js';

    var state = {
        jobId: null,
        jobName: '',
        rows: [],
        filtered: [],
        page: 1,
        pageSize: 50,
        sort: { key: 'title', dir: 1 },
        view: 'table',
        groupBy: false,
        map: null,
        markerLayer: null
    };

    var COLUMNS = [
        { key: 'title', label: 'Business', sortable: true },
        { key: 'keyword', label: 'Search', sortable: true },
        { key: 'category', label: 'Category', sortable: true },
        { key: 'rating', label: 'Rating', sortable: true },
        { key: 'reviews', label: 'Reviews', sortable: true },
        { key: 'phone', label: 'Phone', sortable: true },
        { key: 'website', label: 'Website', sortable: true },
        { key: 'emails', label: 'Email', sortable: true },
        { key: 'social', label: 'Social', sortable: true },
        { key: 'city', label: 'City', sortable: true }
    ];

    var SOCIAL_FIELDS = [
        { key: 'facebook', short: 'f', title: 'Facebook' },
        { key: 'instagram', short: 'ig', title: 'Instagram' },
        { key: 'linkedin', short: 'in', title: 'LinkedIn' },
        { key: 'twitter_x', short: 'X', title: 'X (Twitter)' },
        { key: 'youtube', short: 'yt', title: 'YouTube' },
        { key: 'tiktok', short: 'tt', title: 'TikTok' },
        { key: 'whatsapp', short: 'wa', title: 'WhatsApp' }
    ];

    // ---- small DOM helpers -------------------------------------------------

    function $(sel, root) { return (root || document).querySelector(sel); }

    function el(tag, attrs, children) {
        var node = document.createElement(tag);

        Object.keys(attrs || {}).forEach(function (k) {
            if (k === 'text') { node.textContent = attrs[k]; }
            else if (k === 'html') { node.innerHTML = attrs[k]; }
            else if (k === 'class') { node.className = attrs[k]; }
            else if (k.slice(0, 2) === 'on') { node.addEventListener(k.slice(2), attrs[k]); }
            else if (attrs[k] !== null && attrs[k] !== undefined) { node.setAttribute(k, attrs[k]); }
        });

        (children || []).forEach(function (c) {
            if (c) { node.appendChild(typeof c === 'string' ? document.createTextNode(c) : c); }
        });

        return node;
    }

    // safeLink only produces anchors for http(s) URLs, so a hostile value taken
    // from a scraped page cannot turn into a javascript: link.
    function safeLink(url, label, cls) {
        if (!url || !/^https?:\/\//i.test(url)) { return null; }

        return el('a', {
            href: url,
            class: cls || 'link-cell',
            target: '_blank',
            rel: 'noopener noreferrer',
            title: url,
            text: label || prettyURL(url)
        });
    }

    function prettyURL(url) {
        return String(url).replace(/^https?:\/\//i, '').replace(/^www\./i, '').replace(/\/$/, '');
    }

    function fmtInt(n) { return (n || 0).toLocaleString(); }

    function debounce(fn, wait) {
        var timer = null;

        return function () {
            clearTimeout(timer);
            timer = setTimeout(fn, wait);
        };
    }

    // ---- job selection -----------------------------------------------------

    window.selectJob = function (id, name) {
        state.jobId = id;
        state.jobName = name || id;

        document.querySelectorAll('.jobs-table tr').forEach(function (tr) {
            tr.classList.toggle('selected', tr.dataset.jobId === id);
        });

        $('#results-card').hidden = false;
        $('#results-title').textContent = state.jobName;
        $('#results-body').replaceChildren(loadingState());

        setExportLinks(id);
        fetchResults(id);
        loadJobConfig(id, state.jobName);
    };

    function setExportLinks(id) {
        $('#export-xlsx').href = '/download?id=' + encodeURIComponent(id) + '&format=xlsx';
        $('#export-csv').href = '/download?id=' + encodeURIComponent(id) + '&format=csv';
        $('#export-json').href = '/download?id=' + encodeURIComponent(id) + '&format=json';
    }

    function loadingState() {
        return el('div', { class: 'empty' }, [
            el('div', { class: 'spinner', style: 'display:inline-block;margin:0 auto 10px' }),
            el('p', { text: 'Loading results…' })
        ]);
    }

    function fetchResults(id) {
        fetch('/results?id=' + encodeURIComponent(id))
            .then(function (r) {
                if (!r.ok) { throw new Error('HTTP ' + r.status); }
                return r.json();
            })
            .then(function (data) {
                if (state.jobId !== id) { return; }

                state.rows = data.rows || [];
                renderStats(data);
                renderFilters(data);
                applyFilters();
            })
            .catch(function (err) {
                $('#results-body').replaceChildren(emptyState(
                    'Could not load results', String(err.message || err)));
            });
    }

    // ---- stats -------------------------------------------------------------

    function renderStats(data) {
        var total = data.total || 0;
        var pct = function (n) { return total ? Math.round((n / total) * 100) + '% of results' : '—'; };

        var tiles = [
            ['Businesses', fmtInt(total), 'found by this job'],
            ['Average rating', total ? (data.avg_rating || 0).toFixed(1) : '—', 'across rated places'],
            ['With phone', fmtInt(data.with_phone), pct(data.with_phone)],
            ['With email', fmtInt(data.with_email), pct(data.with_email)],
            ['With social', fmtInt(data.with_social), pct(data.with_social)]
        ];

        $('#results-stats').replaceChildren.apply($('#results-stats'), tiles.map(function (t) {
            return el('div', { class: 'stat' }, [
                el('div', { class: 'stat-label', text: t[0] }),
                el('div', { class: 'stat-value', text: t[1] }),
                el('div', { class: 'stat-sub', text: t[2] })
            ]);
        }));

        $('#results-stats').hidden = false;
    }

    // ---- filters -----------------------------------------------------------

    function renderFilters(data) {
        fillSelect($('#filter-keyword'), 'All searches', data.keywords);
        fillSelect($('#filter-category'), 'All categories', data.categories);
        fillSelect($('#filter-city'), 'All cities', data.cities);

        // Grouping is only meaningful when the job ran more than one search.
        var multi = (data.keywords || []).length > 1;

        $('#group-pill').hidden = !multi;
        $('#filter-keyword').closest('.select').hidden = !multi;

        if (!multi) {
            $('#group-by-search').checked = false;
            state.groupBy = false;
        }

        $('#results-filters').hidden = false;
    }

    function fillSelect(select, allLabel, values) {
        select.replaceChildren(el('option', { value: '', text: allLabel }));

        (values || []).forEach(function (v) {
            select.appendChild(el('option', { value: v, text: v }));
        });

        if (window.GMSDropdown) { window.GMSDropdown.refresh(select); }
    }

    function applyFilters() {
        var q = $('#filter-search').value.trim().toLowerCase();
        var keyword = $('#filter-keyword').value;
        var category = $('#filter-category').value;
        var city = $('#filter-city').value;
        var minRating = parseFloat($('#filter-rating').value) || 0;

        var need = {
            website: $('#has-website').checked,
            email: $('#has-email').checked,
            social: $('#has-social').checked,
            phone: $('#has-phone').checked
        };

        state.filtered = state.rows.filter(function (row) {
            if (keyword && row.keyword !== keyword) { return false; }
            if (category && row.category !== category) { return false; }
            if (city && row.city !== city) { return false; }
            if (minRating && (row.rating || 0) < minRating) { return false; }
            if (need.website && !row.website) { return false; }
            if (need.phone && !row.phone) { return false; }
            if (need.email && !(row.emails || []).length) { return false; }
            if (need.social && !(row.social || []).length) { return false; }

            if (q) {
                var hay = [row.title, row.category, row.keyword, row.address, row.phone, row.website,
                    (row.emails || []).join(' '), (row.social || []).join(' ')]
                    .join(' ').toLowerCase();

                if (hay.indexOf(q) === -1) { return false; }
            }

            return true;
        });

        sortRows();

        // Any change to the filters invalidates the current page number: page 4 of
        // the old result set is meaningless against the new one.
        state.page = 1;
        render();
    }

    function sortRows() {
        var key = state.sort.key;
        var dir = state.sort.dir;

        state.filtered.sort(function (a, b) {
            // Grouping needs rows ordered by search term first; the chosen sort
            // then applies within each group.
            if (state.groupBy) {
                var ga = a.keyword || '';
                var gb = b.keyword || '';

                if (ga !== gb) { return ga.localeCompare(gb); }
            }

            var av = sortValue(a, key);
            var bv = sortValue(b, key);

            if (typeof av === 'number' && typeof bv === 'number') { return (av - bv) * dir; }

            return String(av).localeCompare(String(bv), undefined, { sensitivity: 'base' }) * dir;
        });
    }

    function sortValue(row, key) {
        if (key === 'emails') { return (row.emails || []).length ? row.emails[0] : ''; }
        if (key === 'social') { return (row.social || []).length; }

        var v = row[key];

        return v === null || v === undefined ? '' : v;
    }

    // ---- rendering ---------------------------------------------------------

    function render() {
        $('#results-count').textContent = state.filtered.length === state.rows.length
            ? fmtInt(state.rows.length) + ' results'
            : fmtInt(state.filtered.length) + ' of ' + fmtInt(state.rows.length) + ' results';

        if (state.view === 'map') {
            // The map plots the whole filtered set; paging it would hide pins for
            // no reason.
            $('#pagination').hidden = true;
            renderMap();

            return;
        }

        if (!state.filtered.length) {
            $('#pagination').hidden = true;
            $('#results-body').replaceChildren(emptyState(
                state.rows.length ? 'No matches' : 'No results yet',
                state.rows.length
                    ? 'No business matches the current filters. Try clearing them.'
                    : 'This job has not produced any rows.'));

            return;
        }

        clampPage();
        $('#results-body').replaceChildren(buildTable());
        renderPagination();
    }

    function pageCount() {
        return Math.max(1, Math.ceil(state.filtered.length / state.pageSize));
    }

    function clampPage() {
        state.page = Math.min(Math.max(1, state.page), pageCount());
    }

    function goToPage(n) {
        state.page = n;
        clampPage();
        render();

        var wrap = document.querySelector('.table-wrap');
        if (wrap) { wrap.scrollTop = 0; }
    }

    // pageNumbers returns the buttons to show: the ends, the current neighbourhood,
    // and gaps in between, so 200 pages do not produce 200 buttons.
    function pageNumbers(current, total) {
        if (total <= 7) {
            return Array.apply(null, { length: total }).map(function (_, i) { return i + 1; });
        }

        var out = [1];
        var from = Math.max(2, current - 1);
        var to = Math.min(total - 1, current + 1);

        if (from > 2) { out.push('gap'); }

        for (var i = from; i <= to; i++) { out.push(i); }

        if (to < total - 1) { out.push('gap'); }

        out.push(total);

        return out;
    }

    function renderPagination() {
        var bar = $('#pagination');
        var total = state.filtered.length;
        var pages = pageCount();

        bar.hidden = false;
        bar.replaceChildren();

        var first = (state.page - 1) * state.pageSize + 1;
        var last = Math.min(state.page * state.pageSize, total);

        bar.appendChild(el('span', {
            class: 'page-info',
            text: 'Showing ' + fmtInt(first) + '\u2013' + fmtInt(last) + ' of ' + fmtInt(total)
        }));

        bar.appendChild(el('span', { class: 'page-spacer' }));

        if (pages > 1) {
            var controls = el('div', { class: 'page-controls' });

            var step = function (label, target, disabled, title) {
                return el('button', {
                    type: 'button',
                    class: 'page-btn',
                    text: label,
                    title: title,
                    'aria-label': title,
                    disabled: disabled ? 'disabled' : null,
                    onclick: function () { goToPage(target); }
                });
            };

            controls.appendChild(step('\u00ab', 1, state.page === 1, 'First page'));
            controls.appendChild(step('\u2039', state.page - 1, state.page === 1, 'Previous page'));

            pageNumbers(state.page, pages).forEach(function (n) {
                if (n === 'gap') {
                    controls.appendChild(el('span', { class: 'page-gap', text: '\u2026' }));

                    return;
                }

                var current = n === state.page;

                controls.appendChild(el('button', {
                    type: 'button',
                    class: 'page-btn' + (current ? ' current' : ''),
                    text: String(n),
                    'aria-current': current ? 'page' : null,
                    onclick: function () { goToPage(n); }
                }));
            });

            controls.appendChild(step('\u203a', state.page + 1, state.page === pages, 'Next page'));
            controls.appendChild(step('\u00bb', pages, state.page === pages, 'Last page'));

            bar.appendChild(controls);
        }

        var sizeSelect = el('select', { 'aria-label': 'Rows per page' });

        [25, 50, 100, 250].forEach(function (n) {
            sizeSelect.appendChild(el('option', {
                value: String(n),
                text: String(n) + ' per page',
                selected: n === state.pageSize ? 'selected' : null
            }));
        });

        sizeSelect.addEventListener('change', function () {
            // Keep the first visible row visible when the page size changes.
            var anchor = (state.page - 1) * state.pageSize;

            state.pageSize = parseInt(sizeSelect.value, 10) || 50;
            state.page = Math.floor(anchor / state.pageSize) + 1;
            render();
        });

        var sizeWrap = el('div', { class: 'page-size' }, [sizeSelect]);

        bar.appendChild(sizeWrap);

        if (window.GMSDropdown) { window.GMSDropdown.enhance(sizeSelect); }
    }

    function emptyState(title, message) {
        return el('div', { class: 'empty' }, [
            el('h3', { text: title }),
            el('p', { text: message })
        ]);
    }

    function buildTable() {
        var head = el('tr', {}, COLUMNS.map(function (col) {
            var sorted = state.sort.key === col.key;
            var arrow = el('span', { class: 'arrow', text: sorted ? (state.sort.dir > 0 ? '↑' : '↓') : '↕' });

            return el('th', {
                class: 'sortable' + (sorted ? ' sorted' : ''),
                onclick: function () {
                    state.sort.dir = sorted ? -state.sort.dir : 1;
                    state.sort.key = col.key;
                    sortRows();
                    state.page = 1;
                    render();
                }
            }, [document.createTextNode(col.label), arrow]);
        }));

        var start = (state.page - 1) * state.pageSize;
        var visible = state.filtered.slice(start, start + state.pageSize);
        var body = state.groupBy ? buildGroupedBody(visible) : el('tbody', {}, visible.map(buildRow));

        var table = el('table', { class: 'results-table' }, [el('thead', {}, [head]), body]);

        return el('div', { class: 'table-wrap' }, [table]);
    }

    // buildGroupedBody inserts a header row per search term, so a job covering
    // several searches reads as sections rather than one mixed list.
    function buildGroupedBody(rows) {
        var body = el('tbody');
        var current = null;

        rows.forEach(function (row) {
            var key = row.keyword || 'Other';

            if (key !== current) {
                current = key;

                var count = state.filtered.filter(function (r) {
                    return (r.keyword || 'Other') === key;
                }).length;

                var cell = el('td', { colspan: String(COLUMNS.length), class: 'group-cell' }, [
                    el('div', { class: 'group-inner' }, [
                        el('span', { class: 'group-name', text: key }),
                        el('span', { class: 'group-count', text: fmtInt(count) + ' results' })
                    ])
                ]);

                body.appendChild(el('tr', { class: 'group-row' }, [cell]));
            }

            body.appendChild(buildRow(row));
        });

        return body;
    }

    function buildRow(row) {
        var tr = el('tr', { onclick: function () { openDrawer(row); } });

        tr.appendChild(el('td', {}, [
            el('div', { class: 'cell-title', text: row.title || '—' }),
            row.address ? el('div', { class: 'cell-sub', text: row.address }) : null
        ]));

        tr.appendChild(el('td', { class: 'nowrap muted', text: row.keyword || '' }));
        tr.appendChild(el('td', { text: row.category || '' }));

        tr.appendChild(el('td', {}, [
            row.rating
                ? el('span', { class: 'rating' }, [
                    el('span', { class: 'star', text: '★' }),
                    document.createTextNode(row.rating.toFixed(1))
                ])
                : el('span', { class: 'faint', text: '—' })
        ]));

        tr.appendChild(el('td', { text: row.reviews ? fmtInt(row.reviews) : '—' }));
        tr.appendChild(el('td', { class: 'nowrap', text: row.phone || '—' }));

        tr.appendChild(el('td', {}, [
            safeLink(row.website) || el('span', { class: 'faint', text: '—' })
        ]));

        var email = (row.emails || [])[0];
        tr.appendChild(el('td', {}, [
            email
                ? el('a', { href: 'mailto:' + email, class: 'link-cell', text: email })
                : el('span', { class: 'faint', text: '—' })
        ]));

        tr.appendChild(el('td', {}, [socialChips(row)]));
        tr.appendChild(el('td', { text: row.city || '' }));

        // Clicking a link inside the row must not also open the detail drawer.
        tr.querySelectorAll('a').forEach(function (a) {
            a.addEventListener('click', function (e) { e.stopPropagation(); });
        });

        return tr;
    }

    function socialChips(row) {
        var wrap = el('div', { class: 'social-chips' });

        SOCIAL_FIELDS.forEach(function (net) {
            var url = row[net.key];
            if (!url) { return; }

            var chip = safeLink(url, net.short, 'chip');
            if (chip) {
                chip.title = net.title + ' — ' + url;
                wrap.appendChild(chip);
            }
        });

        if (!wrap.childNodes.length) { wrap.appendChild(el('span', { class: 'faint', text: '—' })); }

        return wrap;
    }

    // ---- detail drawer -----------------------------------------------------

    function openDrawer(row) {
        $('#drawer-title').textContent = row.title || 'Business';
        $('#drawer-sub').textContent = [row.category, row.city].filter(Boolean).join(' · ');

        var body = $('#drawer-body');
        body.replaceChildren();

        body.appendChild(thumbnailFor(row));

        body.appendChild(detailContact(row));

        if ((row.social || []).length) { body.appendChild(detailSocial(row)); }
        if (row.reviews) { body.appendChild(detailRatings(row)); }
        if ((row.hours || []).length) { body.appendChild(detailHours(row)); }
        if ((row.amenities || []).length) { body.appendChild(detailAmenities(row)); }
        if ((row.top_reviews || []).length) { body.appendChild(detailReviews(row)); }

        $('#drawer').classList.add('open');
        $('#drawer-backdrop').classList.add('open');
    }

    window.closeDrawer = function () {
        $('#drawer').classList.remove('open');
        $('#drawer-backdrop').classList.remove('open');
    };

    function group(title, content) {
        return el('div', { class: 'detail-group' }, [el('h4', { text: title }), content]);
    }

    // thumbnailFor returns the business photo, or a placeholder standing in for it.
    //
    // About one place in ten ships no photo in the page data Google serves, even
    // when the Maps website shows one — those listings are thinner all round. An
    // empty gap reads as a broken panel, so the space is filled deliberately. The
    // same placeholder covers an image that fails to load later.
    function thumbnailFor(row) {
        var placeholder = el('div', { class: 'drawer-thumb drawer-thumb-empty' }, [
            el('span', { class: 'drawer-thumb-initial', text: (row.title || '?').trim().charAt(0).toUpperCase() }),
            el('span', { class: 'drawer-thumb-note', text: 'No photo available' })
        ]);

        if (!row.thumbnail || !/^https?:\/\//i.test(row.thumbnail)) {
            return placeholder;
        }

        var img = el('img', {
            class: 'drawer-thumb',
            src: row.thumbnail,
            alt: row.title || '',
            loading: 'lazy'
        });

        img.addEventListener('error', function () {
            if (img.parentNode) { img.parentNode.replaceChild(placeholder, img); }
        });

        return img;
    }

    function detailContact(row) {
        var dl = el('dl', { class: 'kv' });

        var add = function (label, node) {
            if (!node) { return; }
            dl.appendChild(el('dt', { text: label }));
            dl.appendChild(el('dd', {}, [node]));
        };

        add('Address', row.address ? document.createTextNode(row.address) : null);
        add('Phone', row.phone ? el('a', { href: 'tel:' + row.phone, text: row.phone }) : null);
        add('Website', safeLink(row.website));

        (row.emails || []).forEach(function (e, i) {
            add(i === 0 ? 'Email' : '', el('a', { href: 'mailto:' + e, text: e }));
        });

        add('Price', row.price_range ? document.createTextNode(row.price_range) : null);
        add('Maps', safeLink(row.link, 'Open in Google Maps'));

        if (row.latitude || row.longitude) {
            add('Coords', document.createTextNode(row.latitude.toFixed(6) + ', ' + row.longitude.toFixed(6)));
        }

        add('Place ID', row.place_id ? el('code', { text: row.place_id }) : null);

        return group('Contact', dl);
    }

    function detailSocial(row) {
        var list = el('div', { class: 'kv' });

        SOCIAL_FIELDS.forEach(function (net) {
            var link = safeLink(row[net.key], prettyURL(row[net.key]));
            if (!link) { return; }

            list.appendChild(el('dt', { text: net.title }));
            list.appendChild(el('dd', {}, [link]));
        });

        return group('Social profiles', list);
    }

    function detailRatings(row) {
        var wrap = el('div');
        var max = Math.max.apply(null, row.breakdown.concat([1]));

        for (var star = 5; star >= 1; star--) {
            var count = row.breakdown[star - 1] || 0;
            var pct = Math.round((count / max) * 100);

            wrap.appendChild(el('div', { class: 'bar-row' }, [
                el('span', { class: 'bar-label', text: star + '★' }),
                el('span', { class: 'bar-track' }, [
                    el('span', { class: 'bar-fill', style: 'width:' + pct + '%' })
                ]),
                el('span', { class: 'bar-count', text: fmtInt(count) })
            ]));
        }

        return group(row.rating.toFixed(1) + ' from ' + fmtInt(row.reviews) + ' reviews', wrap);
    }

    function detailHours(row) {
        var dl = el('dl', { class: 'kv' });

        row.hours.forEach(function (pair) {
            dl.appendChild(el('dt', { text: pair[0].slice(0, 3) }));
            dl.appendChild(el('dd', { text: pair[1] }));
        });

        return group('Opening hours', dl);
    }

    function detailAmenities(row) {
        var list = el('div', { class: 'tag-list' });

        row.amenities.forEach(function (line) {
            var parts = line.split(': ');
            var values = (parts[1] || '').split(', ');

            values.forEach(function (v) {
                if (v) { list.appendChild(el('span', { class: 'tag', text: v })); }
            });
        });

        return group('Amenities', list);
    }

    function detailReviews(row) {
        var wrap = el('div');

        row.top_reviews.forEach(function (rv) {
            wrap.appendChild(el('div', { class: 'review' }, [
                el('div', { class: 'review-head' }, [
                    el('span', { class: 'review-author', text: rv.author || 'Anonymous' }),
                    el('span', { class: 'rating' }, [
                        el('span', { class: 'star', text: '★' }),
                        document.createTextNode(String(rv.rating || 0))
                    ]),
                    el('span', { class: 'faint small', text: rv.when || '' })
                ]),
                rv.text ? el('div', { class: 'review-text', text: rv.text }) : null
            ]));
        });

        return group('Reviews', wrap);
    }

    // ---- map ---------------------------------------------------------------

    function ensureLeaflet(callback) {
        if (window.L) { callback(); return; }

        if (!document.getElementById('leaflet-css')) {
            document.head.appendChild(el('link', {
                id: 'leaflet-css', rel: 'stylesheet', href: LEAFLET_CSS
            }));
        }

        var script = document.getElementById('leaflet-js');

        if (!script) {
            script = el('script', { id: 'leaflet-js', src: LEAFLET_JS });
            document.head.appendChild(script);
        }

        script.addEventListener('load', callback);
    }

    function renderMap() {
        var host = $('#results-body');
        var container = document.getElementById('results-map');

        // Switching to the table view replaces the map container, so a surviving
        // instance points at a detached node and has to be thrown away.
        if (state.map && (!container || !document.body.contains(container))) {
            state.map.remove();
            state.map = null;
            state.markerLayer = null;
            container = null;
        }

        if (!container) {
            container = el('div', { id: 'results-map' });

            host.replaceChildren(el('div', { class: 'map-wrap' }, [
                container,
                el('div', { id: 'map-empty', class: 'map-empty', hidden: 'hidden' })
            ]));
        }

        ensureLeaflet(function () {
            if (!state.map) {
                state.map = L.map(container);

                L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
                    maxZoom: 19,
                    attribution: '© OpenStreetMap contributors'
                }).addTo(state.map);

                state.markerLayer = L.layerGroup().addTo(state.map);
                state.map.setView([31.95, 35.91], 6);
            }

            updateMarkers();
            state.map.invalidateSize();
        });
    }

    // updateMarkers refreshes the pins on the existing map. Rebuilding the whole
    // Leaflet instance on every keystroke blanked the tiles and discarded the
    // current view, which made a working search look broken.
    function updateMarkers() {
        if (!state.map || !state.markerLayer) { return; }

        state.markerLayer.clearLayers();

        var bounds = [];

        state.filtered.forEach(function (row) {
            if (!row.latitude && !row.longitude) { return; }

            var latlng = [row.latitude, row.longitude];
            bounds.push(latlng);

            L.marker(latlng)
                .bindPopup(popupHTML(row))
                .on('click', function () { openDrawer(row); })
                .addTo(state.markerLayer);
        });

        var empty = document.getElementById('map-empty');

        if (bounds.length) {
            state.map.fitBounds(bounds, { padding: [30, 30], maxZoom: 16 });

            if (empty) { empty.hidden = true; }

            return;
        }

        // Hold the current view rather than jumping out to the whole world, which
        // reads as a broken map instead of an empty result set.
        if (empty) {
            empty.hidden = false;
            empty.textContent = state.rows.length
                ? 'No results match the current filters.'
                : 'This job produced no mappable results.';
        }
    }

    // popupHTML builds the marker popup as detached DOM, so scraped text is
    // inserted as text and never parsed as markup.
    function popupHTML(row) {
        var wrap = el('div', {}, [
            el('strong', { text: row.title || '' }),
            row.address ? el('div', { class: 'small muted', text: row.address }) : null,
            row.phone ? el('div', { class: 'small', text: row.phone }) : null,
            safeLink(row.link, 'Open in Google Maps')
        ]);

        return wrap;
    }

    // ---- location pickers --------------------------------------------------

    // Cell sizes must match coverageCellKm on the server, or the estimate shown
    // here would not describe the job that actually runs.
    var COVERAGE_KM = { quick: 6, balanced: 3, thorough: 1.5 };

    // A single sweep runs no grid at all: one search per term over the city.
    var COVERAGE_SINGLE = 'single';

    // One-click presets. Each sets coverage, depth and enrichment together;
    // touching any of those individually drops the form back to Custom.
    var SCAN_MODES = {
        quick:      { coverage: 'single',   depth: 3,  email: false },
        standard:   { coverage: 'balanced', depth: 5,  email: true },
        everything: { coverage: 'thorough', depth: 10, email: true }
    };

    // Calibrated against completed runs rather than guessed:
    //   sweep of Amman, 34 searches at depth 10 -> 1,965 businesses in 48 minutes
    //   (2,719 sub-jobs, so ~57 jobs/min at concurrency 6).
    // Density is businesses per square kilometre per search term; the range is
    // wide because it varies enormously between a city centre and its outskirts.
    var DENSITY_LOW = 0.3;
    var DENSITY_HIGH = 0.7;

    // A coarse grid misses results in dense districts, where one cell holds more
    // businesses than a single search can return.
    var COVERAGE_QUALITY = { 6: 0.75, 3: 0.90, 1.5: 1.0 };

    // Sub-jobs per minute at the concurrency the server runs with.
    var JOBS_PER_MINUTE = 57;

    var locations = { countries: [], byCode: {} };

    // Tracks whether the run time was chosen by a person rather than defaulted.
    var maxTimeEdited = false;

    // Guards the preset handler from reacting to its own writes.
    var applyingMode = false;

    // Resolves once the country and city dropdowns hold their options.
    var locationsReady = null;
    var markLocationsReady = null;

    function initLocations() {
        locationsReady = new Promise(function (resolve) { markLocationsReady = resolve; });

        fetch('/api/v1/locations')
            .then(function (r) { return r.json(); })
            .then(function (data) {
                locations.countries = data.countries || [];

                locations.countries.forEach(function (c) { locations.byCode[c.code] = c; });

                var countrySelect = $('#country');

                countrySelect.replaceChildren();

                locations.countries.forEach(function (c) {
                    countrySelect.appendChild(el('option', { value: c.code, text: c.name }));
                });

                countrySelect.value = data.default_country || '';

                if (window.GMSDropdown) { window.GMSDropdown.refresh(countrySelect); }

                fillCities();
                applyScanMode($('#scanmode').value);
                snapshotFormDefaults();
                markLocationsReady();
            })
            .catch(function () {
                $('#country').closest('.field').hidden = true;
                $('#city').closest('.field').hidden = true;
                markLocationsReady();
            });

        $('#country').addEventListener('change', fillCities);
        $('#city').addEventListener('change', updateCoverage);
        $('#coverage').addEventListener('change', updateCoverage);
        $('#keywords').addEventListener('input', updateCoverage);

        $('#scanmode').addEventListener('change', function () {
            applyScanMode($('#scanmode').value);
        });

        // Editing any of the three settings a preset controls means the form no
        // longer matches that preset, so it stops claiming to.
        ['#depth', '#email', '#coverage'].forEach(function (sel) {
            $(sel).addEventListener('change', function () {
                if (!applyingMode) { setValue('#scanmode', 'custom'); }

                updateCoverage();
            });
        });

        $('#depth').addEventListener('input', updateCoverage);

        // Once the run time is set by hand it is left alone; until then it tracks
        // the estimate, so the default cannot silently contradict the coverage.
        $('#maxtime').addEventListener('input', function () {
            maxTimeEdited = true;
            updateCoverage();
        });
    }

    function fillCities() {
        var country = locations.byCode[$('#country').value];
        var citySelect = $('#city');

        citySelect.replaceChildren(el('option', {
            value: '',
            text: country ? 'All cities in ' + country.name : 'All cities'
        }));

        (country ? country.cities : []).forEach(function (c) {
            citySelect.appendChild(el('option', { value: c.name, text: c.name }));
        });

        citySelect.value = '';

        if (window.GMSDropdown) { window.GMSDropdown.refresh(citySelect); }

        updateCoverage();
    }

    function selectedCity() {
        var country = locations.byCode[$('#country').value];
        var name = $('#city').value;

        if (!country || !name) { return null; }

        return country.cities.filter(function (c) { return c.name === name; })[0] || null;
    }

    // updateCoverage shows how many searches the selection will produce. Covering
    // a whole city means one search per grid cell per keyword, and that number is
    // the difference between a job that finishes in minutes and one that runs for
    // hours, so it belongs in front of the person starting it.
    function updateCoverage() {
        var country = locations.byCode[$('#country').value];
        var city = selectedCity();
        var field = $('#coverage-field');

        // With a country but no city the job grids every city in that country,
        // so the coverage setting still applies and still has to be shown.
        var areas = city ? [city] : (country ? country.cities : []);

        field.hidden = areas.length === 0;

        if (areas.length === 0) { return; }

        var where = city ? city.name : 'every city in ' + country.name;
        var coverage = $('#coverage').value;
        var single = coverage === COVERAGE_SINGLE;
        var terms = Math.max(1, splitKeywords($('#keywords').value).length);

        var cellKm = COVERAGE_KM[coverage] || COVERAGE_KM.balanced;
        var cells = single ? 1 : areas.reduce(function (sum, a) {
            return sum + estimateCells(a.bbox, cellKm);
        }, 0);
        var searches = cells * terms;

        var depth = parseInt($('#depth').value, 10) || 1;
        var enrich = $('#email').checked;

        var results = estimateResults(areas, cellKm, terms, depth, single);
        var minutes = estimateMinutes(searches, results, enrich);

        $('#est-results').textContent = fmtInt(results.low) + '–' + fmtInt(results.high);
        $('#est-time').textContent = minutes.low === minutes.high
            ? formatMinutes(minutes.high)
            : formatMinutes(minutes.low) + ' – ' + formatMinutes(minutes.high);

        $('#est-detail').textContent =
            fmtInt(cells) + (single ? ' search area' : ' map areas') + ' × ' +
            fmtInt(terms) + ' search' + (terms === 1 ? '' : 'es') + ' = ' +
            fmtInt(searches) + ' searches, depth ' + depth +
            (enrich ? ', website enrichment on' : ', no enrichment') + '.';

        var note = $('#coverage-estimate');

        var scope = single
            ? 'NOT full coverage. One search per term for the whole of ' + where +
              '. Google returns only about 130 results per search, so anything ranked ' +
              'below that is missed no matter how deep it scrolls \u2014 a real business ' +
              'in an outer district will simply not appear. Use this to sample, not to ' +
              'build a complete list.'
            : 'Covers every part of ' + where + ': split into ' +
              fmtInt(cells) + ' areas and each is searched separately, so every district ' +
              'gets its own ~130 result slots. ' + fmtInt(cells) + ' areas \u00d7 ' +
              terms + ' search' + (terms === 1 ? '' : 'es') + ' = ' + fmtInt(searches) + ' searches.';

        note.textContent = scope +
            ' At depth ' + depth + (enrich ? ' with website enrichment' : '') +
            ', roughly ' + formatMinutes(minutes.low) + ' to ' + formatMinutes(minutes.high) + '.';

        labelCoverageOptions(areas);

        // Until the run time is edited by hand, keep it in step with the estimate.
        // Otherwise the shipped default (10m) contradicts the shipped coverage
        // (a whole city), and every job would be cut short by design.
        if (!maxTimeEdited) {
            $('#maxtime').value = suggestDuration(minutes.high);
        }

        // A grid job that outlasts its limit is stopped part-way through and keeps
        // only what it had written, so the run silently returns a slice of the city.
        var budget = parseDurationMinutes($('#maxtime').value);

        note.classList.remove('warn');

        if (budget > 0 && budget < minutes.low) {
            note.classList.add('warn');
            note.textContent += ' Maximum run time is ' + formatMinutes(budget) +
                ', so the job would stop after roughly ' +
                Math.max(1, Math.round((budget / minutes.high) * 100)) + '% of the area. ' +
                'Raise it to ' + suggestDuration(minutes.high) +
                ', lower the depth, or pick a coarser coverage.';
        }
    }

    // areaKm2 approximates the ground covered by a bounding box.
    function areaKm2(bbox) {
        var kmPerDeg = 111.32;
        var midLat = (bbox[0] + bbox[2]) / 2;

        return Math.abs((bbox[2] - bbox[0]) * kmPerDeg) *
            Math.abs((bbox[3] - bbox[1]) * kmPerDeg * Math.cos(midLat * Math.PI / 180));
    }

    // searchCapacity is the most one search can return. Google stops at roughly
    // 130 however deep the list is scrolled, which is the whole reason a grid
    // exists: each area gets its own allowance.
    function searchCapacity(depth) {
        return Math.min(130, 18 + 11 * Math.max(1, depth));
    }

    // estimateResults predicts how many distinct businesses a job will find.
    //
    // A grid is limited by how many businesses are actually there, so it scales
    // with ground area. A sample is limited by Google's per-search cap instead,
    // and in practice returns about half of it, since many searches run out of
    // results first.
    function estimateResults(areas, cellKm, terms, depth, single) {
        var capacity = searchCapacity(depth);

        if (single) {
            return {
                low: Math.round(terms * capacity * 0.40),
                high: Math.round(terms * capacity * 0.65)
            };
        }

        var km2 = areas.reduce(function (sum, a) { return sum + areaKm2(a.bbox); }, 0);
        var quality = COVERAGE_QUALITY[cellKm] || 0.9;

        // A cell can still only hand back so much, whatever is inside it.
        var cells = areas.reduce(function (sum, a) {
            return sum + estimateCells(a.bbox, cellKm);
        }, 0);
        var ceiling = cells * terms * capacity;

        return {
            low: Math.round(Math.min(km2 * terms * DENSITY_LOW * quality, ceiling)),
            high: Math.round(Math.min(km2 * terms * DENSITY_HIGH * quality, ceiling))
        };
    }

    // estimateMinutes derives the run time from the work actually queued: one job
    // per search, one per business found, plus a website fetch for each business
    // when enrichment is on.
    function estimateMinutes(searches, results, enrich) {
        var perResult = enrich ? 1.6 : 1.0;

        var jobs = function (n) { return searches + n * perResult; };

        return {
            low: Math.max(1, Math.ceil(jobs(results.low) / JOBS_PER_MINUTE)),
            high: Math.max(1, Math.ceil(jobs(results.high) / JOBS_PER_MINUTE))
        };
    }

    // parseDurationMinutes reads the Go-style duration in the max run time field.
    function parseDurationMinutes(value) {
        var total = 0;
        var re = /(\d+(?:\.\d+)?)\s*(h|m|s)/g;
        var match = re.exec(String(value || ''));

        while (match) {
            var n = parseFloat(match[1]);

            if (match[2] === 'h') { total += n * 60; }
            else if (match[2] === 'm') { total += n; }
            else { total += n / 60; }

            match = re.exec(String(value || ''));
        }

        return total;
    }

    function formatMinutes(minutes) {
        if (minutes < 60) { return Math.max(1, Math.round(minutes)) + ' minutes'; }

        return (minutes / 60).toFixed(1) + ' hours';
    }

    function suggestDuration(minutes) {
        if (minutes < 60) { return Math.max(5, Math.ceil(minutes / 5) * 5) + 'm'; }

        return Math.ceil(minutes / 30) / 2 + 'h';
    }

    // applyScanMode writes a preset into the individual settings it stands for.
    function applyScanMode(mode) {
        var preset = SCAN_MODES[mode];
        if (!preset) { return; }

        applyingMode = true;

        setValue('#coverage', preset.coverage);
        setValue('#depth', preset.depth);
        $('#email').checked = preset.email;

        applyingMode = false;

        updateCoverage();
    }

    // labelCoverageOptions puts the real number of areas on each option, so the
    // difference between the choices is a number rather than an adjective.
    function labelCoverageOptions(areas) {
        var select = $('#coverage');

        Array.prototype.forEach.call(select.options, function (opt) {
            var base = opt.getAttribute('data-base') || opt.textContent;

            opt.setAttribute('data-base', base);

            if (opt.value === COVERAGE_SINGLE) {
                opt.textContent = base;

                return;
            }

            var n = areas.reduce(function (sum, a) {
                return sum + estimateCells(a.bbox, COVERAGE_KM[opt.value]);
            }, 0);

            opt.textContent = base.replace(/\s*\(\d[\d,]*\sareas\)$/, '') +
                ' (' + fmtInt(n) + ' areas)';
        });

        if (window.GMSDropdown) { window.GMSDropdown.refresh(select); }
    }

    // estimateCells mirrors grid.EstimateCellCount on the server. The grid starts
    // half a cell inside each edge and stops before the far edge, so the count is
    // not simply span / cell size.
    function estimateCells(bbox, cellKm) {
        var kmPerDegLat = 111.32;
        var midLat = (bbox[0] + bbox[2]) / 2;
        var kmPerDegLon = kmPerDegLat * Math.max(Math.abs(Math.cos(midLat * Math.PI / 180)), 0.01);

        var steps = function (span, step) {
            if (span <= 0 || step <= 0) { return 0; }

            return Math.max(0, Math.ceil((span - step / 2) / step));
        };

        var cells = steps(bbox[2] - bbox[0], cellKm / kmPerDegLat) *
            steps(bbox[3] - bbox[1], cellKm / kmPerDegLon);

        return Math.max(1, cells);
    }

    // splitKeywords mirrors the server: newlines or commas both separate searches.
    function splitKeywords(raw) {
        return String(raw || '')
            .split(/[\n,]/)
            .map(function (v) { return v.trim(); })
            .filter(function (v) { return v.length > 0; });
    }

    // ---- job configuration -------------------------------------------------

    // COVERAGE_LABELS mirrors the options in the form; a job scraped before the
    // coverage field existed has none, and the field is simply left alone.
    function loadJobConfig(id, name) {
        fetch('/api/v1/jobs/' + encodeURIComponent(id))
            .then(function (r) {
                if (!r.ok) { throw new Error('HTTP ' + r.status); }

                return r.json();
            })
            .then(function (job) {
                if (state.jobId !== id) { return; }

                applyJobConfig(job, name);
            })
            .catch(function () { /* the form simply keeps whatever it had */ });
    }

    // FORM_FIELDS is every control the job configuration touches. Restoring a job
    // resets all of them first: without that, a value from the previously viewed
    // job survives into one that never set it, and the form quietly lies about
    // what the job ran with.
    var FORM_FIELDS = [
        '#name', '#keywords', '#lang', '#depth', '#zoom', '#radius',
        '#latitude', '#longitude', '#proxies', '#maxtime',
        '#country', '#city', '#coverage', '#email', '#fastmode'
    ];

    var formDefaults = null;

    function snapshotFormDefaults() {
        formDefaults = {};

        FORM_FIELDS.forEach(function (sel) {
            var node = $(sel);
            if (!node) { return; }

            formDefaults[sel] = node.type === 'checkbox' ? node.checked : node.value;
        });
    }

    function resetFormToDefaults() {
        if (!formDefaults) { return; }

        FORM_FIELDS.forEach(function (sel) {
            var node = $(sel);
            if (!node) { return; }

            if (node.type === 'checkbox') {
                node.checked = formDefaults[sel];

                return;
            }

            if (sel === '#city') {
                // The city list depends on the country, so rebuild it rather than
                // assigning a value that may no longer be an option.
                fillCities();

                return;
            }

            setValue(sel, formDefaults[sel]);
        });
    }

    function applyJobConfig(job, name) {
        var d = job.Data || {};

        // The country and city dropdowns are populated from /api/v1/locations, so
        // wait for that before trying to select a value in them.
        locationsReady.then(function () {
            resetFormToDefaults();

            setValue('#name', job.Name || name || '');
            setValue('#keywords', (d.keywords || []).join('\n'));
            setValue('#lang', d.lang || 'en');
            setValue('#depth', d.depth || 1);
            setValue('#zoom', d.zoom || 15);
            setValue('#radius', d.radius || 0);
            setValue('#latitude', d.lat || '0');
            setValue('#longitude', d.lon || '0');
            setValue('#proxies', (d.proxies || []).join('\n'));

            $('#email').checked = !!d.email;
            $('#fastmode').checked = !!d.fast_mode;

            if (d.country) {
                setValue('#country', d.country);
                fillCities();
                setValue('#city', d.city || '');
            }

            var coverage = d.coverage || deriveCoverage(d);

            if (coverage) { setValue('#coverage', coverage); }

            if (d.max_time) {
                // Stored in nanoseconds by Go's time.Duration.
                var runtime = formatDuration(d.max_time / 1e9);

                if (runtime) { setValue('#maxtime', runtime); }
            }

            // These settings were chosen for a job that already ran, so the estimate
            // must not overwrite the run time that was actually used.
            maxTimeEdited = true;

            setValue('#scanmode', 'custom');
            showConfigBanner(job.Name || name || 'this job', !d.country);
            updateCoverage();
        });
    }

    // deriveCoverage reconstructs the coverage choice for jobs stored before it was
    // recorded, so an old job shows what it actually ran rather than the form's
    // default. No bounding box means no grid was used.
    function deriveCoverage(d) {
        if (!d.bbox) { return d.city ? COVERAGE_SINGLE : ''; }

        var names = Object.keys(COVERAGE_KM);

        for (var i = 0; i < names.length; i++) {
            if (Math.abs(COVERAGE_KM[names[i]] - d.cell_size_km) < 0.01) { return names[i]; }
        }

        return '';
    }

    function setValue(selector, value) {
        var node = $(selector);
        if (!node) { return; }

        node.value = value === null || value === undefined ? '' : String(value);

        if (node.tagName === 'SELECT' && window.GMSDropdown) {
            window.GMSDropdown.refresh(node);
        }
    }

    // formatDuration renders a Go duration for the run-time field, rejecting
    // values that cannot be real. A job stored before the API duration units were
    // understood can hold an overflowed number, and echoing it back into the form
    // would produce a job nobody could start.
    var MAX_SANE_RUN_SECONDS = 24 * 3600;

    function formatDuration(seconds) {
        if (!isFinite(seconds) || seconds < 60 || seconds > MAX_SANE_RUN_SECONDS) { return ''; }

        if (seconds % 3600 === 0) { return (seconds / 3600) + 'h'; }
        if (seconds % 60 === 0) { return (seconds / 60) + 'm'; }

        return Math.round(seconds / 60) + 'm';
    }

    function showConfigBanner(name, locationUnknown) {
        $('#config-banner-name').textContent = name;
        $('#config-banner').hidden = false;
        $('#form-title').textContent = 'Job settings';

        var text = 'These are the settings this job ran with. Edit anything and start it again as a new job.';

        // Jobs scraped before the country and city fields existed have none. Saying
        // so is better than leaving the form defaults on screen as if they were the
        // settings that ran.
        if (locationUnknown) {
            text += ' This job predates the location and coverage settings, so those show defaults rather than what it ran with.';
        }

        $('#form-sub').textContent = text;
    }

    function resetConfigBanner() {
        $('#config-banner').hidden = true;
        $('#form-title').textContent = 'New scrape';
        $('#form-sub').textContent =
            'Pick a place, list what to look for, and the results appear here when the job finishes.';
    }

    // ---- job durations -----------------------------------------------------

    // formatElapsed renders a run length the way a person reads a stopwatch: no
    // units they have to divide in their head.
    function formatElapsed(seconds) {
        seconds = Math.max(0, Math.floor(seconds));

        var h = Math.floor(seconds / 3600);
        var m = Math.floor((seconds % 3600) / 60);
        var sec = seconds % 60;

        if (h > 0) { return h + 'h ' + m + 'm'; }
        if (m > 0) { return m + 'm ' + sec + 's'; }

        return sec + 's';
    }

    // paintDurations fills every duration cell. Finished jobs carry their total
    // from the server; a running job is counted up from its start time here, so it
    // ticks without waiting for the ten-second job-list refresh.
    function paintDurations() {
        var now = Date.now() / 1000;

        document.querySelectorAll('.job-duration').forEach(function (cell) {
            var status = cell.dataset.status;
            var total = parseInt(cell.dataset.duration, 10) || 0;
            var started = parseInt(cell.dataset.started, 10) || 0;

            if (status === 'ok' || status === 'failed') {
                cell.textContent = total > 0 ? formatElapsed(total) : '—';
                cell.className = 'nowrap job-duration muted';

                return;
            }

            if (status === 'working' && started > 0) {
                cell.textContent = formatElapsed(now - started);
                cell.className = 'nowrap job-duration running';

                return;
            }

            cell.textContent = '—';
            cell.className = 'nowrap job-duration faint';
        });
    }

    // ---- wiring ------------------------------------------------------------

    function initTheme() {
        var stored = null;

        try { stored = localStorage.getItem('gms-theme'); } catch (e) { /* private mode */ }

        var theme = stored || (window.matchMedia &&
            window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light');

        document.documentElement.setAttribute('data-theme', theme);

        $('#theme-toggle').addEventListener('click', function () {
            var next = document.documentElement.getAttribute('data-theme') === 'dark' ? 'light' : 'dark';
            document.documentElement.setAttribute('data-theme', next);

            try { localStorage.setItem('gms-theme', next); } catch (e) { /* private mode */ }
        });
    }

    function initFilters() {
        // The text box fires on every keystroke; the selects fire once. Debouncing
        // the text box keeps a fast typist from re-rendering the map repeatedly.
        $('#filter-search').addEventListener('input', debounce(applyFilters, 180));

        ['#filter-keyword', '#filter-category', '#filter-city', '#filter-rating']
            .forEach(function (sel) {
                $(sel).addEventListener('input', applyFilters);
            });

        document.querySelectorAll('.toggle-pill input').forEach(function (input) {
            input.addEventListener('change', function () {
                input.closest('.toggle-pill').classList.toggle('active', input.checked);

                if (input.id === 'group-by-search') {
                    state.groupBy = input.checked;
                    sortRows();
                    render();

                    return;
                }

                applyFilters();
            });
        });

        $('#clear-filters').addEventListener('click', function () {
            ['#filter-search', '#filter-keyword', '#filter-category', '#filter-city', '#filter-rating']
                .forEach(function (sel) {
                    $(sel).value = '';

                    if (window.GMSDropdown) { window.GMSDropdown.refresh($(sel)); }
                });

            state.groupBy = false;

            document.querySelectorAll('.toggle-pill input').forEach(function (input) {
                input.checked = false;
                input.closest('.toggle-pill').classList.remove('active');
            });

            applyFilters();
        });
    }

    function initTabs() {
        document.querySelectorAll('.tab').forEach(function (tab) {
            tab.addEventListener('click', function () {
                document.querySelectorAll('.tab').forEach(function (t) { t.classList.remove('active'); });
                tab.classList.add('active');
                state.view = tab.dataset.view;
                render();
            });
        });
    }

    function initDropdown() {
        var menu = $('#export-menu');

        $('#export-button').addEventListener('click', function (e) {
            e.stopPropagation();
            menu.classList.toggle('open');
        });

        document.addEventListener('click', function () { menu.classList.remove('open'); });
    }

    document.addEventListener('DOMContentLoaded', function () {
        initTheme();

        if (window.GMSDropdown) { window.GMSDropdown.enhanceAll(document); }

        initLocations();
        initFilters();
        initTabs();
        initDropdown();

        $('#config-reset').addEventListener('click', function () {
            resetConfigBanner();
            resetFormToDefaults();
            maxTimeEdited = false;
            updateCoverage();
        });

        $('#drawer-backdrop').addEventListener('click', window.closeDrawer);
        $('#drawer-close').addEventListener('click', window.closeDrawer);

        // The job list refreshes itself on a timer, which replaces the rows and
        // loses the selected highlight. Re-apply it, and pick up results as soon
        // as a job that was still running finishes.
        document.body.addEventListener('htmx:afterSwap', function (e) {
            if (e.detail.target.id !== 'job-rows') { return; }

            // Freshly swapped rows arrive with empty duration cells, so fill them
            // immediately rather than leaving them blank until the next tick. This
            // has to happen whether or not a job is currently selected.
            paintDurations();

            if (!state.jobId) { return; }

            var row = document.querySelector('.jobs-table tr[data-job-id="' + state.jobId + '"]');
            if (!row) { return; }

            row.classList.add('selected');

            if (row.dataset.status === 'ok' && !state.rows.length) {
                fetchResults(state.jobId);
            }
        });

        document.addEventListener('keydown', function (e) {
            if (e.key === 'Escape') { window.closeDrawer(); }
        });

        // One timer for the whole table: cheaper than a timer per row, and it
        // survives the job list being swapped out from under it.
        setInterval(paintDurations, 1000);
    });
}());
