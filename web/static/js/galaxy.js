import { bindGalaxyGestures } from "/static/js/galaxy-gestures.js?v=ui-4";
const root = document.querySelector("[data-galaxy]");
if (root) {
  initGalaxyHeader();
  import("/static/js/vendor/three.module.js")
    .then((THREE) => initGalaxy(root, THREE))
    .catch((error) => showGalaxyFallback(root, error));
}

function initGalaxyHeader() {
  let previousY = window.scrollY;
  let ticking = false;
  const update = () => {
    const currentY = window.scrollY;
    const movingDown = currentY > previousY + 5;
    const movingUp = currentY < previousY - 5;
    if (currentY <= 12 || movingUp) document.body.classList.remove("galaxy-nav-hidden");
    else if (movingDown) document.body.classList.add("galaxy-nav-hidden");
    previousY = currentY;
    ticking = false;
  };
  window.addEventListener("scroll", () => {
    if (!ticking) requestAnimationFrame(update);
    ticking = true;
  }, { passive: true });
}

function showGalaxyFallback(root, error) {
  const status = root.querySelector("[data-galaxy-status]");
  const fallback = root.querySelector("[data-galaxy-fallback]");
  if (status) status.textContent = "3D view unavailable. Use the project list below.";
  if (fallback) fallback.hidden = false;
  root.classList.add("galaxy-load-failed");
  if (error) console.error("Galaxy startup failed", error);
}

