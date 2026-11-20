#!/usr/bin/env node
// Look and accent tokens for the makeover (GYST-169, Issue #259; 13B DD-6, DD-7, DD-8, DD-13).
// No dependencies or build step.
//
//   node scripts/build-accents.cjs                    write web/static/css/ui-accents.css
//   node scripts/build-accents.cjs --check            write nothing; fail if anything below is not true
//   node scripts/build-accents.cjs --seeds f.json     use other seeds; writes nothing unless --out is given
//   node scripts/build-accents.cjs --out f.css        write the CSS somewhere else
//   node scripts/build-accents.cjs --r7 dir           also write looks7.css and looks7.json in round 7's own format
//
// Exit 0: every contrast pair holds. Exit 1: a pair fails, or --check found a difference.
// Exit 2: bad arguments or a seeds file that cannot be read.
//
// Why a port and not a copy of the values: 13B "Token tools for R1" asks for the generator in the
// repository so that a value which breaks a contrast pair fails the build. This is a line-for-line port
// of design-makeover/tools-r7/gen_looks7.py. The fixture scripts/fixtures/looks-r7.json holds that
// tool's own output for the round 7 seeds, and --check compares against it byte for byte.
'use strict';
const fs = require('node:fs');
const path = require('node:path');
const crypto = require('node:crypto');
const { spawnSync } = require('node:child_process');

const ROOT = path.join(__dirname, '..');
const CSS_FILE = path.join(ROOT, 'web/static/css/ui-accents.css');
const FIXTURE_FILE = path.join(__dirname, 'fixtures/looks-r7.json');

// ---------------------------------------------------------------------------------------------
// Inputs. These are the round 7 seeds, recorded in 13B. Changing a Look's starting accent is the
// one field `accent` (13B O-2: Aurora stays on Iris until Jay says otherwise).
// ---------------------------------------------------------------------------------------------
const SEEDS = {
  defaultLook: 'aurora',
  // DD-7. Key order is round 7's and decides the order of the output.
  looks: {
    aurora: { label: 'Aurora', hue: 248, satLight: 0.42, satDark: 0.30, accent: 'iris', sectionSat: 0.82, sectionHues: [222, 16, 268, 335, 172, 248], canvasDark: 0.045 },
    sunrise: { label: 'Sunrise', hue: 32, satLight: 0.50, satDark: 0.16, accent: 'ember', sectionSat: 0.80, sectionHues: [206, 12, 282, 346, 174, 30], canvasDark: 0.045 },
    lagoon: { label: 'Lagoon', hue: 188, satLight: 0.40, satDark: 0.34, accent: 'teal', sectionSat: 0.78, sectionHues: [215, 24, 256, 332, 165, 192], canvasDark: 0.04 },
  },
  // DD-6. Fixed, audited values: asserted, never derived. In dark the fill is also the text colour.
  accents: {
    iris: { light: { fill: '#6163F2', text: '#4842D0', soft: '#EDEFFD' }, dark: { fill: '#96A2FE', soft: '#1F2343' } },
    blue: { light: { fill: '#306FEC', text: '#104FCA', soft: '#E9F1FD' }, dark: { fill: '#87B1FD', soft: '#162644' } },
    teal: { light: { fill: '#0F828D', text: '#06626B', soft: '#D6F8FB' }, dark: { fill: '#44D4E2', soft: '#032C31' } },
    volt: { light: { fill: '#5A8209', text: '#436202', soft: '#E9F5DC' }, dark: { fill: '#B2F340', soft: '#1E2C07' } },
    orchid: { light: { fill: '#B149C2', text: '#9126A1', soft: '#FCE9FE' }, dark: { fill: '#EA7FFB', soft: '#341C38' } },
    ember: { light: { fill: '#C4530A', text: '#963D03', soft: '#FEECE4' }, dark: { fill: '#FE8F5B', soft: '#3E1B0B' } },
    graphite: { light: { fill: '#2C2E31', text: '#57595D', soft: '#EEF0F4' }, dark: { fill: '#DCDEE2', soft: '#252729' } },
  },
  // DD-13. State text starts from these and is darkened per Look until it passes; the soft
  // backgrounds are the ones ui-tokens.css already ships.
  state: {
    light: { ok: '#15803D', warn: '#B45309', crit: '#B91C1C', warnsoft: '#FDF1E1', critsoft: '#FDEAEA' },
    dark: { ok: '#6EE7A0', warn: '#FCD062', crit: '#FCA5A5', warnsoft: '#33260C', critsoft: '#3A1618' },
  },
};
const SECTIONS = ['plan', 'work', 'connect', 'health', 'learn', 'system'];
const THEMES = ['light', 'dark'];
const T_TEXT = 4.7, T_STRONG = 7.0; // search targets; the assertion is AA (4.5 text, 3 fills)
const ON_ACCENT = { light: '#FFFFFF', dark: '#0B0D12' }; // DD-6

