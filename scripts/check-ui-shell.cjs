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
  if (c.tag === 'button' && has('gyst-button--icon')) return '.gyst-button--icon';
  if (c.tag === 'button' && has('gyst-button')) return '.gyst-button';
  if (c.tag === 'a' && parent.tag === 'nav' && parent.classes.includes('gyst-primary')) return '.gyst-primary > a';
  if (c.tag === 'a' && parent.classes.includes('gyst-health-sub')) return '.gyst-health-sub > a';
  if (c.tag === 'a' && parent.tag === 'nav' && parent.classes.includes('hv3-tabbar')) return '.hv3-tabbar > a';
  if (within('dialog', 'gyst-command')) {
    if (c.tag === 'a' && within('div', 'command-results')) return ':root .gyst-command .command-results a';
    if (c.tag === 'button' && has('btn')) return ':root .gyst-command .btn';
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
  const templates = {'ui_shell.html': fs.readFileSync('web/templates/ui_shell.html', 'utf8'), 'layout.html': fs.readFileSync('web/templates/layout.html', 'utf8')};
  coverageCheck(templates);
  assert.throws(() => coverageCheck({'ui_shell.html': templates['ui_shell.html'] + '<body><button type="button">Oops</button>'}), /<button> has no checked tap-target selector/);
}
(async()=>{await workerChecks();tapTargetChecks();themeChecks();motionChecks();toastChecks();console.log('UI shell: private/offline cache policy, 44px tap targets, appearance setting, motion preference and completion toast passed');})().catch(error=>{console.error(error);process.exitCode=1;});
