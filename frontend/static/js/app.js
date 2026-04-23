// Handle HX-Redirect responses globally
document.addEventListener('htmx:afterRequest', function(evt) {
  var xhr = evt.detail.xhr;
  if (xhr) {
    var redirect = xhr.getResponseHeader('HX-Redirect');
    if (redirect) {
      window.location.href = redirect;
    }
  }
});

// Collapsible sidebar (persistent on desktop, overlay on mobile)
(function() {
  var panel = document.getElementById('sidebar-panel');
  if (!panel) return;
  var toggle = document.getElementById('sidebar-toggle');
  var backdrop = document.getElementById('sidebar-backdrop');
  var MOBILE_BREAKPOINT = 768;

  function isMobile() { return window.innerWidth <= MOBILE_BREAKPOINT; }

  function showBackdrop() { if (backdrop) backdrop.classList.add('visible'); }
  function hideBackdrop() { if (backdrop) backdrop.classList.remove('visible'); }

  function openSidebar() {
    panel.classList.remove('collapsed');
    if (isMobile()) showBackdrop();
    try { localStorage.setItem('sidebar-open', '1'); } catch(e) {}
  }
  function closeSidebar() {
    panel.classList.add('collapsed');
    hideBackdrop();
    try { localStorage.setItem('sidebar-open', '0'); } catch(e) {}
  }
  function isOpen() { return !panel.classList.contains('collapsed'); }

  // Restore saved state: default closed on mobile, open on desktop
  try {
    var saved = localStorage.getItem('sidebar-open');
    if (isMobile()) {
      closeSidebar();
    } else if (saved === '0') {
      closeSidebar();
    }
  } catch(e) {}

  if (toggle) toggle.addEventListener('click', function() { isOpen() ? closeSidebar() : openSidebar(); });
  if (backdrop) backdrop.addEventListener('click', closeSidebar);

  // Close sidebar on mobile when a nav link is clicked
  panel.addEventListener('click', function(e) {
    if (isMobile() && e.target.closest('a')) {
      closeSidebar();
    }
  });

  // Highlight active nav link based on current path
  var path = window.location.pathname;
  function highlightActiveNav() {
    panel.querySelectorAll('.sidebar-nav-link').forEach(function(link) {
      var linkPath = link.getAttribute('data-path');
      var active = (linkPath === '/notes' && (path === '/notes' || /^\/notes\//.test(path)))
                || (linkPath === '/todos' && path === '/todos')
                || (linkPath === '/archive' && /^\/archive/.test(path));
      link.classList.toggle('active', active);
    });
  }
  document.body.addEventListener('htmx:afterSwap', highlightActiveNav);
  highlightActiveNav();

  window._toggleSidebar = function() { isOpen() ? closeSidebar() : openSidebar(); };
})();

// Shortcuts help modal
(function() {
  var modal = document.getElementById('shortcuts-modal');
  if (!modal) return;

  function openHelp() { modal.classList.add('open'); }
  function closeHelp() { modal.classList.remove('open'); }
  function isHelpOpen() { return modal.classList.contains('open'); }

  modal.querySelector('.shortcuts-backdrop').addEventListener('click', closeHelp);

  // Button in sidebar
  document.addEventListener('click', function(e) {
    if (e.target.closest('#show-shortcuts')) {
      e.preventDefault();
      openHelp();
    }
  });

  window._toggleHelp = function() { isHelpOpen() ? closeHelp() : openHelp(); };
  window._closeHelp = closeHelp;
})();

// Keyboard shortcuts
(function() {
  function isInput(el) {
    if (!el) return false;
    var tag = el.tagName;
    return tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT' || el.isContentEditable;
  }

  document.addEventListener('keydown', function(e) {
    // Escape: close help modal
    if (e.key === 'Escape') {
      if (window._closeHelp) window._closeHelp();
      return;
    }

    // Ctrl+Enter: go to notes list (reader, todos, archive pages)
    if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
      var path = window.location.pathname;
      if ((/^\/notes\/[^/]+$/.test(path) && !path.endsWith('/edit'))
          || path === '/todos' || /^\/archive/.test(path)) {
        e.preventDefault();
        window.location.href = '/notes';
      }
      return;
    }

    if (isInput(document.activeElement)) return;
    if (e.ctrlKey || e.metaKey || e.altKey) return;

    var path = window.location.pathname;

    // "/" — focus search
    if (e.key === '/') {
      e.preventDefault();
      var search = document.getElementById('search-input');
      if (search) { search.focus(); search.select(); }
      else { window.location.href = '/notes'; }
      return;
    }

    // "t" — toggle sidebar
    if (e.key === 't') {
      e.preventDefault();
      if (window._toggleSidebar) window._toggleSidebar();
      return;
    }

    // "d" — go to todos view
    if (e.key === 'd') {
      e.preventDefault();
      window.location.href = '/todos';
      return;
    }

    // "?" — toggle shortcuts help
    if (e.key === '?') {
      e.preventDefault();
      if (window._toggleHelp) window._toggleHelp();
      return;
    }

    // "e" — edit note (reader page only: /notes/{slug} but not /notes/{slug}/edit)
    if (e.key === 'e' && /^\/notes\/[^/]+$/.test(path)) {
      e.preventDefault();
      window.location.href = path + '/edit';
      return;
    }

    // "a" — archive note (reader page only)
    if (e.key === 'a' && /^\/notes\/[^/]+$/.test(path)) {
      e.preventDefault();
      var archiveBtn = document.querySelector('.btn-archive');
      if (archiveBtn) archiveBtn.click();
      return;
    }
  });
})();

// Tag colors
(function() {
  var COLOR_PALETTE = [
    '#001219', '#005f73', '#0a9396', '#94d2bd', '#e9d8a6',
    '#ee9b00', '#ca6702', '#bb3e03', '#ae2012', '#9b2226'
  ];

  function hashColor(name) {
    var h = 0;
    for (var i = 0; i < name.length; i++) {
      h = (h * 31 + name.charCodeAt(i)) >>> 0;
    }
    return COLOR_PALETTE[h % COLOR_PALETTE.length];
  }

  var tagColors = {};
  var colorsLoaded = false;

  function applyTagColors() {
    // Always update all .tag elements (not just new ones) to handle color changes
    document.querySelectorAll('.tag').forEach(function(el) {
      // Skip tags that already have inline style from server (sidebar)
      if (el.hasAttribute('data-server-colored')) return;
      
      var text = el.textContent.trim();
      var match = text.match(/^#([\w-]+)/);
      if (!match) return;
      var name = match[1].toLowerCase();
      var color = tagColors[name] || hashColor(name);
      el.classList.add('tag-colored');
      el.style.backgroundColor = color;
      el.setAttribute('data-tag', name);
      el.setAttribute('data-color', color);
    });
  }

  function loadTagColors() {
    fetch('/tags', { credentials: 'same-origin' })
      .then(function(r) { return r.json(); })
      .then(function(list) {
        (list || []).forEach(function(t) {
          var name = (t.Name || t.name || '').toLowerCase();
          var color = t.Color || t.color;
          if (name && color) tagColors[name] = color;
        });
        colorsLoaded = true;
        applyTagColors();
      })
      .catch(function() {
        colorsLoaded = true;
        applyTagColors();
      });
  }

  // Color picker
  var pickerEl = null;
  var pickerTarget = null;

  function createPicker() {
    if (pickerEl) return;
    pickerEl = document.createElement('div');
    pickerEl.className = 'tag-color-picker';
    var grid = document.createElement('div');
    grid.className = 'tag-color-grid';
    COLOR_PALETTE.forEach(function(c) {
      var swatch = document.createElement('div');
      swatch.className = 'tag-color-swatch';
      swatch.style.backgroundColor = c;
      swatch.setAttribute('data-color', c);
      swatch.addEventListener('click', function(e) {
        e.stopPropagation();
        selectColor(c);
      });
      grid.appendChild(swatch);
    });
    pickerEl.appendChild(grid);
    document.body.appendChild(pickerEl);
  }

  function showPicker(tagEl) {
    createPicker();
    pickerTarget = tagEl;
    var rect = tagEl.getBoundingClientRect();
    pickerEl.style.top = (rect.bottom + window.scrollY + 4) + 'px';
    pickerEl.style.left = (rect.left + window.scrollX) + 'px';
    var current = tagEl.getAttribute('data-color');
    pickerEl.querySelectorAll('.tag-color-swatch').forEach(function(s) {
      s.classList.toggle('selected', s.getAttribute('data-color') === current);
    });
    pickerEl.classList.add('active');
  }

  function hidePicker() {
    if (pickerEl) pickerEl.classList.remove('active');
    pickerTarget = null;
  }

  function selectColor(color) {
    if (!pickerTarget) return;
    var name = pickerTarget.getAttribute('data-tag');
    fetch('/tags/' + encodeURIComponent(name) + '/color', {
      method: 'PUT',
      credentials: 'same-origin',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ color: color })
    }).then(function(r) {
      if (r.ok) {
        tagColors[name] = color;
        document.querySelectorAll('[data-tag="' + name + '"]').forEach(function(el) {
          el.style.backgroundColor = color;
          el.setAttribute('data-color', color);
        });
      }
    });
    hidePicker();
  }

  document.addEventListener('click', function(e) {
    var tagEl = e.target.closest('.tag-colored');
    if (tagEl && tagEl.hasAttribute('data-tag')) {
      if (e.ctrlKey || e.metaKey) {
        e.preventDefault();
        showPicker(tagEl);
        return;
      }
    }
    hidePicker();
  });

  document.addEventListener('DOMContentLoaded', loadTagColors);
  // Re-apply colors after htmx swaps, but only if colors are loaded
  document.body.addEventListener('htmx:afterSwap', function() {
    if (colorsLoaded) applyTagColors();
  });
})();