// ---------------------------------------------------------------------------------------------
// Colour maths. Byte-identical output depends on three details of the Python tool, each kept here:
// round() is half-to-even, colorsys.hls_to_rgb has a fixed operation order, and a colour is cut
// to 8 bits before it is measured.
// ---------------------------------------------------------------------------------------------
const pyRound = x => { const f = Math.floor(x), d = x - f; return d > 0.5 ? f + 1 : d < 0.5 ? f : (f % 2 === 0 ? f : f + 1); };
const hx = c => '#' + c.map(v => pyRound(Math.max(0, Math.min(1, v)) * 255).toString(16).padStart(2, '0')).join('').toUpperCase();
const rgb = h => { const s = h.replace('#', ''); return [0, 2, 4].map(i => parseInt(s.slice(i, i + 2), 16) / 255); };
const ONE_THIRD = 1.0 / 3.0, ONE_SIXTH = 1.0 / 6.0, TWO_THIRD = 2.0 / 3.0;
const pyMod1 = x => { const r = x % 1.0; return r < 0 ? r + 1.0 : r; }; // Python's % never returns a negative
function hueValue(m1, m2, hue) {
  hue = pyMod1(hue);
  if (hue < ONE_SIXTH) return m1 + (m2 - m1) * hue * 6.0;
  if (hue < 0.5) return m2;
  if (hue < TWO_THIRD) return m1 + (m2 - m1) * (TWO_THIRD - hue) * 6.0;
  return m1;
}
function hlsToRgb(h, l, s) {
  if (s === 0.0) return [l, l, l];
  const m2 = l <= 0.5 ? l * (1.0 + s) : l + s - (l * s);
  const m1 = 2.0 * l - m2;
  return [hueValue(m1, m2, h + ONE_THIRD), hueValue(m1, m2, h), hueValue(m1, m2, h - ONE_THIRD)];
}
const hsl = (h, s, l) => hlsToRgb((((h % 360) + 360) % 360) / 360, l, s);
function lum(c) {
  const f = v => (v <= 0.04045 ? v / 12.92 : Math.pow((v + 0.055) / 1.055, 2.4));
  const [r, g, b] = c.map(f);
  return 0.2126 * r + 0.7152 * g + 0.0722 * b;
}
function contrast(a, b) { const la = lum(a), lb = lum(b); return (Math.max(la, lb) + 0.05) / (Math.min(la, lb) + 0.05); }
const q = c => rgb(hx(c));

class NoColour extends Error {}
class BadInput extends Error {}

// Walk lightness from start by step until the colour clears target on every background.
function search(h, s, bgs, target, start, step) {
  let l = start;
  while (l >= 0 && l <= 1) {
    const c = q(hsl(h, s, l));
    if (bgs.every(b => contrast(c, b) >= target)) return c;
    l += step;
  }
  throw new NoColour('no colour for hue ' + h + ' reaches ' + target + ':1');
}

