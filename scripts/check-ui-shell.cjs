// No dependencies or build step. Run: node scripts/check-ui-shell.cjs
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
const worker = fs.readFileSync('web/static/service-worker.js', 'utf8');
async function workerChecks() {
  const handlers = {}, cached = new Map(), removed = [];
  let online = true, networkResponse = () => new Response('network', {headers: {'Content-Type':'text/css'}});
  const context = {
    URL, Response, Promise,
    self: {location:{origin:'https://gyst.test'}, addEventListener:(type,fn)=>handlers[type]=fn, skipWaiting:async()=>{}, clients:{claim:async()=>{}}},
    caches: {
      keys:async()=>['lifeops-shell-v29-ui-0','unrelated-cache'],
      delete:async key=>removed.push(key),
      open:async()=>({addAll:async paths=>{for (const path of paths) {assert(path.startsWith('/static/'));assert(!path.includes('three.'));assert(!path.includes('galaxy.js'));}},put:async(req,res)=>cached.set(req.url,await res.text())}),
      match:async req=>cached.has(req.url)?new Response(cached.get(req.url)):undefined
    },
    fetch:async()=>{if (!online) throw Error('offline');const r=networkResponse();Object.defineProperty(r,'type',{value:'basic'});return r;}
  };
  vm.runInNewContext(worker, context);
  await new Promise((resolve,reject)=>handlers.install({waitUntil:p=>p.then(resolve,reject)}));
  await new Promise((resolve,reject)=>handlers.activate({waitUntil:p=>p.then(resolve,reject)}));
  assert.deepEqual(removed,['lifeops-shell-v29-ui-0']);
  async function request(path,{mode='navigate',method='GET',authorization=false}={}) {
    let response;
    handlers.fetch({request:{url:'https://gyst.test'+path,mode,method,headers:new Headers(authorization?{Authorization:'Bearer test-fixture'}:{})},respondWith:p=>response=p});
    return response;
  }
  for (const path of ['/today','/health','/health/plan','/more','/settings','/login','/logout']) {
    await request(path);assert.equal(cached.size,0,'private navigation cached');
    online=false;const response=await request(path);assert.equal(response.status,503);assert.match(await response.text(),/You’re offline/);online=true;
  }
  assert.equal(await request('/api/health/sync',{mode:'cors'}),undefined);
  assert.equal(await request('/api/health/sync',{mode:'cors',method:'POST',authorization:true}),undefined);
  assert.equal(await request('/static/js/app.js',{mode:'cors',authorization:true}),undefined);
  await request('/static/css/ui-shell.css',{mode:'cors'});assert.equal(cached.size,1);
  online=false;assert.equal(await (await request('/static/css/ui-shell.css',{mode:'cors'})).text(),'network');online=true;
  networkResponse=()=>new Response('private',{headers:{'Content-Type':'text/html'}});
  await request('/static/not-an-asset',{mode:'cors'});assert.equal(cached.size,1,'HTML response cached');
  networkResponse=()=>{const r=new Response('redirected');Object.defineProperty(r,'redirected',{value:true});return r;};
  await request('/static/redirect',{mode:'cors'});assert.equal(cached.size,1,'redirect cached');
  networkResponse=()=>new Response('private',{headers:{'Cache-Control':'no-store'}});
  await request('/static/no-store',{mode:'cors'});assert.equal(cached.size,1,'no-store response cached');
}
// Appearance (#42, DD-1): stored system/light/dark, missing or unreadable → system, System follows the OS live.
function themeChecks() {
  const code=fs.readFileSync('web/static/js/theme.js','utf8');
  function run(initial,denied,osDark,legacyMedia) {
    const state={stored:initial,events:{},changes:0,mediaChange:null};
    const choices=['system','light','dark'].map(value=>({dataset:{themeChoice:value},attrs:{},listeners:{},setAttribute(k,v){this.attrs[k]=v;},addEventListener(n,fn){this.listeners[n]=fn;}}));
    const media={matches:osDark};
    if(legacyMedia)media.addListener=fn=>state.mediaChange=fn;else media.addEventListener=(name,fn)=>{assert.equal(name,'change');state.mediaChange=fn;};
    state.root={dataset:{}};state.meta={content:''};state.choices=choices;state.media=media;
    vm.runInNewContext(code,{
      document:{documentElement:state.root,querySelector:()=>state.meta,querySelectorAll:sel=>sel==='[data-theme-choice]'?choices:[],dispatchEvent:event=>{assert.equal(event.type,'gyst:themechange');state.changes++;},addEventListener:(name,fn)=>state.events[name]=fn},
      localStorage:{getItem:key=>{if(denied)throw Error('denied');return key==='lifeops-theme'?state.stored:null;},setItem:(key,value)=>{if(denied)throw Error('denied');state.stored=value;}},
      window:{addEventListener:(name,fn)=>state.events[name]=fn,matchMedia:query=>{assert.equal(query,'(prefers-color-scheme: dark)');return media;}},Event:class{constructor(type){this.type=type;}}
    });
    return state;
  }
  function expectTheme(state,theme,mode) {
    assert.equal(state.root.dataset.gystTheme,theme);
    assert.equal(state.root.dataset.theme,theme==='light'?'light':undefined,'legacy dark relies on absent data-theme');
    assert.equal(state.meta.content,theme==='dark'?'#0f172a':'#f8fafc');
    if(mode)for(const c of state.choices)assert.equal(c.attrs['aria-pressed'],String(c.dataset.themeChoice===mode));
  }
  const choice=(state,value)=>state.choices.find(c=>c.dataset.themeChoice===value).listeners.click();
  const flipOS=state=>{state.media.matches=!state.media.matches;state.mediaChange();};
  for(const initial of [null,'light','dark','system','invalid']) {
    for(const denied of [false,true]) {
      for(const osDark of [false,true]) {
        for(const legacyMedia of [false,true]) {
          const s=run(initial,denied,osDark,legacyMedia);
          const mode=!denied && (initial==='light'||initial==='dark')?initial:'system';
          const resolved=m=>m==='system'?(s.media.matches?'dark':'light'):m;
          expectTheme(s,resolved(mode));
          assert.ok(s.changes>=1,'gyst:themechange fired on load');
          s.events.DOMContentLoaded();expectTheme(s,resolved(mode),mode);
          // A live OS change moves System only; a manual choice ignores it.
          flipOS(s);expectTheme(s,resolved(mode),mode);
          choice(s,'dark');expectTheme(s,'dark','dark');if(!denied)assert.equal(s.stored,'dark');
          flipOS(s);expectTheme(s,'dark','dark');
          choice(s,'light');expectTheme(s,'light','light');if(!denied)assert.equal(s.stored,'light');
          flipOS(s);expectTheme(s,'light','light');
          choice(s,'system');expectTheme(s,resolved('system'),'system');if(!denied)assert.equal(s.stored,'system');
          flipOS(s);expectTheme(s,resolved('system'),'system');
          // Cross-tab sync for every value, a cleared key and localStorage.clear().
          for(const [key,value,next] of [['lifeops-theme','light','light'],['lifeops-theme','dark','dark'],['lifeops-theme','system','system'],['lifeops-theme','dark','dark'],['lifeops-theme',null,'system'],['lifeops-theme','dark','dark'],[null,null,'system']]) {
            s.events.storage({key,newValue:value});expectTheme(s,resolved(next),next);
          }
          const before=s.root.dataset.gystTheme;s.events.storage({key:'lifeops-motion',newValue:'light'});assert.equal(s.root.dataset.gystTheme,before);
        }
      }
    }
  }
}
function motionChecks() {
  const code=fs.readFileSync('web/static/js/theme.js','utf8');
  for(const [value,denied,expected] of [['reduced',false,'reduced'],['',false,undefined],[null,false,undefined],['reduced',true,undefined]]) {
    const root={dataset:{}};
    vm.runInNewContext(code,{
      document:{documentElement:root,querySelector:()=>({}),querySelectorAll:()=>[],dispatchEvent:()=>{},addEventListener:()=>{}},
      localStorage:{getItem:key=>{if(denied)throw Error('denied');return key==='lifeops-motion'?value:null;},setItem:()=>{}},
      window:{addEventListener:()=>{}},Event:class{}
    });
    assert.equal(root.dataset.motion,expected);
  }
}
// Completion toast (#25): parse and build run in a fake DOM that records any innerHTML write.
function toastChecks() {
  const code=fs.readFileSync('web/static/js/completion-toast.js','utf8');
  const innerHTMLWrites=[];
  class Node {
    constructor(tag){this.tag=tag;this.children=[];this.attrs={};this.listeners={};this.className='';this._text='';this.classList={add:()=>{},remove:()=>{}};}
    set innerHTML(v){innerHTMLWrites.push(v);}
    set textContent(v){this._text=String(v);}
    get textContent(){return this._text+this.children.map(c=>c.textContent).join('');}
    setAttribute(k,v){this.attrs[k]=v;}
    getAttribute(k){return this.attrs[k];}
    appendChild(c){this.children.push(c);return c;}
    addEventListener(n,f){this.listeners[n]=f;}
    find(pred,out=[]){if(pred(this))out.push(this);this.children.forEach(c=>c.find(pred,out));return out;}
  }
  const doc={readyState:'complete',documentElement:{},body:new Node('body'),createElement:t=>new Node(t),addEventListener:()=>{}};
  const sandbox={document:doc,location:{search:'',pathname:'/today',hash:''},history:{replaceState:()=>{}},getComputedStyle:()=>({getPropertyValue:()=>'240ms'}),requestAnimationFrame:()=>{},setTimeout:()=>0,clearTimeout:()=>{},URLSearchParams,Array,parseInt,parseFloat,isNaN};
  sandbox.window=sandbox;
  vm.runInNewContext(code,sandbox);
  const T=sandbox.GystCompletionToast;
  assert(T,'GystCompletionToast exposed');
  assert.equal(innerHTMLWrites.length,0);

  // Parse and build from redirect params.
  const q='?celebrate=task&celebrate_title=Write%20report&celebrate_xp=20&celebrate_undo=%2Ftasks%2F7%2Fundo&celebrate_date=2026-06-24&keep=1';
  const data=T.parse(q);
  assert.equal(data.kind,'task');assert.equal(data.xp,20);assert.equal(data.undo,'/tasks/7/undo');assert.equal(data.date,'2026-06-24');
  const returnTo=T.cleanUrl({pathname:'/today',search:q,hash:'#task-7'});
  assert.equal(returnTo,'/today?keep=1#task-7');
  const toast=T.build(doc,data,returnTo);
  assert.equal(toast.attrs.role,'status');assert.equal(toast.attrs['aria-live'],'polite');
  assert(toast.className.includes('completion-toast'));
  assert(toast.textContent.includes('Done · Write report'));
  assert(toast.textContent.includes('+20 XP'));
  const [form]=toast.find(n=>n.tag==='form');
  assert.equal(form.attrs.action,'/tasks/7/undo');assert.equal(form.attrs.method,'post');
  const inputs=form.find(n=>n.tag==='input');
  assert.equal(inputs.find(i=>i.name==='occurrence_date').value,'2026-06-24');
  assert.equal(inputs.find(i=>i.name==='return_to').value,'/today?keep=1#task-7');
  assert(!inputs.find(i=>i.name==='return_to').value.includes('celebrate'));
  assert.equal(form.find(n=>n.tag==='button')[0].type,'submit');

  // XSS: the title is literal text, never markup.
  const evil='<img src=x onerror=alert(1)>';
  const xss=T.build(doc,T.parse('?celebrate=task&celebrate_title='+encodeURIComponent(evil)+'&celebrate_xp=5'),'/today');
  assert(xss.textContent.includes(evil));
  assert(!xss.find(n=>n.tag==='img').length);
  assert.equal(innerHTMLWrites.length,0);

  // Undo allowlist.
  for (const bad of ['/projects/1/items/2/delete','//evil.example/tasks/1/undo','javascript:alert(1)','/tasks/1/undo?x','https://evil.example/tasks/1/undo']) {
    const d=T.parse('?celebrate=task&celebrate_title=x&celebrate_undo='+encodeURIComponent(bad));
    assert.equal(d.undo,'');
    assert.equal(T.build(doc,d,'/today').find(n=>n.tag==='form').length,0,'no form for '+bad);
  }
  for (const good of ['/projects/1/items/2/undo','/courses/3/blocks/4/undo']) assert.equal(T.parse('?celebrate=task&celebrate_undo='+encodeURIComponent(good)).undo,good);

  // XP and title handling.
  for (const v of ['0','-5','abc']) {
    assert(!T.build(doc,T.parse('?celebrate=task&celebrate_title=x&celebrate_xp='+v),'/').textContent.includes('XP'),'xp '+v);
  }
  assert.equal(T.parse('?celebrate=other'),null);
  assert.equal(T.parse(''),null);
  assert(Array.from(T.parse('?celebrate=task&celebrate_title='+'a'.repeat(100)).title).length<=61);

  // Level-up card.
  const lu=T.buildLevelUp(doc,T.parse('?celebrate=task&celebrate_title=x&celebrate_xp=100&celebrate_level=3'));
  assert(lu.className.includes('levelup-event'));assert.equal(lu.attrs.role,'status');
  assert(lu.textContent.includes('Level 3'));
  assert.equal(T.parse('?celebrate=task&celebrate_level=1').level,0);
  assert.equal(innerHTMLWrites.length,0);
}
// Tap targets (GYST-072): static CSS check, no browser. Real rendered sizes are measured in each UI PR.
const CSS_DIR = 'web/static/css/';
function parseCSS(text, media = null, out = []) {
  text = text.replace(/\/\*[\s\S]*?\*\//g, '');
  let depth = 0, start = 0, prelude = '';
  for (let i = 0; i < text.length; i++) {
    if (text[i] === '{') { if (depth++ === 0) { prelude = text.slice(start, i).trim(); start = i + 1; } }
    else if (text[i] === '}' && --depth === 0) {
      const body = text.slice(start, i); start = i + 1;
      if (/^@(media|supports|layer|container)\b/.test(prelude)) {
        const at = prelude.replace(/^@media\s*/, '').replace(/\s*:\s*/g, ': ').replace(/\s+/g, ' ').trim();
        parseCSS(body, media === null ? at : media + ' | ' + at, out);
      }
      else if (!prelude.startsWith('@')) {
        const decls = {};
        for (const d of body.split(';')) { const k = d.indexOf(':'); if (k > 0) decls[d.slice(0, k).trim().toLowerCase()] = d.slice(k + 1).replace(/!important/i, '').trim(); }
        let paren = 0, from = 0; const sels = [];
        for (let j = 0; j <= prelude.length; j++) {
          if (prelude[j] === '(') paren++; else if (prelude[j] === ')') paren--;
          else if (j === prelude.length || (prelude[j] === ',' && paren === 0)) { sels.push(prelude.slice(from, j).replace(/\s+/g, ' ').trim()); from = j + 1; }
        }
        out.push({selectors: sels, decls, media});
      }
    }
  }
  return out;
}
// [selector, media|null, checks height, checks width]
const TARGETS = [
  ['.gyst-primary > a', null, true, false], ['.gyst-primary > a', '(max-width: 1023px)', true, false],
  ['.gyst-health-sub > a', '(min-width: 1024px)', true, false], ['.hv3-tabbar > a', '(max-width: 1023px)', true, false],
  ['.gyst-brand', null, true, false], ['.gyst-skip', null, true, false],
  ['.gyst-button', null, true, false], ['.gyst-button--icon', null, false, true],
  ['.gyst-row', null, true, false], ['.gyst-segment', null, true, false], ['.gyst-input', null, true, false], ['.gyst-choice', null, true, false],
  [':root .gyst-command .command-results a', null, true, false], [':root .gyst-command input', null, true, false],
  [':root .gyst-command .btn', null, true, true],
  // Galaxy DOM controls, read-only.
  [':root body.galaxy-page .galaxy-stage :is(button,.btn)', null, true, true],
  [':root body.galaxy-page .galaxy-context .galaxy-close', null, true, true],
  [':root body.galaxy-page .galaxy-category-filter select', null, true, false],
  [':root body.galaxy-page .galaxy-list button', null, true, false],
  [':root body.galaxy-page .galaxy-actions a', null, true, false],
];
function checkTargets(files, targets = TARGETS) {
  const rules = files.flatMap(text => parseCSS(text));
  // Every definition of every custom property, in any selector, theme or at-rule.
  const defs = {};
  for (const r of rules) for (const [k, v] of Object.entries(r.decls)) if (k.startsWith('--')) (defs[k] ??= []).push(v);
  const resolveAll = (v, depth = 0) => {
    const m = /var\(\s*(--[\w-]+)\s*(?:,[^)]*)?\)/.exec(v);
    if (!m || depth > 6 || !defs[m[1]]) return [v];
    return defs[m[1]].flatMap(d => resolveAll(v.replace(m[0], d), depth + 1));
  };
  const errors = [];
  for (const [sel, media, needH, needW] of targets) {
    const mine = rules.filter(r => r.selectors.includes(sel));
    for (const [need, props] of [[needH, ['min-height', 'height']], [needW, ['min-width', 'width', 'inline-size']]]) {
      if (!need) continue;
      const label = `${sel}${media ? ' @media ' + media : ''}`;
      const verify = (value, prop, where) => {
        for (const v of resolveAll(value)) {
          const m = /^(\d+(?:\.\d+)?)px$/.exec(v);
          if (!m) errors.push(`${where}: ${prop} ${value} does not resolve to a plain px value`);
          else if (parseFloat(m[1]) < 44) errors.push(`${where}: ${prop} ${value}${v === value ? '' : ' (' + v + ')'} is below 44px`);
        }
      };
      let value = null, prop = null;
      for (const r of mine) if (r.media === media) for (const p of props) if (p in r.decls) { value = r.decls[p]; prop = p; }
      if (value === null) errors.push(`${label}: no ${props[0]} set`); else verify(value, prop, label);
      // Overrides under any other media or at-rule must not shrink the target either.
      for (const r of mine) if (r.media !== media && r.media !== null) for (const p of props) if (p in r.decls) verify(r.decls[p], p, `${sel} @media ${r.media}`);
    }
  }
  if (errors.length) throw new Error('Tap target check failed:\n  ' + [...new Set(errors)].join('\n  '));
}
// Which checked selector covers a control? Returns null when none does.
function controlSelector(c) {
  const has = cls => c.classes.includes(cls), within = (tag, cls) => c.stack.some(a => a.tag === tag && a.classes.includes(cls));
  const parent = c.stack[c.stack.length - 1] || {tag: '', classes: []};
  if (c.tag === 'input' && c.type === 'hidden') return 'hidden';
  if (c.tag === 'a' && has('gyst-skip')) return '.gyst-skip';
  if (c.tag === 'a' && has('gyst-brand')) return '.gyst-brand';
  if (c.tag === 'a' && has('gyst-row')) return '.gyst-row';
  if (c.tag === 'button' && has('gyst-button--icon')) return '.gyst-button--icon';
  if (c.tag === 'button' && has('gyst-button')) return '.gyst-button';
  if (c.tag === 'a' && parent.tag === 'nav' && parent.classes.includes('gyst-primary')) return '.gyst-primary > a';
  if (c.tag === 'a' && parent.classes.includes('gyst-health-sub')) return '.gyst-health-sub > a';
  if (c.tag === 'a' && parent.tag === 'nav' && parent.classes.includes('hv3-tabbar')) return '.hv3-tabbar > a';
  if (within('dialog', 'gyst-command')) {
    if (c.tag === 'a' && within('div', 'command-results')) return ':root .gyst-command .command-results a';
    if (c.tag === 'button' && has('btn')) return ':root .gyst-command .btn';
    if (c.tag === 'button' && has('gyst-segment')) return '.gyst-segment';
    if (c.tag === 'input' || c.tag === 'select') return ':root .gyst-command input';
  }
  return null;
}
function templateControls(html) {
  html = html.replace(/\{\{[\s\S]*?\}\}/g, '').replace(/<!--[\s\S]*?-->/g, '');
  const body = html.indexOf('<body'); if (body >= 0) html = html.slice(body);
  const VOID = new Set(['input', 'meta', 'link', 'br', 'img', 'hr', 'source']), CONTROLS = new Set(['a', 'button', 'input', 'select', 'textarea']);
  const stack = [], found = [];
  for (const m of html.matchAll(/<(\/?)([a-zA-Z][\w-]*)([^>]*)>/g)) {
    const close = m[1], name = m[2].toLowerCase(), attrs = m[3];
    if (close) { const i = stack.map(s => s.tag).lastIndexOf(name); if (i >= 0) stack.length = i; continue; }
    const classes = ((/class="([^"]*)"/.exec(attrs) || [])[1] || '').split(/\s+/).filter(Boolean);
    if (CONTROLS.has(name)) found.push({tag: name, classes, type: (/type="([^"]*)"/.exec(attrs) || [])[1] || '', stack: stack.slice()});
    if (!VOID.has(name)) stack.push({tag: name, classes});
  }
  return found;
}
function coverageCheck(templates, targets = TARGETS) {
  const known = new Set(targets.map(t => t[0]));
  for (const [file, html] of Object.entries(templates)) for (const c of templateControls(html)) {
    const sel = controlSelector(c);
    assert(sel, `${file}: <${c.tag}${c.classes.length ? ' class="' + c.classes.join(' ') + '"' : ''}> has no checked tap-target selector; add one to TARGETS in scripts/check-ui-shell.cjs`);
    assert(sel === 'hidden' || known.has(sel), `${file}: <${c.tag}> maps to ${sel}, which is not in TARGETS`);
  }
}
function tapTargetChecks() {
  const files = ['ui-tokens.css', 'ui-components.css', 'ui-shell.css', 'ui-health-v3.css', 'ui-galaxy.css'].map(f => fs.readFileSync(CSS_DIR + f, 'utf8'));
  checkTargets(files);
  // The check must reject undersized and unresolvable targets.
  const one = (css, sel, h = true, w = false) => () => checkTargets([css], [[sel, null, h, w]]);
  assert.throws(one('.gyst-button{min-height:32px}', '.gyst-button'), /\.gyst-button.*32px/);
  assert.throws(one('.gyst-button{min-height:calc(40px + 2px)}', '.gyst-button'), /\.gyst-button.*calc/);
  assert.throws(one('.gyst-button--icon{width:40px}', '.gyst-button--icon', false, true), /\.gyst-button--icon.*40px/);
  assert.throws(one('.x{min-height:44px}', '.gyst-button'), /no min-height/);
  checkTargets([':root{--gyst-tab-height:56px}@media (max-width:1023px){.t > a{min-height:var(--gyst-tab-height)}}'], [['.t > a', '(max-width: 1023px)', true, false]]);
  assert.throws(one('.gyst-button{min-height:44px}@media (max-width:767px){.gyst-button{min-height:32px}}', '.gyst-button'), /\.gyst-button.*32px/);
  assert.throws(one('.gyst-button{min-height:44px}@supports (display:grid){.gyst-button{min-height:32px}}', '.gyst-button'), /\.gyst-button.*32px/);
  const tok = extra => () => checkTargets([':root{--tab:56px}.t{min-height:var(--tab)}' + extra], [['.t', null, true, false]]);
  tok('')();
  assert.throws(tok('[data-gyst-theme="dark"]{--tab:30px}'), /\.t.*30px/);
  assert.throws(tok('@media (max-width:600px){:root{--tab:30px}}'), /\.t.*30px/);
  assert.throws(tok(':root{--tab:30px}'), /\.t.*30px/);
  const templates = {'ui_shell.html': fs.readFileSync('web/templates/ui_shell.html', 'utf8'), 'layout.html': fs.readFileSync('web/templates/layout.html', 'utf8'), 'archive.html': fs.readFileSync('web/templates/archive.html', 'utf8'), 'analytics.html': fs.readFileSync('web/templates/analytics.html', 'utf8')};
  coverageCheck(templates);
  assert.throws(() => coverageCheck({'ui_shell.html': templates['ui_shell.html'] + '<body><button type="button">Oops</button>'}), /<button> has no checked tap-target selector/);
}
function overlayChecks() {
  const src = fs.readFileSync('web/static/js/ui-overlay.js', 'utf8');
  assert(!/\bconfirm\s*\(/.test(src), 'overlay module must not call window.confirm');
  const ctx = {}; ctx.globalThis = ctx; vm.runInNewContext(src, ctx);
  const O = ctx.GYSTOverlay;
  vm.runInNewContext(src, ctx); assert.equal(ctx.GYSTOverlay, O, 'second load must keep the first instance');
  assert(/if \(root\.GYSTOverlay\) return;/.test(src), 'double-load guard');
  assert.equal(O.mode(390), 'sheet'); assert.equal(O.mode(1023), 'sheet'); assert.equal(O.mode(1024), 'popover'); assert.equal(O.mode(1600), 'popover');
  const vp = {width: 1280, height: 800}, size = {width: 220, height: 240};
  for (const [l, t] of [[0, 0], [1200, 0], [0, 740], [1200, 740], [640, 400], [1280, 800]]) {
    const trig = {left: l, top: t, right: Math.min(l + 80, 1280), bottom: Math.min(t + 44, 800), width: 80, height: 44};
    const at = O.place(trig, size, vp);
    assert(at.left >= 0 && at.top >= 0 && at.left + size.width <= vp.width && at.top + size.height <= vp.height, 'popover outside viewport for trigger ' + l + ',' + t + ': ' + JSON.stringify(at));
  }
  assert.equal(O.step(2, 1, 3), 0); assert.equal(O.step(0, -1, 3), 2); assert.equal(O.step(1, 1, 3), 2);
  assert.equal(O.step(-1, 1, 3), 0); assert.equal(O.step(-1, -1, 3), 2); assert.equal(O.step(0, 1, 0), -1);
}
// Launcher v2 (GYST-102): pure helpers run in a vm with no DOM; the source is scanned for unsafe writes.
function launcherChecks() {
  const src = fs.readFileSync('web/static/js/launcher.js', 'utf8');
  assert(!/innerHTML|outerHTML|insertAdjacentHTML|document\.write/.test(src), 'launcher must not write HTML strings');
  assert(!/location\s*(\.href)?\s*=[^=]/.test(src) && !/location\.(assign|replace)\(/.test(src), 'launcher must not navigate through location');
  // Only the Cmd/Ctrl-K opener may listen on document; every other key handler is on the dialog or input.
  assert.equal((src.match(/document\.addEventListener\("keydown"/g) || []).length, 1, 'one document keydown (the opener)');
  assert(!/palette\.showModal\(\)/.test(src.replace(/function open[\s\S]*?function close/, '')), 'only open() shows the dialog');
  const sandbox = {Array, JSON, String, Math}; sandbox.window = sandbox; sandbox.globalThis = sandbox;
  vm.runInNewContext(src, sandbox);
  const L = sandbox.GystLauncher;
  const same = (a, b) => assert.equal(JSON.stringify(a), JSON.stringify(b));
  assert(L, 'GystLauncher exposed without a DOM');
  assert.equal(L.safeHref('/projects/3', false), '/projects/3');
  for (const bad of ['//evil.example', '//evil.example/x', 'javascript:alert(1)', 'data:text/html,hi', 'http://', 'http:///x', 'https://', '/\\evil.example', 'projects', '', null, undefined, 5, '/a b', '/x\ny', 'https://u:p@evil.example/', 'ftp://example.com/']) {
    assert.equal(L.safeHref(bad, false), '', 'internal rejects ' + String(bad));
    assert.equal(L.safeHref(bad, true), '', 'external rejects ' + String(bad));
  }
  assert.equal(L.safeHref('https://example.com/a?b=1#c', true), 'https://example.com/a?b=1#c');
  assert.equal(L.safeHref('http://box.ts.net:9443/dash', true), 'http://box.ts.net:9443/dash');
  assert.equal(L.safeHref('https://example.com/', false), '', 'an absolute URL is not an internal path');
  assert.equal(L.safeHref('/x', true), '', 'external needs http(s)');
  // Recents: at most 5, newest first, no duplicates, invalid entries dropped.
  let list = [];
  for (let i = 0; i < 8; i++) list = L.addRecent(list, {label: 'Item ' + i, href: '/p/' + i});
  assert.equal(list.length, 5); assert.equal(list[0].href, '/p/7'); assert.equal(list[4].href, '/p/3');
  list = L.addRecent(list, {label: 'Again', href: '/p/5'});
  same(list.map(e => e.href), ['/p/5', '/p/7', '/p/6', '/p/4', '/p/3']);
  const dirty = L.cleanRecents([{label: 'ok', href: '/ok'}, {label: 'x', href: '//evil.example'}, {label: 'js', href: 'javascript:1'}, {label: '', href: '/e'}, {href: '/n'}, null, 'str', {label: 'dup', href: '/ok'}, {label: 'ext', href: 'https://example.com/'}]);
  same(dirty.map(e => e.href), ['/ok']);
  same(L.cleanRecents('nope'), []); same(L.cleanRecents(null), []);
  assert.equal(L.cleanRecents(Array.from({length: 20}, (_, i) => ({label: 'L' + i, href: '/' + i}))).length, 5);
  // '+' as the first character switches Go to Capture and is stripped; never the other way, never mid-text.
  same({...L.parseInput('+call sam', 'go')}, {mode: 'capture', text: 'call sam'});
  same({...L.parseInput('+ call sam', 'go')}, {mode: 'capture', text: 'call sam'});
  same({...L.parseInput('a+b', 'go')}, {mode: 'go', text: 'a+b'});
  same({...L.parseInput('+x', 'capture')}, {mode: 'capture', text: '+x'});
  // Enter never creates in Go, with or without results; Capture submits only with a title.
  assert.equal(L.enterAction('go', false, 'buy milk'), 'none');
  assert.equal(L.enterAction('go', false, ''), 'none');
  assert.equal(L.enterAction('go', true, 'buy milk'), 'open');
  assert.equal(L.enterAction('go', true, 'garden', true), 'defer', 'stale results never open');
  assert.equal(L.enterAction('go', false, 'garden', true), 'defer');
  assert.equal(L.enterAction('capture', true, 'x', true), 'submit');
  assert.equal(L.enterAction('capture', false, 'buy milk'), 'submit');
  assert.equal(L.enterAction('capture', true, '  '), 'none');
  // Arrow keys wrap and an empty list has no selection.
  assert.equal(L.step(-1, 1, 3), 0); assert.equal(L.step(2, 1, 3), 0); assert.equal(L.step(0, -1, 3), 2); assert.equal(L.step(-1, -1, 3), 2); assert.equal(L.step(0, 1, 0), -1);
  // Capture chips.
  same({...L.parseCapture('Call Sam tomorrow !high', '2026-10-12', '2026-10-13')}, {title: 'Call Sam', due: '2026-10-13', priority: 'high'});
  same({...L.parseCapture('Pay rent today !low', '2026-10-12', '2026-10-13')}, {title: 'Pay rent', due: '2026-10-12', priority: 'low'});
  same({...L.parseCapture('Plain', 'a', 'b')}, {title: 'Plain', due: '', priority: ''});
  // The shell no longer carries the old palette, and the new script ships with the page and the worker.
  assert(!/initCommandPalette/.test(fs.readFileSync('web/static/js/app.js', 'utf8')), 'old palette removed from app.js');
  assert(/launcher\.js\?v=/.test(fs.readFileSync('web/static/service-worker.js', 'utf8')), 'launcher.js is precached');
}
// Launcher v2 race checks: drives the real launcher.js in a fake DOM with fake timers and a fetch the test answers by hand.
async function launcherRaceChecks() {
  const src = fs.readFileSync('web/static/js/launcher.js', 'utf8');
  const strip = s => s.slice(1, -1);
  let clicked = [], submitted = 0;
  function el(tag) {
    const e = {tag, attrs: {}, children: [], listeners: {}, dataset: {}, hidden: false, value: '', className: '', id: '', textContent: '', href: '', target: '',
      classList: {set: new Set(), add(c) { this.set.add(c); }, remove(c) { this.set.delete(c); }},
      setAttribute(k, v) { this.attrs[k] = String(v); },
      getAttribute(k) { return k === 'href' ? this.href : this.attrs[k]; },
      removeAttribute(k) { delete this.attrs[k]; },
      addEventListener(t, f) { (this.listeners[t] ||= []).push(f); },
      dispatch(t, ev) { ev = Object.assign({type: t, preventDefault() {}, target: this, currentTarget: this}, ev); (this.listeners[t] || []).forEach(f => f(ev)); return ev; },
      appendChild(c) { c.parent = this; this.children.push(c); return c; },
      removeChild(c) { this.children.splice(this.children.indexOf(c), 1); },
      get firstChild() { return this.children[0] || null; },
      focus() {}, querySelector() { return null; }, isConnected: true, scrollIntoView() {}, requestSubmit() { submitted++; },
      closest() { let n = this; while (n) { if (n.className === 'launcher-item') return n; n = n.parent; } return null; },
      click() { clicked.push(this.href); let n = this; while (n) { (n.listeners.click || []).forEach(f => f({target: this, preventDefault() {}})); n = n.parent; } }};
    return e;
  }
  const input = el('input'), results = el('div'), form = el('form'), title = el('input'), date = el('input'), priority = el('select'), ret = el('input'), status = el('p'), closeBtn = el('button');
  const goBtn = el('button'), capBtn = el('button'); goBtn.dataset.launcherMode = 'go'; capBtn.dataset.launcherMode = 'capture';
  const goPanel = el('div'), capPanel = el('div'); goPanel.dataset.launcherPanel = 'go'; capPanel.dataset.launcherPanel = 'capture';
  const palette = el('dialog'); palette.open = false; palette.dataset.today = '2026-10-12'; palette.dataset.tomorrow = '2026-10-13';
  palette.showModal = function () { this.open = true; };
  palette.close = function () { this.open = false; this.dispatch('close'); };
  const byName = {'data-command-search': input, 'data-command-results': results, 'data-quick-task': form, 'data-quick-title': title, 'data-quick-date': date, 'data-quick-priority': priority, 'data-command-return': ret, 'data-launcher-status': status, 'data-command-close': closeBtn};
  palette.querySelector = s => byName[strip(s)] || null;
  palette.querySelectorAll = s => strip(s) === 'data-launcher-mode' ? [goBtn, capBtn] : strip(s) === 'data-launcher-panel' ? [goPanel, capPanel] : [];
  const opener = el('button');
  const fakeDoc = {readyState: 'complete', activeElement: opener, querySelector: s => strip(s) === 'data-command-palette' ? palette : null, querySelectorAll: s => strip(s) === 'data-command-open' ? [opener] : [], createElement: t => el(t), addEventListener() {}};
  let now = 0, timers = [], tid = 0;
  const fakeSetTimeout = (f, ms) => { tid++; timers.push({id: tid, at: now + ms, f}); return tid; };
  const fakeClearTimeout = id => { timers = timers.filter(t => t.id !== id); };
  const inflight = [];
  const fakeFetch = (url, init) => new Promise((res, rej) => {
    inflight.push({q: decodeURIComponent(url.split('q=')[1]), res});
    if (init.signal) init.signal.addEventListener('abort', () => { const e = new Error('aborted'); e.name = 'AbortError'; rej(e); });
  });
  const reply = (q, groups) => { const i = inflight.findIndex(r => r.q === q); if (i < 0) return false; inflight.splice(i, 1)[0].res({status: 200, ok: true, json: () => ({groups})}); return true; };
  const answer = (q, external) => reply(q, q ? [{label: 'Top hit', items: [{label: 'Hit for ' + q, href: external ? 'https://example.com/' + q : '/hit/' + q, external: !!external}]}] : []);
  const flush = () => new Promise(r => setImmediate(r));
  async function advance(ms) { const end = now + ms; for (;;) { timers.sort((a, b) => a.at - b.at); const t = timers[0]; if (!t || t.at > end) break; timers.shift(); now = t.at; t.f(); await flush(); } now = end; await flush(); }
  const sandbox = {document: fakeDoc, fetch: fakeFetch, setTimeout: fakeSetTimeout, clearTimeout: fakeClearTimeout, AbortController, localStorage: {getItem() { return null; }, setItem() {}}, location: {pathname: '/today', search: '', hash: ''}, Array, JSON, String, Math, Date, Error};
  sandbox.window = sandbox; sandbox.globalThis = sandbox;
  vm.runInNewContext(src, sandbox);
  const type = v => { input.value = v; input.dispatch('input'); };
  const key = k => input.dispatch('keydown', {key: k});
  const got = () => clicked.join(',') || '(nothing)';
  async function reopen() { if (palette.open) palette.close(); inflight.length = 0; timers = []; clicked = []; submitted = 0; opener.dispatch('click'); await flush(); answer(''); await flush(); }
  const is = (label, want) => assert.equal(got(), want, label + ': opened ' + got());

  await reopen();
  type('gar'); key('Enter'); await flush(); answer('gar'); await flush();
  is('Enter inside the debounce with nothing in flight opens the current top hit', '/hit/gar');
  // The reviewer's race: "g" in flight, user types "ga", the "g" answer lands, Enter.
  await reopen();
  type('g'); await advance(130); type('ga'); await advance(20);
  answer('g'); await flush(); key('Enter'); await flush();
  is('an older answer must not open', '(nothing)');
  await advance(200); answer('ga'); await flush();
  is('the current query hit opens once it renders', '/hit/ga');
  await reopen();
  type('g'); await advance(130); type('ga'); await advance(20);
  key('Enter'); answer('g'); await flush();
  is('Enter before the older answer', '(nothing)');
  await advance(200); answer('ga'); await flush();
  is('then only the current hit', '/hit/ga');
  await reopen();
  type('ab'); key('Enter'); type('abc'); answer('ab'); await flush();
  is('Enter then more typing opens nothing', '(nothing)');
  await advance(200); answer('abc'); await flush();
  is('and no second Enter means still nothing', '(nothing)');
  await reopen();
  type('xy'); key('Enter'); palette.dispatch('keydown', {key: 'Escape'}); answer('xy'); await flush();
  is('Escape while deferred', '(nothing)'); assert.equal(palette.open, false);
  await reopen();
  type('zz'); key('Enter'); key('Tab'); answer('zz'); await flush();
  is('Tab to Capture while deferred', '(nothing)'); assert.equal(submitted, 0);
  await reopen();
  type('qq'); key('Enter'); type('+qq'); answer('qq'); await flush();
  is('plus to Capture while deferred', '(nothing)'); assert.equal(submitted, 0);
  await reopen();
  type('nomatch'); key('Enter'); reply('nomatch', []); await flush();
  is('empty answer', '(nothing)'); assert.equal(submitted, 0);
  await reopen();
  type('app'); key('Enter'); answer('app', true); await flush();
  is('a deferred Enter never opens an external new-tab link', '(nothing)');
}
// Day reload (#78, #193): Health Today reloads once the server's day has
// ended (data-day-ends). The device's zone rules are simulated as stale:
// Winnipeg still UTC-06 after 2026-11-01, as before tzdb 2026e. With
// data-day-ends they must not matter; without it the browser fallback runs.
function dayReloadChecks() {
  const code = fs.readFileSync('web/static/js/app.js', 'utf8');
  const RealDate = Date;
  function run(attrs, now) {
    const clock = {now: RealDate.parse(now)}, docEvents = {}, winEvents = {};
    let reloads = 0;
    class FakeDate extends RealDate {
      constructor(...args) { if (args.length) super(...args); else super(clock.now); }
      static now() { return clock.now; }
    }
    const staleIntl = {DateTimeFormat: function () {
      return {formatToParts: d => {
        const s = new RealDate(d.getTime() - 6 * 3600e3).toISOString();
        return [{type: 'month', value: s.slice(5, 7)}, {type: 'day', value: s.slice(8, 10)}, {type: 'year', value: s.slice(0, 4)}];
      }};
    }};
    const page = {getAttribute: k => (k in attrs ? attrs[k] : null)};
    const sandbox = {
      document: {readyState: 'complete', visibilityState: 'visible', querySelector: sel => (sel === '[data-day-page]' ? page : null), querySelectorAll: () => [], addEventListener: (n, f) => { docEvents[n] = f; }},
      Date: FakeDate, Intl: staleIntl, URL, location: {href: 'https://gyst.test/health', reload: () => { reloads++; }},
      history: {state: null, replaceState() {}}, navigator: {onLine: true}, setTimeout, clearTimeout,
      addEventListener: (n, f) => { winEvents[n] = f; },
    };
    sandbox.window = sandbox;
    vm.runInNewContext(code, sandbox);
    return {
      resume(at) { if (at) clock.now = RealDate.parse(at); docEvents.visibilitychange(); return reloads; },
      pageshow() { winEvents.pageshow({persisted: true}); return reloads; },
    };
  }
  const tz = 'America/Winnipeg';
  // A Nov 1 page left open past midnight (00:30 Nov 2, UTC-05): stale rules
  // still say Nov 1, but the day has ended, so it reloads.
  let p = run({'data-day-page': '2026-11-01', 'data-day-tz': tz, 'data-day-ends': '2026-11-02T05:00:00Z'}, '2026-11-02T04:59:59Z');
  assert.equal(p.resume(), 0, 'reloaded before the day ended');
  assert.equal(p.resume('2026-11-02T05:30:00Z'), 1, 'yesterday stayed on screen after the day ended');
  // The fresh Nov 2 page at 00:30: stale rules say Nov 1, but no reload.
  p = run({'data-day-page': '2026-11-02', 'data-day-tz': tz, 'data-day-ends': '2026-11-03T05:00:00Z'}, '2026-11-02T05:30:00Z');
  assert.equal(p.resume(), 0, 'stale device rules forced a reload');
  assert.equal(p.pageshow(), 0, 'stale device rules forced a reload on pageshow');
  assert.equal(p.resume('2026-11-03T05:00:00Z'), 1, 'no reload at the end of the day');
  // Without data-day-ends (no server day, render tests) the browser's own
  // reading of the app timezone still drives the reload.
  for (const ends of [undefined, '']) {
    const attrs = {'data-day-page': '2026-11-01', 'data-day-tz': tz};
    if (ends !== undefined) attrs['data-day-ends'] = ends;
    p = run(attrs, '2026-11-02T05:30:00Z');
    assert.equal(p.resume(), 0, 'fallback reloaded while the browser still reads Nov 1');
    assert.equal(p.resume('2026-11-02T07:00:00Z'), 1, 'fallback did not reload on the browser\'s next day');
  }
}
(async()=>{await workerChecks();tapTargetChecks();themeChecks();motionChecks();toastChecks();overlayChecks();launcherChecks();await launcherRaceChecks();dayReloadChecks();console.log('UI shell: private/offline cache policy, 44px tap targets, appearance setting, motion preference, completion toast, overlay helpers, launcher and day reload passed');})().catch(error=>{console.error(error);process.exitCode=1;});
