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
function themeChecks() {
  const code=fs.readFileSync('web/static/js/theme.js','utf8');
  for(const initial of [null,'light','dark','invalid']) {
    for(const denied of [false,true]) {
      let stored=initial;const events={},toggle={checked:false,addEventListener:(name,fn)=>events.change=fn};
      const root={dataset:{}},meta={content:''};
      vm.runInNewContext(code,{
        document:{documentElement:root,querySelector:()=>meta,querySelectorAll:()=>[toggle],dispatchEvent:()=>{},addEventListener:(name,fn)=>events[name]=fn},
        localStorage:{getItem:()=>{if(denied)throw Error('denied');return stored;},setItem:(key,value)=>{if(denied)throw Error('denied');stored=value;}},
        window:{addEventListener:(name,fn)=>events[name]=fn},Event:class{}
      });
      const expected=!denied && initial==='dark'?'dark':'light';
      assert.equal(root.dataset.gystTheme,expected);
      events.DOMContentLoaded();assert.equal(toggle.checked,expected==='light');
      toggle.checked=false;events.change();assert.equal(root.dataset.gystTheme,'dark');assert.equal(root.dataset.theme,undefined);
      if(!denied)assert.equal(stored,'dark');
      events.storage({key:'lifeops-theme',newValue:'light'});assert.equal(root.dataset.theme,'light');
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
(async()=>{await workerChecks();themeChecks();motionChecks();console.log('UI shell: private/offline cache policy, theme persistence and motion preference passed');})().catch(error=>{console.error(error);process.exitCode=1;});