// ---------------------------------------------------------------------------------------------
// Derive every Look in both themes and measure every pair.
// `checks` are round 7's own pairs, in round 7's order (714 for the round 7 seeds).
// `extra` are pairs round 7 did not have, which part 2 of R1 (GYST-168) relies on.
// ---------------------------------------------------------------------------------------------
function derive(seeds) {
  const WHITE = rgb('#FFFFFF'), INK_D = rgb('#0B0D12');
  const looks = {}, checks = [], extra = [];
  const push = (list, look, theme, what, a, b, need) => { const r = contrast(a, b); list.push({ r, look, theme, what, need, ok: r >= need }); };
  for (const [key, L] of Object.entries(seeds.looks)) {
    looks[key] = { label: L.label, accent: L.accent };
    for (const theme of THEMES) {
      const chk = (what, a, b, need) => push(checks, key, theme, what, a, b, need);
      const t = {};
      let n;
      if (theme === 'light') {
        n = { canvas: q(hsl(L.hue, L.satLight, 0.957)), surface: WHITE, surface2: q(hsl(L.hue, L.satLight * 0.8, 0.935)), surface3: q(hsl(L.hue, L.satLight * 0.7, 0.895)), elevated: WHITE, ink: q(hsl(L.hue, 0.30, 0.065)) };
        n.ink2 = search(L.hue, 0.14, [n.surface2, n.canvas], T_STRONG, 0.5, -0.005);
        n.ink3 = search(L.hue, 0.10, [n.surface2, n.canvas, n.surface], T_TEXT + 0.2, 0.6, -0.005);
      } else {
        const d = L.canvasDark;
        n = { canvas: q(hsl(L.hue, L.satDark, d)), surface: q(hsl(L.hue, L.satDark, d + 0.05)), surface2: q(hsl(L.hue, L.satDark, d + 0.09)), surface3: q(hsl(L.hue, L.satDark * 0.9, d + 0.135)), elevated: q(hsl(L.hue, L.satDark, d + 0.095)), ink: q(hsl(L.hue, 0.12, 0.955)) };
        n.ink2 = search(L.hue, 0.10, [n.surface2, n.elevated], T_STRONG, 0.5, 0.005);
        n.ink3 = search(L.hue, 0.08, [n.surface2, n.elevated, n.canvas], T_TEXT + 0.2, 0.4, 0.005);
      }
      const bgs = ['canvas', 'surface', 'surface2', 'elevated'].map(k => n[k]);
      for (const k of ['ink', 'ink2', 'ink3']) t[k] = hx(n[k]);
      for (const k of ['canvas', 'surface', 'surface2', 'surface3', 'elevated']) t[k] = hx(n[k]);
      for (const b of bgs) { chk('ink-3 on neutral', n.ink3, b, 4.5); chk('ink-2 on neutral', n.ink2, b, 4.5); }

      for (const [sk, sh, ssat] of [['ok', 142, 0.72], ['warn', 26, 0.92], ['crit', 0, 0.74]]) {
        const cur = rgb(seeds.state[theme][sk]);
        const sb = bgs.concat(sk !== 'ok' ? [rgb(seeds.state[theme][sk + 'soft'])] : []);
        const text = sb.every(b => contrast(cur, b) >= 4.6) ? cur : search(sh, ssat, sb, T_TEXT, theme === 'light' ? 0.40 : 0.6, theme === 'light' ? -0.004 : 0.004);
        t[sk + '-text'] = hx(text);
        for (const b of sb) chk(sk + '-text on neutral or its soft', text, b, 4.5);
      }

      // Every accent must hold on this Look's neutrals, not only the Look's own.
      const on = theme === 'light' ? WHITE : INK_D;
      for (const [an, a] of Object.entries(seeds.accents)) {
        const fill = rgb(a[theme].fill), text = rgb(theme === 'light' ? a.light.text : a.dark.fill), soft = rgb(a[theme].soft);
        chk('on-accent on accent (' + an + ')', on, fill, 4.5);
        chk('accent-text on accent-soft (' + an + ')', text, soft, 4.5);
        for (const b of bgs) chk('accent-text on neutral (' + an + ')', text, b, 4.5);
        chk('accent fill vs surface (' + an + ')', fill, n.surface, 3.0);
        // GYST-168 points legacy fills at --vn-accent-text and keeps --vn-on-accent as their text.
        push(extra, key, theme, 'on-accent on accent-text (' + an + ')', on, text, 4.5);
      }

      SECTIONS.forEach((name, i) => {
        const h = L.sectionHues[i], s = name !== 'system' ? L.sectionSat : 0.10;
        let soft, fill, text;
        if (theme === 'light') {
          soft = q(hsl(h, Math.min(1, s + 0.1), name !== 'system' ? 0.945 : 0.93));
          fill = search(h, s, [WHITE], T_TEXT, 0.62, -0.005);
          text = search(h, s, bgs.concat([soft]), T_TEXT, 0.5, -0.005);
        } else {
          soft = q(hsl(h, s * 0.55, name !== 'system' ? 0.17 : 0.2));
          text = search(h, Math.min(1, s + 0.1), bgs.concat([soft]), T_TEXT + 0.6, 0.55, 0.005);
          fill = text;
        }
        t['s-' + name] = hx(fill); t['s-' + name + '-t'] = hx(text); t['s-' + name + '-s'] = hx(soft);
        chk('tile icon on section fill (' + name + ')', on, fill, 4.5);
        chk('section text on section soft (' + name + ')', text, soft, 4.5);
        for (const b of bgs) chk('section text on neutral (' + name + ')', text, b, 4.5);
        chk('section fill vs surface (' + name + ')', fill, n.surface, 3.0);
        chk('ink on section soft (' + name + ')', n.ink, soft, 4.5);
      });

      // Body text is strong text: GYST-168 points --gyst-text and the legacy --text at --vn-ink.
      for (const k of ['canvas', 'surface', 'surface2', 'elevated']) push(extra, key, theme, 'ink on neutral (' + k + ')', n.ink, n[k], 7.0);

      t.separator = theme === 'light' ? hx(n.surface3) : 'rgb(255 255 255 / 9%)';
      const [r, g, b] = n.surface.map(v => pyRound(v * 255));
      t.glass = `rgb(${r} ${g} ${b} / ${theme === 'light' ? '74%' : '70%'})`;
      looks[key][theme] = t;
    }
  }
  return { looks, checks, extra };
}

