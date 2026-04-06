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
