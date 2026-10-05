/* GYST Health v3 prototype — NON-PRODUCTION. Fictional data only; no network, no backend, no LLM.
   Scenarios (?data=full|partial|empty|long) exist to review empty, partial and overflow states. */
(() => {
  'use strict';

  /* ---------- Fictional scenarios ---------- */
  const FOODS = [
    { id: 'pp', name: 'Protein powder', serving: '1 scoop (30 g)', kcal: 120, p: 24, c: 3, f: 1.5, fib: 1, notes: 'Vanilla. Mix with water or milk.' },
    { id: 'wr', name: 'High-protein wrap', serving: '1 wrap (62 g)', kcal: 160, p: 12, c: 18, f: 4, fib: 9, notes: '' },
    { id: 'gy', name: 'Greek yogurt', serving: '175 g pot', kcal: 105, p: 18, c: 6, f: 0.5, fib: 0, notes: '0% fat, plain.' },
    { id: 'tb', name: 'Turkey breast', serving: '100 g', kcal: 110, p: 24, c: 1, f: 1, fib: 0, notes: 'Sliced, deli counter.' },
    { id: 'ps', name: 'Protein shake', serving: '1 bottle (330 ml)', kcal: 150, p: 25, c: 9, f: 1.5, fib: 2, notes: 'Ready to drink.' }
  ];
  const CHECKS = [
    { id: 'mv', name: 'Multivitamin', done: true },
    { id: 'vc', name: 'Vitamin C', done: true },
    { id: 'mg', name: 'Magnesium', done: false },
    { id: 'cr', name: 'Creatine', done: true }
  ];
  const BODY_GROUPS = {
    Legs: ['quadriceps', 'hamstrings', 'glutes', 'calves', 'adductors', 'abductors'],
    Back: ['lats', 'upper_back', 'lower_back', 'traps'],
    Chest: ['chest'], Shoulders: ['shoulders'], Arms: ['biceps', 'triceps', 'forearms'], Core: ['abdominals']
  };
  const weightSeries = (start, end, n) => Array.from({ length: n }, (_, i) => +(start + (end - start) * (i / (n - 1)) + (i % 3 === 1 ? 0.3 : i % 4 === 2 ? -0.2 : 0)).toFixed(1));

  const full = () => ({
    plan: { title: 'Fat loss', day: 13, days: 182, start: 'Sep 22', end: 'Mar 22' },
    targets: { kcal: 2290, protein: 190, carbs: 220, fat: 70, fiber: 30, steps: 10000, workouts: 3, water: 3.0 },
    food: { kcal: 1450, protein: 132, carbs: 140, fat: 48, fiber: 18 },
    energy: { active: 420, resting: 1450 },
    steps: { watch: 7840, phone: 6920 },
    exercise: { min: 46, week: [30, 0, 52, 0, 0, 38, 46] },
    water: 1.8,
    weight: { start: 95.5, startLabel: 'Sep 22', series: weightSeries(95.5, 92.4, 12), lastLabel: 'Today' },
    sync: { apple: '6:40 PM', stale: false },
    meals: [['Breakfast', 420], ['Lunch', 610], ['Snacks', 420]],
    hevy: {
      connected: true, synced: '7:05 PM', doneOffsets: [0, 3], volume: 14820, timeMin: 92,
      progress: [
        { name: 'Leg Press', pct: { 4: 6, 12: 18, 26: 31 }, best: '150 kg × 10', base: 110 },
        { name: 'Lat Pulldown', pct: { 4: 4, 12: 11, 26: 19 }, best: '62.5 kg × 10', base: 50 },
        { name: 'Chest Press', pct: { 4: 3, 12: 8, 26: 14 }, best: '55 kg × 8', base: 45 },
        { name: 'Romanian Deadlift', pct: { 4: 5, 12: 9, 26: 16 }, best: '80 kg × 8', base: 62 },
        { name: 'Seated Cable Row', pct: { 4: 0, 12: 6, 26: 12 }, best: '57.5 kg × 10', base: 48 }
      ],
      focus: { Legs: 35, Back: 25, Chest: 20, Shoulders: 10, Arms: 10, Core: 0 },
      muscleSets: [['Quadriceps', 24, 6], ['Lats', 16, 4], ['Chest', 16, 3], ['Glutes', 12, 14], ['Hamstrings', 10, 8], ['Shoulders', 8, 12], ['Biceps', 4, 10], ['Triceps', 4, 9]],
      unmapped: 2,
      latest: { title: 'Full Body A – Fat Loss', min: 45, when: 'Today', exercises: [['Leg Press', '3 × 10 · 150 kg'], ['Lat Pulldown', '3 × 10 · 62.5 kg'], ['Chest Press', '3 × 8 · 55 kg'], ['Romanian Deadlift', '3 × 8 · 80 kg'], ['Plank', '3 × 45 s']] },
      weeks: [2, 3, 1, 3, 2, 3, 3, 0, 2, 3, 3, 2, 3, 3, 2, 1, 3, 3, 2, 3, 3, 2, 3, 3, 3, 2]
    },
    checks: CHECKS.map((c) => ({ ...c })),
    foods: FOODS.map((f) => ({ ...f }))
  });

  const SCENARIOS = {
    full,
    partial: () => {
      const d = full();
      d.food = null; d.energy = { active: 180, resting: null }; d.steps = { watch: 2140, phone: null };
      d.exercise = { min: 0, week: [30, 0, 52, 0, 0, 38, 0] }; d.water = null; d.meals = null;
      d.weight = { start: 92.4, startLabel: 'Today', series: [92.4], lastLabel: 'Today' };
      d.hevy = { ...d.hevy, doneOffsets: [3], volume: 5120, timeMin: 44, progress: [], latest: { ...d.hevy.latest, when: 'Thursday' }, weeks: [0, 0, 0, 1] };
      return d;
    },
    empty: () => ({
      plan: null, targets: null, food: null, energy: null, steps: null, exercise: null, water: null, weight: null,
      sync: null, meals: null, hevy: { connected: false }, checks: [], foods: []
    }),
    long: () => {
      const d = full();
      d.food = { kcal: 12345, protein: 1234, carbs: 2210, fat: 970, fiber: 305 };
      d.energy = { active: 12450, resting: 2100 }; d.steps = { watch: 123456, phone: 98765 };
      d.exercise = { min: 1234, week: [300, 0, 520, 0, 0, 380, 1234] };
      d.weight = { start: 199.9, startLabel: 'Sep 22', series: weightSeries(199.9, 188.8, 12), lastLabel: 'Today' };
      d.plan.title = 'Six-month fat loss with progressive strength and daily walking';
      d.hevy.latest.title = 'Full Body A – Fat Loss with Extended Posterior Chain Accessory Work';
      d.hevy.progress[0].name = 'Single-arm Cable Lateral Raise (Behind the Back)';
      d.hevy.volume = 1234567; d.hevy.timeMin = 1234;
      d.foods[0].name = 'Unflavoured whey isolate protein powder, extra-large tub (2.27 kg)';
      d.sync.stale = true;
      return d;
    }
  };

  /* ---------- Prototype settings (URL first, then per-device storage) ---------- */
  const params = new URLSearchParams(location.search);
  const store = {
    get(k) { try { return localStorage.getItem('hv3-' + k); } catch { return null; } },
    set(k, v) { try { localStorage.setItem('hv3-' + k, v); } catch { /* storage unavailable */ } }
  };
  let scenario = params.get('data') || store.get('data') || 'full';
  if (!SCENARIOS[scenario]) scenario = 'full';
  let theme = params.get('theme') || store.get('theme') || 'system';
  let D = SCENARIOS[scenario]();
  const page = document.body.dataset.page || 'today';

  function applyTheme() {
    if (theme === 'light' || theme === 'dark') document.documentElement.dataset.theme = theme;
    else delete document.documentElement.dataset.theme;
  }
  applyTheme();

  /* ---------- Icons (Lucide-style line icons, drawn inline) ---------- */
  const ICONS = {
    today: '<circle cx="12" cy="12" r="4"/><path d="M12 2v2M12 20v2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M2 12h2M20 12h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4"/>',
    dumbbell: '<path d="M6.5 6.5v11M17.5 6.5v11M3 9.5v5M21 9.5v5M6.5 12h11"/>',
    apple: '<path d="M12 6.5c-1.5-1-4.5-1.2-6 1-1.8 2.6-.6 7.6 1.6 10.2 1.2 1.4 2.4 1.6 3.4 1 .6-.3 1.4-.3 2 0 1 .6 2.2.4 3.4-1 2.2-2.6 3.4-7.6 1.6-10.2-1.5-2.2-4.5-2-6-1Z"/><path d="M12 6.5c0-2 1-3.5 3-4"/>',
    chat: '<path d="M21 12a8 8 0 0 1-11.8 7L4 20.5l1.5-4.8A8 8 0 1 1 21 12Z"/>',
    more: '<circle cx="5" cy="12" r="1.2"/><circle cx="12" cy="12" r="1.2"/><circle cx="19" cy="12" r="1.2"/>',
    chevR: '<path d="m9 6 6 6-6 6"/>', chevL: '<path d="m15 6-6 6 6 6"/>',
    flame: '<path d="M12 3c1 3.5 5 5.5 5 10a5 5 0 0 1-10 0c0-2 1-3.5 2-4.5 0 2 1 3 2 3 0-3-1-5 1-8.5Z"/>',
    steps: '<path d="M8 4c1.7 0 2.5 1.8 2.5 4S9.5 12 8 12 5.5 10.2 5.5 8 6.3 4 8 4ZM6 15h4v2a2 2 0 0 1-4 0v-2ZM16 8c1.7 0 2.5 1.8 2.5 4s-1 4-2.5 4-2.5-1.8-2.5-4S14.3 8 16 8ZM14 19h4v.5a2 2 0 0 1-4 0V19Z"/>',
    timer: '<circle cx="12" cy="13" r="8"/><path d="M12 9v4l2.5 2M9 2h6"/>',
    scale: '<rect x="3" y="4" width="18" height="16" rx="4"/><path d="M8 9.5a5 5 0 0 1 8 0L13 12"/>',
    plus: '<path d="M12 5v14M5 12h14"/>', x: '<path d="M6 6l12 12M18 6 6 18"/>',
    check: '<path d="m5 12.5 4.5 4.5L19 7.5"/>', send: '<path d="M12 19V5M5.5 11.5 12 5l6.5 6.5"/>',
    target: '<circle cx="12" cy="12" r="8"/><circle cx="12" cy="12" r="4"/><circle cx="12" cy="12" r=".8"/>',
    list: '<path d="M9 6h11M9 12h11M9 18h11"/><path d="m3.5 6 1 1 2-2M3.5 12l1 1 2-2M3.5 18l1 1 2-2"/>',
    box: '<path d="M21 8 12 3 3 8v8l9 5 9-5V8Z"/><path d="m3 8 9 5 9-5M12 13v8"/>',
    link: '<path d="M10 14a4 4 0 0 0 5.7 0l3-3a4 4 0 0 0-5.7-5.7l-1 1"/><path d="M14 10a4 4 0 0 0-5.7 0l-3 3a4 4 0 0 0 5.7 5.7l1-1"/>',
    ext: '<path d="M14 4h6v6M20 4l-9 9M18 14v5a1 1 0 0 1-1 1H5a1 1 0 0 1-1-1V7a1 1 0 0 1 1-1h5"/>',
    info: '<circle cx="12" cy="12" r="9"/><path d="M12 11v5M12 8h.01"/>',
    refresh: '<path d="M20 11a8 8 0 0 0-14.5-4.5L4 8M4 4v4h4M4 13a8 8 0 0 0 14.5 4.5L20 16M20 20v-4h-4"/>',
    up: '<path d="M7 17 17 7M9 7h8v8"/>', down: '<path d="M7 7l10 10M17 9v8H9"/>',
    drop: '<path d="M12 3s6 6.5 6 11a6 6 0 0 1-12 0c0-4.5 6-11 6-11Z"/>',
    sparkle: '<path d="M12 3v4M12 17v4M3 12h4M17 12h4M6 6l2.5 2.5M15.5 15.5 18 18M6 18l2.5-2.5M15.5 8.5 18 6"/>',
    home: '<path d="M4 11 12 4l8 7v8a1 1 0 0 1-1 1h-5v-6h-4v6H5a1 1 0 0 1-1-1v-8Z"/>',
    folder: '<path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V7Z"/>',
    heart: '<path d="M12 20s-7-4.4-7-10a4 4 0 0 1 7-2.6A4 4 0 0 1 19 10c0 5.6-7 10-7 10Z"/>',
    book: '<path d="M4 5a2 2 0 0 1 2-2h13v16H6a2 2 0 0 0-2 2V5Z"/><path d="M4 19a2 2 0 0 1 2-2h13"/>',
    menu: '<path d="M4 7h16M4 12h16M4 17h16"/>',
    cpu: '<rect x="6" y="6" width="12" height="12" rx="2"/><path d="M9 2v4M15 2v4M9 18v4M15 18v4M2 9h4M2 15h4M18 9h4M18 15h4"/>',
    shield: '<path d="M12 3 5 6v6c0 4 3 7 7 9 4-2 7-5 7-9V6l-7-3Z"/>'
  };
  const ic = (name, cls = '') => `<svg class="i ${cls}" aria-hidden="true" viewBox="0 0 24 24">${ICONS[name] || ''}</svg>`;
  document.querySelectorAll('[data-icon]').forEach((el) => { el.insertAdjacentHTML('afterbegin', ic(el.dataset.icon, el.dataset.iconClass || '')); });

  /* ---------- Helpers ---------- */
  const esc = (s) => String(s).replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
  const n = (v, digits = 0) => v == null ? '—' : Number(v).toLocaleString('en-US', { minimumFractionDigits: digits, maximumFractionDigits: digits });
  const signed = (v, digits = 0) => (v > 0 ? '+' : v < 0 ? '−' : '') + n(Math.abs(v), digits);
  const LIMIT = { carbs: true, fat: true };
  const macroState = (k, v, tg) => v == null || !tg ? '' : LIMIT[k] ? (v > tg ? 'warn' : '') : (v >= tg ? 'ok' : '');
  const pct = (v, t) => (v == null || !t) ? null : Math.max(0, Math.min(100, (v / t) * 100));
  const minutes = (m) => m == null ? '—' : (m >= 60 ? `${Math.floor(m / 60)} h ${m % 60} min` : `${m} min`);
  const $ = (sel, root = document) => root.querySelector(sel);
  const view = $('#view');

  function ring({ value, size = 56, sw = 6, cls = '', label, center = '' }) {
    const r = (size - sw) / 2, c = 2 * Math.PI * r;
    const missing = value == null;
    const off = missing ? c : c * (1 - value / 100);
    return `<div class="ring ${cls}" style="width:${size}px;height:${size}px" role="img" aria-label="${esc(label)}">
      <svg width="${size}" height="${size}" aria-hidden="true"><circle class="track${missing ? ' dashed' : ''}" cx="${size / 2}" cy="${size / 2}" r="${r}" stroke-width="${sw}"/>${missing ? '' : `<circle class="val" cx="${size / 2}" cy="${size / 2}" r="${r}" stroke-width="${sw}" stroke-dasharray="${c.toFixed(2)}" stroke-dashoffset="${off.toFixed(2)}"/>`}</svg>
      ${center ? `<div class="center">${center}</div>` : ''}</div>`;
  }
  function spark(vals, h = 40, label = '') {
    if (!vals || vals.length < 2) return '';
    const w = 100, lo = Math.min(...vals), hi = Math.max(...vals), span = hi - lo || 1;
    const pts = vals.map((v, i) => [(i / (vals.length - 1)) * w, h - 4 - ((v - lo) / span) * (h - 8)]);
    const line = pts.map((p) => p.map((x) => x.toFixed(1)).join(',')).join(' ');
    return `<svg class="spark" viewBox="0 0 ${w} ${h}" preserveAspectRatio="none" style="height:${h}px" ${label ? `role="img" aria-label="${esc(label)}"` : 'aria-hidden="true"'}>
      <polygon class="area" points="0,${h} ${line} ${w},${h}"/><polyline class="ln" points="${line}"/></svg>`;
  }
  const progSeries = (base, pctUp, len) => Array.from({ length: len }, (_, i) => base * (1 + (pctUp / 100) * (i / (len - 1))) + (i % 3 === 1 ? -base * 0.015 : 0));

  /* ---------- Sheet, toast ---------- */
  const sheet = $('#sheet');
  let lastFocus = null;
  function openSheet(title, body, onReady) {
    lastFocus = document.activeElement;
    sheet.setAttribute('aria-labelledby', 'sheet-title');
    sheet.innerHTML = `<div class="sheet-in"><span class="grab" aria-hidden="true"></span>
      <div class="sheet-h"><h2 id="sheet-title">${esc(title)}</h2><button class="icon-btn quiet" type="button" data-close aria-label="Close">${ic('x')}</button></div>${body}</div>`;
    sheet.showModal();
    if (onReady) onReady(sheet);
  }
  sheet.addEventListener('click', (e) => { if (e.target === sheet || e.target.closest('[data-close]')) sheet.close(); });
  sheet.addEventListener('close', () => { if (lastFocus && lastFocus.focus) lastFocus.focus(); });
  let toastTimer;
  function toast(msg) {
    const t = $('#toast');
    t.textContent = msg; t.hidden = false;
    clearTimeout(toastTimer); toastTimer = setTimeout(() => { t.hidden = true; }, 2600);
  }

  /* ---------- Shared bits ---------- */
  function greeting() {
    const h = new Date().getHours();
    return h < 12 ? 'Good morning' : h < 18 ? 'Good afternoon' : 'Good evening';
  }
  const dateLabel = () => new Date().toLocaleDateString('en-US', { weekday: 'long', month: 'long', day: 'numeric' });
  function header(title, eyebrow, extra = '', trailing = '') {
    return `<header class="ph"><div class="copy">${eyebrow ? `<p class="eyebrow">${eyebrow}</p>` : ''}<h1>${title}</h1>${extra}</div>${trailing}</header>`;
  }
  function protoBar() {
    const opt = (v, l, cur) => `<option value="${v}"${v === cur ? ' selected' : ''}>${l}</option>`;
    return `<aside class="proto" aria-label="Prototype controls"><b>Prototype</b><span>fictional data</span>
      <label>Data <select data-proto="data">${opt('full', 'Full day', scenario)}${opt('partial', 'Partial', scenario)}${opt('empty', 'Empty', scenario)}${opt('long', 'Long values', scenario)}</select></label>
      <label>Theme <select data-proto="theme">${opt('system', 'System', theme)}${opt('light', 'Light', theme)}${opt('dark', 'Dark', theme)}</select></label></aside>`;
  }
  function weightCard() {
    const w = D.weight;
    if (!w) {
      return `<section class="card stat" aria-labelledby="w-h"><p class="card-k" id="w-h">${ic('scale', 'i-16')}Weight</p>
        <p class="card-v missing">—</p><p class="card-s">No readings yet</p>
        <div class="card-actions"><span></span><button class="btn small secondary" type="button" data-act="log-weight">${ic('plus', 'i-16')}Log weight</button></div></section>`;
    }
    const cur = w.series[w.series.length - 1], diff = +(cur - w.start).toFixed(1);
    return `<section class="card stat" aria-labelledby="w-h"><p class="card-k" id="w-h">${ic('scale', 'i-16')}Weight</p>
      <div class="weight"><div><p class="card-v num">${n(cur, 1)} <small>kg</small></p>
        <p class="delta">${w.series.length > 1 ? `${ic(diff <= 0 ? 'down' : 'up', 'i-16')}${signed(diff, 1)} kg since ${esc(w.startLabel)}` : 'First reading'}</p></div>
        <div>${spark(w.series, 40, `Weight trend, ${n(w.start, 1)} to ${n(cur, 1)} kg`)}</div></div>
      <div class="card-actions"><span class="card-s">${esc(w.lastLabel)}</span><button class="btn small secondary" type="button" data-act="log-weight">${ic('plus', 'i-16')}Log weight</button></div></section>`;
  }
  function macroTiles() {
    const t = D.targets || {}, f = D.food || {};
    const m = [['Protein', 'protein'], ['Carbs', 'carbs'], ['Fat', 'fat'], ['Fiber', 'fiber']];
    return m.map(([label, k]) => {
      const v = f[k], tg = t[k], p = pct(v, tg);
      const st = macroState(k, v, tg);
      const amt = v == null ? '<span class="missing">—</span>' : `${n(v)}${tg ? ` <small>/ ${n(tg)} g</small>` : ' <small>g</small>'}`;
      return `<div class="card macro">${tg ? ring({ value: p, size: 44, sw: 5, cls: st, label: v == null ? `${label}: not received` : `${label}: ${n(v)} of ${n(tg)} grams${st === 'warn' ? ', over limit' : st === 'ok' ? ', target met' : ''}` }) : ''}
        <div class="copy"><span class="name">${label}</span><span class="amt num">${amt}</span></div></div>`;
    }).join('');
  }

  /* ---------- Today ---------- */
  function renderToday() {
    const t = D.targets, f = D.food, e = D.energy;
    let status;
    if (!D.sync) status = `<span class="status-line"><span class="dot"></span>No health data connected yet</span>`;
    else if (!f) status = `<span class="status-line"><span class="dot"></span>Waiting for today's food</span>`;
    else if (t && f.kcal > t.kcal) status = `<span class="status-line"><span class="dot warn"></span>Over your calorie target</span>`;
    else status = `<span class="status-line"><span class="dot ok"></span>On track so far</span>`;

    let hero;
    if (!D.sync) {
      hero = `<section class="card hero empty" aria-labelledby="cal-h"><span class="tile">${ic('link')}</span><h2 class="title" id="cal-h">Connect your data</h2>
        <p>Food comes from MyFitnessPal and activity from Apple Health.</p><a class="btn" href="settings.html#sources">Set up sources</a></section>`;
    } else {
      const left = t && f ? t.kcal - f.kcal : null, over = left != null && left < 0;
      const balance = f && e && e.resting != null && e.active != null ? f.kcal - (e.resting + e.active) : null;
      const center = !f ? `<span class="big missing">—</span><span class="big-l">No food logged yet</span>`
        : t ? `<span class="big num">${n(Math.abs(left))}</span><span class="big-l">${over ? 'kcal over' : 'kcal left'}</span>`
          : `<span class="big num">${n(f.kcal)}</span><span class="big-l">kcal eaten</span>`;
      hero = `<section class="card hero" aria-labelledby="cal-h"><h2 class="sr" id="cal-h">Calories</h2><div class="cal">
        ${ring({ value: f && t ? pct(f.kcal, t.kcal) : null, size: 184, sw: 14, cls: over ? 'warn' : '', label: !f ? 'Calories: no food logged yet' : t ? `${n(f.kcal)} of ${n(t.kcal)} calories eaten` : `${n(f.kcal)} calories eaten, no target set`, center })}
        <dl class="cal-stats">
          <div><dt>Eaten</dt><dd class="num">${f ? n(f.kcal) : '<span class="missing">—</span>'}</dd></div>
          <div><dt>Target</dt><dd class="num">${t ? n(t.kcal) : '<span class="missing">Not set</span>'}</dd></div>
          <div><dt>Active</dt><dd class="num">${e && e.active != null ? n(e.active) : '<span class="missing">—</span>'}</dd></div>
        </dl>
        <div class="balance"><div class="l"><b>Energy balance so far</b><span>${balance == null ? (f ? 'Waiting for resting energy' : 'Needs food and energy data') : 'Observed today, not a final deficit'}</span></div>
          <div style="display:flex;align-items:center;gap:4px"><span class="v num">${balance == null ? '<span class="missing">—</span>' : `${signed(balance)} kcal`}</span>
          <button class="icon-btn quiet" type="button" data-act="energy-info" aria-label="How energy balance is calculated">${ic('info')}</button></div></div>
      </div></section>`;
    }

    const s = D.steps, x = D.exercise;
    const activity = !D.sync ? '' : `<section class="section span" aria-labelledby="act-h"><div class="section-h"><h2 id="act-h">Activity</h2></div><div class="grid-2">
      <div class="card stat"><p class="card-k">${ic('steps', 'i-16')}Steps</p>
        <p class="card-v num">${s && s.watch != null ? n(s.watch) : '<span class="missing">—</span>'}${s && s.watch != null ? ' <small>Watch</small>' : ''}</p>
        ${D.targets && s && s.watch != null ? `<div class="bar${s.watch >= D.targets.steps ? ' ok' : ''}" role="img" aria-label="${n(s.watch)} of ${n(D.targets.steps)} steps"><i style="--v:${pct(s.watch, D.targets.steps)}%"></i></div>` : ''}
        <p class="card-s">${s && s.phone != null ? `Phone ${n(s.phone)} · counted separately` : 'Phone · not received'}</p></div>
      <div class="card stat"><p class="card-k">${ic('timer', 'i-16')}Exercise</p>
        <p class="card-v num">${x ? n(x.min) : '<span class="missing">—</span>'} <small>min</small></p>
        ${x ? `<div class="mini-bars" role="img" aria-label="Exercise minutes, last 7 days: ${x.week.join(', ')}">${x.week.map((v, i) => `<span class="${v ? 'on' : ''}${i === 6 ? ' today' : ''}" style="height:${Math.max(8, (v / Math.max(...x.week, 1)) * 100)}%"></span>`).join('')}</div>` : ''}
        <p class="card-s">${x && x.min === 0 ? 'Recorded zero today' : 'Last 7 days'}</p></div>
    </div></section>`;

    view.innerHTML = `${header(`${greeting()}, Jay`, dateLabel(), status)}
      ${D.sync && D.sync.stale ? `<div class="banner warn" role="status">${ic('refresh')}<span class="grow">Apple Health last synced ${esc(D.sync.apple)}</span><span class="chip warn">Stale</span></div>` : ''}
      <div class="today">
        ${hero}
        ${D.sync ? `<section class="section macros" aria-labelledby="mac-h"><div class="section-h"><h2 id="mac-h">Macros</h2><a href="nutrition.html">Nutrition</a></div><div class="grid-4">${macroTiles()}</div></section>` : ''}
        ${activity}
        <div class="span">${weightCard()}</div>
      </div>${protoBar()}`;
  }

  /* ---------- Workout ---------- */
  let wkView = 'overview', period = 12;
  function bodyMap(focus) {
    const heat = {};
    Object.entries(BODY_GROUPS).forEach(([g, ids]) => { const v = focus[g] || 0; ids.forEach((id) => { heat[id] = v >= 30 ? 3 : v >= 15 ? 2 : v > 0 ? 1 : 0; }); });
    const outline = 'M88 65 L87 76 L67 82 Q55 87 51 103 L39 143 L31 185 Q32 195 39 192 L49 169 L58 141 L66 120 L67 169 L60 198 L65 247 L68 287 L65 322 Q71 332 86 325 L88 286 L95 242 L100 214 L105 242 L112 286 L114 325 Q129 332 135 322 L132 287 L135 247 L140 198 L133 169 L134 120 L142 141 L151 169 L161 192 Q168 195 169 185 L161 143 L149 103 Q145 87 133 82 L113 76 L112 65 Z';
    const front = [['chest', 'M70 92 Q84 86 98 93 L98 119 Q82 126 69 114 Z M102 93 Q116 86 130 92 L131 114 Q118 126 102 119 Z'], ['shoulders', 'M65 85 L77 86 L68 109 L55 113 Q55 93 65 85 Z M123 86 L135 85 Q145 93 145 113 L132 109 Z'], ['biceps', 'M55 115 L67 112 L63 138 L51 150 L47 145 Z M133 112 L145 115 L153 145 L149 150 L137 138 Z'], ['forearms', 'M47 150 L56 148 L47 174 L39 188 L35 185 Z M144 148 L153 150 L165 185 L161 188 L153 174 Z'], ['abdominals', 'M87 126 L98 124 L98 169 L89 180 L82 165 Z M102 124 L113 126 L118 165 L111 180 L102 169 Z'], ['quadriceps', 'M72 190 L88 185 L95 202 L88 241 L82 258 L71 246 L66 209 Z M112 185 L128 190 L134 209 L129 246 L118 258 L112 241 L105 202 Z'], ['adductors', 'M91 184 L99 190 L98 215 L91 239 L93 207 Z M101 190 L109 184 L107 207 L109 239 L102 215 Z'], ['calves', 'M72 265 L84 265 L82 298 L76 314 L71 290 Z M116 265 L128 265 L129 290 L124 314 L118 298 Z']];
    const back = [['shoulders', 'M65 85 L77 86 L68 109 L55 113 Q55 93 65 85 Z M123 86 L135 85 Q145 93 145 113 L132 109 Z'], ['traps', 'M85 80 L100 88 L115 80 L124 91 L100 112 L76 91 Z'], ['upper_back', 'M75 97 L97 114 L97 135 L76 120 Z M103 114 L125 97 L124 120 L103 135 Z'], ['lats', 'M70 120 L82 129 L96 140 L93 163 L79 153 Z M118 129 L130 120 L121 153 L107 163 L104 140 Z'], ['lower_back', 'M88 156 L100 168 L112 156 L119 176 L100 187 L81 176 Z'], ['triceps', 'M55 115 L67 112 L63 138 L51 150 L47 145 Z M133 112 L145 115 L153 145 L149 150 L137 138 Z'], ['glutes', 'M73 180 L98 188 L98 211 Q79 223 66 205 Z M102 188 L127 180 L134 205 Q121 223 102 211 Z'], ['hamstrings', 'M68 214 L92 217 L88 244 L81 259 L70 249 Z M108 217 L132 214 L130 249 L119 259 L112 244 Z'], ['calves', 'M72 262 L84 263 L85 283 L77 301 L71 290 Z M116 263 L128 262 L129 290 L123 301 L115 283 Z']];
    const side = (list, label) => `<text x="100" y="22">${label}</text><ellipse cx="100" cy="48" rx="14" ry="18" class="outline"/><path class="outline" d="${outline}"/>${list.map(([id, d]) => `<path class="m${heat[id] ? ' h' + heat[id] : ''}" d="${d}"/>`).join('')}`;
    return `<svg class="body-map" viewBox="20 0 360 336" role="img" aria-label="Body map: darker areas had more primary work sets. Exact shares are listed alongside."><g>${side(front, 'Front')}</g><g transform="translate(200 0)">${side(back, 'Back')}</g></svg>`;
  }
  function renderWorkout() {
    const h = D.hevy;
    const sync = h.connected ? `<button class="btn small secondary" type="button" data-act="sync">${ic('refresh', 'i-16')}Sync</button>` : '';
    if (!h.connected) {
      view.innerHTML = `${header('Workout', 'Hevy', '', '')}
        <section class="card hero empty" aria-labelledby="hv-h"><span class="tile idle">${ic('dumbbell')}</span><h2 class="title" id="hv-h">Hevy isn't connected</h2>
          <p>Log workouts in Hevy. GYST shows your consistency, focus and progress.</p><a class="btn" href="settings.html#sources">Connect Hevy</a></section>${protoBar()}`;
      return;
    }
    const todayIdx = (new Date().getDay() + 6) % 7;
    const done = new Set(h.doneOffsets.map((o) => todayIdx - o).filter((i) => i >= 0));
    const labels = ['M', 'T', 'W', 'T', 'F', 'S', 'S'], full = ['Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday', 'Sunday'];
    const tgt = D.targets ? D.targets.workouts : null;
    const tabs = `<div class="seg" role="tablist" aria-label="Workout view">
      <button type="button" role="tab" id="tab-overview" aria-controls="panel" aria-selected="${wkView === 'overview'}" tabindex="${wkView === 'overview' ? 0 : -1}">Overview</button>
      <button type="button" role="tab" id="tab-progress" aria-controls="panel" aria-selected="${wkView === 'progress'}" tabindex="${wkView === 'progress' ? 0 : -1}">Progress</button></div>`;

    let panel;
    if (wkView === 'overview') {
      const focus = Object.entries(h.focus).filter(([, v]) => v > 0).sort((a, b) => b[1] - a[1]);
      const prog = h.progress.slice(0, 3);
      panel = `<div class="wk-grid">
        <section class="card week span" aria-labelledby="wk-h"><div class="week-top"><h2 class="card-k" id="wk-h">This week</h2><span class="chip nodot">${esc(D.plan ? 'Target ' + tgt + ' / week' : 'No weekly target')}</span></div>
          <p class="card-v num">${done.size}${tgt ? ` <small>/ ${tgt} workouts</small>` : ' <small>workouts</small>'}</p>
          <ol class="days">${labels.map((l, i) => `<li class="${i === todayIdx ? 'is-today' : ''}"><span class="d${done.has(i) ? ' on' : ''}${i === todayIdx ? ' today' : ''}">${done.has(i) ? ic('check', 'i-16') : ''}</span><span aria-hidden="true">${l}</span><span class="sr">${full[i]}: ${done.has(i) ? 'workout logged' : i > todayIdx ? 'upcoming' : 'no workout'}</span></li>`).join('')}</ol></section>
        <div class="grid-2 span">
          <div class="card stat"><p class="card-k">${ic('dumbbell', 'i-16')}Volume</p><p class="card-v num">${n(h.volume)} <small>kg</small></p><p class="card-s">This week · work sets</p></div>
          <div class="card stat"><p class="card-k">${ic('timer', 'i-16')}Training time</p><p class="card-v num">${minutes(h.timeMin)}</p><p class="card-s">This week</p></div>
        </div>
        <section class="section span" aria-labelledby="tp-h"><div class="section-h"><h2 id="tp-h">Top progress · 12 weeks</h2>${prog.length ? '<button type="button" data-act="to-progress">All</button>' : ''}</div>
          ${prog.length ? `<div class="carousel" role="list">${prog.map((p) => `<div class="card prog" role="listitem"><span class="name">${esc(p.name)}</span>
            <span class="pct num">${signed(p.pct[12])}%<small>best set</small></span>${spark(progSeries(p.base, p.pct[12], 10), 40, `${p.name} trend, up ${p.pct[12]} percent`)}<span class="card-s num">${esc(p.best)}</span></div>`).join('')}</div>`
            : `<div class="card empty"><p class="title">Not enough history yet</p><p>Progress appears after an exercise is logged in two or more sessions.</p></div>`}</section>
        <section class="card focus" aria-labelledby="mf-h"><div class="week-top"><h2 class="card-k" id="mf-h">Muscle focus</h2><span class="card-s">Last 30 days</span></div>
          <div class="focus-body">${bodyMap(h.focus)}<ul class="focus-bars">${focus.map(([g, v]) => `<li><span>${g}</span><b>${v}%</b><div class="bar" aria-hidden="true"><i style="--v:${v}%"></i></div></li>`).join('')}</ul></div>
          <details class="more"><summary>Details ${ic('chevR', 'i-16')}</summary>
            <table class="tbl"><caption class="sr">Work sets by muscle, last 30 days</caption><thead><tr><th scope="col">Muscle</th><th scope="col">Primary sets</th><th scope="col">Secondary</th></tr></thead>
            <tbody>${h.muscleSets.map(([m, p, s]) => `<tr><th scope="row">${m}</th><td>${p}</td><td>${s}</td></tr>`).join('')}</tbody></table>
            <p class="foot" style="padding:8px 0 0">Shares count primary work sets; warmups excluded.${h.unmapped ? ` ${h.unmapped} sets have no muscle mapping.` : ''}</p></details></section>
        <section class="section" aria-labelledby="rw-h"><div class="section-h"><h2 id="rw-h">Latest workout</h2></div>
          <div class="card"><div class="workout-row"><span class="tile">${ic('dumbbell')}</span><div class="copy"><span class="title">${esc(h.latest.title)}</span>
            <span class="meta num">${minutes(h.latest.min)}<span class="sep"></span>${h.latest.exercises.length} exercises<span class="sep"></span>${esc(h.latest.when)}</span></div></div>
            <div class="card-actions"><span class="card-s">From Hevy · ${esc(h.synced)}</span><button class="btn small secondary" type="button" data-act="workout-detail">View details</button></div></div></section>
      </div>`;
    } else {
      const len = period === 4 ? 4 : period === 12 ? 12 : 26;
      const weeks = h.weeks.slice(-len), max = Math.max(...weeks, 1), total = weeks.reduce((a, b) => a + b, 0);
      const per = (p) => `<button type="button" aria-pressed="${period === p}" data-period="${p}">${p === 26 ? '6 mo' : p + ' wk'}</button>`;
      panel = `<div class="wk-grid">
        <div class="seg span" role="group" aria-label="Progress period">${per(4)}${per(12)}${per(26)}</div>
        <section class="card" aria-labelledby="wc-h"><div class="week-top"><h2 class="card-k" id="wc-h">Workouts per week</h2><span class="card-s num">${total} total</span></div>
          <div class="week-chart" role="img" aria-label="Workouts per week over ${len} weeks: ${weeks.join(', ')}" style="margin-top:12px">${weeks.map((v) => `<span class="${v ? '' : 'zero'}" style="height:${Math.max(3, (v / max) * 100)}%"></span>`).join('')}</div>
          <div class="axis"><span>${len} wk ago</span><span>This week</span></div></section>
        <section class="card" aria-labelledby="ex-h"><h2 class="card-k" id="ex-h" style="margin-bottom:8px">Exercise progress</h2>
          ${h.progress.length ? `<ul class="plist">${h.progress.map((p) => `<li><span class="copy"><span class="title">${esc(p.name)}</span><span class="card-s num">${esc(p.best)}</span></span>${spark(progSeries(p.base, p.pct[period], Math.min(len, 12)), 28)}<span class="chg num">${signed(p.pct[period])}%</span></li>`).join('')}</ul>`
            : `<div class="empty"><p class="title">Not enough history yet</p><p>Log an exercise in two or more sessions.</p></div>`}
          <p class="foot" style="padding:12px 0 0">Change in best set (estimated 1-rep max) over the period.</p></section>
      </div>`;
    }
    view.innerHTML = `${header('Workout', 'Hevy · synced ' + esc(h.synced), '', sync)}${tabs}<div id="panel" role="tabpanel" aria-labelledby="tab-${wkView}">${panel}</div>${protoBar()}`;
  }

  /* ---------- Nutrition ---------- */
  function renderNutrition() {
    const t = D.targets, f = D.food;
    if (!D.sync) {
      view.innerHTML = `${header('Nutrition', dateLabel())}
        <section class="card hero empty"><span class="tile idle">${ic('apple')}</span><h2 class="title">No food data yet</h2><p>Log food in MyFitnessPal and share it with Apple Health.</p><a class="btn" href="settings.html#sources">Set up sources</a></section>${protoBar()}`;
      return;
    }
    const over = t && f && f.kcal > t.kcal;
    const hero = `<section class="card hero" aria-labelledby="nc-h"><h2 class="sr" id="nc-h">Calories</h2><div class="cal">
      ${ring({ value: f && t ? pct(f.kcal, t.kcal) : null, size: 168, sw: 14, cls: over ? 'warn' : '', label: f ? `${n(f.kcal)} of ${t ? n(t.kcal) : 'no target'} calories` : 'No food logged yet',
        center: f ? `<span class="big num">${n(f.kcal)}</span><span class="big-l">${t ? `of ${n(t.kcal)} kcal` : 'kcal'}</span>` : `<span class="big missing">—</span><span class="big-l">No food logged yet</span>` })}
      <dl class="cal-stats"><div><dt>Consumed</dt><dd class="num">${f ? n(f.kcal) : '<span class="missing">—</span>'}</dd></div>
        <div><dt>${over ? 'Over' : 'Left'}</dt><dd class="num">${f && t ? n(Math.abs(t.kcal - f.kcal)) : '<span class="missing">—</span>'}</dd></div>
        <div><dt>Target</dt><dd class="num">${t ? n(t.kcal) : '<span class="missing">Not set</span>'}</dd></div></dl></div></section>`;
    const rows = [['Protein', 'protein'], ['Carbs', 'carbs'], ['Fat', 'fat'], ['Fiber', 'fiber']].map(([l, k]) => {
      const v = f ? f[k] : null, tg = t ? t[k] : null, p = pct(v, tg);
      return `<li class="measure"><div class="copy"><div class="top"><span class="title">${l}</span><span class="num">${v == null ? '<span class="missing">— not received</span>' : `<b>${n(v)}</b>${tg ? ` / ${n(tg)} g` : ' g'}`}</span></div>
        ${tg ? `<div class="bar${v == null ? ' missing' : ' ' + macroState(k, v, tg)}" aria-hidden="true"><i style="--v:${p || 0}%"></i></div>` : ''}
        ${v != null && tg ? `<span class="card-s num">${v > tg && LIMIT[k] ? `${n(v - tg)} g over` : v >= tg ? 'Target met' : `${n(tg - v)} g left`}</span>` : ''}</div></li>`;
    }).join('');
    const meals = D.meals ? `<section class="section" aria-labelledby="ml-h"><div class="section-h"><h2 id="ml-h">Meals</h2><span class="tag">Only when the source sends meals</span></div>
      <ul class="group">${D.meals.map(([m, k]) => `<li class="row"><span class="copy"><span class="title">${m}</span></span><span class="trail"><span class="value">${n(k)} kcal</span></span></li>`).join('')}</ul></section>` : '';
    view.innerHTML = `${header('Nutrition', dateLabel())}
      <div class="wk-grid">${hero}
        <section class="section" aria-labelledby="mc-h"><div class="section-h"><h2 id="mc-h">Macros</h2></div><ul class="group">${rows}</ul></section>
        <section class="card" aria-labelledby="src-h"><h2 class="card-k" id="src-h" style="margin-bottom:12px">Source</h2>
          <p class="flow"><span class="app-tile mfp" aria-hidden="true">MFP</span><b>MyFitnessPal</b>${ic('chevR', 'i-16')}<span class="app-tile ah" aria-hidden="true">AH</span><b>Apple Health</b>${ic('chevR', 'i-16')}<b>GYST</b></p>
          <p class="card-s" style="margin-top:12px">Last sync ${esc(D.sync.apple)} ${D.sync.stale ? '<span class="chip warn">Stale</span>' : ''}</p>
          <a class="btn block" style="margin-top:16px" href="https://www.myfitnesspal.com/" target="_blank" rel="noopener noreferrer">Open MyFitnessPal ${ic('ext', 'i-16')}</a></section>
        ${meals}</div>${protoBar()}`;
  }

  /* ---------- Coach ---------- */
  const PROMPTS = ["I haven't eaten breakfast. What should I have?", 'How much protein do I have left today?', 'What should I eat for lunch?', 'How has my training been this month?', "I'm sore. Should I train today?", 'What muscle groups have I neglected?'];
  const chatLog = [];
  let clOpen = null;
  function coachReply(q) {
    const s = q.toLowerCase(), t = D.targets, f = D.food, h = D.hevy;
    const pLeft = t && f ? Math.max(0, t.protein - f.protein) : null, kLeft = t && f ? t.kcal - f.kcal : null;
    const byProtein = [...D.foods].sort((a, b) => b.p / b.kcal - a.p / a.kcal).slice(0, 2);
    if (!D.sync && !h.connected) return { text: 'I don\'t have any of your health data yet. Connect MyFitnessPal, Apple Health and Hevy in More, then ask again.', ctx: 'None available' };
    if (/breakfast/.test(s)) return { text: `You have ${n(kLeft)} kcal and ${n(pLeft)} g protein left today. From foods you keep:`, list: byProtein.map((x) => `${x.name} · ${x.serving} · ${n(x.kcal)} kcal, ${n(x.p)} g protein`), ctx: 'Calories, protein, My foods' };
    if (/protein/.test(s)) return { text: pLeft == null ? 'No food has synced today, so I can\'t tell yet.' : `${n(pLeft)} g left of your ${n(t.protein)} g target (${n(f.protein)} g so far).`, list: pLeft ? byProtein.map((x) => `${x.name}: ${n(x.p)} g per ${x.serving}`) : null, ctx: 'Protein, protein target, My foods' };
    if (/lunch/.test(s)) return { text: `About ${n(kLeft)} kcal left. A lunch of 500–650 kcal with 40 g+ protein keeps dinner open. Options from your foods:`, list: ['High-protein wrap + turkey breast (100 g) · 270 kcal, 36 g protein', 'Greek yogurt + protein powder · 225 kcal, 42 g protein'], ctx: 'Calories, macros, My foods' };
    if (/train|month|workout/.test(s)) return { text: 'Last 30 days: 11 workouts, about 2.6 a week against your target of 3. Most work went to legs (35%) and back (25%). Leg press is up 18% over 12 weeks.', ctx: 'Hevy workouts, muscle focus, progress' };
    if (/sore/.test(s)) return { text: 'Your last session was heavy on legs. A lighter upper-body day or a walk keeps you moving without loading the same muscles. If pain is sharp, or lasts more than a few days, check with a professional.', ctx: 'Latest Hevy workout, muscle focus' };
    if (/neglect|muscle/.test(s)) return { text: 'Over 30 days, core got no primary sets and arms and shoulders 10% each. Adding 2–3 sets of a core exercise to each session would balance it.', ctx: 'Muscle focus, Hevy workouts' };
    return { text: 'In the working version I\'ll answer this from your health data. This prototype only has sample replies for the suggested questions.', ctx: 'Prototype' };
  }
  function checklistHTML() {
    const t = D.targets, f = D.food;
    const measured = [];
    if (t) measured.push({ name: 'Protein target', v: f ? f.protein : null, tg: t.protein, unit: 'g', src: 'MyFitnessPal' });
    if (t) measured.push({ name: 'Water target', v: D.water, tg: t.water, unit: 'L', src: 'Apple Health', digits: 1 });
    const met = measured.filter((m) => m.v != null && m.v >= m.tg).length + D.checks.filter((c) => c.done).length;
    const total = measured.length + D.checks.length;
    if (!total) return `<div class="empty" style="padding:16px"><p class="title">No checklist yet</p><p>Add your own daily habits.</p><a class="btn small secondary" href="settings.html#checklist">Set up checklist</a></div>`;
    return `<p class="card-s" id="cl-count" style="padding:0 16px">${met} of ${total} done</p><ul>
      ${measured.map((m) => `<li class="measure"><span class="tile sm ${m.v != null && m.v >= m.tg ? 'ok' : 'idle'}">${ic(m.unit === 'L' ? 'drop' : 'target')}</span><div class="copy"><div class="top"><span class="title">${m.name}</span><span class="num">${m.v == null ? '<span class="missing">—</span>' : `<b>${n(m.v, m.digits || 0)}</b>`} / ${n(m.tg, m.digits || 0)} ${m.unit}</span></div>
        <div class="bar${m.v == null ? ' missing' : m.v >= m.tg ? ' ok' : ''}" aria-hidden="true"><i style="--v:${pct(m.v, m.tg) || 0}%"></i></div><span class="card-s">From ${m.src}</span></div></li>`).join('')}
      ${D.checks.map((c) => `<li><label class="check"><input type="checkbox" data-check="${c.id}"${c.done ? ' checked' : ''}><span class="box">${ic('check', 'i-16')}</span><span class="copy"><span class="title">${esc(c.name)}</span></span></label></li>`).join('')}</ul>`;
  }
  function renderCoach() {
    const wide = clOpen ?? matchMedia('(min-width: 1024px)').matches;
    const msgs = chatLog.map((m) => m.me ? `<p class="msg me">${esc(m.text)}</p>`
      : m.typing ? `<div class="msg ai" aria-label="Coach is typing"><span class="typing"><i></i><i></i><i></i></span></div>`
        : `<div class="msg ai"><p>${esc(m.text)}</p>${m.list ? `<ul>${m.list.map((x) => `<li>${esc(x)}</li>`).join('')}</ul>` : ''}<p class="ctx">Used: ${esc(m.ctx)} · prototype reply</p></div>`).join('');
    view.innerHTML = `${header('Health Coach', '', '<p class="status-line">Your nutrition &amp; training assistant</p>')}
      <div class="coach">
        <div class="side"><details class="card" style="padding:0" ${wide ? 'open' : ''} id="checklist"><summary class="row" style="border-radius:16px"><span class="tile sm">${ic('list')}</span><span class="copy"><span class="title">Today's checklist</span></span>${ic('chevR', 'chev')}</summary><div style="padding-bottom:8px">${checklistHTML()}</div></details></div>
        <section class="chat" aria-label="Conversation">
          ${!D.sync && !D.hevy.connected ? `<div class="banner">${ic('info')}<span class="grow">Connect your sources so the coach has context.</span></div>` : ''}
          <div class="chat" role="log" aria-live="polite" id="log">${msgs}</div>
          ${chatLog.length ? '' : `<div class="prompts">${PROMPTS.map((p) => `<button class="prompt" type="button" data-prompt="${esc(p)}">${ic('chat', 'i-16')}<span>${esc(p)}</span></button>`).join('')}</div>`}
          <form class="composer" id="composer"><label class="sr" for="q">Ask the Health Coach</label><textarea id="q" rows="1" placeholder="Ask about food, training or progress"></textarea>
            <button class="icon-btn" type="submit" aria-label="Send" disabled>${ic('send')}</button></form>
        </section>
      </div>${protoBar()}`;
    const cl = $('#checklist'); if (cl) cl.addEventListener('toggle', () => { clOpen = cl.open; });
  }
  function ask(q) {
    chatLog.push({ me: true, text: q }, { typing: true });
    renderCoach();
    setTimeout(() => {
      chatLog.pop(); chatLog.push(coachReply(q)); renderCoach();
      const box = $('#q'); if (box) box.focus();
    }, 700);
  }

  /* ---------- Settings (More) ---------- */
  function renderSettings() {
    const p = D.plan, t = D.targets, w = D.weight, row = (icon, title, meta, trail, attrs = '', tag = 'div') =>
      `<li><${tag} class="row" ${attrs}><span class="tile sm${icon[1] ? ' ' + icon[1] : ''}">${ic(icon[0])}</span><span class="copy"><span class="title">${title}</span>${meta ? `<span class="meta">${meta}</span>` : ''}</span><span class="trail">${trail}</span></${tag}></li>`;
    const chev = ic('chevR', 'chev');
    const src = [['mfp', 'MFP', 'MyFitnessPal', 'Food logging', D.sync ? 'Logging' : 'Not connected', !!D.sync],
      ['ah', 'AH', 'Apple Health', 'Nutrition and activity bridge: steps, exercise, energy', D.sync ? `Synced ${D.sync.apple}` : 'Not connected', !!D.sync],
      ['hv', 'HV', 'Hevy', 'Routines and workouts', D.hevy.connected ? `Synced ${D.hevy.synced}` : 'Not connected', D.hevy.connected],
      ['gy', 'G', 'GYST', 'Targets, weight, progress, checklist, coach', 'Owner', true]];
    view.innerHTML = `${header('More', 'Health settings')}
      <div class="settings">
        <section class="section" aria-labelledby="pl-h"><div class="section-h"><h2 id="pl-h">Health plan</h2></div><ul class="group">
          ${p ? row(['target'], esc(p.title), `${p.start} – ${p.end}<span class="sep"></span><span class="num">Day ${p.day} of ${p.days}</span>`, chev, 'href="#"', 'a')
            + row(['apple'], 'Nutrition targets', `<span class="num">${n(t.kcal)} kcal · ${n(t.protein)} g protein</span>`, chev, 'href="#"', 'a')
            + row(['steps'], 'Activity targets', `<span class="num">${n(t.steps)} steps · ${t.workouts} workouts a week</span>`, chev, 'href="#"', 'a')
            : row(['target', 'idle'], 'No plan yet', 'Set a goal period and targets', '<span class="btn small secondary">Create plan</span>', 'href="#"', 'a')}</ul></section>
        <section class="section" aria-labelledby="wt-h"><div class="section-h"><h2 id="wt-h">Weight</h2></div><ul class="group">
          ${row(['scale'], 'Latest', w ? esc(w.lastLabel) : 'No readings yet', w ? `<span class="value num">${n(w.series[w.series.length - 1], 1)} kg</span>` : '<span class="missing">—</span>')}
          ${row(['plus'], 'Log weight', '', chev, 'type="button" data-act="log-weight"', 'button')}
          ${row(['scale', 'idle'], 'Units', '', '<span>kg</span>')}</ul></section>
        <section class="section" id="checklist" aria-labelledby="cl-h"><div class="section-h"><h2 id="cl-h">Daily checklist</h2><button type="button" data-act="add-check">Add</button></div><ul class="group">
          ${D.checks.length ? D.checks.map((c) => row(['check', 'idle'], esc(c.name), '', '<span>Daily</span>')).join('') : row(['list', 'idle'], 'No items yet', 'Your own habits, set by you', '')}
          ${t ? row(['target'], 'Protein target', 'Measured from MyFitnessPal', '<span>Auto</span>') + row(['drop'], 'Water target', 'Measured from Apple Health', '<span>Auto</span>') : ''}</ul>
          <p class="foot">Items are yours. GYST doesn't suggest supplements or doses.</p></section>
        <section class="section" id="foods" aria-labelledby="fd-h"><div class="section-h"><h2 id="fd-h">My foods &amp; products</h2><button type="button" data-act="add-food">Add</button></div><ul class="group">
          ${D.foods.length ? D.foods.map((f) => row(['box'], esc(f.name), `<span class="num">${esc(f.serving)}<span class="sep"></span>${n(f.kcal)} kcal<span class="sep"></span>${n(f.p, f.p % 1 ? 1 : 0)} g protein</span>`, chev, `type="button" data-food="${f.id}"`, 'button')).join('')
            : row(['box', 'idle'], 'No foods yet', 'Add what you usually have at home', '')}</ul>
          <p class="foot">The coach uses these for suggestions like "make breakfast from what I have".</p></section>
        <section class="section" id="sources" aria-labelledby="ds-h"><div class="section-h"><h2 id="ds-h">Data sources</h2></div><ul class="group">
          ${src.map(([c, m, name, role, state, ok]) => `<li><div class="row"><span class="app-tile ${c}" aria-hidden="true">${m}</span><span class="copy"><span class="title">${name}</span><span class="meta">${role}</span></span><span class="trail"><span class="dot${ok ? ' ok' : ''}"></span>${esc(state)}</span></div></li>`).join('')}</ul>
          <p class="foot">Log food in MyFitnessPal and workouts in Hevy. GYST reads them and never logs twice.</p></section>
        <section class="section" aria-labelledby="ai-h"><div class="section-h"><h2 id="ai-h">AI coach</h2></div><ul class="group">
          ${row(['cpu', 'idle'], 'Provider', 'Claude or OpenAI · key stays on the server', '<span class="dot"></span>Not connected')}
          <li><details class="more" style="padding:0 16px"><summary>What the coach can see ${ic('chevR', 'i-16')}</summary>
            <ul class="meta" style="display:grid;gap:4px;padding:0 0 12px;list-style:disc;padding-left:20px"><li>Today's calories, macros and targets</li><li>Steps, exercise and weight trend</li><li>Plan goal and dates</li><li>Hevy routines, workouts, progress and muscle focus</li><li>My foods and checklist</li></ul>
            <p class="meta" style="padding-bottom:12px">Never sent: passwords, keys, other GYST areas or the whole database.</p></details></li></ul></section>
      </div>${protoBar()}`;
  }

  /* ---------- Sheets ---------- */
  function logWeightSheet() {
    const last = D.weight ? D.weight.series[D.weight.series.length - 1] : '';
    openSheet('Log weight', `<form id="wf" novalidate><div class="field"><label for="wkg">Weight</label><div class="input-unit"><input class="input" id="wkg" inputmode="decimal" autocomplete="off" value="${last}" aria-describedby="wkg-err"><span>kg</span></div><p class="err" id="wkg-err" hidden></p></div>
      <div class="field" style="margin-top:12px"><label for="wdt">When</label><input class="input" id="wdt" type="datetime-local" value="${new Date(Date.now() - new Date().getTimezoneOffset() * 60000).toISOString().slice(0, 16)}"></div>
      <button class="btn block" style="margin-top:24px" type="submit">Save</button></form>`, (s) => {
      const input = $('#wkg', s); input.select();
      $('#wf', s).addEventListener('submit', (e) => {
        e.preventDefault();
        const v = parseFloat(input.value.replace(',', '.')), err = $('#wkg-err', s);
        if (!(v > 20 && v < 400)) { err.textContent = 'Enter a weight between 20 and 400 kg.'; err.hidden = false; input.setAttribute('aria-invalid', 'true'); input.focus(); return; }
        if (!D.weight) D.weight = { start: v, startLabel: 'Today', series: [], lastLabel: 'Today' };
        D.weight.series.push(+v.toFixed(1)); D.weight.lastLabel = 'Just now';
        sheet.close(); render(); toast(`Weight logged · ${n(v, 1)} kg`);
      });
    });
  }
  function foodSheet(id) {
    const f = D.foods.find((x) => x.id === id); if (!f) return;
    openSheet(f.name, `<p class="meta">${esc(f.serving)}</p><dl class="kv num"><div><dt>Calories</dt><dd>${n(f.kcal)}</dd></div><div><dt>Protein</dt><dd>${n(f.p, f.p % 1 ? 1 : 0)} g</dd></div><div><dt>Carbs</dt><dd>${n(f.c, f.c % 1 ? 1 : 0)} g</dd></div><div><dt>Fat</dt><dd>${n(f.f, f.f % 1 ? 1 : 0)} g</dd></div><div><dt>Fiber</dt><dd>${n(f.fib)} g</dd></div></dl>
      ${f.notes ? `<p class="meta">${esc(f.notes)}</p>` : ''}<div style="display:flex;gap:8px"><button class="btn secondary" type="button" data-close>Done</button><button class="btn plain" type="button" data-close>Edit</button></div>`);
  }
  function addFoodSheet() {
    const fld = (id, l, mode = 'decimal') => `<div class="field"><label for="${id}">${l}</label><input class="input" id="${id}" ${mode ? `inputmode="${mode}"` : ''} autocomplete="off"></div>`;
    openSheet('Add food or product', `<form id="ff" style="display:grid;gap:12px">${fld('fn', 'Name', '')}${fld('fs', 'Serving size', '')}
      <div class="grid-2">${fld('fk', 'Calories')}${fld('fp', 'Protein (g)')}${fld('fc', 'Carbs (g)')}${fld('ft', 'Fat (g)')}</div>${fld('fb', 'Fiber (g)')}
      <p class="err" id="ff-err" hidden></p><button class="btn block" type="submit">Save</button></form>`, (s) => {
      $('#fn', s).focus();
      $('#ff', s).addEventListener('submit', (e) => {
        e.preventDefault();
        const name = $('#fn', s).value.trim(), num = (x) => parseFloat($(x, s).value) || 0;
        if (!name) { const er = $('#ff-err', s); er.textContent = 'Add a name.'; er.hidden = false; $('#fn', s).focus(); return; }
        D.foods.push({ id: 'u' + Date.now(), name, serving: $('#fs', s).value.trim() || '1 serving', kcal: num('#fk'), p: num('#fp'), c: num('#fc'), f: num('#ft'), fib: num('#fb'), notes: '' });
        sheet.close(); render(); toast('Food added');
      });
    });
  }
  function addCheckSheet() {
    openSheet('Add checklist item', `<form id="cf" style="display:grid;gap:12px"><div class="field"><label for="cn">Name</label><input class="input" id="cn" autocomplete="off" placeholder="e.g. Stretch 10 minutes"></div><button class="btn block" type="submit">Add</button></form>`, (s) => {
      $('#cn', s).focus();
      $('#cf', s).addEventListener('submit', (e) => { e.preventDefault(); const v = $('#cn', s).value.trim(); if (!v) return; D.checks.push({ id: 'c' + Date.now(), name: v, done: false }); sheet.close(); render(); toast('Item added'); });
    });
  }
  function energySheet() {
    const e = D.energy || {}, f = D.food;
    openSheet('Energy balance so far', `<dl class="kv num"><div><dt>Eaten</dt><dd>${f ? n(f.kcal) : '—'}</dd></div><div><dt>Resting</dt><dd>${n(e.resting)}</dd></div><div><dt>Active</dt><dd>${n(e.active)}</dd></div></dl>
      <p class="meta">Eaten − (resting + active), observed so far today. It changes until the day ends, so it isn't a final deficit. Steps are never converted to calories.</p>`);
  }
  function workoutSheet() {
    const l = D.hevy.latest;
    openSheet(l.title, `<p class="meta num">${minutes(l.min)}<span class="sep"></span>${l.exercises.length} exercises<span class="sep"></span>${esc(l.when)}</p>
      <ul class="group">${l.exercises.map(([x, s]) => `<li class="row"><span class="copy"><span class="title">${esc(x)}</span><span class="meta num">${esc(s)}</span></span></li>`).join('')}</ul>
      <p class="foot">Logged in Hevy. GYST shows it read-only.</p>`);
  }

  /* ---------- Render + events ---------- */
  const RENDER = { today: renderToday, workout: renderWorkout, nutrition: renderNutrition, coach: renderCoach, settings: renderSettings };
  function render() { (RENDER[page] || renderToday)(); }
  render();

  function syncLinks() {
    const q = new URLSearchParams();
    if (scenario !== 'full') q.set('data', scenario);
    if (theme !== 'system') q.set('theme', theme);
    const qs = q.toString();
    document.querySelectorAll('a[data-keep]').forEach((a) => { const base = a.getAttribute('href').split('?')[0]; a.setAttribute('href', base + (qs ? '?' + qs : '')); });
    try { history.replaceState(null, '', location.pathname + (qs ? '?' + qs : '') + location.hash); } catch { /* sandboxed viewer */ }
  }
  syncLinks();

  document.addEventListener('change', (e) => {
    const sel = e.target.closest('[data-proto]');
    if (sel) {
      if (sel.dataset.proto === 'data') { scenario = sel.value; store.set('data', scenario); D = SCENARIOS[scenario](); chatLog.length = 0; }
      else { theme = sel.value; store.set('theme', theme); applyTheme(); }
      syncLinks(); render();
      const again = $(`[data-proto="${sel.dataset.proto}"]`); if (again) again.focus();
      return;
    }
    const cb = e.target.closest('[data-check]');
    if (cb) {
      const c = D.checks.find((x) => x.id === cb.dataset.check); if (c) c.done = cb.checked;
      const t = D.targets, f = D.food, count = $('#cl-count');
      const measured = t ? [[f ? f.protein : null, t.protein], [D.water, t.water]] : [];
      const met = measured.filter(([v, tg]) => v != null && v >= tg).length + D.checks.filter((x) => x.done).length;
      if (count) count.textContent = `${met} of ${measured.length + D.checks.length} done`;
    }
  });

  document.addEventListener('click', (e) => {
    const a = e.target.closest('[data-act]');
    if (a) {
      const act = a.dataset.act;
      if (act === 'log-weight') logWeightSheet();
      else if (act === 'energy-info') energySheet();
      else if (act === 'workout-detail') workoutSheet();
      else if (act === 'add-food') addFoodSheet();
      else if (act === 'add-check') addCheckSheet();
      else if (act === 'sync') toast('Synced with Hevy · prototype');
      else if (act === 'to-progress') { wkView = 'progress'; render(); $('#tab-progress').focus(); }
      return;
    }
    const food = e.target.closest('[data-food]'); if (food) { foodSheet(food.dataset.food); return; }
    const tab = e.target.closest('[role="tab"]'); if (tab) { wkView = tab.id.replace('tab-', ''); render(); $('#' + tab.id).focus(); return; }
    const per = e.target.closest('[data-period]'); if (per) { period = +per.dataset.period; render(); $(`[data-period="${period}"]`).focus(); return; }
    const pr = e.target.closest('[data-prompt]'); if (pr) { ask(pr.dataset.prompt); return; }
  });

  document.addEventListener('keydown', (e) => {
    const tab = e.target.closest('[role="tab"]');
    if (tab && (e.key === 'ArrowRight' || e.key === 'ArrowLeft')) { e.preventDefault(); wkView = wkView === 'overview' ? 'progress' : 'overview'; render(); $('#tab-' + wkView).focus(); }
    if (e.target.id === 'q' && e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); $('#composer').requestSubmit(); }
  });
  document.addEventListener('input', (e) => {
    if (e.target.id === 'q') { const b = $('#composer button'); b.disabled = !e.target.value.trim(); e.target.style.height = 'auto'; e.target.style.height = Math.min(140, e.target.scrollHeight) + 'px'; }
  });
  document.addEventListener('submit', (e) => {
    if (e.target.id !== 'composer') return;
    e.preventDefault(); const q = $('#q').value.trim(); if (q) ask(q);
  });

  const topbar = $('.topbar');
  const onScroll = () => topbar && topbar.classList.toggle('scrolled', scrollY > 40);
  addEventListener('scroll', onScroll, { passive: true }); onScroll();
  matchMedia('(min-width: 1024px)').addEventListener('change', () => { if (page === 'coach') render(); });
})();