// ---------------------------------------------------------------------------------------------
// Round 7's own two files, reproduced so the port can be compared with the Python tool.
// ---------------------------------------------------------------------------------------------
const R7_KEYS = t => Object.keys(t).filter(k => k !== 'separator' && k !== 'glass'); // round 7 kept these two out of its JSON

function r7Css(result) {
  const lines = [];
  for (const [key, look] of Object.entries(result.looks)) {
    for (const theme of THEMES) {
      const t = look[theme];
      const sel = theme === 'light' ? `:root[data-look="${key}"]` : `:root[data-theme="dark"][data-look="${key}"], [data-look="${key}"] .space`;
      const d = [
        `--canvas: ${t.canvas}; --surface: ${t.surface}; --surface-2: ${t.surface2}; --surface-3: ${t.surface3}; --elevated: ${t.elevated};`,
        `--ink: ${t.ink}; --ink-2: ${t.ink2}; --ink-3: ${t.ink3}; --separator: ${t.separator};`,
        `--glass: ${t.glass};`,
        R7_KEYS(t).filter(k => k.startsWith('s-') || k.endsWith('-text')).map(k => `--${k}: ${t[k]};`).join(' '),
      ];
      lines.push(sel + ' { ' + d.join(' ') + ' }');
    }
  }
  return '/* ---------- Looks: generated by gen_looks.py; every pair asserted at AA ---------- */\n' + lines.join('\n') + '\n';
}

// Python prints a whole-number float as "3.0"; JSON.stringify would print "3".
const pyFloat = x => (Number.isInteger(x) ? x.toFixed(1) : String(x));
const round2 = x => Number(x.toFixed(2));
const cmp = (a, b) => (a < b ? -1 : a > b ? 1 : 0);

function lowestPairs(checks) {
  // Python sorts the tuples (ratio rounded to 2 places, look, theme, what, need).
  return checks.map(c => [round2(c.r), c.look, c.theme, c.what, c.need])
    .sort((a, b) => cmp(a[0], b[0]) || cmp(a[1], b[1]) || cmp(a[2], b[2]) || cmp(a[3], b[3]) || cmp(a[4], b[4]));
}

// json.dump(..., indent=1) of {"looks": ..., "lowest": first 12, "count": n}, with no final newline.
function r7Json(result) {
  const pad = n => ' '.repeat(n);
  const str = s => JSON.stringify(s);
  const out = ['{', ' "looks": {'];
  const looks = Object.entries(result.looks);
  looks.forEach(([key, look], li) => {
    out.push(`${pad(2)}${str(key)}: {`, `${pad(3)}"label": ${str(look.label)},`, `${pad(3)}"accent": ${str(look.accent)},`);
    THEMES.forEach((theme, ti) => {
      out.push(`${pad(3)}${str(theme)}: {`);
      const keys = R7_KEYS(look[theme]);
      keys.forEach((k, ki) => out.push(`${pad(4)}${str(k)}: ${str(look[theme][k])}${ki < keys.length - 1 ? ',' : ''}`));
      out.push(`${pad(3)}}${ti < THEMES.length - 1 ? ',' : ''}`);
    });
    out.push(`${pad(2)}}${li < looks.length - 1 ? ',' : ''}`);
  });
  out.push(' },', ' "lowest": [');
  const lowest = lowestPairs(result.checks).slice(0, 12);
  lowest.forEach(([r, look, theme, what, need], i) => {
    out.push(`${pad(2)}[`, `${pad(3)}${pyFloat(r)},`, `${pad(3)}${str(look)},`, `${pad(3)}${str(theme)},`, `${pad(3)}${str(what)},`, `${pad(3)}${pyFloat(need)}`, `${pad(2)}]${i < lowest.length - 1 ? ',' : ''}`);
  });
  out.push(' ],', ` "count": ${result.checks.length}`, '}');
  return out.join('\n');
}

