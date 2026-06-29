(function () {
  var THEME_KEY = "lifeops-theme";

  function cssVar(name) {
    return getComputedStyle(document.documentElement).getPropertyValue(name).trim();
  }

  function parseList(value) {
    if (!value) return [];
    return value.split("|").filter(Boolean);
  }

  function parseValues(value) {
    return parseList(value).map(function (item) {
      var parsed = Number(item);
      return Number.isFinite(parsed) ? parsed : 0;
    });
  }

  function setupCanvas(canvas) {
    var ratio = window.devicePixelRatio || 1;
    var rect = canvas.getBoundingClientRect();
    var mini = canvas.dataset.mini === "true";
    var width = Math.max(mini ? 120 : 320, Math.floor(rect.width));
    var height = Math.max(mini ? 72 : 220, Math.floor(rect.height || (mini ? 78 : 260)));
    canvas.width = width * ratio;
    canvas.height = height * ratio;
    canvas.style.height = height + "px";
    var ctx = canvas.getContext("2d");
    ctx.setTransform(ratio, 0, 0, ratio, 0, 0);
    return { ctx: ctx, width: width, height: height, mini: mini };
  }

  function palette() {
    return {
      text: cssVar("--text") || "#eef7ff",
      muted: cssVar("--muted") || "#92a4b7",
      border: cssVar("--border") || "rgba(166, 213, 255, 0.16)",
      cyan: cssVar("--cyan") || "#65e8ff",
      blue: cssVar("--blue") || "#78a7ff",
      purple: cssVar("--purple") || "#a98cff"
    };
  }

  function drawEmpty(ctx, width, height, colors, mini, text) {
    ctx.clearRect(0, 0, width, height);
    // Soft dashed baseline so an empty card reads as "no data yet" rather than
    // looking broken. Honest placeholder, no fabricated values.
    var pad = mini ? 4 : 44;
    var baseline = height - (mini ? 14 : 42);
    ctx.save();
    ctx.strokeStyle = colors.border;
    ctx.lineWidth = 1;
    if (ctx.setLineDash) ctx.setLineDash([4, 5]);
    ctx.beginPath();
    ctx.moveTo(pad, baseline);
    ctx.lineTo(width - pad, baseline);
    ctx.stroke();
    ctx.restore();

    ctx.fillStyle = colors.muted;
    ctx.font = (mini ? "11px" : "14px") + " system-ui, sans-serif";
    ctx.textAlign = "center";
    var message = text || (mini ? "No data yet" : "No chart data yet");
    ctx.fillText(message, width / 2, (mini ? baseline - 16 : height / 2));
  }

  function drawAxes(ctx, left, top, right, bottom, colors) {
    ctx.strokeStyle = colors.border;
    ctx.lineWidth = 1;
    ctx.beginPath();
    ctx.moveTo(left, top);
    ctx.lineTo(left, bottom);
    ctx.lineTo(right, bottom);
    ctx.stroke();
  }

  function drawMiniGrid(ctx, left, top, right, bottom, colors) {
    ctx.strokeStyle = colors.border;
    ctx.lineWidth = 1;
    [0.33, 0.66].forEach(function (mark) {
      var y = top + (bottom - top) * mark;
      ctx.beginPath();
      ctx.moveTo(left, y);
      ctx.lineTo(right, y);
      ctx.stroke();
    });
  }

  function chartMax(canvas, values, fallback) {
    var forcedMax = Number(canvas.dataset.max);
    if (Number.isFinite(forcedMax) && forcedMax > 0) return forcedMax;
    return Math.max.apply(null, values.concat([fallback || 1]));
  }

  function drawBar(canvas, labels, values) {
    var setup = setupCanvas(canvas);
    var ctx = setup.ctx;
    var width = setup.width;
    var height = setup.height;
    var mini = setup.mini;
    var colors = palette();
    ctx.clearRect(0, 0, width, height);
    if (!values.length) {
      drawEmpty(ctx, width, height, colors, mini, canvas.dataset.empty);
      return;
    }

    var left = mini ? 4 : 44;
    var right = width - (mini ? 4 : 20);
    var top = mini ? 8 : 18;
    var bottom = height - (mini ? 10 : 42);
    var max = chartMax(canvas, values, 1);
    var gap = 8;
    var slot = (right - left) / values.length;
    var barWidth = Math.max(5, slot - gap);
    var gradient = ctx.createLinearGradient(0, top, 0, bottom);
    gradient.addColorStop(0, colors.cyan);
    gradient.addColorStop(1, colors.purple);

    if (mini) drawMiniGrid(ctx, left, top, right, bottom, colors);
    else drawAxes(ctx, left, top, right, bottom, colors);

    ctx.fillStyle = gradient;
    var radius = Math.min(6, barWidth / 2);
    values.forEach(function (value, index) {
      var barHeight = ((bottom - top) * value) / max;
      var x = left + index * slot + gap / 2;
      var y = bottom - barHeight;
      // Rounded tops read as premium pills rather than flat blocks. Guard tiny
      // bars so the radius never exceeds the height.
      var r = Math.min(radius, barHeight);
      ctx.beginPath();
      if (ctx.roundRect) {
        ctx.roundRect(x, y, barWidth, barHeight, [r, r, 0, 0]);
      } else {
        ctx.moveTo(x, bottom);
        ctx.lineTo(x, y + r);
        ctx.quadraticCurveTo(x, y, x + r, y);
        ctx.lineTo(x + barWidth - r, y);
        ctx.quadraticCurveTo(x + barWidth, y, x + barWidth, y + r);
        ctx.lineTo(x + barWidth, bottom);
        ctx.closePath();
      }
      ctx.fill();
    });

    if (!mini) {
      ctx.fillStyle = colors.muted;
      ctx.font = "11px system-ui, sans-serif";
      ctx.textAlign = "center";
      var step = Math.max(1, Math.ceil(labels.length / 10));
      labels.forEach(function (label, index) {
        if (index % step !== 0 && index !== labels.length - 1) return;
        var x = left + index * slot + slot / 2;
        ctx.fillText(label, x, height - 16);
      });

      ctx.textAlign = "right";
      ctx.fillText(String(max), left - 8, top + 4);
      ctx.fillText("0", left - 8, bottom + 4);
    }
  }

  function drawLine(canvas, labels, values) {
    var setup = setupCanvas(canvas);
    var ctx = setup.ctx;
    var width = setup.width;
    var height = setup.height;
    var mini = setup.mini;
    var colors = palette();
    ctx.clearRect(0, 0, width, height);
    if (!values.length) {
      drawEmpty(ctx, width, height, colors, mini, canvas.dataset.empty);
      return;
    }

    var left = mini ? 4 : 44;
    var right = width - (mini ? 4 : 20);
    var top = mini ? 8 : 18;
    var bottom = height - (mini ? 10 : 42);
    var max = chartMax(canvas, values, 1);
    var min = 0;
    var range = Math.max(1, max - min);
    var slot = values.length === 1 ? 0 : (right - left) / (values.length - 1);

    if (mini) drawMiniGrid(ctx, left, top, right, bottom, colors);
    else drawAxes(ctx, left, top, right, bottom, colors);

    if (!mini) {
      ctx.strokeStyle = colors.border;
      ctx.lineWidth = 1;
      [25, 50, 75, 100].forEach(function (mark) {
        if (mark > max) return;
        var y = bottom - ((bottom - top) * mark) / range;
        ctx.beginPath();
        ctx.moveTo(left, y);
        ctx.lineTo(right, y);
        ctx.stroke();
      });
    }

    var points = [];
    values.forEach(function (value, index) {
      var x = values.length === 1 ? (left + right) / 2 : left + index * slot;
      var y = bottom - ((bottom - top) * (value - min)) / range;
      points.push({ x: x, y: y });
    });

    // Trace the route once so both the area fill and the stroke follow the same
    // smoothed path. skipMove omits the leading moveTo so the fill can keep its
    // baseline anchor (otherwise moveTo starts a new subpath and the fill closes
    // diagonally instead of dropping straight down to the baseline).
    function tracePath(skipMove) {
      points.forEach(function (point, index) {
        if (index === 0) {
          if (!skipMove) ctx.moveTo(point.x, point.y);
          return;
        }
        var prev = points[index - 1];
        var midX = (prev.x + point.x) / 2;
        ctx.quadraticCurveTo(prev.x, prev.y, midX, (prev.y + point.y) / 2);
        ctx.quadraticCurveTo(point.x, point.y, point.x, point.y);
      });
    }

    // Soft area fill under the route reads as accumulated progress — premium,
    // no heavy blur. Skipped when there's a single point (nothing to fill).
    if (points.length > 1) {
      ctx.save();
      ctx.beginPath();
      ctx.moveTo(points[0].x, bottom);
      ctx.lineTo(points[0].x, points[0].y);
      tracePath(true);
      ctx.lineTo(points[points.length - 1].x, bottom);
      ctx.closePath();
      var fill = ctx.createLinearGradient(0, top, 0, bottom);
      fill.addColorStop(0, "rgba(101, 232, 255, 0.26)");
      fill.addColorStop(1, "rgba(101, 232, 255, 0)");
      ctx.fillStyle = fill;
      ctx.fill();
      ctx.restore();
    }

    var gradient = ctx.createLinearGradient(left, 0, right, 0);
    gradient.addColorStop(0, colors.cyan);
    gradient.addColorStop(1, colors.blue);
    ctx.strokeStyle = gradient;
    ctx.lineWidth = mini ? 2.4 : 3;
    ctx.lineJoin = "round";
    ctx.lineCap = "round";
    ctx.beginPath();
    tracePath();
    ctx.stroke();

    // Current point: the last value, highlighted with a soft halo so the eye
    // lands on "where you are now" along the trajectory.
    var last = points[points.length - 1];
    ctx.save();
    ctx.fillStyle = "rgba(101, 232, 255, 0.18)";
    ctx.beginPath();
    ctx.arc(last.x, last.y, mini ? 5 : 8, 0, Math.PI * 2);
    ctx.fill();
    ctx.fillStyle = colors.cyan;
    ctx.shadowColor = colors.cyan;
    ctx.shadowBlur = mini ? 4 : 8;
    ctx.beginPath();
    ctx.arc(last.x, last.y, mini ? 2.6 : 4, 0, Math.PI * 2);
    ctx.fill();
    ctx.restore();

    if (!mini) {
      // Milestone nodes for the earlier points (the current one is already the
      // glowing marker above).
      ctx.fillStyle = colors.blue;
      points.slice(0, -1).forEach(function (point) {
        ctx.beginPath();
        ctx.arc(point.x, point.y, 3, 0, Math.PI * 2);
        ctx.fill();
      });

      ctx.fillStyle = colors.muted;
      ctx.font = "11px system-ui, sans-serif";
      ctx.textAlign = "center";
      var step = Math.max(1, Math.ceil(labels.length / 8));
      labels.forEach(function (label, index) {
        if (index % step !== 0 && index !== labels.length - 1) return;
        var x = values.length === 1 ? (left + right) / 2 : left + index * slot;
        ctx.fillText(label, x, height - 16);
      });

      ctx.textAlign = "right";
      ctx.fillText(String(max), left - 8, top + 4);
      ctx.fillText("0", left - 8, bottom + 4);
    }
  }

  function drawCharts() {
    document.querySelectorAll("canvas[data-chart]").forEach(function (canvas) {
      var labels = parseList(canvas.dataset.labels);
      var values = parseValues(canvas.dataset.values);
      if (canvas.dataset.chart === "line") drawLine(canvas, labels, values);
      else drawBar(canvas, labels, values);
    });
  }

  function scheduleDrawCharts(delay) {
    window.clearTimeout(window.__lifeOpsChartTimer);
    window.__lifeOpsChartTimer = window.setTimeout(function () {
      window.requestAnimationFrame(drawCharts);
    }, delay || 0);
  }

  function openDrawer() {
    document.body.classList.add("drawer-open");
    var drawer = document.querySelector(".more-drawer");
    if (drawer) drawer.setAttribute("aria-hidden", "false");
  }

  function closeDrawer() {
    document.body.classList.remove("drawer-open");
    var drawer = document.querySelector(".more-drawer");
    if (drawer) drawer.setAttribute("aria-hidden", "true");
  }

  function initDrawer() {
    document.querySelectorAll("[data-more-open]").forEach(function (button) {
      button.addEventListener("click", openDrawer);
    });
    document.querySelectorAll("[data-more-close]").forEach(function (button) {
      button.addEventListener("click", closeDrawer);
    });
    document.addEventListener("keydown", function (event) {
      if (event.key === "Escape") closeDrawer();
    });
  }

  var TRACKER_STATUSES = ["done", "not_done", "skipped", "rest"];

  function currentTrackerStatus(button) {
    for (var i = 0; i < TRACKER_STATUSES.length; i += 1) {
      if (button.classList.contains("status-" + TRACKER_STATUSES[i])) {
        return TRACKER_STATUSES[i];
      }
    }
    return null;
  }

  function nextTrackerStatus(button, current) {
    if (button.classList.contains("gym-day")) {
      if (current === "done") return "not_done";
      if (current === "not_done") return "rest";
      return "done";
    }
    return current === "done" ? "not_done" : "done";
  }

  function setTrackerStatus(button, status) {
    TRACKER_STATUSES.forEach(function (name) {
      button.classList.remove("status-" + name);
    });
    if (status) button.classList.add("status-" + status);
    var title = button.getAttribute("title");
    if (title && title.indexOf(":") !== -1) {
      button.setAttribute("title", title.slice(0, title.lastIndexOf(":") + 1) + " " + (status || ""));
    }
  }

  function percent(done, total) {
    if (!total) return null;
    return Math.floor((done * 100) / total);
  }

  // Recompute the mission scores straight from the current DOM. Used for the
  // optimistic update that fires the instant a toggle is clicked (works on
  // every page, even ones without the dashboard orb/signal cards). The server
  // response then refines these with authoritative numbers.
  function missionFromDom() {
    var data = {};
    var prayerTotal = document.querySelectorAll(".prayer-row .tracker-dot").length;
    if (prayerTotal) {
      data.prayer = percent(document.querySelectorAll(".prayer-row .tracker-dot.status-done").length, prayerTotal);
    }
    var athkarTotal = document.querySelectorAll(".athkar-row .tracker-dot").length;
    if (athkarTotal) {
      data.athkar = percent(document.querySelectorAll(".athkar-row .tracker-dot.status-done").length, athkarTotal);
    }
    var deenTotal = document.querySelectorAll(".tracker-row .tracker-dot").length;
    if (deenTotal) {
      data.deen = percent(document.querySelectorAll(".tracker-row .tracker-dot.status-done").length, deenTotal);
    }
    if (document.querySelectorAll(".gym-strip .gym-day").length) {
      // Gym completion is always measured against a 7-day week (matches server).
      data.gym = percent(document.querySelectorAll(".gym-strip .gym-day.status-done").length, 7);
    }
    return data;
  }

  function applyMissionSummary(data) {
    if (!data) return;
    ["prayer", "athkar", "deen", "gym"].forEach(function (field) {
      var value = data[field];
      if (typeof value !== "number") return;
      document.querySelectorAll('[data-mission="' + field + '"]').forEach(function (el) {
        el.textContent = value + "%";
      });
    });
    if (typeof data.deen === "number") {
      document.querySelectorAll(".progress-orb").forEach(function (orb) {
        orb.style.setProperty("--value", data.deen);
      });
    }
  }

  // Only used for the failure path; success feedback is the in-place pulseSaved.
  function showToast(message) {
    var toast = document.querySelector(".lifeops-toast");
    if (!toast) {
      toast = document.createElement("div");
      toast.className = "lifeops-toast";
      document.body.appendChild(toast);
    }
    toast.textContent = message;
    window.requestAnimationFrame(function () {
      toast.classList.add("is-visible");
    });
    window.clearTimeout(window.__lifeOpsToastTimer);
    window.__lifeOpsToastTimer = window.setTimeout(function () {
      toast.classList.remove("is-visible");
    }, 2600);
  }

  function pulseSaved(button) {
    button.classList.remove("is-saved");
    // force reflow so the animation restarts on rapid repeated toggles
    void button.offsetWidth;
    button.classList.add("is-saved");
    window.setTimeout(function () {
      button.classList.remove("is-saved");
    }, 500);
  }

  function toggleTracker(button, form) {
    if (button.dataset.busy === "1") return;
    var previous = currentTrackerStatus(button);
    setTrackerStatus(button, nextTrackerStatus(button, previous));
    applyMissionSummary(missionFromDom());
    button.dataset.busy = "1";

    var body = new URLSearchParams();
    var dateInput = form.querySelector('input[name="log_date"]');
    if (dateInput) body.set("log_date", dateInput.value);

    fetch(form.getAttribute("action"), {
      method: "POST",
      headers: {
        "X-Requested-With": "fetch",
        "Content-Type": "application/x-www-form-urlencoded"
      },
      body: body.toString()
    })
      .then(function (response) {
        if (!response.ok) throw new Error("request failed");
        return response.json();
      })
      .then(function (data) {
        if (data && data.status) setTrackerStatus(button, data.status);
        applyMissionSummary(data);
        button.dataset.busy = "";
        pulseSaved(button);
      })
      .catch(function () {
        setTrackerStatus(button, previous);
        applyMissionSummary(missionFromDom());
        button.dataset.busy = "";
        showToast("Could not save. Reverted.");
      });
  }

  function initTrackers() {
    document.addEventListener("click", function (event) {
      var button = event.target.closest(".tracker-dot, .gym-day");
      if (!button) return;
      var form = button.closest('form[action*="/toggle"]');
      if (!form) return;
      event.preventDefault();
      event.stopPropagation();
      toggleTracker(button, form);
    });
  }

  function applyTheme(theme) {
    if (theme === "light") document.documentElement.dataset.theme = "light";
    else delete document.documentElement.dataset.theme;
    scheduleDrawCharts(0);
  }

  function initTheme() {
    var stored = localStorage.getItem(THEME_KEY);
    applyTheme(stored);
    var toggle = document.querySelector("[data-theme-toggle]");
    if (!toggle) return;
    toggle.checked = stored === "light";
    toggle.addEventListener("change", function () {
      var next = toggle.checked ? "light" : "dark";
      if (next === "light") localStorage.setItem(THEME_KEY, "light");
      else localStorage.removeItem(THEME_KEY);
      applyTheme(next);
    });
  }

  // Scroll memory: a no-JS POST already returns to the acted-on item via a
  // #fragment (see return_to). This is the belt-and-suspenders fallback for the
  // forms that redirect back WITHOUT an anchor (status changes, deletes): keep
  // the user's place instead of snapping to the top. It defers entirely to the
  // anchor when one is present so the two never fight.
  function initScrollMemory() {
    var key = "lifeops-scroll:" + window.location.pathname;
    document.addEventListener("submit", function (event) {
      var form = event.target;
      if (!form || (form.method || "").toLowerCase() !== "post") return;
      try {
        sessionStorage.setItem(key, String(window.scrollY));
      } catch (err) {
        /* storage unavailable; ignore */
      }
    });
    try {
      var saved = sessionStorage.getItem(key);
      sessionStorage.removeItem(key);
      if (saved !== null && !window.location.hash) {
        window.scrollTo(0, parseInt(saved, 10) || 0);
      }
    } catch (err) {
      /* storage unavailable; ignore */
    }
  }

  window.addEventListener("load", function () {
    initTheme();
    initDrawer();
    initTrackers();
    initScrollMemory();
    scheduleDrawCharts(0);
  });

  window.addEventListener("resize", function () {
    scheduleDrawCharts(140);
  });
})();
