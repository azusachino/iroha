import {
  BufferAttribute,
  BufferGeometry,
  Group,
  LineBasicMaterial,
  LineSegments,
  PerspectiveCamera,
  Points,
  PointsMaterial,
  Scene,
  WebGLRenderer,
} from "three";
import type { AmbientFactory } from "./renderer";

// A flat survey chart, not a globe: scattered "dust" points read as
// unmarked terrain, larger points as legend/waypoint markers (Atlas's own
// --accent-2 amber, per themes.css's "legend-style semantic colors"
// comment), with a handful of faint triangulation lines between nearby
// markers. Camera is far back with a narrow FOV so the scene reads planar,
// like looking straight down at a chart, rather than a 3D object.
const DUST_COUNT = 350;
const MARKER_COUNT = 40;
const MAX_LINK_DISTANCE = 3;
const MAX_LINKS_PER_MARKER = 2;

function scatter(count: number, spreadX: number, spreadY: number, spreadZ: number) {
  const positions = new Float32Array(count * 3);
  for (let i = 0; i < count; i += 1) {
    positions[i * 3] = (Math.random() * 2 - 1) * spreadX;
    positions[i * 3 + 1] = (Math.random() * 2 - 1) * spreadY;
    positions[i * 3 + 2] = (Math.random() * 2 - 1) * spreadZ;
  }
  return positions;
}

function buildLinkSegments(markerPositions: Float32Array): Float32Array {
  const segments: number[] = [];
  const linkCounts = new Array(MARKER_COUNT).fill(0);
  for (let i = 0; i < MARKER_COUNT; i += 1) {
    if (linkCounts[i] >= MAX_LINKS_PER_MARKER) continue;
    const ax = markerPositions[i * 3];
    const ay = markerPositions[i * 3 + 1];
    const az = markerPositions[i * 3 + 2];
    let bestIndex = -1;
    let bestDistance = MAX_LINK_DISTANCE;
    for (let j = 0; j < MARKER_COUNT; j += 1) {
      if (j === i || linkCounts[j] >= MAX_LINKS_PER_MARKER) continue;
      const bx = markerPositions[j * 3];
      const by = markerPositions[j * 3 + 1];
      const bz = markerPositions[j * 3 + 2];
      const distance = Math.hypot(ax - bx, ay - by, az - bz);
      if (distance < bestDistance) {
        bestDistance = distance;
        bestIndex = j;
      }
    }
    if (bestIndex === -1) continue;
    segments.push(ax, ay, az, markerPositions[bestIndex * 3], markerPositions[bestIndex * 3 + 1], markerPositions[bestIndex * 3 + 2]);
    linkCounts[i] += 1;
    linkCounts[bestIndex] += 1;
  }
  return new Float32Array(segments);
}

export const createAtlasAmbientScene: AmbientFactory = (canvas, getColors) => {
  const scene = new Scene();
  const camera = new PerspectiveCamera(45, 1, 0.1, 50);
  camera.position.set(0, 0, 8);
  camera.lookAt(0, 0, 0);

  const renderer = new WebGLRenderer({ canvas, alpha: true, antialias: true });
  renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));

  const group = new Group();
  scene.add(group);

  const dustPositions = scatter(DUST_COUNT, 7, 4, 0.5);
  const dustGeometry = new BufferGeometry();
  dustGeometry.setAttribute("position", new BufferAttribute(dustPositions, 3));
  const dustMaterial = new PointsMaterial({
    size: 0.045,
    sizeAttenuation: true,
    transparent: true,
    opacity: 0.55,
  });
  group.add(new Points(dustGeometry, dustMaterial));

  const markerPositions = scatter(MARKER_COUNT, 6.5, 3.5, 0.3);
  const markerGeometry = new BufferGeometry();
  markerGeometry.setAttribute("position", new BufferAttribute(markerPositions, 3));
  const markerMaterial = new PointsMaterial({
    size: 0.09,
    sizeAttenuation: true,
    transparent: true,
    opacity: 0.9,
  });
  group.add(new Points(markerGeometry, markerMaterial));

  const linkGeometry = new BufferGeometry();
  linkGeometry.setAttribute(
    "position",
    new BufferAttribute(buildLinkSegments(markerPositions), 3),
  );
  const linkMaterial = new LineBasicMaterial({ transparent: true, opacity: 0.2 });
  group.add(new LineSegments(linkGeometry, linkMaterial));

  return {
    resize(width, height) {
      camera.aspect = width / Math.max(1, height);
      camera.updateProjectionMatrix();
      renderer.setSize(width, height, false);
    },
    render() {
      const colors = getColors();
      dustMaterial.color.set(colors.accent);
      markerMaterial.color.set(colors.accent2);
      linkMaterial.color.set(colors.accent2);
      group.rotation.z += 0.00006;
      renderer.render(scene, camera);
    },
    dispose() {
      dustGeometry.dispose();
      dustMaterial.dispose();
      markerGeometry.dispose();
      markerMaterial.dispose();
      linkGeometry.dispose();
      linkMaterial.dispose();
      renderer.dispose();
    },
  };
};
