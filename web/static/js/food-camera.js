// AI food camera (Issue #211). Pick or take a photo, shrink it here to a
// JPEG of at most 1024 px (a canvas re-encode also drops EXIF and turns iPhone
// HEIC into JPEG), send it for analysis, then review every item. Nothing is
// saved until "Save meal". Text from the AI is untrusted: it is only ever put
// into the page with textContent / value, never as HTML. No keys live here.
(function () {
  'use strict';
  var input = document.getElementById('camera-input');
  var pick = document.getElementById('camera-pick');
  var panel = document.getElementById('camera-review');
  if (!input || !pick || !panel) return;

  var MAX_EDGE = 1024;
  var statusEl = document.getElementById('cam-status');
  var errorEl = document.getElementById('cam-error');
  var errorText = document.getElementById('cam-error-text');
  var errorLink = document.getElementById('cam-error-link');
  var body = document.getElementById('cam-body');
  var notes = document.getElementById('cam-notes');
  var list = document.getElementById('cam-items');
  var addBtn = document.getElementById('cam-add');
  var form = document.getElementById('cam-form');
  var slot = document.getElementById('cam-slot');
  var saveBtn = document.getElementById('cam-save');
  var cancelBtn = document.getElementById('cam-cancel');
  var nonce = '';
  var busy = false;

  pick.hidden = false;

  function show(el, on) { el.hidden = !on; }
  function reset() {
    nonce = '';
    list.textContent = '';
    notes.textContent = '';
    show(body, false); show(statusEl, false); show(errorEl, false);
    show(panel, false);
    input.value = '';
  }
  function setStatus(msg) {
    show(errorEl, false);
    statusEl.textContent = msg;
    show(statusEl, !!msg);
  }
  function setError(msg, settingsLink) {
    show(statusEl, false);
    errorText.textContent = msg;
    show(errorLink, !!settingsLink);
    show(errorEl, true);
    errorEl.scrollIntoView({block: 'nearest'});
  }

  // Decode, scale to <= 1024 px on the long edge and re-encode as JPEG.
  function toJPEG(file) {
    return decode(file).then(function (img) {
      var w = img.width, h = img.height;
      if (!w || !h) throw new Error('decode');
      var k = Math.min(1, MAX_EDGE / Math.max(w, h));
      var cw = Math.max(1, Math.round(w * k)), ch = Math.max(1, Math.round(h * k));
      var canvas = document.createElement('canvas');
      canvas.width = cw; canvas.height = ch;
      var ctx = canvas.getContext('2d');
      ctx.fillStyle = '#fff'; ctx.fillRect(0, 0, cw, ch); // PNG transparency
      ctx.drawImage(img, 0, 0, cw, ch);
      if (img.close) img.close();
      return new Promise(function (resolve, reject) {
        canvas.toBlob(function (blob) { blob ? resolve(blob) : reject(new Error('encode')); }, 'image/jpeg', 0.85);
      });
    });
  }
  function decode(file) {
    if (window.createImageBitmap) {
      return createImageBitmap(file, {imageOrientation: 'from-image'}).catch(function () {
        return createImageBitmap(file);
      }).catch(function () { return viaImage(file); });
    }
    return viaImage(file);
  }
  function viaImage(file) {
    return new Promise(function (resolve, reject) {
      var url = URL.createObjectURL(file);
      var img = new Image();
      img.onload = function () { URL.revokeObjectURL(url); resolve(img); };
      img.onerror = function () { URL.revokeObjectURL(url); reject(new Error('decode')); };
      img.src = url;
    });
  }

  input.addEventListener('change', function () {
    var file = input.files && input.files[0];
    if (!file || busy) return;
    busy = true;
    show(panel, true); show(body, false); setStatus('Preparing photo…');
    list.textContent = '';
    panel.scrollIntoView({block: 'start'});
    toJPEG(file).then(function (blob) {
      setStatus('Analysing photo…');
      var fd = new FormData();
      fd.append('photo', blob, 'meal.jpg');
      return fetch('/health/nutrition/photo/analyze', {method: 'POST', body: fd, credentials: 'same-origin'});
    }, function () {
      throw {local: "This photo couldn't be read. Try a different photo."};
    }).then(function (res) {
      return res.json().catch(function () { return {}; }).then(function (data) { return {res: res, data: data}; });
    }).then(function (r) {
      if (!r.res.ok) {
        var msg = r.data && r.data.error;
        if (r.res.status === 503) throw {msg: 'Photo analysis isn’t set up yet. Add an API key in', link: true};
        if (r.res.status === 413) throw {msg: 'That photo is too large. Try a smaller photo.'};
        if (r.res.status === 415) throw {msg: msg || 'Use a JPEG or PNG photo.'};
        throw {msg: msg || 'Photo analysis failed. Nothing was saved. Try again.'};
      }
      render(r.data);
    }).catch(function (e) {
      if (e && e.local) setError(e.local);
      else if (e && e.msg) setError(e.msg, e.link);
      else setError('Could not reach GYST. Nothing was saved. Check your connection and try again.');
    }).then(function () { busy = false; input.value = ''; });
  });

  function el(tag, cls, text) {
    var n = document.createElement(tag);
    if (cls) n.className = cls;
    if (text != null) n.textContent = text;
    return n;
  }
  function tag(text, warn) { return el('span', 'hv3-chip' + (warn ? ' warn' : ''), text); }
  function num(v) {
    return v == null || !isFinite(v) ? '' : String(Math.round(v * 10) / 10);
  }
  var FIELDS = [['kcal', 'kcal'], ['protein_g', 'Protein g'], ['carbs_g', 'Carbs g'], ['fat_g', 'Fat g'], ['fiber_g', 'Fiber g']];

  function field(label, name, mode, extra) {
    var l = el('label', extra || '');
    l.appendChild(el('span', '', label));
    var i = document.createElement('input');
    i.name = name;
    if (mode === 'text') { i.type = 'text'; i.maxLength = 120; i.autocomplete = 'off'; }
    else { i.type = 'text'; i.inputMode = 'decimal'; i.autocomplete = 'off'; }
    l.appendChild(i);
    return {label: l, input: i};
  }

  // addRow builds one editable item. it.idx ties it to the server-held AI
  // estimate; a hand-added row has no idx.
  function addRow(it) {
    var li = el('li', 'hv3-cam-item');
    li._idx = it && typeof it.idx === 'number' ? it.idx : null;
    var est = it && it.est_grams != null ? it.est_grams : null;
    var match = it && it.match ? it.match : null;
    var tags = el('div', 'hv3-cam-tags');
    var grid = el('div', 'hv3-cam-grid');
    var name = field('Name', 'name', 'text', 'wide');
    var grams = field('Grams', 'grams', 'num');
    grid.appendChild(name.label); grid.appendChild(grams.label);
    var nut = {};
    FIELDS.forEach(function (f) { var x = field(f[1], f[0], 'num'); nut[f[0]] = x.input; grid.appendChild(x.label); });
    name.input.value = it ? it.name : '';
    grams.input.value = num(est);
    var touched = {};

    function fillFromMatch() {
      var g = parseFloat((grams.input.value || '').replace(',', '.'));
      if (!match || !(g > 0)) return;
      FIELDS.forEach(function (f) {
        var per = match.per_100g[f[0]];
        if (!touched[f[0]]) nut[f[0]].value = per == null ? '' : num(per * g / 100);
      });
    }
    function refreshTags() {
      tags.textContent = '';
      if (it) {
        tags.appendChild(tag('Identified' + (it.visible === false ? ' (probably hidden)' : '')));
        if (it.confidence < 0.5 || it.visible === false) tags.appendChild(tag('Uncertain', true));
      }
      var g = parseFloat((grams.input.value || '').replace(',', '.'));
      if (!(g > 0)) tags.appendChild(tag('Enter grams', true));
      else if (est != null && g === est) tags.appendChild(tag('AI estimate'));
      else tags.appendChild(tag('You confirmed grams'));
      if (match && !Object.keys(touched).length) tags.appendChild(tag('Nutrition from ' + match.source + ': ' + match.name));
      else if (FIELDS.some(function (f) { return nut[f[0]].value !== ''; })) tags.appendChild(tag('Nutrition: yours'));
      else tags.appendChild(tag('No nutrition yet', true));
    }
    grams.input.addEventListener('input', function () { fillFromMatch(); refreshTags(); });
    FIELDS.forEach(function (f) {
      nut[f[0]].addEventListener('input', function () { touched[f[0]] = true; refreshTags(); });
    });
    var rm = el('button', 'hv3-btn small secondary', 'Remove');
    rm.type = 'button';
    rm.addEventListener('click', function () { li.remove(); });
    li.appendChild(tags); li.appendChild(grid); li.appendChild(rm);
    li._read = function () {
      var out = {idx: li._idx, name: name.input.value.trim(), grams: parseNum(grams.input.value)};
      FIELDS.forEach(function (f) { out[f[0]] = nut[f[0]].value.trim() === '' ? null : parseNum(nut[f[0]].value); });
      return out;
    };
    li._focus = function () { grams.input.focus(); };
    refreshTags();
    fillFromMatch();
    refreshTags();
    list.appendChild(li);
    return li;
  }
  function parseNum(v) {
    var s = String(v).trim().replace(',', '.');
    return /^\d+(\.\d+)?$/.test(s) ? parseFloat(s) : NaN;
  }

  function render(data) {
    nonce = data.nonce;
    setStatus('');
    notes.textContent = '';
    (data.questions || []).concat(data.uncertainties || []).forEach(function (q) { notes.appendChild(el('li', '', q)); });
    list.textContent = '';
    (data.items || []).forEach(addRow);
    if (!(data.items || []).length) {
      setStatus('No food was found in that photo. Add it by hand below, or take another photo.');
    }
    show(body, true);
  }

  addBtn.addEventListener('click', function () { addRow(null)._focus(); });
  cancelBtn.addEventListener('click', reset);

  form.addEventListener('submit', function (ev) {
    ev.preventDefault();
    if (busy) return;
    var rows = Array.prototype.slice.call(list.children).map(function (li) { return li._read(); });
    if (!rows.length) return setError('Add at least one food, or discard the photo.');
    for (var i = 0; i < rows.length; i++) {
      var r = rows[i];
      if (!r.name) return setError('Every food needs a name.');
      if (!(r.grams > 0)) return setError('Enter grams for ' + r.name + '.');
      for (var f = 0; f < FIELDS.length; f++) {
        if (r[FIELDS[f][0]] !== null && isNaN(r[FIELDS[f][0]])) return setError('Nutrition for ' + r.name + ' must be a number, or left blank.');
      }
    }
    busy = true; saveBtn.disabled = true;
    setStatus('Saving…');
    fetch('/health/nutrition/photo/confirm', {
      method: 'POST', credentials: 'same-origin',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({nonce: nonce, slot: slot.value, items: rows})
    }).then(function (res) {
      return res.json().catch(function () { return {}; }).then(function (data) {
        if (res.ok && data.redirect) { window.location.href = data.redirect; return; }
        setError(data.error || 'Could not save the meal. Nothing was saved. Try again.');
      });
    }).catch(function () {
      setError('Could not reach GYST. Nothing was saved. Try again.');
    }).then(function () { busy = false; saveBtn.disabled = false; });
  });
})();
