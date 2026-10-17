/*
 * Architecture reference behaviour: theme toggle, sidebar filter, scroll spy,
 * copy buttons and the Ctrl/Cmd-K command palette.
 *
 * Everything here reads the document that is already in the file. There is no
 * fetch, no framework and no build step: the whole point of this export is
 * that it keeps working from an email attachment years from now.
 */
(function () {
  'use strict';

  var doc = document;
  var root = doc.documentElement;

  /* ---------------------------------------------------------------- theme */

  function currentTheme() {
    var chosen = root.getAttribute('data-theme');
    if (chosen) {
      return chosen;
    }
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
  }

  function applyTheme(theme) {
    root.setAttribute('data-theme', theme);
    try {
      localStorage.setItem('cqrs-architecture-theme', theme);
    } catch (error) {
      /* nothing to persist to; the choice still applies for this visit */
    }
  }

  var themeBtn = doc.getElementById('theme-btn');
  if (themeBtn) {
    themeBtn.addEventListener('click', function () {
      applyTheme(currentTheme() === 'dark' ? 'light' : 'dark');
    });
  }

  /* -------------------------------------------------------- sidebar filter */

  var filter = doc.getElementById('nav-filter');
  var empty = doc.getElementById('nav-empty');
  var groups = doc.querySelectorAll('.nav-group');

  function applyFilter() {
    var query = filter.value.trim().toLowerCase();
    var visible = 0;

    for (var i = 0; i < groups.length; i++) {
      var items = groups[i].querySelectorAll('.item');
      var shown = 0;

      for (var j = 0; j < items.length; j++) {
        var haystack = (items[j].textContent + ' ' + (items[j].dataset.alt || '')).toLowerCase();
        var hit = query === '' || haystack.indexOf(query) !== -1;
        items[j].hidden = !hit;
        shown += hit ? 1 : 0;
      }

      groups[i].hidden = shown === 0;
      visible += shown;
    }

    if (empty) {
      empty.hidden = visible !== 0;
    }
  }

  if (filter) {
    filter.addEventListener('input', applyFilter);
  }

  /* ----------------------------------------------------------- scroll spy */

  var navLinks = doc.querySelectorAll('.sidenav .item');
  var byAnchor = {};
  for (var n = 0; n < navLinks.length; n++) {
    var href = navLinks[n].getAttribute('href') || '';
    if (href.charAt(0) === '#') {
      byAnchor[href.slice(1)] = navLinks[n];
    }
  }

  function markActive(anchor) {
    for (var key in byAnchor) {
      if (Object.prototype.hasOwnProperty.call(byAnchor, key)) {
        byAnchor[key].classList.toggle('active', key === anchor);
      }
    }
  }

  if (window.IntersectionObserver) {
    var observer = new IntersectionObserver(function (entries) {
      entries.forEach(function (entry) {
        if (entry.isIntersecting) {
          markActive(entry.target.id);
        }
      });
    }, { rootMargin: '-58px 0px -70% 0px', threshold: 0 });

    doc.querySelectorAll('section.section').forEach(function (section) {
      observer.observe(section);
    });
  }

  /* --------------------------------------------------------- copy buttons */

  doc.querySelectorAll('.copy').forEach(function (button) {
    button.addEventListener('click', function () {
      var block = button.closest('.code');
      var code = block && block.querySelector('pre');
      if (!code || !navigator.clipboard) {
        return;
      }

      navigator.clipboard.writeText(code.innerText).then(function () {
        var label = button.querySelector('span');
        if (!label) {
          return;
        }
        var previous = label.textContent;
        label.textContent = 'Copied';
        window.setTimeout(function () { label.textContent = previous; }, 1400);
      }, function () {
        /* clipboard denied; the code is selectable anyway */
      });
    });
  });

  /* ------------------------------------------------------ command palette */

  var palette = doc.getElementById('cmdk');
  var scrim = doc.getElementById('cmdk-scrim');
  var input = doc.getElementById('cmdk-input');
  var results = doc.getElementById('cmdk-results');
  var openBtn = doc.getElementById('search-btn');
  var index = null;
  var matches = [];
  var selected = 0;

  function entries() {
    if (index === null) {
      var payload = doc.getElementById('search-index');
      index = payload ? JSON.parse(payload.textContent) : [];
    }
    return index;
  }

  function escapeHtml(value) {
    return String(value).replace(/[&<>"']/g, function (character) {
      return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[character];
    });
  }

  function search(query) {
    var needle = query.trim().toLowerCase();
    var kept = [];

    entries().forEach(function (entry) {
      var haystack = (entry.t + ' ' + entry.a + ' ' + entry.s + ' ' + entry.d).toLowerCase();
      if (needle === '' || haystack.indexOf(needle) !== -1) {
        kept.push(entry);
      }
    });

    return kept.slice(0, 60);
  }

  function render() {
    matches = search(input.value);
    selected = 0;

    if (!matches.length) {
      results.innerHTML = '<div class="cmdk-none">Nothing matches that search.</div>';
      return;
    }

    var html = '';
    var group = null;
    var number = 0;

    matches.forEach(function (entry) {
      if (entry.g !== group) {
        group = entry.g;
        html += '<div class="cmdk-group">' + escapeHtml(group) + '</div>';
      }

      html += '<div class="cmdk-item" role="option" data-n="' + number + '" aria-selected="' +
        (number === 0) + '"><span class="k ' + escapeHtml(entry.c) + '">' + escapeHtml(entry.k) +
        '</span><span class="tt"><b>' + escapeHtml(entry.t) + '</b><span>' + escapeHtml(entry.s) +
        '</span></span><span class="dom">' + escapeHtml(entry.d) + '</span></div>';
      number++;
    });

    results.innerHTML = html;
  }

  function move(delta) {
    if (!matches.length) {
      return;
    }

    var previous = results.querySelector('[data-n="' + selected + '"]');
    if (previous) {
      previous.setAttribute('aria-selected', 'false');
    }

    selected = (selected + delta + matches.length) % matches.length;

    var current = results.querySelector('[data-n="' + selected + '"]');
    if (current) {
      current.setAttribute('aria-selected', 'true');
      current.scrollIntoView({ block: 'nearest' });
    }
  }

  function go(number) {
    var entry = matches[number];
    if (!entry) {
      return;
    }

    close(false);
    var target = doc.getElementById(entry.h);
    if (!target) {
      return;
    }

    target.scrollIntoView({ behavior: 'smooth', block: 'start' });
    history.replaceState(null, '', '#' + entry.h);
    target.classList.add('flash');
    window.setTimeout(function () { target.classList.remove('flash'); }, 1200);
  }

  function open() {
    if (!palette) {
      return;
    }

    palette.hidden = false;
    if (scrim) {
      scrim.hidden = false;
    }
    input.value = '';
    render();
    input.focus();
  }

  function close(refocus) {
    if (!palette) {
      return;
    }

    palette.hidden = true;
    if (scrim) {
      scrim.hidden = true;
    }
    if (refocus !== false && openBtn) {
      openBtn.focus();
    }
  }

  if (openBtn) {
    openBtn.addEventListener('click', open);
  }
  if (scrim) {
    scrim.addEventListener('click', function () { close(); });
  }

  if (input) {
    input.addEventListener('input', render);
    input.addEventListener('keydown', function (event) {
      if (event.key === 'ArrowDown') {
        event.preventDefault();
        move(1);
      } else if (event.key === 'ArrowUp') {
        event.preventDefault();
        move(-1);
      } else if (event.key === 'Enter') {
        event.preventDefault();
        go(selected);
      } else if (event.key === 'Escape') {
        event.preventDefault();
        close();
      }
    });
  }

  if (results) {
    results.addEventListener('click', function (event) {
      var item = event.target.closest('.cmdk-item');
      if (item) {
        go(Number(item.dataset.n));
      }
    });
  }

  doc.addEventListener('keydown', function (event) {
    if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'k') {
      event.preventDefault();
      if (palette && palette.hidden) {
        open();
      } else {
        close();
      }
      return;
    }

    var typing = /^(INPUT|TEXTAREA|SELECT)$/.test((doc.activeElement || {}).tagName || '');
    if (event.key === '/' && palette && palette.hidden && !typing) {
      event.preventDefault();
      open();
    }
    if (event.key === 'Escape' && palette && !palette.hidden) {
      close();
    }
  });

  if (/Mac|iPhone|iPad/.test(navigator.userAgent)) {
    var hint = doc.getElementById('kbd-hint');
    if (hint) {
      hint.textContent = '⌘K';
    }
  }
})();