// ---------------------------------------------------------------------------------------------
// The production file. Names take the --vn- prefix and the theme attribute is the app's own
// data-gyst-theme (13B "Token tools for R1").
// ---------------------------------------------------------------------------------------------
function neutralLines(t) {
  const sections = SECTIONS.map(s => `--vn-s-${s}: ${t['s-' + s]}; --vn-s-${s}-t: ${t['s-' + s + '-t']}; --vn-s-${s}-s: ${t['s-' + s + '-s']};`);
  return [
    `--vn-canvas: ${t.canvas}; --vn-surface: ${t.surface}; --vn-surface-2: ${t.surface2}; --vn-surface-3: ${t.surface3}; --vn-elevated: ${t.elevated};`,
    `--vn-ink: ${t.ink}; --vn-ink-2: ${t.ink2}; --vn-ink-3: ${t.ink3}; --vn-separator: ${t.separator};`,
    `--vn-glass: ${t.glass};`,
    `--vn-ok-text: ${t['ok-text']}; --vn-warn-text: ${t['warn-text']}; --vn-crit-text: ${t['crit-text']};`,
    ...sections,
  ];
}
function accentLine(a, theme) {
  const text = theme === 'light' ? a.light.text : a.dark.fill;
  return `--vn-accent: ${a[theme].fill}; --vn-accent-text: ${text}; --vn-accent-soft: ${a[theme].soft}; --vn-on-accent: ${ON_ACCENT[theme]};`;
}
const block = (selector, lines, indent = '') => `${indent}${selector} {\n${lines.map(l => `${indent}  ${l}`).join('\n')}\n${indent}}`;
const DARK_MEDIA = '@media (prefers-color-scheme: dark)';
// Three blocks per set of values, the same shape ui-tokens.css uses: light, dark when the OS is dark
// and nothing is chosen (no JS), and dark when it is chosen.
function themed(light, systemDark, dark, lines) {
  return [block(light, lines('light')), `${DARK_MEDIA} {\n${block(systemDark, lines('dark'), '  ')}\n}`, block(dark, lines('dark'))];
}

function accentsCss(seeds, result) {
  const dl = seeds.defaultLook;
  const withAccent = key => theme => neutralLines(result.looks[key][theme]).concat(accentLine(seeds.accents[seeds.looks[key].accent], theme));
  const out = [
    '/* GENERATED by scripts/build-accents.cjs. Do not edit: change the seeds there and run it again.',
    `   Look and accent tokens (13B DD-6, DD-7, DD-8, DD-13). ${result.checks.length} contrast pairs asserted at AA, plus ${result.extra.length} for the aliases.`,
    '   GYST-169: no page links this file yet. GYST-168 links it and points the older tokens here. */',
    '',
    `/* With no Look chosen: ${seeds.looks[dl].label} and its starting accent, ${seeds.looks[dl].accent}.`,
    '   The dark values are also on a plain [data-gyst-theme="dark"], with no :root: the sign-in page',
    '   stays dark through that attribute alone, without JS (GYST-168), and the UI gallery pins panels with it. */',
    ...themed(':root, [data-gyst-theme="light"]', ':root:not([data-gyst-theme="light"])', '[data-gyst-theme="dark"]', withAccent(dl)),
    '',
    '/* A chosen Look, each with its starting accent. Nothing sets data-look yet (Settings, R5). */',
  ];
  for (const key of Object.keys(seeds.looks)) {
    const r = `:root[data-look="${key}"]`;
    out.push(...themed(`${r}, ${r} [data-gyst-theme="light"]`, `${r}:not([data-gyst-theme="light"])`, `${r}[data-gyst-theme="dark"], ${r} [data-gyst-theme="dark"]`, withAccent(key)));
  }
  out.push('', '/* A chosen accent. These come after the Looks and match their specificity, so the choice wins',
    '   over a Look\'s starting accent. Nothing sets data-accent yet (Settings, R5). */');
  for (const [name, a] of Object.entries(seeds.accents)) {
    const r = `:root[data-accent="${name}"]`;
    out.push(...themed(`${r}, ${r} [data-gyst-theme="light"]`, `${r}:not([data-gyst-theme="light"])`, `${r}[data-gyst-theme="dark"], ${r} [data-gyst-theme="dark"]`, theme => [accentLine(a, theme)]));
  }
  return out.join('\n') + '\n';
}

