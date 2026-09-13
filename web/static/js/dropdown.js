/* Accessible dropdown built on top of a real <select>.
 *
 * Native option lists are drawn by the operating system and cannot be styled, so
 * a long country list looks out of place next to the rest of the interface. This
 * enhances a <select> rather than replacing it: the original element stays in the
 * DOM and keeps its value, so form submission, validation and the no-JavaScript
 * fallback all continue to work. Only the presentation is taken over.
 */
(function () {
    'use strict';

    // Lists longer than this get a search box; below it, scanning is faster than typing.
    var SEARCH_THRESHOLD = 8;

    var openInstance = null;
    var idCounter = 0;

    function el(tag, attrs, children) {
        var node = document.createElement(tag);

        Object.keys(attrs || {}).forEach(function (k) {
            if (k === 'text') { node.textContent = attrs[k]; }
            else if (k === 'class') { node.className = attrs[k]; }
            else if (k.slice(0, 2) === 'on') { node.addEventListener(k.slice(2), attrs[k]); }
            else if (attrs[k] !== null && attrs[k] !== undefined) { node.setAttribute(k, attrs[k]); }
        });

        (children || []).forEach(function (c) {
            if (c) { node.appendChild(typeof c === 'string' ? document.createTextNode(c) : c); }
        });

        return node;
    }

    function Dropdown(select) {
        this.select = select;
        this.id = 'dd-' + (++idCounter);
        this.activeIndex = -1;
        this.options = [];

        this.root = el('div', { class: 'select' });
        this.button = el('button', {
            type: 'button',
            class: 'select-button',
            id: this.id + '-button',
            'aria-haspopup': 'listbox',
            'aria-expanded': 'false'
        });

        this.label = el('span', { class: 'select-value' });
        this.button.appendChild(this.label);
        this.button.appendChild(el('span', { class: 'select-caret', 'aria-hidden': 'true' }));

        this.panel = el('div', { class: 'select-panel', hidden: 'hidden' });
        this.list = el('div', {
            class: 'select-list',
            role: 'listbox',
            id: this.id + '-list',
            tabindex: '-1'
        });

        this.searchWrap = el('div', { class: 'select-search' });
        this.search = el('input', {
            type: 'text',
            class: 'select-search-input',
            placeholder: 'Search…',
            autocomplete: 'off',
            'aria-label': 'Filter options'
        });
        this.searchWrap.appendChild(this.search);

        this.panel.appendChild(this.searchWrap);
        this.panel.appendChild(this.list);

        this.root.appendChild(this.button);
        this.root.appendChild(this.panel);

        select.parentNode.insertBefore(this.root, select);
        this.root.appendChild(select);
        select.classList.add('select-native');
        select.setAttribute('tabindex', '-1');
        select.setAttribute('aria-hidden', 'true');

        // A label pointing at the select should focus the visible control instead.
        if (select.id) {
            var labelEl = document.querySelector('label[for="' + select.id + '"]');
            if (labelEl) {
                this.button.setAttribute('aria-labelledby', labelEl.id || (labelEl.id = this.id + '-label'));
                labelEl.addEventListener('click', function (e) {
                    e.preventDefault();
                    this.button.focus();
                }.bind(this));
            }
        }

        this.bind();
        this.refresh();
    }

    Dropdown.prototype.bind = function () {
        var self = this;

        this.button.addEventListener('click', function () {
            if (self.isOpen()) { self.close(); } else { self.open(); }
        });

        this.button.addEventListener('keydown', function (e) {
            if (e.key === 'ArrowDown' || e.key === 'ArrowUp' || e.key === 'Enter' || e.key === ' ') {
                e.preventDefault();
                self.open();
            }
        });

        this.search.addEventListener('input', function () { self.renderOptions(); });

        this.panel.addEventListener('keydown', function (e) { self.onPanelKey(e); });

        // The native select can still change programmatically; mirror it.
        this.select.addEventListener('change', function () { self.syncLabel(); });
    };

    Dropdown.prototype.onPanelKey = function (e) {
        var visible = this.visibleOptions();

        switch (e.key) {
            case 'Escape':
                e.preventDefault();
                this.close(true);
                break;
            case 'ArrowDown':
                e.preventDefault();
                this.moveActive(1, visible);
                break;
            case 'ArrowUp':
                e.preventDefault();
                this.moveActive(-1, visible);
                break;
            case 'Home':
                e.preventDefault();
                this.setActive(0, visible);
                break;
            case 'End':
                e.preventDefault();
                this.setActive(visible.length - 1, visible);
                break;
            case 'Enter':
                e.preventDefault();
                if (visible[this.activeIndex]) { this.choose(visible[this.activeIndex].value); }
                break;
            case 'Tab':
                this.close(true);
                break;
            default:
                break;
        }
    };

    Dropdown.prototype.isOpen = function () { return !this.panel.hidden; };

    Dropdown.prototype.open = function () {
        if (openInstance && openInstance !== this) { openInstance.close(); }

        openInstance = this;

        this.panel.hidden = false;
        this.button.setAttribute('aria-expanded', 'true');
        this.root.classList.add('open');
        this.search.value = '';

        var searchable = this.options.length > SEARCH_THRESHOLD;
        this.searchWrap.hidden = !searchable;

        this.renderOptions();
        this.dropUpIfNeeded();

        if (searchable) { this.search.focus(); } else { this.list.focus(); }
    };

    // dropUpIfNeeded flips the panel above the control when there is not enough
    // room below, so a country list near the bottom of the sidebar stays usable.
    Dropdown.prototype.dropUpIfNeeded = function () {
        this.root.classList.remove('drop-up');

        var rect = this.button.getBoundingClientRect();
        var below = window.innerHeight - rect.bottom;

        if (below < 260 && rect.top > below) { this.root.classList.add('drop-up'); }
    };

    Dropdown.prototype.close = function (focusButton) {
        this.panel.hidden = true;
        this.button.setAttribute('aria-expanded', 'false');
        this.root.classList.remove('open', 'drop-up');
        this.activeIndex = -1;

        if (openInstance === this) { openInstance = null; }
        if (focusButton) { this.button.focus(); }
    };

    // refresh re-reads the native select, for when options are replaced at runtime
    // (choosing a country repopulates the city list).
    Dropdown.prototype.refresh = function () {
        this.options = Array.prototype.map.call(this.select.options, function (o) {
            return { value: o.value, label: o.textContent, disabled: o.disabled };
        });

        this.syncLabel();

        if (this.isOpen()) { this.renderOptions(); }
    };

    Dropdown.prototype.syncLabel = function () {
        var selected = this.select.options[this.select.selectedIndex];
        var text = selected ? selected.textContent : '';

        this.label.textContent = text || this.select.getAttribute('data-placeholder') || 'Select…';
        this.label.classList.toggle('is-placeholder', !this.select.value);
        this.button.disabled = this.select.disabled || this.options.length === 0;
    };

    Dropdown.prototype.visibleOptions = function () {
        var q = this.search.value.trim().toLowerCase();
        if (!q) { return this.options; }

        return this.options.filter(function (o) {
            return o.label.toLowerCase().indexOf(q) !== -1;
        });
    };

    Dropdown.prototype.renderOptions = function () {
        var self = this;
        var visible = this.visibleOptions();

        this.list.replaceChildren();

        if (!visible.length) {
            this.list.appendChild(el('div', { class: 'select-empty', text: 'No matches' }));
            this.activeIndex = -1;

            return;
        }

        visible.forEach(function (opt, i) {
            var selected = opt.value === self.select.value;

            if (selected) { self.activeIndex = i; }

            var item = el('div', {
                class: 'select-option' + (selected ? ' selected' : ''),
                role: 'option',
                id: self.id + '-opt-' + i,
                'aria-selected': selected ? 'true' : 'false',
                text: opt.label,
                onclick: function () { self.choose(opt.value); }
            });

            if (opt.disabled) { item.classList.add('disabled'); }

            item.addEventListener('mousemove', function () { self.setActive(i, visible); });

            self.list.appendChild(item);
        });

        if (this.activeIndex < 0 || this.activeIndex >= visible.length) { this.activeIndex = 0; }

        this.setActive(this.activeIndex, visible);
    };

    Dropdown.prototype.setActive = function (index, visible) {
        if (!visible.length) { return; }

        this.activeIndex = Math.max(0, Math.min(index, visible.length - 1));

        var items = this.list.querySelectorAll('.select-option');

        items.forEach(function (node, i) {
            node.classList.toggle('active', i === this.activeIndex);
        }.bind(this));

        var active = items[this.activeIndex];

        if (active) {
            this.list.setAttribute('aria-activedescendant', active.id);
            active.scrollIntoView({ block: 'nearest' });
        }
    };

    Dropdown.prototype.moveActive = function (delta, visible) {
        this.setActive(this.activeIndex + delta, visible);
    };

    Dropdown.prototype.choose = function (value) {
        if (this.select.value !== value) {
            this.select.value = value;
            this.select.dispatchEvent(new Event('change', { bubbles: true }));
            this.select.dispatchEvent(new Event('input', { bubbles: true }));
        }

        this.syncLabel();
        this.close(true);
    };

    document.addEventListener('mousedown', function (e) {
        if (openInstance && !openInstance.root.contains(e.target)) { openInstance.close(); }
    });

    window.addEventListener('resize', function () {
        if (openInstance) { openInstance.close(); }
    });

    window.GMSDropdown = {
        // enhance upgrades one select and returns the instance, or the existing
        // one when called twice on the same element.
        enhance: function (select) {
            if (!select) { return null; }
            if (select.__dropdown) { return select.__dropdown; }

            select.__dropdown = new Dropdown(select);

            return select.__dropdown;
        },
        // enhanceAll upgrades every select inside root that has not been done yet.
        enhanceAll: function (root) {
            (root || document).querySelectorAll('select').forEach(function (s) {
                window.GMSDropdown.enhance(s);
            });
        },
        // refresh re-reads a select whose options were replaced.
        refresh: function (select) {
            if (select && select.__dropdown) { select.__dropdown.refresh(); }
        }
    };
}());
