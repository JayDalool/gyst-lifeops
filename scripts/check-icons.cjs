#!/usr/bin/env node
// Icon sprite check (GYST-169, Issue #259; 13B DD-15). No dependencies or build step.
// Run: node scripts/check-icons.cjs
//
// Why: the sprite is loaded with <use href="/static/icons.svg#i-name">, so a missing symbol draws
// nothing and nothing reports it. This fails the build instead. It also keeps the file inert:
// an SVG can carry script and outside references, and this one must never do either.
'use strict';
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');

const ROOT = path.join(__dirname, '..');
const SPRITE = path.join(ROOT, 'web/static/icons.svg');
const TEMPLATES = path.join(ROOT, 'web/templates');

// The symbols shipped so far, in round 7's order: the round 7 drawing for each icon the app uses
// (the name on the right is the Lucide name in ui_icons.html, where one exists). 13B DD-15 lets a
// chunk ship only the symbols in use; each adds its own from the same round 7 set, here and in the
// sprite. Today's waves, bike, arrow-left and circle-dashed have no round 7 drawing.
const IDS = [
  'search',  // search
  'check',   // check
  'plus',    // plus
  'x',       // x
  'home',    // house
  'folder',  // folder-heart
  'heart',   // heart
  'book',    // book-open
  'more',    // ellipsis
  'camera',  // camera
  'flame',   // flame
  'steps',   // footprints
  'chev',    // chevron-right
  'sun',     // sun
  'gym',     // dumbbell
  'clock',   // clock, timer
  'drop',    // droplet
  'gear',    // settings
  'bell',    // bell
  'repeat',  // refresh-cw
  'alert',   // circle-alert, triangle-alert
  'offline', // wifi-off
  'ship',    // rocket
  'tasks',   // GYST-178: the Tasks tab of the phone tab bar (13B DD-18)
  'sec',     // GYST-174: the Coach row on Health, as round 7 draws it (today's message-circle elsewhere)
  'scale',   // weight
  'trend',   // GYST-174: the Progress tab of the Health switcher (no Lucide twin in use)
  'fork',    // utensils
  'target',  // GYST-174: the Plan row on Health (no Lucide twin in use)
  'apple',   // apple
  'inbox',   // inbox
  'info',    // info
  'ext',     // external-link
  'lock',    // lock
  'undo',    // undo-2
  'all',     // GYST-178: the All tab of the phone tab bar, a 3x3 grid of dots (13B DD-18)
  'bars',    // chart-column
  'orbit',   // GYST-186: the Galaxy tab of the Tasks hub switch
  'flag',    // GYST-187: the Goals tile of All pages (round 7's drawing)
  'cal',     // GYST-187: Monthly view
  'user',    // GYST-187: Coach profile
  'spark',   // GYST-187: Coach
  'upload',  // GYST-187: Import a plan
  'box',     // GYST-187: Archive
  'out',     // GYST-188: the signed-out banner of the sign-in page (round 7's drawing)
  'eye',     // GYST-188: the show-password button (round 7's drawing)
];

// Round 7's sprite uses only these. Anything else (script, image, use, foreignObject, style, a, ...)
// is refused, and so is any attribute that can run code or point somewhere: href, style, on*.
const ELEMENTS = new Set(['path', 'circle', 'rect', 'ellipse']);
const ATTRIBUTES = new Set(['d', 'cx', 'cy', 'r', 'rx', 'ry', 'x', 'y', 'width', 'height', 'fill', 'stroke-width', 'transform']);
const VALUE = /^[A-Za-z0-9 .,()-]*$/; // numbers, path letters, rotate(...), currentColor; no url(), no quotes, no ':'

