"""Galaxy baseline harness (GYST-VN-143). Observation only.

Drives DISPOSABLE local GYST instances seeded with fictional data
(see seed-fictional-galaxy.sh) and records behaviour and relative performance
of the real production Galaxy. It never changes application code; it only
navigates, taps, clicks, drags, scrolls, presses keys and reads the DOM.

Run (from the repository root) with the Playwright image and host networking:

  docker run --rm --network host -v "$PWD/docs/design/galaxy-baseline":/work -w /work \
    mcr.microsoft.com/playwright/python:v1.58.0-noble sh -c \
    'pip install -q --break-system-packages playwright==1.58.0 && python3 measure_galaxy.py'

(The image ships the browsers but not the Python package; it's installed inside
the throwaway container only.)

Environment: GB_URLS='{"0":"http://127.0.0.1:18090",...}' maps world counts
to instance URLs (defaults below). Output: results.json and media/*.jpg.

Headless browsers render WebGL in software (SwiftShader), so frame timings are
RELATIVE ONLY and not representative of a real iPhone or desktop GPU.
"""
import json, os, time, statistics
from playwright.sync_api import sync_playwright

URLS = json.loads(os.environ.get("GB_URLS", json.dumps({
    "0": "http://127.0.0.1:18090", "1": "http://127.0.0.1:18091", "3": "http://127.0.0.1:18092",
    "10": "http://127.0.0.1:18093", "30": "http://127.0.0.1:18080"})))
for u in URLS.values():
    assert u.startswith(("http://127.0.0.1:", "http://localhost:")), "local instances only"
OUT = {}


def save():
    json.dump(OUT, open("results.json", "w"), indent=1, default=str)

os.makedirs("media", exist_ok=True)

INIT = r"""
(() => {
  window.__draws = 0; window.__drawsPerFrame = []; window.__longtasks = [];
  for (const C of [window.WebGLRenderingContext, window.WebGL2RenderingContext]) {
    if (!C) continue;
    for (const m of ['drawElements','drawArrays','drawElementsInstanced','drawArraysInstanced']) {
      const o = C.prototype[m]; if (!o) continue;
      C.prototype[m] = function (...a) { window.__draws++; return o.apply(this, a); };
    }
  }
  const raf = window.requestAnimationFrame.bind(window);
  (function tick() { window.__drawsPerFrame.push(window.__draws); window.__draws = 0; raf(tick); })();
  try { new PerformanceObserver(l => l.getEntries().forEach(e => window.__longtasks.push([Math.round(e.startTime), Math.round(e.duration)])))
        .observe({type: 'longtask', buffered: true}); } catch (e) { window.__longtasks = 'unsupported'; }
})();
"""

JS_LABELS = """() => [...document.querySelectorAll('.galaxy-planet-label')].map(l => {
  const m = l.style.transform.match(/translate\\(([-\\d.]+)px,\\s*([-\\d.]+)px\\)\\s*$/);
  return {name: l.textContent, hidden: l.hidden, x: m ? +m[1] : null, y: m ? +m[2] + 18 : null}; })"""
JS_STAGE = "() => { const r = document.querySelector('[data-galaxy-stage]').getBoundingClientRect(); return {x: r.left, y: r.top, w: r.width, h: r.height}; }"
JS_STATUS = "() => document.querySelector('[data-galaxy-status]')?.textContent || ''"
JS_FRAMES = """async (ms) => { const ts = []; const end = performance.now() + ms;
  await new Promise(res => { function f(t) { ts.push(t); if (t < end) requestAnimationFrame(f); else res(); } requestAnimationFrame(f); });
  const d = ts.slice(1).map((t, i) => t - ts[i]).sort((a, b) => a - b);
  const p = q => +d[Math.floor(q * (d.length - 1))].toFixed(1);
  return {frames: d.length, fps: +(d.length / (ms / 1000)).toFixed(1), median_ms: p(.5), p95_ms: p(.95), max_ms: +d[d.length - 1].toFixed(1)}; }"""


