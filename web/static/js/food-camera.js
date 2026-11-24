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
  var portionEl = document.getElementById('cam-portion');
  var portionOut = document.getElementById('cam-portion-out');
  var portionReset = document.getElementById('cam-portion-reset');
  var totalEl = document.getElementById('cam-total');
  var liveEl = document.getElementById('cam-live');
  var find = document.getElementById('cam-find');
  var findQ = document.getElementById('cam-find-q');
  var findList = document.getElementById('cam-find-list');
  var findNote = document.getElementById('cam-find-note');
  var findOwn = document.getElementById('cam-find-own');
  var findClose = document.getElementById('cam-find-close');
  var M = window.FoodCameraMath;
  var nonce = '';
  var busy = false;
  var portion = 1;

  pick.hidden = false;

  function show(el, on) { el.hidden = !on; }
  function reset() {
    nonce = '';
    setPortion(1);
    closeFind();
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
    closeFind();
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
  var num = M.num; // pure maths, shared with the Go golden-table test
  var FIELDS = [['kcal', 'kcal'], ['protein_g', 'Protein g'], ['carbs_g', 'Carbs g'], ['fat_g', 'Fat g'], ['fiber_g', 'Fiber g']];
  var AI_FIELDS = ['kcal', 'protein_g', 'carbs_g', 'fat_g'];
  var STEP = 10, MAX_GRAMS = 100000;
  function round1(v) { return Math.round(v * 10) / 10; }
  var parseNum = M.parseAmount;

  // One polite live region for every spoken change; debounced so holding a
  // stepper does not queue a sentence per tap.
  var liveTimer = 0;
  function announce(msg) {
    clearTimeout(liveTimer);
    liveTimer = setTimeout(function () { liveEl.textContent = ''; liveEl.textContent = msg; }, 400);
  }

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

  var rowSeq = 0;
  function rowEls() { return Array.prototype.slice.call(list.children); }

  // addRow builds one editable item. it.idx ties it to the server-held AI
  // estimate; a hand-added row has no idx. GYST-179: grams are kept as
  // baseGrams (grams at portion x1); the shown grams are base x portion, and a
  // typed or stepped value sets base = shown / portion, so the slider and the
  // stepper never scale twice. A database food (match) is grams-only: its
  // nutrition is shown, not edited, and the server recomputes it on save.
  function addRow(it, match0) {
    var li = el('li', 'hv3-cam-item');
    li._idx = it && typeof it.idx === 'number' ? it.idx : null;
    var est = it && it.est_grams != null ? it.est_grams : null;
    var match = match0 || (it && it.match ? it.match : null);
    var base = null;               // grams at x1
    var ratio = {};                // free-form: nutrient per gram when last set
    // GYST-180: the AI's own nutrition for this food at `est` grams, only for a
    // food with no database match. aiKeys marks the fields still showing it;
    // typing in a field hands that field to the owner for good.
    var aiEst = !match && est != null && it && it.est_nutrition ? it.est_nutrition : null;
    var aiKeys = {};
    if (aiEst) AI_FIELDS.forEach(function (k) { if (aiEst[k] != null) aiKeys[k] = true; });
    var tags = el('div', 'hv3-cam-tags');
    var grid = el('div', 'hv3-cam-grid');
    var name = field('Name', 'name', 'text', 'wide');
    grid.appendChild(name.label);
    name.input.value = it ? it.name : (match ? match.name : '');

    // Grams stepper: 44 px buttons, typed grams, ArrowUp/Down = +/-10 g.
    var step = el('div', 'hv3-step');
    step.setAttribute('role', 'group');
    var less = el('button', 'hv3-step-btn', '−'); less.type = 'button';
    var more = el('button', 'hv3-step-btn', '+'); more.type = 'button';
    var gl = el('label', 'hv3-sr');
    var gi = document.createElement('input');
    gi.className = 'hv3-step-val'; gi.name = 'grams'; gi.type = 'text'; gi.inputMode = 'decimal'; gi.autocomplete = 'off';
    gi.id = 'cam-grams-' + (++rowSeq);
    gl.htmlFor = gi.id;
    var glText = document.createTextNode('');
    gl.appendChild(glText);
    step.appendChild(less); step.appendChild(gl); step.appendChild(gi); step.appendChild(more);
    var gramsWrap = el('div', 'hv3-cam-gramsrow');
    gramsWrap.appendChild(el('span', 'hv3-cam-gramslabel', 'Grams'));
    gramsWrap.appendChild(step);

    var macroBox = el('div', 'hv3-cam-macrobox');
    var nut = {};
    var dl = null, dd = {};

    function label() { return name.input.value.trim() || 'this food'; }
    function relabel() {
      var n = label();
      step.setAttribute('aria-label', 'Grams of ' + n);
      glText.nodeValue = 'Grams of ' + n;
      less.setAttribute('aria-label', n + ', 10 grams less');
      more.setAttribute('aria-label', n + ', 10 grams more');
      swapBtn.lastChild.textContent = ' ' + n;
      rm.lastChild.textContent = ' ' + n;
    }
    function gramsNow() { var g = parseNum(gi.value); return g > 0 ? g : NaN; }

    function buildMacros() {
      macroBox.textContent = '';
      nut = {}; dl = null; dd = {};
      if (match) {
        dl = el('dl', 'hv3-cam-macros');
        FIELDS.forEach(function (f) {
          var box = el('div');
          box.appendChild(el('dt', '', f[1]));
          dd[f[0]] = el('dd');
          box.appendChild(dd[f[0]]);
          dl.appendChild(box);
        });
        macroBox.appendChild(dl);
      } else {
        var g2 = el('div', 'hv3-cam-grid');
        FIELDS.forEach(function (f) {
          var x = field(f[1], f[0], 'num');
          nut[f[0]] = x.input;
          // The owner's typing is never rewritten: only the ratio is noted, and
          // the other fields move when the grams or the portion change.
          x.input.addEventListener('input', function () {
            delete aiKeys[f[0]];
            var r = M.typedRatio(x.input.value, gramsNow());
            if (r != null) ratio[f[0]] = r; else delete ratio[f[0]];
            refreshTags(); updateTotal();
          });
          g2.appendChild(x.label);
        });
        macroBox.appendChild(g2);
        if (aiEst) macroBox.appendChild(el('p', 'hv3-s', 'AI estimate for these grams: a guess, so change any value that looks wrong.'));
      }
    }
    // followGrams moves the macros with the grams: a database food by its
    // per-100 g data, a free-form row by the per-gram ratio of its last typed values.
    function followGrams() {
      var g = gramsNow();
      FIELDS.forEach(function (f) {
        var k = f[0];
        if (match) {
          var v = g > 0 ? M.prefill(match.per_100g[k], g) : '';
          dd[k].textContent = v === '' ? '—' : v;
          dd[k]._v = v;
        } else if (g > 0 && aiKeys[k]) {
          nut[k].value = M.prefillEst(aiEst[k], est, g);
        } else if (g > 0 && ratio[k] != null) {
          nut[k].value = M.followValues(ratio, g)[k];
        }
      });
    }
    // captureNow fixes a ratio for nutrients typed before any grams, at the
    // grams as they stand now (before a step or the slider moves them).
    function captureNow() {
      if (match) return;
      var texts = {};
      // A field still showing the AI estimate follows the estimate, not a ratio.
      FIELDS.forEach(function (f) { if (!aiKeys[f[0]]) texts[f[0]] = nut[f[0]].value; });
      ratio = M.captureRatios(ratio, texts, gramsNow());
    }
    function macroVal(k) {
      if (match) return dd[k]._v === '' || dd[k]._v == null ? null : parseFloat(dd[k]._v);
      var t = nut[k].value.trim();
      return t === '' ? null : parseNum(t);
    }
    li._macro = macroVal;
    // aiDriven: every estimated field is still the AI's number and nothing else
    // was typed, which is exactly when the server stores the item as an AI
    // estimate. Fibre is not estimated, so a typed fibre value makes it Jay's.
    function aiDriven() {
      return !match && aiEst != null && AI_FIELDS.every(function (k) { return aiKeys[k]; }) && nut.fiber_g.value.trim() === '';
    }
    li._aiKcal = function () {
      if (!aiDriven()) return undefined;
      var k = macroVal('kcal');
      return k == null || isNaN(k) ? null : k;
    };
    function refreshTags() {
      tags.textContent = '';
      if (it) {
        tags.appendChild(tag('Identified' + (it.visible === false ? ' (probably hidden)' : '')));
        if (it.confidence < 0.5 || it.visible === false) tags.appendChild(tag('Uncertain', true));
      }
      var g = gramsNow();
      if (!(g > 0)) tags.appendChild(tag('Enter grams', true));
      else if (est != null && portion === 1 && g === round1(est)) tags.appendChild(tag('AI estimate'));
      else tags.appendChild(tag('You confirmed grams'));
      if (match) tags.appendChild(tag('Nutrition from ' + match.source + ': ' + match.name));
      else if (aiDriven()) tags.appendChild(tag('Nutrition: AI estimate'));
      else if (aiEst) tags.appendChild(tag('Nutrition: yours (edited from an AI estimate)'));
      else if (FIELDS.some(function (f) { return nut[f[0]].value !== ''; })) tags.appendChild(tag('Nutrition: yours'));
      else tags.appendChild(tag('No nutrition yet', true));
    }
    function updateSteppers() {
      var g = gramsNow();
      var lessOff = g > 0 ? g - STEP <= 0 : true;
      var moreOff = g > 0 ? g + STEP > MAX_GRAMS : false;
      [[less, lessOff], [more, moreOff]].forEach(function (p) {
        if (p[1] && document.activeElement === p[0]) gi.focus(); // a disabled button would drop focus
        p[0].disabled = p[1];
      });
    }
    function refresh(quiet) {
      followGrams(); refreshTags(); updateSteppers(); updateTotal();
      if (!quiet) speak();
    }
    function speak() {
      var g = gramsNow();
      if (!(g > 0)) return;
      var k = macroVal('kcal');
      announce(label() + ': ' + num(g) + ' grams' + (k != null && !isNaN(k) ? ', ' + num(k) + ' kcal' : ''));
    }
    // setShown writes grams the user (or the slider) chose and keeps base in step.
    function setShown(g, fromSlider) {
      if (!(g > 0)) { gi.value = ''; if (!fromSlider) base = null; return; }
      gi.value = num(g);
      if (!fromSlider) base = g / portion;
    }
    li._applyPortion = function () {
      if (base == null) return;
      captureNow();
      gi.value = num(round1(base * portion));
      refresh(true);
    };
    function typed(g) { base = g > 0 ? g / portion : null; }

    gi.addEventListener('input', function () { typed(gramsNow()); refresh(false); });
    gi.addEventListener('keydown', function (ev) {
      if (ev.key !== 'ArrowUp' && ev.key !== 'ArrowDown') return;
      ev.preventDefault();
      nudge(ev.key === 'ArrowUp' ? STEP : -STEP);
    });
    function nudge(d) {
      var g = gramsNow();
      if (!(g > 0)) return;
      var n = round1(g + d);
      if (n <= 0 || n > MAX_GRAMS) return;
      captureNow();
      setShown(n, false);
      refresh(false);
    }
    less.addEventListener('click', function () { nudge(-STEP); });
    more.addEventListener('click', function () { nudge(STEP); });
    name.input.addEventListener('input', relabel);

    var actions = el('div', 'hv3-cam-actions');
    var swapBtn = el('button', 'hv3-btn small secondary', 'Swap food');
    swapBtn.type = 'button'; swapBtn.appendChild(el('span', 'hv3-sr', ' '));
    var rm = el('button', 'hv3-btn small secondary', 'Remove');
    rm.type = 'button'; rm.appendChild(el('span', 'hv3-sr', ' '));
    actions.appendChild(swapBtn); actions.appendChild(rm);
    swapBtn.addEventListener('click', function () { openFind(li, swapBtn); });
    rm.addEventListener('click', function () {
      var n = label();
      var next = li.nextElementSibling || li.previousElementSibling;
      if (findTarget === li) closeFind(); // its Swap chooser must not outlive the row
      li.remove();
      announce('Removed ' + n);
      updateTotal();
      var target = next && next._focus ? next : null;
      if (target) target._focus(); else addBtn.focus();
    });
    // swapTo replaces the food, keeping the grams and the photo item (idx), so
    // the AI's own grams estimate stays attached to this row.
    li._swapTo = function (m) {
      match = m; ratio = {}; aiEst = null; aiKeys = {}; // a database food replaces the estimate
      name.input.value = m.name;
      buildMacros(); relabel(); refresh(true);
      announce('Swapped to ' + m.name);
    };
    li._read = function () {
      var out = {idx: li._idx, food_ref: match ? match.food_ref : '', name: name.input.value.trim(), grams: parseNum(gi.value)};
      FIELDS.forEach(function (f) { var v = macroVal(f[0]); out[f[0]] = v; });
      return out;
    };
    li._focus = function () { gi.focus(); };

    li.appendChild(tags); li.appendChild(grid); li.appendChild(gramsWrap); li.appendChild(macroBox); li.appendChild(actions);
    gi.value = it && it.ask_grams ? '' : num(est);
    base = gramsNow() > 0 ? gramsNow() / portion : null;
    if (base != null && portion !== 1) { gi.value = num(round1(base * portion)); }
    buildMacros(); relabel(); refresh(true);
    list.appendChild(li);
    updateTotal();
    return li;
  }

  function updateTotal() {
    var kcal = 0, prot = 0, anyK = false, anyP = false, missK = false;
    var aiItems = 0, aiKcal = 0, aiAnyK = false;
    rowEls().forEach(function (li) {
      var a = li._aiKcal();
      if (a !== undefined) { aiItems++; if (a != null) { aiKcal += a; aiAnyK = true; } }
      var k = li._macro('kcal'), p = li._macro('protein_g');
      if (k == null || isNaN(k)) missK = true; else { kcal += k; anyK = true; }
      if (p != null && !isNaN(p)) { prot += p; anyP = true; }
    });
    var t = 'Meal: ' + (anyK ? Math.round(kcal) + ' kcal' : 'no kcal yet');
    if (anyP) t += ', ' + num(prot) + ' g protein';
    if (anyK && missK) t += ' (plus foods with no kcal)';
    // GYST-180: same sentence, same rounding, as the day total after saving.
    if (aiItems > 0) t += ' · ' + M.estimateNote(aiItems, aiAnyK ? aiKcal : null);
    totalEl.textContent = t;
  }

  // Whole-meal portion: x0.5 to x2 in 5% steps. Each row keeps its own base
  // grams, so going back to x1 restores what was there before the slider.
  function setPortion(v) {
    portion = v;
    portionEl.value = String(v);
    portionOut.textContent = 'x' + (Math.round(v * 100) / 100).toFixed(2).replace(/0$/, '');
    var pct = Math.round((v - 1) * 100);
    portionEl.setAttribute('aria-valuetext', 'x' + portionOut.textContent.slice(1) + (pct === 0 ? ', no change' : ', ' + Math.abs(pct) + ' percent ' + (pct > 0 ? 'more' : 'less')));
    rowEls().forEach(function (li) { li._applyPortion(); });
    updateTotal();
  }
  portionEl.addEventListener('input', function () {
    setPortion(parseFloat(portionEl.value));
    announce('Portion ' + portionOut.textContent);
  });
  portionReset.addEventListener('click', function () { setPortion(1); announce('Portion x1.0'); });

  // Swap / Add chooser over the catalogue search (POST, so the text never
  // sits in a URL). Result text goes in with textContent only.
  var findTarget = null, findTrigger = null, findTimer = 0, findReq = M.latest();
  function openFind(row, trigger) {
    clearTimeout(findTimer); findReq.invalidate(); // answers for the previous row or mode are stale
    findTarget = row; findTrigger = trigger;
    findOwn.hidden = !!row;
    findQ.value = ''; findList.textContent = ''; findNote.textContent = '';
    find.hidden = false;
    find.scrollIntoView({block: 'nearest'});
    findQ.focus();
  }
  function closeFind() {
    clearTimeout(findTimer); findReq.invalidate();
    if (find.hidden) return;
    find.hidden = true;
    var t = findTrigger; findTrigger = null; findTarget = null;
    if (t && document.body.contains(t)) t.focus();
  }
  function runFind() {
    var q = findQ.value.trim();
    var seq = findReq.next();
    findList.textContent = '';
    if (!q) { findNote.textContent = ''; return; }
    if (q.length > 80) { findNote.textContent = 'Use 80 characters or fewer.'; return; }
    findNote.textContent = 'Searching…';
    fetch('/health/nutrition/photo/foods', {
      method: 'POST', credentials: 'same-origin',
      headers: {'Content-Type': 'application/json'}, body: JSON.stringify({q: q})
    }).then(function (res) {
      if (!res.ok) throw new Error('search');
      return res.json();
    }).then(function (items) {
      if (!findReq.isCurrent(seq)) return;
      findList.textContent = '';
      findNote.textContent = items.length ? items.length + (items.length === 1 ? ' food found' : ' foods found') : 'No match. Try another word.';
      items.forEach(function (m) {
        var li = document.createElement('li');
        var b = el('button', 'hv3-cam-result');
        b.type = 'button';
        b.appendChild(el('span', 'hv3-strong', m.name));
        b.appendChild(el('span', 'hv3-s', m.source));
        b.addEventListener('click', function () { choose(m); });
        li.appendChild(b); findList.appendChild(li);
      });
    }).catch(function () {
      if (findReq.isCurrent(seq)) findNote.textContent = 'Search is not available. Nothing was changed.';
    });
  }
  function choose(m) {
    var target = findTarget;
    if (target) {
      target._swapTo(m);
      var t = findTrigger; findTrigger = null; findTarget = null; find.hidden = true; if (t) t.focus();
      return;
    }
    findTrigger = null; find.hidden = true;
    var row = addRow(null, m);
    announce('Added ' + m.name + '. Enter grams.');
    row._focus();
  }
  findQ.addEventListener('input', function () {
    // A changed query drops the old results and any answer still on its way.
    clearTimeout(findTimer); findReq.invalidate();
    findList.textContent = ''; findNote.textContent = '';
    findTimer = setTimeout(runFind, 250);
  });
  find.addEventListener('keydown', function (ev) { if (ev.key === 'Escape') { ev.preventDefault(); closeFind(); } });
  findClose.addEventListener('click', closeFind);
  findOwn.addEventListener('click', function () {
    findTrigger = null; find.hidden = true;
    var row = addRow(null); updateTotal(); row._focus();
    announce('Added a blank food. Enter its name, grams and nutrition.');
  });

  function render(data) {
    closeFind(); // a new photo replaces the review; its old chooser goes with it
    nonce = data.nonce;
    setStatus('');
    notes.textContent = '';
    (data.questions || []).concat(data.uncertainties || []).forEach(function (q) { notes.appendChild(el('li', '', q)); });
    list.textContent = '';
    setPortion(1);
    (data.items || []).forEach(function (it) { addRow(it); });
    updateTotal();
    if (!(data.items || []).length) {
      setStatus('No food was found in that photo. Add it by hand below, or take another photo.');
    }
    show(body, true);
  }

  addBtn.addEventListener('click', function () { openFind(null, addBtn); });
  cancelBtn.addEventListener('click', reset);

  form.addEventListener('submit', function (ev) {
    ev.preventDefault();
    if (busy) return;
    var rows = rowEls().map(function (li) { return li._read(); });
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
      body: JSON.stringify({nonce: nonce, slot: slot.value, portion: portion, items: rows})
    }).then(function (res) {
      return res.json().catch(function () { return {}; }).then(function (data) {
        if (res.status === 409) { setStatus('Still saving… wait a moment, then press Save meal again if the page does not move on.'); return; }
        if (res.ok && data.redirect) { window.location.href = data.redirect; return; }
        setError(data.error || 'Could not save the meal. Nothing was saved. Try again.');
      });
    }).catch(function () {
      setError('Could not reach GYST. Nothing was saved. Try again.');
    }).then(function () { busy = false; saveBtn.disabled = false; });
  });
})();