function checkSprite(text, ids = IDS) {
  const problems = [];
  const body = text.replace(/^\s*<!--[^>]*-->\s*/, ''); // one leading comment is allowed
  const m = /^<svg xmlns="http:\/\/www\.w3\.org\/2000\/svg">\n([\s\S]*)\n<\/svg>\n?$/.exec(body);
  if (!m) return ['the sprite must be one <svg xmlns="http://www.w3.org/2000/svg"> element, with nothing around it but a first comment'];
  const found = [];
  for (const line of m[1].split('\n')) {
    const s = /^\s*<symbol id="i-([a-z0-9-]+)" viewBox="([^"]*)">(.*)<\/symbol>$/.exec(line);
    if (!s) { problems.push('not a round 7 symbol line: ' + line.trim().slice(0, 80)); continue; }
    const [, id, viewBox, inner] = s;
    found.push(id);
    if (viewBox !== '0 0 24 24') problems.push(`i-${id}: viewBox is "${viewBox}", must be "0 0 24 24"`);
    const rest = inner.replace(/<([a-zA-Z][\w:-]*)((?:\s+[\w:-]+="[^"]*")*)\s*\/>/g, (_, tag, attrs) => {
      if (!ELEMENTS.has(tag)) problems.push(`i-${id}: <${tag}> is not allowed`);
      for (const [, name, value] of attrs.matchAll(/\s+([\w:-]+)="([^"]*)"/g)) {
        if (!ATTRIBUTES.has(name)) problems.push(`i-${id}: attribute ${name} is not allowed`);
        else if (!VALUE.test(value)) problems.push(`i-${id}: ${name}="${value.slice(0, 40)}" is not a plain value`);
      }
      return '';
    });
    if (rest.trim()) problems.push(`i-${id}: only self-closed shapes are allowed, found: ${rest.trim().slice(0, 60)}`);
  }
  const dup = found.filter((id, i) => found.indexOf(id) !== i);
  if (dup.length) problems.push('symbol defined twice: ' + [...new Set(dup)].join(', '));
  const missing = ids.filter(id => !found.includes(id)), extra = found.filter(id => !ids.includes(id));
  if (missing.length) problems.push('missing from the sprite: ' + missing.join(', '));
  if (extra.length) problems.push('in the sprite but not in this check\'s list: ' + extra.join(', '));
  if (!missing.length && !extra.length && !dup.length && found.join() !== ids.join()) problems.push('symbols are not in the list\'s order');
  return problems;
}