def ready(page, timeout=20000):
    t0 = time.time()
    page.wait_for_function("() => /Drag to explore|No active planets|unavailable/.test(document.querySelector('[data-galaxy-status]')?.textContent||'')", timeout=timeout)
    return round((time.time() - t0) * 1000)


def settle(page, name, max_s=12):
    """Seconds until the named world's label stops moving (< 0.5 px over 0.5 s)."""
    t0 = time.time(); last = None; still_since = None
    while time.time() - t0 < max_s:
        lab = next((l for l in page.evaluate(JS_LABELS) if l["name"] == name), None)
        pos = (lab["x"], lab["y"]) if lab and lab["x"] is not None else None
        now = time.time()
        if pos and last and abs(pos[0] - last[0]) < 0.5 and abs(pos[1] - last[1]) < 0.5:
            still_since = still_since or now
            if now - still_since >= 0.5:
                return round(still_since - t0, 2)
        else:
            still_since = None
        last = pos; time.sleep(0.05)
    return None


def reset(page):
    page.locator("[data-galaxy-stage]").press("Escape")
    try:
        page.wait_for_function("() => /Drag to explore/.test(document.querySelector('[data-galaxy-status]').textContent)", timeout=15000)
    except Exception:
        pass
    time.sleep(0.4)


def open_galaxy(ctx, url):
    page = ctx.new_page(); errors = []
    page.on("console", lambda m: errors.append(m.text) if m.type == "error" else None)
    page.on("pageerror", lambda e: errors.append(str(e)))
    t0 = time.time(); page.goto(url + "/galaxy"); r = ready(page)
    return page, errors, round((time.time() - t0) * 1000), r


def url_of(page):
    base = page.url.split("/projects")[0].split("/galaxy")[0]
    return base + "/galaxy"


def hit_sweep(page, mode, worlds=2):
    """Binary-search the effective hit edge around visible worlds in 4 directions."""
    res = []
    names = [l["name"] for l in page.evaluate(JS_LABELS) if not l["hidden"]][:worlds]
    def tap(x, y):
        # Selecting a world scrolls the page on phones, so re-anchor every time.
        page.evaluate("() => { const r = document.querySelector('[data-galaxy-stage]').getBoundingClientRect(); window.scrollBy(0, r.top); }")
        time.sleep(0.05); stage = page.evaluate(JS_STAGE)
        time.sleep(0.65)  # > double-click interval: two quick clicks/taps on a world open it (Galaxy's dblclick handler)
        if mode == "touch": page.touchscreen.tap(stage["x"] + x, stage["y"] + y)
        else:
            page.mouse.click(stage["x"] + x, stage["y"] + y)
            page.mouse.move(stage["x"] + stage["w"] / 2, stage["y"] + stage["h"] / 2)  # dead zone: stop hover roam
    for name in names:
        row = {"world": name}
        for d, dx, dy in (("right", 1, 0), ("left", -1, 0), ("up", 0, -1), ("down", 0, 1)):
            lo, hi, other = 0.0, 100.0, None
            while hi - lo > 2:
                mid = (lo + hi) / 2
                lab = next((l for l in page.evaluate(JS_LABELS) if l["name"] == name), None)
                if not lab or lab["x"] is None: break
                try:
                    tap(lab["x"] + dx * mid, lab["y"] + dy * mid); time.sleep(0.3)
                    if "/galaxy" not in page.url: raise RuntimeError("navigated")
                    s = page.evaluate(JS_STATUS)
                except Exception:  # guard: an unintended open navigated away; restart this probe
                    page.goto(url_of(page)); ready(page); time.sleep(0.5); continue
                hit = s.startswith(name)
                if not hit and not s.startswith("Drag to explore"): other = s.split(" · ")[0]
                if not s.startswith("Drag to explore"): reset(page)
                lo, hi = (mid, hi) if hit else (lo, mid)
            row[d] = {"edge_px": round(lo), "lost_to": other}
        res.append(row)
    return res


