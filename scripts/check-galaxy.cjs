const assert = require('node:assert/strict');
const fs = require('node:fs');
(async () => {
  const source = fs.readFileSync('web/static/js/galaxy-gestures.js', 'utf8');
  const {bindGalaxyGestures} = await import('data:text/javascript;base64,' + Buffer.from(source).toString('base64'));
  const handlers = {}, captured = new Set(), calls = [];
  const canvas = {addEventListener: (name, fn) => handlers[name] = fn, setPointerCapture: id => captured.add(id), hasPointerCapture: id => captured.has(id), releasePointerCapture: id => captured.delete(id)};
  bindGalaxyGestures(canvas, {tap: () => calls.push('tap'), drag: () => calls.push('drag'), zoom: factor => calls.push(factor)});
  const send = (type, id, x, y) => handlers[type]({pointerId: id, clientX: x, clientY: y, button: 0, pointerType: 'touch'});
  send('pointerdown', 1, 50, 50); send('pointerup', 1, 50, 50); assert.deepEqual(calls, ['tap']); calls.length = 0;
  send('pointerdown', 1, 50, 50); send('pointermove', 1, 70, 50); send('pointerup', 1, 70, 50); assert.deepEqual(calls, ['drag']); calls.length = 0;
  send('pointerdown', 1, 50, 50); send('pointerdown', 2, 100, 50); send('pointermove', 2, 150, 50); send('pointerup', 2, 150, 50); send('pointerup', 1, 50, 50); assert.deepEqual(calls, [0.5]); calls.length = 0;
  send('pointerdown', 1, 50, 50); send('pointercancel', 1, 50, 50); send('pointerup', 1, 50, 50); assert.deepEqual(calls, []);
  send('pointerdown', 1, 50, 50); send('lostpointercapture', 1, 50, 50); send('pointerup', 1, 50, 50); assert.deepEqual(calls, []);
  send('pointerdown', 1, 50, 50); send('pointerup', 1, 50, 50); assert.deepEqual(calls, ['tap']);
  console.log('Galaxy gestures: tap, drag, pinch, cancellation and recovery passed');
})();

(async () => {
 const source = fs.readFileSync('web/static/js/galaxy-camera.js','utf8');
 const {galaxyCameraFit} = await import('data:text/javascript;base64,'+Buffer.from(source).toString('base64'));
 for (const [width,height] of [[320,568],[390,600],[844,280],[1440,800]]) {
  const radius=90, distance=galaxyCameraFit(radius,width,height);
  const vertical=55*Math.PI/360, horizontal=Math.atan(Math.tan(vertical)*width/height);
  assert.ok(distance*Math.sin(Math.min(vertical,horizontal))>radius,'all worlds fit in narrower dimension');
 }
 assert.ok(galaxyCameraFit(90,320,600)>galaxyCameraFit(90,1440,800));
 console.log('Galaxy camera: portrait, landscape and desktop bounds passed');
})();

(() => {
 const html = fs.readFileSync('web/templates/galaxy.html','utf8');
 assert.ok(!html.includes('Drag to explore'), 'galaxy.html must not repeat the status-line hint');
 assert.ok(html.includes('galaxy-screen'), 'galaxy.html wraps hub and stage in .galaxy-screen');
 console.log('Galaxy template: single hint and screen wrapper passed');
})();