async function initGalaxy(root, THREE) {
  const stage = root.querySelector("[data-galaxy-stage]");
  const canvas = root.querySelector("[data-galaxy-canvas]");
  const status = root.querySelector("[data-galaxy-status]");
  const list = root.querySelector("[data-galaxy-list]");
  const fallback = root.querySelector("[data-galaxy-fallback]");
  const openButton = root.querySelector("[data-galaxy-open]");
	const contextPanel = root.querySelector("[data-galaxy-context]");
  const reduced = matchMedia("(prefers-reduced-motion: reduce)").matches || document.documentElement.dataset.motion === "reduced";
  let planets;
  try {
    const controller = new AbortController();
    const timeout = window.setTimeout(() => controller.abort(), 10000);
    const response = await fetch("/galaxy/data", { headers: { Accept: "application/json" }, signal: controller.signal });
    window.clearTimeout(timeout);
    if (!response.ok) throw new Error("Galaxy data could not be loaded");
    planets = await response.json();
  } catch (error) {
    showGalaxyFallback(root, error);
    return;
  }

  const categoryFilter = root.querySelector("[data-galaxy-filter]");
  [...new Set(planets.map(planet => planet.category || "Other"))].sort().forEach(category => {
    const option = document.createElement("option"); option.value = category; option.textContent = category; categoryFilter.appendChild(option);
  });
  categoryFilter.addEventListener("change", () => {
    [...list.children].forEach(item => { item.hidden = !!categoryFilter.value && item.dataset.category !== categoryFilter.value; });
  });
  planets.forEach((planet) => {
    const item = document.createElement("li");
    const link = document.createElement("a");
    link.href = planet.url;
    link.innerHTML = `<span></span><b></b><small></small>`;
    link.querySelector("span").className = `galaxy-state state-${planet.state}`;
    link.querySelector("b").textContent = planet.name;
    link.querySelector("small").textContent = `${planet.progress}% · ${planet.state}`;
    item.dataset.category = planet.category || "Other";
    link.querySelector("small").textContent += " · " + item.dataset.category;
    item.appendChild(link);
    const focus = document.createElement("button"); focus.type = "button"; focus.textContent = "Preview";
    focus.setAttribute("aria-label", "Preview " + planet.name); focus.dataset.galaxyFocus = planet.id; focus.disabled = true;
    item.appendChild(focus);
    if (planet.moons?.length) {
      const children = document.createElement("ul");
      planet.moons.forEach(moon => { const child = document.createElement("li"); const a = document.createElement("a"); a.href = moon.url; a.textContent = moon.name + " · " + moon.progress + "%"; child.appendChild(a); children.appendChild(child); });
      item.appendChild(children);
    }
    list.appendChild(item);
  });
  if (!planets.length) {
	status.textContent = "No active planets. Completed worlds have left the Galaxy.";
	list.innerHTML = '<li><a href="/projects/new">Create a new project</a></li>';
    return;
  }

  let renderer;
  try {
    renderer = new THREE.WebGLRenderer({ canvas, antialias: true, alpha: true, powerPreference: "high-performance" });
  } catch (error) {
    showGalaxyFallback(root, error);
    return;
  }
  try {
  const scene = new THREE.Scene();
  scene.background = new THREE.Color(0x03050b);
  scene.fog = new THREE.FogExp2(0x050812, 0.0032);
  const camera = new THREE.PerspectiveCamera(55, 1, 0.1, 600);
  camera.position.set(0, 8, 125);
	 scene.add(camera);
	 const oliverTexture = new THREE.TextureLoader().load("/static/images/oliver-galaxy-pixel.png");
	 oliverTexture.colorSpace = THREE.SRGBColorSpace;
	 oliverTexture.magFilter = THREE.NearestFilter;
	 oliverTexture.minFilter = THREE.LinearMipmapLinearFilter;
	 const ship = new THREE.Sprite(new THREE.SpriteMaterial({ map: oliverTexture, transparent: true, alphaTest: 0.08, depthTest: false, depthWrite: false }));
	 ship.scale.set(0.62, 0.465, 1);
	 ship.position.set(0, -3.5, -8);
	 ship.renderOrder = 20;
	 camera.add(ship);
	 const shipTarget = new THREE.Vector3(0, -3.5, -8);
  scene.add(new THREE.AmbientLight(0x65749a, 0.72));
  const sun = new THREE.PointLight(0xb9e8ff, 2600, 350);
  sun.position.set(0, 4, 0);
  scene.add(sun);

	const core = new THREE.Mesh(new THREE.SphereGeometry(3.8, 32, 24), new THREE.MeshBasicMaterial({ color: 0xdff8ff }));
	scene.add(core);
	const hubRings = [6.2, 8.5, 11].map((radius, index) => {
	  const ring = new THREE.Mesh(
	    new THREE.TorusGeometry(radius, index === 1 ? 0.1 : 0.055, 8, 96),
	    new THREE.MeshBasicMaterial({ color: index === 1 ? 0x8bddff : 0x416d91, transparent: true, opacity: index === 1 ? 0.48 : 0.24, blending: THREE.AdditiveBlending, depthWrite: false })
	  );
	  ring.rotation.set(Math.PI * (0.42 + index * 0.09), index * 0.55, index * 0.28);
	  scene.add(ring);
	  return ring;
	});
	const coreGlow = new THREE.Sprite(new THREE.SpriteMaterial({ map: glowTexture(THREE, [125, 218, 255]), color: 0x8bddff, transparent: true, blending: THREE.AdditiveBlending, depthWrite: false }));
	coreGlow.scale.set(30, 30, 1);
	scene.add(coreGlow);
	[[0x4361ee, -38, 5, -26, 62], [0x8f5bd7, 42, -7, 18, 52]].forEach((cloud) => {
	  const sprite = new THREE.Sprite(new THREE.SpriteMaterial({ map: glowTexture(THREE, [(cloud[0] >> 16) & 255, (cloud[0] >> 8) & 255, cloud[0] & 255]), transparent: true, opacity: 0.24, blending: THREE.AdditiveBlending, depthWrite: false }));
	  sprite.position.set(cloud[1], cloud[2], cloud[3]); sprite.scale.set(cloud[4] * 1.55, cloud[4] * 0.82, 1); scene.add(sprite);
	});

  const starGeometry = new THREE.BufferGeometry();
	const starCount = 850;
  const starPositions = new Float32Array(starCount * 3);
	for (let i = 0; i < starCount; i++) {
	  const arm = i % 4;
	  const radius = 4 + Math.pow(Math.random(), 0.68) * 180;
	  const angle = radius * 0.055 + arm * Math.PI / 2 + (Math.random() - 0.5) * 0.58;
	  starPositions[i * 3] = Math.cos(angle) * radius + (Math.random() - 0.5) * 7;
	  starPositions[i * 3 + 1] = (Math.random() - 0.5) * Math.max(2, 15 - radius * 0.055);
	  starPositions[i * 3 + 2] = Math.sin(angle) * radius + (Math.random() - 0.5) * 7;
	}
  starGeometry.setAttribute("position", new THREE.BufferAttribute(starPositions, 3));
  scene.add(new THREE.Points(starGeometry, new THREE.PointsMaterial({ color: 0xc9ddff, size: 0.38, transparent: true, opacity: 0.48, blending: THREE.AdditiveBlending, depthWrite: false })));

  const colors = { active: 0x4dbfd0, current: 0x4dbfd0, done: 0x57c98b, locked: 0x596273, upcoming: 0xd9a054 };
	const projectColors = { cyan: 0x4dbfd0, violet: 0x8b6ee8, emerald: 0x43c98b, amber: 0xe0aa45, rose: 0xe46f91, blue: 0x4f83e8, orange: 0xe88145, magenta: 0xc85bd6 };
  const objects = [];
	const labels = [];
  planets.forEach((planet) => {
	const planetColor = projectColors[(planet.color || "cyan").toLowerCase()] || colors[planet.state] || projectColors.cyan;
	const material = new THREE.MeshStandardMaterial({ color: planetColor, roughness: 0.58, metalness: 0.18, emissive: planet.state === "done" ? planetColor : 0x07101a, emissiveIntensity: planet.state === "done" ? 0.42 : 0.18, transparent: true, opacity: planet.state === "locked" ? 0.48 : 1 });
    const mesh = new THREE.Mesh(new THREE.SphereGeometry(Math.max(1.5, planet.size * 1.35), 32, 20), material);
    mesh.position.set(planet.x, planet.y * 0.16, planet.z);
	 if (planet.state === "locked") mesh.position.multiplyScalar(1.35);
    mesh.userData = planet;
    scene.add(mesh);
    objects.push(mesh);
	const atmosphere = new THREE.Mesh(new THREE.SphereGeometry(Math.max(1.8, planet.size * 1.52), 24, 16), new THREE.MeshBasicMaterial({ color: planetColor, transparent: true, opacity: planet.state === "locked" ? 0.05 : 0.13, blending: THREE.AdditiveBlending, side: THREE.BackSide }));
	atmosphere.position.copy(mesh.position); scene.add(atmosphere);
	const orbitRadius = Math.hypot(mesh.position.x, mesh.position.z);
	const orbit = new THREE.Mesh(new THREE.RingGeometry(Math.max(2, orbitRadius - 0.08), orbitRadius + 0.08, 128), new THREE.MeshBasicMaterial({ color: planetColor, transparent: true, opacity: 0.09, side: THREE.DoubleSide, depthWrite: false }));
	orbit.rotation.x = Math.PI / 2; scene.add(orbit);
	if (planet.state === "upcoming" || planet.state === "locked") {
	  const ring = new THREE.Mesh(new THREE.RingGeometry(planet.size * 1.7, planet.size * 2.05, 64), new THREE.MeshBasicMaterial({ color: planetColor, transparent: true, opacity: 0.48, side: THREE.DoubleSide }));
	  ring.rotation.x = Math.PI * 0.68; ring.position.copy(mesh.position); scene.add(ring);
	}
	const label = document.createElement("span"); label.className = "galaxy-planet-label"; label.textContent = planet.name; stage.appendChild(label); labels.push({ element: label, object: mesh });
    planet.moons.forEach((moon, index) => {
      const angle = (index / Math.max(1, planet.moons.length)) * Math.PI * 2;
      const moonMesh = new THREE.Mesh(new THREE.SphereGeometry(Math.max(0.45, moon.size), 18, 12), new THREE.MeshStandardMaterial({ color: colors[moon.state] || 0x8893a5, roughness: 0.9 }));
      moonMesh.position.copy(mesh.position).add(new THREE.Vector3(Math.cos(angle) * (planet.size * 2.2 + 3), Math.sin(angle) * 2, Math.sin(angle) * (planet.size * 2.2 + 3)));
      scene.add(moonMesh);
    });
  });

  const pointer = new THREE.Vector2();
	const raycaster = new THREE.Raycaster();
	const cameraTarget = camera.position.clone();
	const lookTarget = new THREE.Vector3();
	const focusDirection = new THREE.Vector3();
	const selectionHalo = new THREE.Mesh(
	  new THREE.RingGeometry(1.12, 1.42, 64),
	  new THREE.MeshBasicMaterial({ color: 0x6eeaff, transparent: true, opacity: 0.72, side: THREE.DoubleSide, blending: THREE.AdditiveBlending, depthWrite: false })
	);
	selectionHalo.visible = false;
	selectionHalo.renderOrder = 8;
	scene.add(selectionHalo);
	let orbitRadius = 125;
	let orbitYaw = 0;
	let orbitPitch = 0.08;
	let roamInputX = 0;
	let roamInputY = 0;
	let recentering = false;
	const deadZone = (value) => Math.abs(value) < 0.12 ? 0 : Math.sign(value) * (Math.abs(value) - 0.12) / 0.88;
	function updateOrbitTarget() {
	  const horizontalRadius = Math.cos(orbitPitch) * orbitRadius;
	  cameraTarget.set(Math.sin(orbitYaw) * horizontalRadius, Math.sin(orbitPitch) * orbitRadius, Math.cos(orbitYaw) * horizontalRadius);
	}
	updateOrbitTarget();
  let hovered = null;
  function preview(object) {
	if (hovered === object) return;
	if (hovered) hovered.material.emissiveIntensity = hovered.userData._baseEmissiveIntensity;
    hovered = object || null;
    root.classList.toggle("galaxy-has-selection", !!hovered);
	if (!object) {
	  selectionHalo.visible = false;
	  status.textContent = "Drag to explore · tap a world to preview";
	  contextPanel.hidden = true;
	  openButton.hidden = true;
	  return;
	}
	object.userData._baseEmissiveIntensity ??= object.material.emissiveIntensity;
	object.material.emissiveIntensity = Math.max(0.78, object.userData._baseEmissiveIntensity);
	focusDirection.copy(camera.position).sub(object.position).normalize();
	recentering = false;
	selectionHalo.visible = true;
	selectionHalo.position.copy(object.position);
	roamInputX = 0;
	roamInputY = 0;
    status.textContent = `${object.userData.name} · ${object.userData.progress}% · ${object.userData.state} · use Open project to enter`;
    openButton.href = object.userData.url;
    openButton.hidden = false;
	contextPanel.hidden = false;
	contextPanel.querySelector("[data-galaxy-category]").textContent = object.userData.category || "Project";
	contextPanel.querySelector("[data-galaxy-name]").textContent = object.userData.name;
	contextPanel.querySelector("[data-galaxy-summary]").textContent = object.userData.summary || "No project brief yet.";
	contextPanel.querySelector("[data-galaxy-progress]").style.width = object.userData.progress + "%";
	contextPanel.querySelector("[data-galaxy-meta]").textContent = object.userData.missions + " missions · " + object.userData.progress + "% complete" + (object.userData.target ? " · target " + object.userData.target : "");
	contextPanel.querySelector("[data-galaxy-next]").textContent = object.userData.next ? "Next mission: " + object.userData.next : "No pending mission queued.";
  }
  function planetAtPointer() {
	raycaster.setFromCamera(pointer, camera);
	const directHit = raycaster.intersectObjects(objects, false)[0]?.object || null;
	if (directHit) return directHit;

	const width = stage.clientWidth;
	const height = stage.clientHeight;
	const pointerX = (pointer.x * 0.5 + 0.5) * width;
	const pointerY = (-pointer.y * 0.5 + 0.5) * height;
	const cameraUp = new THREE.Vector3().setFromMatrixColumn(camera.matrixWorld, 1);
	let match = null;
	let bestEdge = Infinity;
	objects.forEach((object) => {
	  const center = object.position.clone().project(camera);
	  if (center.z <= -1 || center.z >= 1) return;
	  const radius = object.geometry.parameters.radius;
	  const edge = object.position.clone().addScaledVector(cameraUp, radius).project(camera);
	  const centerX = (center.x * 0.5 + 0.5) * width;
	  const centerY = (-center.y * 0.5 + 0.5) * height;
	  const edgeX = (edge.x * 0.5 + 0.5) * width;
	  const edgeY = (-edge.y * 0.5 + 0.5) * height;
	  const screenRadius = Math.max(5, Math.hypot(edgeX - centerX, edgeY - centerY));
	  const contactPadding = object === hovered ? 44 : 28;
	  const edgeDistance = Math.hypot(pointerX - centerX, pointerY - centerY) - screenRadius;
	  if (edgeDistance >= 0 && edgeDistance <= contactPadding && edgeDistance < bestEdge) {
		match = object;
		bestEdge = edgeDistance;
	  }
	});
	return match;
  }
	function updatePointer(event) {
	  const rect = stage.getBoundingClientRect();
	  pointer.set(
		((event.clientX - rect.left) / rect.width) * 2 - 1,
		1 - ((event.clientY - rect.top) / rect.height) * 2
	  );
	  const shipHalfHeight = Math.tan(THREE.MathUtils.degToRad(camera.fov * 0.5)) * 8;
	  shipTarget.set(pointer.x * shipHalfHeight * camera.aspect, pointer.y * shipHalfHeight, -8);
	}
  function recenterGalaxy() {
	preview(null);
	orbitRadius = 125;
	orbitYaw = 0;
	orbitPitch = 0.08;
	roamInputX = 0;
	roamInputY = 0;
	updateOrbitTarget();
	recentering = true;
	status.textContent = "Returning to the Galaxy map";
  }
  function directionalPlanet(directionX, directionY) {
	if (!hovered) return null;
	const origin = hovered.position.clone().project(camera);
	let match = null;
	let bestScore = Infinity;
	objects.forEach((object) => {
	  if (object === hovered) return;
	  const point = object.position.clone().project(camera);
	  if (point.z <= -1 || point.z >= 1) return;
	  const deltaX = point.x - origin.x;
	  const deltaY = point.y - origin.y;
	  const forward = deltaX * directionX + deltaY * directionY;
	  if (forward <= 0.025) return;
	  const sideways = Math.abs(deltaX * directionY - deltaY * directionX);
	  const score = forward + sideways * 2.4;
	  if (score < bestScore) {
		match = object;
		bestScore = score;
	  }
	});
	return match;
  }
  bindGalaxyGestures(canvas, {
    tap(event) {
      updatePointer(event); const object = planetAtPointer();
      if (object) {
        if (hovered === object && event.pointerType === "mouse") location.href = object.userData.url;
        else preview(object);
      }
    },
    drag(dx, dy) {
      preview(null); recentering = false; roamInputX = 0; roamInputY = 0;
      orbitYaw -= dx * 0.006;
      orbitPitch = Math.max(-1.05, Math.min(1.05, orbitPitch + dy * 0.006)); updateOrbitTarget();
    },
    zoom(factor) { zoomGalaxy(factor); }
  });
  function zoomGalaxy(factor) {
    preview(null); orbitRadius = Math.max(58, Math.min(175, orbitRadius * factor)); updateOrbitTarget();
  }
  root.querySelectorAll("[data-galaxy-zoom]").forEach(button => {
    button.disabled = false; button.addEventListener("click", () => zoomGalaxy(button.dataset.galaxyZoom === "in" ? 0.85 : 1.15));
  });
  const resetButton = root.querySelector("[data-galaxy-reset]"); resetButton.disabled = false;
  resetButton.addEventListener("click", recenterGalaxy);
  root.querySelector("[data-galaxy-close]").addEventListener("click", () => { recenterGalaxy(); stage.focus({preventScroll: true}); });
  root.querySelectorAll("[data-galaxy-focus]").forEach(button => {
    button.disabled = false; button.addEventListener("click", () => {
      preview(objects.find(object => String(object.userData.id) === button.dataset.galaxyFocus));
      stage.scrollIntoView({behavior: "instant", block: "start"});
      openButton.focus({preventScroll: true});
    });
  });
	canvas.addEventListener("dblclick", (event) => {
	  event.preventDefault();
	  updatePointer(event);
	  const object = planetAtPointer();
	  if (object) location.href = object.userData.url;
	  else if (hovered) recenterGalaxy();
	});
	canvas.addEventListener("contextmenu", (event) => {
	  updatePointer(event);
	  const object = planetAtPointer();
	  if (!object) return;
	  event.preventDefault();
	  location.href = object.userData.url;
	});
  if (matchMedia("(pointer: fine)").matches) {
    canvas.addEventListener("pointermove", (event) => {
      if (event.pointerType !== "mouse" || event.buttons) return;
	  updatePointer(event);
	  const normalizedX = pointer.x;
	  const normalizedY = pointer.y;
	  recentering = false;
	  roamInputX = deadZone(normalizedX);
	  roamInputY = deadZone(normalizedY);
    });
  }
	window.addEventListener("blur", () => {
	  roamInputX = 0;
	  roamInputY = 0;
	});
	stage.addEventListener("keydown", (event) => {
      if (event.key === "Escape") { recenterGalaxy(); stage.focus({preventScroll: true}); return; }
      if (event.target !== stage || !hovered) return;
	  const directions = { ArrowLeft: [-1, 0], ArrowRight: [1, 0], ArrowUp: [0, 1], ArrowDown: [0, -1] };
	  const direction = directions[event.key];
	  if (!direction) return;
	  event.preventDefault();
	  const next = directionalPlanet(direction[0], direction[1]);
	  if (next) preview(next);
	});
	stage.addEventListener("pointerleave", () => {
	  roamInputX = 0; roamInputY = 0;
	});
	canvas.addEventListener("wheel", (event) => {
	  event.preventDefault();
	  zoomGalaxy(Math.exp(event.deltaY * 0.001));
	}, { passive: false });

  function resize() {
    const width = stage.clientWidth, height = stage.clientHeight;
    renderer.setPixelRatio(Math.min(devicePixelRatio, 2)); renderer.setSize(width, height, false);
    camera.aspect = width / height; camera.updateProjectionMatrix();
  }
  new ResizeObserver(resize).observe(stage); resize();
  status.textContent = "Drag to explore · tap a world to preview";
  function frame() {
	if (hovered) {
	  const focusDistance = Math.max(15, hovered.geometry.parameters.radius * 5.2);
	  cameraTarget.copy(hovered.position).addScaledVector(focusDirection, focusDistance);
	  lookTarget.lerp(hovered.position, reduced ? 1 : 0.045);
	} else if (recentering) {
	  updateOrbitTarget();
	  lookTarget.lerp(scene.position, reduced ? 1 : 0.075);
	  if (camera.position.distanceTo(cameraTarget) < 0.35) recentering = false;
	} else {
	  orbitYaw += roamInputX * 0.0032;
	  orbitPitch = Math.max(-1.05, Math.min(1.05, orbitPitch + roamInputY * 0.0018));
	  updateOrbitTarget();
	  lookTarget.lerp(scene.position, reduced ? 1 : 0.08);
	}
	camera.position.lerp(cameraTarget, reduced ? 1 : hovered ? 0.022 : 0.065);
	if (!reduced) {
	  const flap = Math.sin(performance.now() * 0.012);
	  ship.scale.y = 0.465 + flap * 0.022;
	  ship.material.rotation = flap * 0.025;
	  hubRings.forEach((ring, index) => { ring.rotation.z += 0.00045 * (index % 2 ? -1 : 1); });
	}
	 camera.lookAt(lookTarget);
	 camera.updateMatrixWorld();
	 scene.updateMatrixWorld();
	ship.position.copy(shipTarget);
	objects.forEach((object) => {
	  const targetScale = object === hovered ? 1.14 : 1;
	  const nextScale = reduced ? targetScale : THREE.MathUtils.lerp(object.scale.x, targetScale, 0.13);
	  object.scale.setScalar(nextScale);
	});
	if (hovered) {
	  const pulse = reduced ? 1 : 1 + Math.sin(performance.now() * 0.007) * 0.08;
	  selectionHalo.position.copy(hovered.position);
	  selectionHalo.lookAt(camera.position);
	  selectionHalo.scale.setScalar(hovered.geometry.parameters.radius * pulse);
	  selectionHalo.material.opacity = reduced ? 0.72 : 0.62 + Math.sin(performance.now() * 0.007) * 0.18;
	}
	labels.forEach((label) => {
	  const point = label.object.position.clone().project(camera);
	  const visible = point.z < 1 && point.z > -1;
	  label.element.hidden = !visible;
	  if (visible) label.element.style.transform = "translate(-50%, -50%) translate(" + ((point.x * 0.5 + 0.5) * stage.clientWidth).toFixed(1) + "px," + ((-point.y * 0.5 + 0.5) * stage.clientHeight - 18).toFixed(1) + "px)";
	  label.element.classList.toggle("selected", hovered === label.object);
	});
    renderer.render(scene, camera);
    requestAnimationFrame(frame);
  }
  frame();
  } catch (error) {
    showGalaxyFallback(root, error);
  }
}

function glowTexture(THREE, color) {
  const canvas = document.createElement("canvas"); canvas.width = canvas.height = 256;
  const context = canvas.getContext("2d");
  const gradient = context.createRadialGradient(128, 128, 3, 128, 128, 128);
  gradient.addColorStop(0, `rgba(${color[0]},${color[1]},${color[2]},1)`);
  gradient.addColorStop(0.18, `rgba(${color[0]},${color[1]},${color[2]},.45)`);
  gradient.addColorStop(0.55, `rgba(${color[0]},${color[1]},${color[2]},.12)`);
  gradient.addColorStop(1, `rgba(${color[0]},${color[1]},${color[2]},0)`);
  context.fillStyle = gradient; context.fillRect(0, 0, 256, 256);
  return new THREE.CanvasTexture(canvas);
}
