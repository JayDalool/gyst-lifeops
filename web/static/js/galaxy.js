const root = document.querySelector("[data-galaxy]");
if (root) {
  import("/static/js/vendor/three.module.js")
    .then((THREE) => initGalaxy(root, THREE))
    .catch((error) => showGalaxyFallback(root, error));
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

  planets.forEach((planet) => {
    const item = document.createElement("li");
    const link = document.createElement("a");
    link.href = planet.url;
    link.innerHTML = `<span></span><b></b><small></small>`;
    link.querySelector("span").className = `galaxy-state state-${planet.state}`;
    link.querySelector("b").textContent = planet.name;
    link.querySelector("small").textContent = `${planet.progress}% · ${planet.state}`;
    item.appendChild(link);
    list.appendChild(item);
  });
  if (!planets.length) {
    status.textContent = "No projects yet.";
    list.innerHTML = '<li><a href="/projects/new">Create your first project</a></li>';
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
	 ship.scale.set(2.9, 2.18, 1);
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
	const categoryColors = { career: 0x4dbfd0, learning: 0x7d8cff, health: 0x57c98b, finance: 0xd9a054, deen: 0x9b7de5, personal: 0xe47b9b, other: 0x89a2c7 };
  const objects = [];
	const labels = [];
  planets.forEach((planet) => {
    const planetColor = categoryColors[(planet.category || "other").toLowerCase()] || colors[planet.state] || categoryColors.other;
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

  const raycaster = new THREE.Raycaster();
  const pointer = new THREE.Vector2();
  let selected = null;
  let target = new THREE.Vector3(0, 4, 110);
  function choose(object) {
    if (!object) return;
    if (selected === object) { location.href = object.userData.url; return; }
    selected = object;
    status.textContent = `${object.userData.name} · ${object.userData.progress}% · ${object.userData.state}`;
    openButton.href = object.userData.url;
    openButton.hidden = false;
	contextPanel.hidden = false;
	contextPanel.querySelector("[data-galaxy-category]").textContent = object.userData.category || "Project";
	contextPanel.querySelector("[data-galaxy-name]").textContent = object.userData.name;
	contextPanel.querySelector("[data-galaxy-summary]").textContent = object.userData.summary || "No project brief yet.";
	contextPanel.querySelector("[data-galaxy-progress]").style.width = object.userData.progress + "%";
	contextPanel.querySelector("[data-galaxy-meta]").textContent = object.userData.missions + " missions · " + object.userData.progress + "% complete" + (object.userData.target ? " · target " + object.userData.target : "");
	contextPanel.querySelector("[data-galaxy-next]").textContent = object.userData.next ? "Next mission: " + object.userData.next : "No pending mission queued.";
    target.copy(object.position).add(new THREE.Vector3(0, 3, 18));
  }
  function pick(event) {
    const rect = canvas.getBoundingClientRect();
    pointer.set(((event.clientX - rect.left) / rect.width) * 2 - 1, -((event.clientY - rect.top) / rect.height) * 2 + 1);
    raycaster.setFromCamera(pointer, camera);
    choose(raycaster.intersectObjects(objects, false)[0]?.object);
  }
  canvas.addEventListener("pointerup", pick);
  if (matchMedia("(pointer: fine)").matches && !reduced) {
    stage.addEventListener("pointermove", (event) => {
      const rect = stage.getBoundingClientRect();
      target.x += (((event.clientX - rect.left) / rect.width) - 0.5) * 0.08;
      target.y += (0.5 - ((event.clientY - rect.top) / rect.height)) * 0.05;
	  shipTarget.x = (((event.clientX - rect.left) / rect.width) - 0.5) * 10;
	  shipTarget.y = (0.5 - ((event.clientY - rect.top) / rect.height)) * 6;
    });
  }
  let dragging = false, lastX = 0, lastY = 0;
  stage.addEventListener("pointerdown", (event) => { dragging = true; lastX = event.clientX; lastY = event.clientY; });
  stage.addEventListener("pointermove", (event) => {
    if (!dragging || matchMedia("(pointer: fine)").matches) return;
    target.x -= (event.clientX - lastX) * 0.08; target.y += (event.clientY - lastY) * 0.08;
    lastX = event.clientX; lastY = event.clientY;
  });
  stage.addEventListener("pointerup", () => { dragging = false; });

  function resize() {
    const width = stage.clientWidth, height = stage.clientHeight;
    renderer.setPixelRatio(Math.min(devicePixelRatio, 2)); renderer.setSize(width, height, false);
    camera.aspect = width / height; camera.updateProjectionMatrix();
  }
  new ResizeObserver(resize).observe(stage); resize();
  status.textContent = "Select a planet";
  function frame() {
    if (!reduced) camera.position.lerp(target, selected ? 0.045 : 0.018);
    else camera.position.copy(target);
	if (!reduced) {
	  ship.position.lerp(shipTarget, 0.09);
	  const flap = Math.sin(performance.now() * 0.012);
	  ship.scale.y = 2.18 + flap * 0.09;
	  ship.material.rotation = flap * 0.025;
	}
    camera.lookAt(selected ? selected.position : scene.position);
	labels.forEach((label) => {
	  const point = label.object.position.clone().project(camera);
	  const visible = point.z < 1 && point.z > -1;
	  label.element.hidden = !visible || selected !== label.object;
	  if (visible) label.element.style.transform = "translate(-50%, -50%) translate(" + ((point.x * 0.5 + 0.5) * stage.clientWidth).toFixed(1) + "px," + ((-point.y * 0.5 + 0.5) * stage.clientHeight - 18).toFixed(1) + "px)";
	  label.element.classList.toggle("selected", selected === label.object);
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
