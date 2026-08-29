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

  const pointer = new THREE.Vector2();
	const raycaster = new THREE.Raycaster();
	const cameraTarget = camera.position.clone();
	const lookTarget = new THREE.Vector3();
	const focusDirection = new THREE.Vector3();
	const travelTarget = new THREE.Vector3();
	const travelLookTarget = new THREE.Vector3();
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
	let pointerActive = false;
	let pointerDirty = false;
	let pendingObject;
	let pendingSince = 0;
	const selectionAnchor = new THREE.Vector2();
	let travelling = false;
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
	if (!object) {
	  selectionHalo.visible = false;
	  status.textContent = "Free roam · click a planet to enter";
	  contextPanel.hidden = true;
	  openButton.hidden = true;
	  return;
	}
	object.userData._baseEmissiveIntensity ??= object.material.emissiveIntensity;
	object.material.emissiveIntensity = Math.max(0.78, object.userData._baseEmissiveIntensity);
	focusDirection.copy(camera.position).sub(object.position).normalize();
	selectionAnchor.copy(pointer);
	travelling = false;
	selectionHalo.visible = true;
	selectionHalo.position.copy(object.position);
	roamInputX = 0;
	roamInputY = 0;
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
  }
  function planetAtPointer() {
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
	  const contactPadding = object === hovered ? 36 : 28;
	  const edgeDistance = Math.hypot(pointerX - centerX, pointerY - centerY) - screenRadius;
	  if (edgeDistance <= contactPadding && edgeDistance < bestEdge) {
		match = object;
		bestEdge = edgeDistance;
	  }
	});
	return match;
  }
  function queuePreview(object) {
	if (object === hovered) {
	  pendingObject = undefined;
	  return;
	}
	if (object !== pendingObject) {
	  pendingObject = object;
	  pendingSince = performance.now();
	}
  }
  canvas.addEventListener("pointerup", () => {
	const object = planetAtPointer();
	if (object) {
	  location.href = object.userData.url;
	  return;
	}
	raycaster.setFromCamera(pointer, camera);
	preview(null);
	pendingObject = undefined;
	travelTarget.copy(camera.position).addScaledVector(raycaster.ray.direction, 34);
	travelLookTarget.copy(camera.position).addScaledVector(raycaster.ray.direction, 100);
	travelling = true;
	status.textContent = "Travelling through the galaxy";
  });
  if (matchMedia("(pointer: fine)").matches) {
    stage.addEventListener("pointermove", (event) => {
      const rect = stage.getBoundingClientRect();
	  const normalizedX = ((event.clientX - rect.left) / rect.width) * 2 - 1;
	  const normalizedY = 1 - ((event.clientY - rect.top) / rect.height) * 2;
	  shipTarget.x = normalizedX * 7;
	  shipTarget.y = normalizedY * 4;
	  roamInputX = deadZone(normalizedX);
	  roamInputY = deadZone(normalizedY);
	  pointer.set(normalizedX, normalizedY);
	  pointerActive = true;
	  const anchorDistance = hovered
		? Math.hypot((pointer.x - selectionAnchor.x) * rect.width * 0.5, (pointer.y - selectionAnchor.y) * rect.height * 0.5)
		: Infinity;
	  pointerDirty = !hovered || anchorDistance > 48;
    });
  }
	stage.addEventListener("pointerleave", () => {
	  roamInputX = 0; roamInputY = 0; pointerActive = false; pendingObject = undefined; preview(null);
	});
	stage.addEventListener("wheel", (event) => {
	  event.preventDefault();
	  orbitRadius = Math.max(58, Math.min(175, orbitRadius + event.deltaY * 0.08));
	  updateOrbitTarget();
	}, { passive: false });

  function resize() {
    const width = stage.clientWidth, height = stage.clientHeight;
    renderer.setPixelRatio(Math.min(devicePixelRatio, 2)); renderer.setSize(width, height, false);
    camera.aspect = width / height; camera.updateProjectionMatrix();
  }
  new ResizeObserver(resize).observe(stage); resize();
  status.textContent = "Select a planet";
  function frame() {
	if (hovered) {
	  const focusDistance = Math.max(15, hovered.geometry.parameters.radius * 5.2);
	  cameraTarget.copy(hovered.position).addScaledVector(focusDirection, focusDistance);
	  lookTarget.lerp(hovered.position, reduced ? 1 : 0.045);
	} else if (travelling) {
	  cameraTarget.copy(travelTarget);
	  lookTarget.lerp(travelLookTarget, reduced ? 1 : 0.075);
	  if (camera.position.distanceTo(travelTarget) < 0.35) {
		travelling = false;
		orbitRadius = camera.position.length();
		orbitYaw = Math.atan2(camera.position.x, camera.position.z);
		orbitPitch = Math.asin(THREE.MathUtils.clamp(camera.position.y / orbitRadius, -1, 1));
	  }
	} else {
	  orbitYaw += roamInputX * 0.0032;
	  orbitPitch = Math.max(-1.05, Math.min(1.05, orbitPitch + roamInputY * 0.0018));
	  updateOrbitTarget();
	  lookTarget.lerp(scene.position, reduced ? 1 : 0.08);
	}
	camera.position.lerp(cameraTarget, reduced ? 1 : hovered ? 0.022 : 0.065);
	ship.position.copy(shipTarget);
	if (!reduced) {
	  const flap = Math.sin(performance.now() * 0.012);
	  ship.scale.y = 0.465 + flap * 0.022;
	  ship.material.rotation = flap * 0.025;
	}
	 camera.lookAt(lookTarget);
	 camera.updateMatrixWorld();
	 scene.updateMatrixWorld();
	 if (pointerActive && pointerDirty) {
	   queuePreview(planetAtPointer());
	   pointerDirty = false;
	 }
	const selectionDelay = pendingObject ? 250 : 80;
	if (pendingObject !== undefined && performance.now() - pendingSince >= selectionDelay) {
	  preview(pendingObject);
	  pendingObject = undefined;
	}
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