// files: {name: text}. Every use of the helper must name a symbol that exists.
function checkTemplates(files, ids = IDS) {
  const problems = [];
  const helper = files['ui_icons.html'] || '';
  const def = /\{\{define "vn\.icon"\}\}([\s\S]*?)\{\{end\}\}/.exec(helper);
  if (!def) problems.push('ui_icons.html does not define "vn.icon"');
  // Same-origin: one leading slash means this site, whatever its host is. The version makes a
  // changed sprite a new URL, as every other static file here does.
  else if (!/<use href="\/static\/icons\.svg\?v=[\w.-]+#i-\{\{\.\}\}"\/>/.test(def[1])) problems.push('vn.icon must use href="/static/icons.svg?v=<version>#i-{{.}}"');
  for (const [name, text] of Object.entries(files)) {
    for (const [, arg] of text.matchAll(/\{\{-?\s*template\s+"vn\.icon"\s*([^}]*?)\s*-?\}\}/g)) {
      const literal = /^"([^"]*)"$/.exec(arg);
      if (literal) { if (!ids.includes(literal[1])) problems.push(`${name}: vn.icon "${literal[1]}" is not in the sprite`); }
      // The gallery lists the sprite itself (ui_gallery_test.go reads the ids from icons.svg), so a
      // name there cannot be wrong. Anywhere else a computed name could not be checked here.
      else if (name !== 'ui_gallery.html') problems.push(`${name}: vn.icon needs a literal name, found ${arg || 'nothing'}`);
    }
    // The sprite has one way in. A hand-written reference would skip this check and the version.
    // GYST-188: login.html is parsed alone by renderLogin (the sign-in handler is not changed to load the helper),
    // so it names the sprite itself; each reference must carry the helper's own version and a symbol that exists.
    if (name === 'login.html') {
      const version = def && /icons\.svg\?v=([\w.-]+)#/.exec(def[1]);
      for (const [, v, id] of text.matchAll(/icons\.svg\?v=([\w.-]+)#i-([a-z0-9-]+)/g)) {
        if (!version || v !== version[1]) problems.push(`${name}: sprite version ${v} is not the helper's ${version && version[1]}`);
        if (!ids.includes(id)) problems.push(`${name}: sprite symbol "${id}" is not in the sprite`);
      }
      if ((text.match(/icons\.svg/g) || []).length !== (text.match(/icons\.svg\?v=[\w.-]+#i-[a-z0-9-]+/g) || []).length) problems.push(`${name}: every sprite reference needs a version and a symbol`);
    } else if (name !== 'ui_icons.html' && /icons\.svg[?#]/.test(text)) problems.push(`${name}: reference the sprite through {{template "vn.icon" "name"}} only`);
  }
  return problems;
}

function main() {
  const sprite = fs.readFileSync(SPRITE, 'utf8');
  const files = Object.fromEntries(fs.readdirSync(TEMPLATES).filter(f => f.endsWith('.html')).map(f => [f, fs.readFileSync(path.join(TEMPLATES, f), 'utf8')]));
  const problems = checkSprite(sprite).concat(checkTemplates(files));

  // The check must be able to fail: each broken copy below has to be caught.
  const caught = (list, re, what) => { if (!list.some(p => re.test(p))) problems.push('self-test: ' + what + ' was not caught'); };
  const one = '  <symbol id="i-check" viewBox="0 0 24 24"><path d="m5 12.5 4.5 4.5L19 7.5"/></symbol>';
  assert.ok(sprite.includes(one + '\n'), 'self-test needs the i-check line as round 7 wrote it');
  const swap = to => sprite.replace(one, to);
  caught(checkSprite(sprite.replace(one + '\n', '')), /missing from the sprite: check/, 'a removed symbol');
  caught(checkSprite(swap(one.replace('i-check', 'i-tick'))), /not in this check's list: tick/, 'an unlisted symbol');
  caught(checkSprite(swap(one.replace('0 0 24 24', '0 0 20 20'))), /viewBox/, 'a wrong viewBox');
  caught(checkSprite(swap(one.replace('<path d=', '<path onclick="x" d='))), /attribute onclick/, 'an event attribute');
  caught(checkSprite(swap(one.replace('<path d=', '<path style="x" d='))), /attribute style/, 'a style attribute');
  caught(checkSprite(swap(one.replace('<path ', '<path fill="url(https://example.test/x)" '))), /not a plain value/, 'an outside reference in a value');
  caught(checkSprite(swap(one.replace('</symbol>', '<use href="https://example.test/x.svg#a"/></symbol>'))), /<use> is not allowed/, 'a <use> element');
  caught(checkSprite(swap(one.replace('</symbol>', '<script>1</script></symbol>'))), /only self-closed shapes/, 'a script element');
  caught(checkSprite(sprite.replace('</svg>', '</svg><script>1</script>')), /one <svg/, 'content after the sprite');
  const good = { 'ui_icons.html': files['ui_icons.html'] };
  caught(checkTemplates({ ...good, 'x.html': '{{template "vn.icon" "nope"}}' }), /"nope" is not in the sprite/, 'an unknown icon name');
  caught(checkTemplates({ ...good, 'x.html': '{{template "vn.icon" .Icon}}' }), /needs a literal name/, 'a computed icon name');
  caught(checkTemplates({ ...good, 'x.html': '<use href="/static/icons.svg#i-check"/>' }), /through \{\{template/, 'a hand-written sprite reference');
  caught(checkTemplates({ 'ui_icons.html': (files['ui_icons.html'] || '').replace('/static/icons.svg', 'https://example.test/icons.svg') }), /must use href/, 'a sprite on another origin');
  assert.deepEqual(checkTemplates({ ...good, 'x.html': '{{template "vn.icon" "check"}}' }), [], 'self-test: a known icon name was refused');
  caught(checkTemplates({ ...good, 'login.html': '<use href="/static/icons.svg?v=old-1#i-eye"/>' }), /is not the helper's/, 'a stale sprite version on the sign-in page');
  caught(checkTemplates({ ...good, 'login.html': '<use href="/static/icons.svg?v=r7-6#i-nope"/>' }), /"nope" is not in the sprite/, 'an unknown symbol on the sign-in page');
  assert.deepEqual(checkTemplates({ ...good, 'login.html': '<use href="/static/icons.svg?v=r7-6#i-eye"/>' }), [], 'self-test: the sign-in page reference was refused');

  for (const p of problems) console.error('FAIL ' + p);
  if (problems.length) return 1;
  console.log(`check-icons: ${IDS.length} symbols, same-origin, no script or outside reference; every vn.icon names one`);
  return 0;
}

if (require.main === module) process.exitCode = main();
module.exports = { IDS, checkSprite, checkTemplates };