with sync_playwright() as p:
    chromium = p.chromium.launch(args=["--use-angle=swiftshader", "--enable-unsafe-swiftshader", "--ignore-gpu-blocklist"])
    webkit = p.webkit.launch()
    desktop = dict(viewport={"width": 1280, "height": 800})

    # 1. Load, draw calls, idle/drag/flight frames per dataset (desktop Chromium)
    perf = {}
    for n, url in URLS.items():
        ctx = chromium.new_context(**desktop); ctx.add_init_script(INIT)
        page, errors, nav_ms, ready_ms = open_galaxy(ctx, url)
        time.sleep(1.5)
        d = page.evaluate("() => window.__drawsPerFrame.slice(-60)")
        row = {"load_to_ready_ms": nav_ms, "worlds_in_data": len(page.evaluate("async () => (await (await fetch('/galaxy/data')).json())")),
               "labels": len(page.evaluate(JS_LABELS)), "visible_labels": sum(1 for l in page.evaluate(JS_LABELS) if not l["hidden"]),
               "draw_calls_per_frame_median": statistics.median(d) if d else None,
               "longtasks": page.evaluate("() => window.__longtasks"), "status": page.evaluate(JS_STATUS)}
        if row["worlds_in_data"]:
            page.mouse.move(640, 400)
            row["idle"] = page.evaluate(JS_FRAMES, 3000)
            st = page.evaluate(JS_STAGE)
            import threading
            page.evaluate("() => { window.__fr = null; }")
            page.evaluate("(ms) => { const ts=[]; const end=performance.now()+ms; (function f(t){ ts.push(t); if (performance.now()<end) requestAnimationFrame(f); else window.__fr=ts; })(performance.now()); }", 2200)
            page.mouse.move(st["x"] + st["w"] * .3, st["y"] + st["h"] * .5); page.mouse.down()
            for i in range(40): page.mouse.move(st["x"] + st["w"] * (.3 + i * .01), st["y"] + st["h"] * .5); time.sleep(0.04)
            page.mouse.up(); time.sleep(0.6)
            row["drag"] = page.evaluate("() => { const ts=window.__fr||[]; const d=ts.slice(1).map((t,i)=>t-ts[i]).sort((a,b)=>a-b); return d.length? {frames:d.length, median_ms:+d[Math.floor(d.length/2)].toFixed(1), p95_ms:+d[Math.floor(.95*(d.length-1))].toFixed(1), max_ms:+d[d.length-1].toFixed(1)} : null; }")
            reset(page)
            first = page.locator("[data-galaxy-focus]").first; name = first.get_attribute("aria-label").replace("Preview ", "")
            t0 = time.time(); first.click(); row["preview_status_ms"] = round((time.time() - t0) * 1000)
            row["flight_settle_s"] = settle(page, name)
        row["console_errors"] = errors
        perf[n] = row; ctx.close()
    OUT["performance_desktop_chromium_swiftshader"] = perf; save()

    # 2. Hit targets: desktop mouse and phone touch (10 and 30 worlds)
    hits = {}
    for n in ("10", "30"):
        ctx = chromium.new_context(**desktop); page, *_ = open_galaxy(ctx, URLS[n]); hits[f"{n}_desktop_mouse_1280"] = hit_sweep(page, "mouse"); ctx.close()
        ctx = chromium.new_context(viewport={"width": 390, "height": 844}, has_touch=True, is_mobile=True, device_scale_factor=3)
        page, *_ = open_galaxy(ctx, URLS[n]); hits[f"{n}_phone_touch_390"] = hit_sweep(page, "touch"); ctx.close()
    OUT["hit_targets"] = hits; save()

    # 3. Selected world: select it with a click on the world, then find how far from its centre a
    #    second click still opens the project (desktop, 10 worlds)
    ctx = chromium.new_context(**desktop); page, *_ = open_galaxy(ctx, URLS["10"])
    name = next(l["name"] for l in page.evaluate(JS_LABELS) if not l["hidden"])
    diag = []
    def select_then_click(offset):
        page.goto(URLS["10"] + "/galaxy"); ready(page); time.sleep(0.5)
        st = page.evaluate(JS_STAGE); lab = next(l for l in page.evaluate(JS_LABELS) if l["name"] == name)
        page.mouse.click(st["x"] + lab["x"], st["y"] + lab["y"])
        page.mouse.move(st["x"] + st["w"] / 2, st["y"] + st["h"] / 2)
        selected = page.evaluate(JS_STATUS).startswith(name); settle(page, name)
        st = page.evaluate(JS_STAGE); lab = next(l for l in page.evaluate(JS_LABELS) if l["name"] == name)
        page.mouse.click(st["x"] + lab["x"] + offset, st["y"] + lab["y"]); time.sleep(1.0)
        opened = "/projects/" in page.url
        diag.append({"offset": round(offset), "selected_first": selected, "opened": opened})
        return opened
    lo, hi = 0.0, 400.0
    if select_then_click(0):
        while hi - lo > 4:
            mid = (lo + hi) / 2
            lo, hi = (mid, hi) if select_then_click(mid) else (lo, mid)
    OUT["selected_world_click_open_desktop"] = {"world": name, "edge_px_from_centre": round(lo), "trials": diag}
    ctx.close()

    # 4. Ship, roam, keyboard, zoom, reduced motion (desktop, 10 worlds)
    ctx = chromium.new_context(**desktop); page, *_ = open_galaxy(ctx, URLS["10"]); st = page.evaluate(JS_STAGE)
    def crop(x, y, name):
        return page.screenshot(clip={"x": x - 40, "y": y - 40, "width": 80, "height": 80}, path=f"media/{name}.png")
    ax, ay = st["x"] + st["w"] * .25, st["y"] + st["h"] * .75
    bx, by = st["x"] + st["w"] * .52, st["y"] + st["h"] * .55  # inside the roam dead zone
    page.mouse.move(ax, ay); time.sleep(0.6); a1 = crop(ax, ay, "_ship_at_A")
    page.mouse.move(bx, by); time.sleep(0.6); a2 = crop(ax, ay, "_A_after_move")
    def diff(b1, b2):
        return sum(1 for i, j in zip(b1, b2) if i != j) / max(1, len(b1))
    before = {l["name"]: (l["x"], l["y"]) for l in page.evaluate(JS_LABELS)}
    time.sleep(2); centre = {l["name"]: (l["x"], l["y"]) for l in page.evaluate(JS_LABELS)}
    page.mouse.move(st["x"] + st["w"] * .97, st["y"] + st["h"] * .5); time.sleep(2)
    edge = {l["name"]: (l["x"], l["y"]) for l in page.evaluate(JS_LABELS)}
    moved = lambda a, b: max(abs(a[k][0] - b[k][0]) for k in a if a[k][0] is not None and b.get(k) and b[k][0] is not None)
    OUT["ship_and_roam_desktop"] = {
        "ship_crop_bytes_differ_after_pointer_moves_away": round(diff(a1, a2), 3),
        "label_max_shift_px_pointer_centre_2s": round(moved(before, centre), 1),
        "label_max_shift_px_pointer_right_edge_2s": round(moved(centre, edge), 1)}
    page.mouse.move(ax, ay); time.sleep(0.6)
    flap = [crop(ax, ay, f"_flap{i}") for i in range(4) if not time.sleep(0.07)]
    OUT["ship_and_roam_desktop"]["ship_crop_changes_while_pointer_still"] = round(max(diff(flap[0], f) for f in flap[1:]), 3)
    page.mouse.move(bx, by); reset(page)
    # zoom
    b = {l["name"]: l for l in page.evaluate(JS_LABELS)}
    page.locator('[data-galaxy-zoom="in"]').click(); page.locator('[data-galaxy-zoom="in"]').click(); time.sleep(2)
    a = {l["name"]: l for l in page.evaluate(JS_LABELS)}
    spread = lambda L: statistics.pstdev([v["x"] for v in L.values() if v["x"] is not None])
    OUT["zoom_buttons_label_spread_px"] = {"before": round(spread(b), 1), "after_2x_zoom_in": round(spread(a), 1)}
    page.locator("[data-galaxy-reset]").click(); time.sleep(2)
    # keyboard
    page.locator("[data-galaxy-focus]").first.click(); first_name = page.evaluate("() => document.querySelector('[data-galaxy-name]').textContent")
    focused = page.evaluate("() => document.activeElement.textContent.trim()")
    page.locator("[data-galaxy-stage]").focus(); page.keyboard.press("ArrowRight"); time.sleep(0.2)
    after_arrow = page.evaluate("() => document.querySelector('[data-galaxy-name]').textContent")
    page.keyboard.press("Escape"); time.sleep(0.2)
    OUT["keyboard"] = {"preview_button_focus_moves_to": focused, "arrow_right_changes_selection": first_name != after_arrow,
                       "escape_closes_panel": page.evaluate("() => document.querySelector('[data-galaxy-context]').hidden")}
    tab_order = []
    page.evaluate("() => document.querySelector('[data-galaxy]').scrollIntoView()"); page.locator(".galaxy-hub a").first.focus()
    for i in range(10):
        tab_order.append(page.evaluate("() => { const e=document.activeElement; return (e.getAttribute('aria-label')||e.textContent||e.tagName).trim().slice(0,40) }"))
        page.keyboard.press("Tab")
    OUT["keyboard"]["tab_order_from_hub"] = tab_order
    # contrast of text over the dark scene
    OUT["contrast"] = page.evaluate("""() => { const lum=c=>{const v=c.match(/\\d+(\\.\\d+)?/g).slice(0,3).map(Number).map(x=>{x/=255;return x<=.03928?x/12.92:Math.pow((x+.055)/1.055,2.4)});return .2126*v[0]+.7152*v[1]+.0722*v[2]};
      const bg=lum('rgb(3,5,11)'); const r=s=>{const e=document.querySelector(s); if(!e) return null; const L=lum(getComputedStyle(e).color); return +((Math.max(L,bg)+.05)/(Math.min(L,bg)+.05)).toFixed(2)};
      document.querySelector('[data-galaxy-focus]').click();
      const c=document.querySelector('[data-galaxy-close]'); const cs=getComputedStyle(c); const bgc=cs.backgroundColor;
      const cl=lum(cs.color), bl=(/rgba?\\(0, 0, 0, 0\\)|transparent/.test(bgc)) ? lum(getComputedStyle(document.querySelector('[data-galaxy-context]')).backgroundColor) : lum(bgc);
      const closeRatio=+((Math.max(cl,bl)+.05)/(Math.min(cl,bl)+.05)).toFixed(2);
      document.querySelector('[data-galaxy-close]').click();
      return {close_button_fg_vs_bg: closeRatio, close_button_colors: [cs.color, bgc], planet_label: r('.galaxy-planet-label'), status: r('[data-galaxy-status]'), legend: r('.galaxy-legend span'), assumed_background: '#03050b (scene)'}; }""")
    ctx.close()
    # reduced motion (OS preference and the app setting)
    rm = {}
    for label, kw, init in (("os_prefers_reduced_motion", {"reduced_motion": "reduce"}, None),
                            ("app_setting_lifeops_motion", {}, "localStorage.setItem('lifeops-motion','reduced')")):
        ctx = chromium.new_context(**desktop, **kw)
        if init: ctx.add_init_script(init)
        page, *_ = open_galaxy(ctx, URLS["10"])
        btn = page.locator("[data-galaxy-focus]").first; name = btn.get_attribute("aria-label").replace("Preview ", "")
        btn.click(); rm[label] = {"flight_settle_s": settle(page, name),
                                  "dataset_motion": page.evaluate("() => document.documentElement.dataset.motion || null")}
        reset(page); st = page.evaluate(JS_STAGE); x, y = st["x"] + st["w"] * .25, st["y"] + st["h"] * .75
        page.mouse.move(x, y); time.sleep(0.6)
        shots = []
        for i in range(4):
            shots.append(page.screenshot(clip={"x": x - 40, "y": y - 40, "width": 80, "height": 80})); time.sleep(0.07)
        rm[label]["ship_crop_changes_while_pointer_still"] = round(max(sum(1 for a, b in zip(shots[0], f) if a != b) / len(f) for f in shots[1:]), 3)
        ctx.close()
    OUT["reduced_motion"] = rm; save()

    # 5. Phone: touch drag, tap-vs-drag, pinch, preview obstruction, scroll-hide (Chromium mobile + WebKit iPhone emulation)
    phone = {}
    for engine, browser, dev in (("chromium", chromium, None), ("webkit_iphone13_emulated", webkit, p.devices["iPhone 13"])):
        for w, h in ((390, 844), (430, 932)):
            kw = dict(dev) if dev else dict(has_touch=True, is_mobile=True, device_scale_factor=3)
            kw["viewport"] = {"width": w, "height": h}
            ctx = browser.new_context(**kw); page, errors, nav_ms, _ = open_galaxy(ctx, URLS["10"]); st = page.evaluate(JS_STAGE)
            r = {"load_to_ready_ms": nav_ms, "console_errors": errors}
            page.locator("[data-galaxy-focus]").first.tap(); time.sleep(1.2)
            r["open_project_centre_hits_button"] = page.evaluate("""() => { const o=document.querySelector('[data-galaxy-open]').getBoundingClientRect();
                const e=document.elementFromPoint(o.left+o.width/2,o.top+o.height/2); return !!(e && e.closest('[data-galaxy-open]')); }""")
            r["open_project_obscured_by"] = page.evaluate("""() => { const o=document.querySelector('[data-galaxy-open]').getBoundingClientRect();
                const e=document.elementFromPoint(o.left+o.width/2,o.top+o.height/2); return e ? (e.closest('nav') ? 'bottom tab bar' : e.tagName) : null; }""")
            page.screenshot(path=f"media/phone-{engine}-{w}-selected.jpg", type="jpeg", quality=70)
            reset(page); page.evaluate("() => { const r = document.querySelector('[data-galaxy-stage]').getBoundingClientRect(); window.scrollBy(0, r.top); }")
            time.sleep(0.3); st = page.evaluate(JS_STAGE)
            lab = next(l for l in page.evaluate(JS_LABELS) if not l["hidden"])
            page.touchscreen.tap(st["x"] + lab["x"], st["y"] + lab["y"]); time.sleep(1.5)
            r["world_tap_selected"] = page.evaluate(JS_STATUS).startswith(lab["name"])
            r["world_tap_open_project_centre_hits_button"] = page.evaluate("""() => { const o=document.querySelector('[data-galaxy-open]').getBoundingClientRect();
                const e=document.elementFromPoint(o.left+o.width/2,o.top+o.height/2); return !!(e && e.closest('[data-galaxy-open]')); }""")
            r["world_tap_open_project_obscured_by"] = page.evaluate("""() => { const o=document.querySelector('[data-galaxy-open]').getBoundingClientRect();
                const e=document.elementFromPoint(o.left+o.width/2,o.top+o.height/2); return e ? (e.closest('nav') ? 'bottom tab bar' : (e.closest('[data-galaxy-open]') ? 'none (button itself)' : e.tagName)) : 'offscreen'; }""")
            r["world_tap_open_project_rect_vs_viewport"] = page.evaluate("() => { const o=document.querySelector('[data-galaxy-open]').getBoundingClientRect(); return [Math.round(o.top), Math.round(o.bottom), innerHeight]; }")
            page.screenshot(path=f"media/phone-{engine}-{w}-world-tap.jpg", type="jpeg", quality=70)
            page.evaluate("() => window.scrollTo(0,0)"); time.sleep(0.3)
            page.evaluate("() => window.scrollBy(0, 500)"); time.sleep(0.6)
            r["scroll_down_body_has_galaxy_nav_hidden"] = page.evaluate("() => document.body.classList.contains('galaxy-nav-hidden')")
            r["top_header_hidden_after_scroll"] = page.evaluate("() => { const h=document.querySelector('header.gyst-topbar'); if(!h) return null; const cs=getComputedStyle(h); return cs.opacity==='0' || cs.transform.includes('matrix') && h.getBoundingClientRect().bottom<=0; }")
            r["bottom_bar_still_visible_after_scroll"] = page.evaluate("() => { const n=document.querySelector('nav.gyst-primary'); return !!n && getComputedStyle(n).opacity!=='0' && n.getBoundingClientRect().top < innerHeight; }")
            ctx.close()
            phone[f"{engine}_{w}"] = r
    # pinch + drag via CDP touch (Chromium)
    ctx = chromium.new_context(viewport={"width": 390, "height": 844}, has_touch=True, is_mobile=True, device_scale_factor=3)
    page, *_ = open_galaxy(ctx, URLS["10"]); st = page.evaluate(JS_STAGE); cdp = ctx.new_cdp_session(page)
    cx, cy = st["x"] + st["w"] / 2, st["y"] + st["h"] / 2
    def touch(t, pts): cdp.send("Input.dispatchTouchEvent", {"type": t, "touchPoints": [{"x": x, "y": y, "id": i} for i, (x, y) in enumerate(pts)]})
    b = {l["name"]: l for l in page.evaluate(JS_LABELS)}
    touch("touchStart", [(cx - 30, cy), (cx + 30, cy)])
    for k in range(1, 11): touch("touchMove", [(cx - 30 - k * 8, cy), (cx + 30 + k * 8, cy)]); time.sleep(0.03)
    touch("touchEnd", []); time.sleep(2)
    a = {l["name"]: l for l in page.evaluate(JS_LABELS)}
    s0 = page.evaluate(JS_STATUS)
    touch("touchStart", [(cx, cy)])
    for k in range(1, 11): touch("touchMove", [(cx + k * 10, cy)]); time.sleep(0.03)
    touch("touchEnd", []); time.sleep(1.5)
    c = {l["name"]: l for l in page.evaluate(JS_LABELS)}
    reset(page); page.evaluate("() => { const r = document.querySelector('[data-galaxy-stage]').getBoundingClientRect(); window.scrollBy(0, r.top); }")
    time.sleep(0.3); st = page.evaluate(JS_STAGE); lab = next(l for l in page.evaluate(JS_LABELS) if not l["hidden"])
    page.touchscreen.tap(st["x"] + lab["x"], st["y"] + lab["y"]); time.sleep(0.12); page.touchscreen.tap(st["x"] + lab["x"], st["y"] + lab["y"]); time.sleep(1.5)
    phone["double_tap_world_opens_project_390"] = "/projects/" in page.url
    page.goto(URLS["10"] + "/galaxy"); ready(page); st = page.evaluate(JS_STAGE)
    phone["cdp_touch_390"] = {"pinch_out_label_spread_px": [round(spread(b), 1), round(spread(a), 1)],
                              "drag_label_max_shift_px": round(max(abs(a[k]["x"] - c[k]["x"]) for k in a if a[k]["x"] is not None and c[k]["x"] is not None), 1),
                              "drag_did_not_select": page.evaluate(JS_STATUS).startswith("Drag to explore")}
    ctx.close()
    OUT["phone"] = phone; save()

    # 6. Layout per viewport (Chromium, 30 worlds)
    layout = {}
    for w, h in ((390, 844), (430, 932), (768, 1024), (1280, 800), (1440, 900), (1920, 1080)):
        ctx = chromium.new_context(viewport={"width": w, "height": h}); page, *_ = open_galaxy(ctx, URLS["30"])
        layout[w] = page.evaluate("""() => { const R=s=>{const e=document.querySelector(s); if(!e) return null; const r=e.getBoundingClientRect(); return [Math.round(r.width),Math.round(r.height)]};
            document.querySelector('[data-galaxy-focus]').click();
            return {overflow_x: document.documentElement.scrollWidth > innerWidth, stage: R('[data-galaxy-stage]'),
                    visible_labels: [...document.querySelectorAll('.galaxy-planet-label')].filter(l=>!l.hidden).length,
                    zoom_in: R('[data-galaxy-zoom="in"]'), reset: R('[data-galaxy-reset]'), close: R('[data-galaxy-close]'), open: R('[data-galaxy-open]'),
                    preview_button: R('[data-galaxy-focus]'), nav_items: [...document.querySelectorAll('nav a')].filter(a=>a.offsetParent).map(a=>a.textContent.trim().replace(/\\s+/g,' '))}; }""")
        ctx.close()
    OUT["layout_30_worlds"] = layout; save()

    # 7. Media: fictional data only
    shots = [("1280x800", {"width": 1280, "height": 800}, {}), ("390x844", {"width": 390, "height": 844}, dict(has_touch=True, is_mobile=True, device_scale_factor=2))]
    for tag, vp, extra in shots:
        for n, state in (("10", "idle"), ("10", "selected"), ("10", "zoomed"), ("0", "empty"), ("1", "sparse-1"), ("3", "sparse-3")):
            ctx = chromium.new_context(viewport=vp, **extra); page, *_ = open_galaxy(ctx, URLS[n]); time.sleep(1)
            if state == "selected":
                page.locator("[data-galaxy-focus]").first.click(); time.sleep(4); page.evaluate("() => document.querySelector('[data-galaxy]').scrollIntoView()")
            if state == "zoomed":
                for _ in range(3): page.locator('[data-galaxy-zoom="in"]').click()
                time.sleep(3)
            page.screenshot(path=f"media/galaxy-{tag}-{state}.jpg", type="jpeg", quality=70); ctx.close()
    nogl = p.chromium.launch(args=["--disable-webgl", "--disable-3d-apis"])
    for tag, vp, extra in shots:
        ctx = nogl.new_context(viewport=vp, **extra); page, errors, *_ = open_galaxy(ctx, URLS["10"])
        OUT.setdefault("webgl_fallback", {})[tag] = {"status": page.evaluate(JS_STATUS), "fallback_visible": page.evaluate("() => !document.querySelector('[data-galaxy-fallback]').hidden"),
                                                     "navigator_items": page.evaluate("() => document.querySelectorAll('[data-galaxy-list] > li').length")}
        page.screenshot(path=f"media/galaxy-{tag}-webgl-fallback.jpg", type="jpeg", quality=70); ctx.close()
    nogl.close()
    # accessibility snapshot (structure only)
    ctx = chromium.new_context(**desktop); page, *_ = open_galaxy(ctx, URLS["3"])
    OUT["accessibility_snapshot_excerpt"] = page.locator("[data-galaxy]").aria_snapshot()[:3000]
    ctx.close()
    chromium.close(); webkit.close()

for f in os.listdir("media"):
    if f.startswith("_"): os.remove(os.path.join("media", f))
json.dump(OUT, open("results.json", "w"), indent=1, default=str)
print(json.dumps(OUT, indent=1, default=str)[:20000])
