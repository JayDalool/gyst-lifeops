/* GYST overlays (GYST-VN-023, 13 §6-§9): sheet, popover, confirm dialog and inspector.
   Vanilla, no dependencies. Overlays are native <dialog class="gyst-overlay">.
   Triggers: [data-gyst-open=id], [data-gyst-close], form[data-gyst-confirm=id],
   a[data-gyst-inspect=panel] inside [data-gyst-inspector]. Without JS, rows stay links. */
(function (root) {
  'use strict';
  if (root.GYSTOverlay) return; // loaded twice: keep the first instance and its listeners

  // Pure helpers, exposed for node checks.
  function mode(width) { return width >= 1024 ? 'popover' : 'sheet'; }
  function place(trigger, size, viewport) {
    var gap = 8, margin = 8;
    var left = trigger.left;
    if (left + size.width > viewport.width - margin) left = trigger.right - size.width;
    left = Math.max(margin, Math.min(left, viewport.width - size.width - margin));
    var top = trigger.bottom + gap;
    if (top + size.height > viewport.height - margin) {
      var above = trigger.top - gap - size.height;
      top = above >= margin ? above : Math.max(margin, viewport.height - size.height - margin);
    }
    return { left: left, top: top };
  }
  function step(index, delta, count) {
    if (count <= 0) return -1;
    if (index < 0) return delta > 0 ? 0 : count - 1;
    return ((index + delta) % count + count) % count;
  }
  root.GYSTOverlay = { mode: mode, place: place, step: step };

  var doc = root.document;
  if (!doc || !root.matchMedia) return;

  var FOCUSABLE = 'a[href],button:not([disabled]),input:not([disabled]):not([type="hidden"]),select:not([disabled]),textarea:not([disabled]),[tabindex]:not([tabindex="-1"])';
  var stack = []; // open overlay records, last is topmost
  var lockCount = 0, lockY = 0;

  function recordFor(dlg) { for (var i = 0; i < stack.length; i++) if (stack[i].dlg === dlg) return stack[i]; return null; }
  function focusables(dlg) {
    return Array.prototype.filter.call(dlg.querySelectorAll(FOCUSABLE), function (el) {
      return !el.hidden && !el.closest('[hidden]') && el.getClientRects().length > 0;
    });
  }

  function lockScroll() {
    if (lockCount++ > 0) return;
    lockY = root.scrollY || 0;
    doc.documentElement.classList.add('gyst-scroll-lock');
    doc.body.style.top = (-lockY) + 'px';
  }
  function unlockScroll() {
    if (lockCount === 0 || --lockCount > 0) return;
    doc.documentElement.classList.remove('gyst-scroll-lock');
    doc.body.style.top = '';
    root.scrollTo(0, lockY);
  }

  function setExpanded(inv, value) {
    if (inv && inv.hasAttribute && inv.hasAttribute('data-gyst-open')) inv.setAttribute('aria-expanded', value);
  }
  // An overlay removed from the DOM while open never fires 'close'; release it here.
  function prune() {
    stack.slice().forEach(function (rec) { if (!rec.dlg.isConnected || !rec.dlg.open) onClosed(rec.dlg); });
  }

  function open(dlg, invoker, extra) {
    prune();
    if (!dlg || dlg.open || typeof dlg.showModal !== 'function') return null;
    var isConfirm = dlg.classList.contains('gyst-confirm');
    var isPopover = dlg.classList.contains('gyst-popover');
    var asPopover = isPopover && mode(root.innerWidth) === 'popover';
    // One sheet at a time; a confirm may open over a sheet.
    if (!isConfirm) {
      stack.slice().forEach(function (rec) { if (!rec.confirm) { rec.noFocus = true; rec.dlg.close(); } });
    }
    var rec = { dlg: dlg, invoker: invoker || null, popover: asPopover, confirm: isConfirm, noFocus: false, converted: false, at: Date.now() };
    if (extra) for (var k in extra) rec[k] = extra[k];
    stack.push(rec);
    setExpanded(invoker, 'true');
    if (asPopover) {
      dlg.show();
      var r = invoker.getBoundingClientRect();
      var size = dlg.getBoundingClientRect();
      var at = place(r, { width: size.width, height: size.height }, { width: root.innerWidth, height: root.innerHeight });
      dlg.style.left = at.left + 'px';
      dlg.style.top = at.top + 'px';
    } else {
      if (isPopover) { dlg.classList.add('gyst-sheet'); dlg.style.left = ''; dlg.style.top = ''; rec.converted = true; }
      lockScroll();
      dlg.showModal();
    }
    var target = dlg.querySelector('[autofocus]') || focusables(dlg)[0];
    if (target && !dlg.contains(doc.activeElement)) target.focus();
    return rec;
  }

  function onClosed(dlg) {
    var rec = recordFor(dlg);
    if (!rec) return;
    stack.splice(stack.indexOf(rec), 1);
    if (rec.converted) dlg.classList.remove('gyst-sheet');
    if (!rec.popover) unlockScroll();
    var inv = rec.invoker;
    setExpanded(inv, 'false');
    if (!rec.noFocus && inv && inv.isConnected && inv.focus) inv.focus();
  }

  doc.addEventListener('close', function (e) { if (e.target.classList && e.target.classList.contains('gyst-overlay')) onClosed(e.target); }, true);

  function closeTop() { var rec = stack[stack.length - 1]; if (rec) rec.dlg.close(); }

  doc.addEventListener('click', function (e) {
    var t = e.target;
    if (!t.closest) return;
    prune();
    var opener = t.closest('[data-gyst-open]');
    if (opener) {
      var dlg = doc.getElementById(opener.getAttribute('data-gyst-open'));
      if (dlg && dlg.open) dlg.close(); else open(dlg, opener);
      return;
    }
    var ok = t.closest('[data-gyst-confirm-ok]');
    if (ok) {
      var crec = recordFor(ok.closest('dialog'));
      var form = crec && crec.form;
      if (crec) { crec.noFocus = !!form; crec.dlg.close(); }
      if (form && form.isConnected) {
        form.setAttribute('data-gyst-confirmed', 'true');
        try {
          if (typeof form.requestSubmit !== 'function') { form.removeAttribute('data-gyst-confirmed'); form.submit(); }
          else { try { form.requestSubmit(crec.submitter || undefined); } catch (err) { form.requestSubmit(); } }
        } catch (err2) { form.removeAttribute('data-gyst-confirmed'); }
      }
      return;
    }
    var closer = t.closest('[data-gyst-close]');
    if (closer) { var d = closer.closest('dialog.gyst-overlay'); if (d) d.close(); return; }
    // Backdrop click: the event targets the dialog itself, outside its box.
    if (t.classList && t.classList.contains('gyst-overlay') && (e.clientX || e.clientY)) {
      var b = t.getBoundingClientRect();
      if (e.clientX < b.left || e.clientX > b.right || e.clientY < b.top || e.clientY > b.bottom) t.close();
    }
  });

  // Non-modal popover: outside press closes it.
  doc.addEventListener('pointerdown', function (e) {
    stack.slice().forEach(function (rec) {
      if (rec.popover && !rec.dlg.contains(e.target) && !(rec.invoker && rec.invoker.contains(e.target))) rec.dlg.close();
    });
  });
  function closePopovers(e) {
    stack.slice().forEach(function (rec) {
      if (!rec.popover) return;
      if (e && e.type === 'scroll' && ((e.target && e.target.nodeType && rec.dlg.contains(e.target)) || Date.now() - rec.at < 300)) return;
      rec.dlg.close();
    });
  }
  root.addEventListener('resize', closePopovers);
  root.addEventListener('scroll', closePopovers, true);

  doc.addEventListener('keydown', function (e) {
    prune();
    var top = stack[stack.length - 1];
    if (top) {
      if (e.key === 'Escape' && top.popover) { e.preventDefault(); top.dlg.close(); return; }
      if (e.key === 'Tab') {
        var items = focusables(top.dlg);
        if (!items.length) { e.preventDefault(); return; }
        var first = items[0], last = items[items.length - 1], active = doc.activeElement;
        if (e.shiftKey && (active === first || active === top.dlg || !top.dlg.contains(active))) { e.preventDefault(); last.focus(); }
        else if (!e.shiftKey && (active === last || !top.dlg.contains(active))) { e.preventDefault(); first.focus(); }
        return;
      }
      if ((e.key === 'ArrowDown' || e.key === 'ArrowUp') && top.popover) {
        var list = focusables(top.dlg).filter(function (el) { return el.classList.contains('gyst-menu-item'); });
        if (list.length) { e.preventDefault(); list[step(list.indexOf(doc.activeElement), e.key === 'ArrowDown' ? 1 : -1, list.length)].focus(); }
      }
      return;
    }
    inspectorKey(e);
  });

  // Confirm: cancel the first submit, ask, then resubmit once.
  doc.addEventListener('submit', function (e) {
    var form = e.target;
    if (!form.getAttribute) return;
    var id = form.getAttribute('data-gyst-confirm');
    if (!id) return;
    if (form.getAttribute('data-gyst-confirmed')) { form.removeAttribute('data-gyst-confirmed'); return; }
    var dlg = doc.getElementById(id);
    if (!dlg) return;
    e.preventDefault();
    var invoker = e.submitter || doc.activeElement;
    open(dlg, invoker, { form: form, submitter: e.submitter || null });
  });

  // Inspector: 1280px and up, pointer click selects, keyboard click navigates.
  var inspectMQ = root.matchMedia('(min-width: 1280px)');
  function rows(host) { return Array.prototype.slice.call(host.querySelectorAll('a[data-gyst-inspect]')); }
  function select(host, row) {
    rows(host).forEach(function (a) { if (a === row) a.setAttribute('aria-current', 'true'); else a.removeAttribute('aria-current'); });
    var key = row ? row.getAttribute('data-gyst-inspect') : null;
    Array.prototype.forEach.call(host.querySelectorAll('[data-gyst-inspect-panel]'), function (p) { p.hidden = p.getAttribute('data-gyst-inspect-panel') !== key; });
    var empty = host.querySelector('[data-gyst-inspect-empty]');
    if (empty) empty.hidden = !!row;
  }
  function clearAll() { Array.prototype.forEach.call(doc.querySelectorAll('[data-gyst-inspector]'), function (h) { select(h, null); }); }
  function inspectorKey(e) {
    if (!inspectMQ.matches || !e.target.closest) return;
    var host = e.target.closest('[data-gyst-inspector]');
    if (!host) return;
    if (e.key === 'Escape') { select(host, null); return; }
    if ((e.key === 'ArrowDown' || e.key === 'ArrowUp') && e.target.matches('a[data-gyst-inspect]')) {
      var all = rows(host);
      var next = all[step(all.indexOf(e.target), e.key === 'ArrowDown' ? 1 : -1, all.length)];
      if (next) { e.preventDefault(); next.focus(); select(host, next); }
    }
  }
  doc.addEventListener('click', function (e) {
    if (!inspectMQ.matches || !e.target.closest || e.detail === 0 || e.button !== 0 || e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) return;
    var row = e.target.closest('a[data-gyst-inspect]');
    var host = row && row.closest('[data-gyst-inspector]');
    if (!host) return;
    e.preventDefault();
    select(host, row);
  });
  if (inspectMQ.addEventListener) inspectMQ.addEventListener('change', clearAll);
  else if (inspectMQ.addListener) inspectMQ.addListener(clearAll);
})(typeof window !== 'undefined' ? window : globalThis);
