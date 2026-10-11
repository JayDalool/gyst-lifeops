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
(async()=>{await workerChecks();themeChecks();motionChecks();toastChecks();overlayChecks();console.log('UI shell: private/offline cache policy, appearance setting, motion preference, completion toast and overlay helpers passed');})().catch(error=>{console.error(error);process.exitCode=1;});
