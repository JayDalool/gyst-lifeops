// Canvas-only gestures: a drag, pinch or cancellation can never become a tap.
export function bindGalaxyGestures(canvas, actions) {
  const pointers = new Map();
  let moved = false;
  let distance = 0;
  const separation = () => {
    const [a, b] = [...pointers.values()];
    return b ? Math.hypot(a.x - b.x, a.y - b.y) : 0;
  };
  canvas.addEventListener("pointerdown", event => {
    if (event.button !== 0) return;
    if (!pointers.size) moved = false;
    pointers.set(event.pointerId, {x: event.clientX, y: event.clientY, startX: event.clientX, startY: event.clientY});
    canvas.setPointerCapture(event.pointerId);
    if (pointers.size > 1) { moved = true; distance = separation(); }
  });
  canvas.addEventListener("pointermove", event => {
    const point = pointers.get(event.pointerId);
    if (!point) return;
    const dx = event.clientX - point.x, dy = event.clientY - point.y;
    point.x = event.clientX; point.y = event.clientY;
    if (pointers.size > 1) {
      const next = separation();
      if (distance > 0 && next > 0) actions.zoom(distance / next);
      distance = next;
    } else if (moved || Math.hypot(point.x - point.startX, point.y - point.startY) > 8) {
      moved = true; actions.drag(dx, dy);
    }
  });
  const finish = (event, cancelled) => {
    const point = pointers.get(event.pointerId);
    if (!point) return;
    pointers.delete(event.pointerId);
    if (cancelled) moved = true;
    if (!cancelled && !moved && !pointers.size && Math.hypot(event.clientX - point.startX, event.clientY - point.startY) <= 8) actions.tap(event);
    if (canvas.hasPointerCapture(event.pointerId)) canvas.releasePointerCapture(event.pointerId);
  };
  canvas.addEventListener("pointerup", event => finish(event, false));
  canvas.addEventListener("pointercancel", event => finish(event, true));
  canvas.addEventListener("lostpointercapture", event => finish(event, true));
}