// ---------------------------------------------------------------------------------------------
// Input checking (exit 2). A seeds file has the same shape as SEEDS above.
// ---------------------------------------------------------------------------------------------
function validateSeeds(s) {
  const bad = m => { throw new BadInput('seeds: ' + m); };
  const obj = (v, name) => { if (!v || typeof v !== 'object' || Array.isArray(v)) bad(name + ' must be an object'); return v; };
  const num = (v, name, lo, hi) => { if (typeof v !== 'number' || !Number.isFinite(v) || v < lo || v > hi) bad(`${name} must be a number from ${lo} to ${hi}`); };
  const hex = (v, name) => { if (typeof v !== 'string' || !/^#[0-9A-Fa-f]{6}$/.test(v)) bad(name + ' must be a colour like #1A2B3C'); };
  const id = (v, name) => { if (!/^[a-z][a-z0-9-]*$/.test(v)) bad(`${name} "${v}" must be lower-case letters, digits or hyphens`); }; // it becomes part of a CSS selector
  obj(s, 'the file');
  const accents = obj(s.accents, 'accents'), looks = obj(s.looks, 'looks'), state = obj(s.state, 'state');
  if (!Object.keys(accents).length) bad('accents is empty');
  for (const [name, a] of Object.entries(accents)) {
    id(name, 'accent name');
    obj(a, 'accents.' + name); obj(a.light, `accents.${name}.light`); obj(a.dark, `accents.${name}.dark`);
    for (const k of ['fill', 'text', 'soft']) hex(a.light[k], `accents.${name}.light.${k}`);
    for (const k of ['fill', 'soft']) hex(a.dark[k], `accents.${name}.dark.${k}`);
  }
  if (!Object.keys(looks).length) bad('looks is empty');
  for (const [key, L] of Object.entries(looks)) {
    id(key, 'look name');
    obj(L, 'looks.' + key);
    if (typeof L.label !== 'string' || !L.label) bad(`looks.${key}.label must be text`);
    num(L.hue, `looks.${key}.hue`, 0, 360);
    for (const k of ['satLight', 'satDark', 'sectionSat']) num(L[k], `looks.${key}.${k}`, 0, 1);
    num(L.canvasDark, `looks.${key}.canvasDark`, 0, 0.5);
    if (!Array.isArray(L.sectionHues) || L.sectionHues.length !== SECTIONS.length) bad(`looks.${key}.sectionHues must list ${SECTIONS.length} hues (${SECTIONS.join(', ')})`);
    L.sectionHues.forEach((h, i) => num(h, `looks.${key}.sectionHues[${i}]`, 0, 360));
    if (!Object.hasOwn(accents, L.accent)) bad(`looks.${key}.accent "${L.accent}" is not one of the accents`);
  }
  if (!Object.hasOwn(looks, s.defaultLook)) bad(`defaultLook "${s.defaultLook}" is not one of the looks`);
  for (const theme of THEMES) {
    obj(state[theme], 'state.' + theme);
    for (const k of ['ok', 'warn', 'crit', 'warnsoft', 'critsoft']) hex(state[theme][k], `state.${theme}.${k}`);
  }
  return s;
}

function parseArgs(argv) {
  const o = { check: false, seeds: null, out: null, r7: null };
  for (let i = 0; i < argv.length; i++) {
    const a = argv[i];
    if (a === '--check') { o.check = true; continue; }
    if (a === '--seeds' || a === '--out' || a === '--r7') {
      const v = argv[++i];
      if (v === undefined || v.startsWith('--')) throw new BadInput(a + ' needs a path');
      o[a.slice(2)] = v;
      continue;
    }
    throw new BadInput('unknown argument ' + a);
  }
  if (o.check && (o.seeds || o.out || o.r7)) throw new BadInput('--check takes no other argument');
  return o;
}

function loadSeeds(file) {
  let text;
  try { text = fs.readFileSync(file, 'utf8'); } catch (e) { throw new BadInput('cannot read ' + file + ': ' + e.code); }
  let data;
  try { data = JSON.parse(text); } catch (e) { throw new BadInput(file + ' is not JSON: ' + e.message); }
  return validateSeeds(data);
}

// ---------------------------------------------------------------------------------------------
// Running.
// ---------------------------------------------------------------------------------------------
const fmt = c => `${c.look} ${c.theme}: ${c.what} is ${c.r.toFixed(2)}:1, needs ${c.need}:1`;

// Returns the result, or null after printing why a pair cannot hold.
function run(seeds, log = console.error) {
  let result;
  try { result = derive(seeds); } catch (e) {
    if (!(e instanceof NoColour)) throw e;
    log('FAIL ' + e.message);
    return null;
  }
  const failed = result.checks.concat(result.extra).filter(c => !c.ok);
  for (const c of failed) log('FAIL ' + fmt(c));
  return failed.length ? null : result;
}

function summary(result) {
  const low = result.checks.concat(result.extra).reduce((a, b) => (b.r < a.r ? b : a));
  return `${result.checks.length} pairs asserted, plus ${result.extra.length} for the aliases; lowest: ${fmt(low).replace(/, needs.*/, '')}`;
}

const sha256 = s => crypto.createHash('sha256').update(s).digest('hex');
const clone = v => JSON.parse(JSON.stringify(v));

// One rule's declarations from the generated CSS, by its exact selector at the top level.
function topLevelRule(css, selector) {
  let depth = 0, start = 0;
  for (let i = 0; i < css.length; i++) {
    if (css[i] === '{') {
      if (depth === 0 && css.slice(start, i).replace(/\/\*[\s\S]*?\*\//g, '').trim() === selector) return css.slice(i + 1, css.indexOf('}', i));
      depth++;
    } else if (css[i] === '}') { depth--; start = i + 1; }
  }
  return null;
}

function check() {
  const problems = [];
  const expect = (ok, message) => { if (!ok) problems.push(message); };
  const result = run(SEEDS);
  if (!result) return 1;
  const css = accentsCss(SEEDS, result);

  // 1. Byte-identical to the Python tool for the round 7 seeds.
  const fx = JSON.parse(fs.readFileSync(FIXTURE_FILE, 'utf8'));
  const wantCss = fx.css.join('\n') + '\n', gotCss = r7Css(result), gotJson = r7Json(result);
  expect(sha256(wantCss) === fx.sha256['looks7.css'], 'the fixture\'s css lines do not match its own looks7.css hash');
  expect(gotCss === wantCss, 'the round 7 CSS differs from the Python tool\'s (scripts/fixtures/looks-r7.json, css)');
  expect(sha256(gotJson) === fx.sha256['looks7.json'], 'the round 7 JSON differs from the Python tool\'s (fixture sha256, looks7.json)');
  expect(JSON.stringify(JSON.parse(gotJson)) === JSON.stringify(fx.json), 'the round 7 values differ from the Python tool\'s (fixture json)');
  expect(result.checks.length === fx.json.count, `asserted ${result.checks.length} pairs, the Python tool asserts ${fx.json.count}`);

  // 2. The committed file is this generator's output.
  const committed = fs.existsSync(CSS_FILE) ? fs.readFileSync(CSS_FILE, 'utf8') : null;
  expect(committed === css, 'web/static/css/ui-accents.css is not the generator\'s output; run node scripts/build-accents.cjs');

  // 3. Dark values on the plain attribute selector (Issue #259 lead note): sign-in depends on it.
  const dark = topLevelRule(css, '[data-gyst-theme="dark"]');
  const want = neutralLines(result.looks[SEEDS.defaultLook].dark).concat(accentLine(SEEDS.accents[SEEDS.looks[SEEDS.defaultLook].accent], 'dark'));
  expect(dark !== null, 'no plain [data-gyst-theme="dark"] rule');
  if (dark !== null) for (const line of want) expect(dark.includes(line), 'the plain [data-gyst-theme="dark"] rule lacks: ' + line);

  // 4. The assertion can fail. A lighter Iris fill leaves white text on it under 4.5:1.
  const broken = clone(SEEDS);
  broken.accents.iris.light.fill = '#9FA1F7';
  const said = [];
  expect(run(broken, m => said.push(m)) === null && said.some(m => m.includes('on-accent on accent (iris)')), 'a contrast-breaking accent was not reported');

  // 5. Exit codes, as a caller sees them.
  const tmp = fs.mkdtempSync(path.join(require('node:os').tmpdir(), 'gyst-accents-'));
  try {
    const self = (...args) => spawnSync(process.execPath, [__filename, ...args], { encoding: 'utf8' });
    const write = (name, text) => { const f = path.join(tmp, name); fs.writeFileSync(f, text); return f; };
    const good = self('--seeds', write('r7.json', JSON.stringify(SEEDS)), '--out', path.join(tmp, 'r7.css'));
    expect(good.status === 0, 'exit ' + good.status + ' on the round 7 seeds, want 0');
    expect(good.status !== 0 || fs.readFileSync(path.join(tmp, 'r7.css'), 'utf8') === css, 'the round 7 seeds read from a file give different CSS');
    const fail = self('--seeds', write('broken.json', JSON.stringify(broken)), '--out', path.join(tmp, 'broken.css'));
    expect(fail.status === 1, 'exit ' + fail.status + ' on a contrast-breaking accent, want 1');
    expect(/FAIL aurora light: on-accent on accent \(iris\) is \d\.\d\d:1, needs 4\.5:1/.test(fail.stderr), 'the failing pair was not named');
    expect(!fs.existsSync(path.join(tmp, 'broken.css')), 'CSS was written although a pair failed');
    // A Look whose seed cannot be solved at all is also a contrast failure, not a crash.
    const unsolvable = clone(SEEDS);
    unsolvable.state.light.warnsoft = '#777777';
    expect(self('--seeds', write('unsolvable.json', JSON.stringify(unsolvable))).status === 1, 'an unsolvable seed did not exit 1');
    const noAccent = clone(SEEDS);
    noAccent.looks.aurora.accent = 'mauve';
    const badHex = clone(SEEDS);
    badHex.accents.iris.dark.fill = 'purple';
    for (const [label, args] of [
      ['an unknown argument', ['--nope']],
      ['--seeds without a path', ['--seeds']],
      ['--check with another argument', ['--check', '--out', path.join(tmp, 'x.css')]],
      ['a missing seeds file', ['--seeds', path.join(tmp, 'missing.json')]],
      ['a seeds file that is not JSON', ['--seeds', write('not.json', '{')]],
      ['a Look naming an unknown accent', ['--seeds', write('noaccent.json', JSON.stringify(noAccent))]],
      ['a colour that is not hex', ['--seeds', write('badhex.json', JSON.stringify(badHex))]],
    ]) {
      const r = self(...args);
      expect(r.status === 2, `exit ${r.status} on ${label}, want 2`);
    }
  } finally { fs.rmSync(tmp, { recursive: true, force: true }); }

  for (const p of problems) console.error('FAIL ' + p);
  if (problems.length) return 1;
  console.log('build-accents: ' + summary(result));
  console.log('build-accents: byte-identical to gen_looks7.py for the round 7 seeds; ui-accents.css is current; exit codes 0, 1 and 2 hold');
  return 0;
}

function main(argv) {
  let opts, seeds;
  try {
    opts = parseArgs(argv);
    seeds = opts.seeds ? loadSeeds(opts.seeds) : SEEDS;
  } catch (e) {
    if (!(e instanceof BadInput)) throw e;
    console.error('build-accents: ' + e.message);
    console.error('usage: node scripts/build-accents.cjs [--check] [--seeds file.json] [--out file.css] [--r7 dir]');
    return 2;
  }
  if (opts.check) return check();
  const result = run(seeds);
  if (!result) return 1;
  // Other seeds are a trial: they never replace the committed file unless --out names it.
  const out = opts.out || (opts.seeds ? null : CSS_FILE);
  if (out) fs.writeFileSync(out, accentsCss(seeds, result));
  if (opts.r7) {
    fs.mkdirSync(opts.r7, { recursive: true });
    fs.writeFileSync(path.join(opts.r7, 'looks7.css'), r7Css(result));
    fs.writeFileSync(path.join(opts.r7, 'looks7.json'), r7Json(result));
  }
  console.log('build-accents: ' + summary(result) + (out ? '; wrote ' + path.relative(process.cwd(), out) : ''));
  return 0;
}

if (require.main === module) process.exitCode = main(process.argv.slice(2));
module.exports = { SEEDS, derive, r7Css, r7Json, accentsCss, validateSeeds };
