import * as THREE from "/static/js/vendor/three.module.js";

const root = document.querySelector("[data-galaxy]");
if (root) initGalaxy(root);

async function initGalaxy(root) {
  const stage = root.querySelector("[data-galaxy-stage]");
  const canvas = root.querySelector("[data-galaxy-canvas]");
  const status = root.querySelector("[data-galaxy-status]");
  const list = root.querySelector("[data-galaxy-list]");
  const fallback = root.querySelector("[data-galaxy-fallback]");
  const openButton = root.querySelector("[data-galaxy-open]");
  const reduced = matchMedia("(prefers-reduced-motion: reduce)").matches || document.documentElement.dataset.motion === "reduced";
  let planets;
  try {
    const response = await fetch("/galaxy/data", { headers: { Accept: "application/json" } });
    if (!response.ok) throw new Error("Galaxy data could not be loaded");
    planets = await response.json();
  } catch (error) {
    status.textContent = error.message;
    fallback.hidden = false;
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
  } catch (_) {
    stage.hidden = true;
    fallback.hidden = false;
    return;
  }
  const scene = new THREE.Scene();
  scene.fog = new THREE.FogExp2(0x070b12, 0.006);
  const camera = new THREE.PerspectiveCamera(55, 1, 0.1, 600);
  camera.position.set(0, 8, 125);
	 scene.add(camera);
	 const ship = new THREE.Mesh(new THREE.ConeGeometry(0.38, 1.4, 5), new THREE.MeshBasicMaterial({ color: 0xe5f7ff }));
	 ship.rotation.z = -Math.PI / 2;
	 ship.position.set(0, -3.5, -8);
	 camera.add(ship);
	 const shipTarget = new THREE.Vector3(0, -3.5, -8);
  scene.add(new THREE.AmbientLight(0x7187a8, 1.2));
  const sun = new THREE.PointLight(0xffffff, 1700, 300);
  sun.position.set(5, 20, 80);
  scene.add(sun);

  const starGeometry = new THREE.BufferGeometry();
  const starPositions = new Float32Array(1800 * 3);
  for (let i = 0; i < starPositions.length; i++) starPositions[i] = (Math.random() - 0.5) * 420;
  starGeometry.setAttribute("position", new THREE.BufferAttribute(starPositions, 3));
  scene.add(new THREE.Points(starGeometry, new THREE.PointsMaterial({ color: 0xb9d8ff, size: 0.45, transparent: true, opacity: 0.7 })));

  const colors = { active: 0x4dbfd0, current: 0x4dbfd0, done: 0x57c98b, locked: 0x596273, upcoming: 0xd9a054 };
  const objects = [];
  planets.forEach((planet) => {
    const material = new THREE.MeshStandardMaterial({ color: colors[planet.state] || colors.upcoming, roughness: 0.72, metalness: 0.12, emissive: planet.state === "done" ? 0x163d2a : 0x000000, emissiveIntensity: 0.8, transparent: true, opacity: planet.state === "locked" ? 0.45 : 1 });
    const mesh = new THREE.Mesh(new THREE.SphereGeometry(Math.max(1.5, planet.size * 1.35), 32, 20), material);
    mesh.position.set(planet.x, planet.y, planet.z);
	 if (planet.state === "locked") mesh.position.multiplyScalar(1.35);
    mesh.userData = planet;
    scene.add(mesh);
    objects.push(mesh);
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
	if (!reduced) ship.position.lerp(shipTarget, 0.09);
    camera.lookAt(selected ? selected.position : scene.position);
    renderer.render(scene, camera);
    requestAnimationFrame(frame);
  }
  frame();
}
