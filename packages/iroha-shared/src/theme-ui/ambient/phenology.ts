import {
  BufferAttribute,
  BufferGeometry,
  Group,
  LineBasicMaterial,
  LineLoop,
  LineSegments,
  PerspectiveCamera,
  Points,
  PointsMaterial,
  Scene,
  WebGLRenderer,
} from "three";
import type { AmbientFactory } from "./renderer";

// A phase wheel, not a compass: a ring of evenly spaced dial ticks with four
// larger season markers hubbed to the center by spokes, the whole wheel
// turning slowly -- "a cyclical language ... unfolding patterns" per
// themes.ts's own description, and the one ambient scene where rotation is
// the primary motion rather than a near-imperceptible touch (contrast
// Atlas's planar drift, Field Journal's vertical scroll).
const RING_COUNT = 48;
const RING_RADIUS = 3.2;
const SEASON_EVERY = 12;
const ROTATION_SPEED = 0.0015;

function buildRing(radius: number): Float32Array {
  const positions = new Float32Array(RING_COUNT * 3);
  for (let i = 0; i < RING_COUNT; i += 1) {
    const angle = (i / RING_COUNT) * Math.PI * 2;
    positions[i * 3] = Math.cos(angle) * radius;
    positions[i * 3 + 1] = Math.sin(angle) * radius;
    positions[i * 3 + 2] = 0;
  }
  return positions;
}

function buildSeasonMarkers(radius: number): {
  points: Float32Array;
  spokes: Float32Array;
} {
  const points: number[] = [];
  const spokes: number[] = [];
  for (let i = 0; i < RING_COUNT; i += SEASON_EVERY) {
    const angle = (i / RING_COUNT) * Math.PI * 2;
    const x = Math.cos(angle) * radius;
    const y = Math.sin(angle) * radius;
    points.push(x, y, 0);
    spokes.push(0, 0, 0, x, y, 0);
  }
  return { points: new Float32Array(points), spokes: new Float32Array(spokes) };
}

export const createPhenologyAmbientScene: AmbientFactory = (canvas, getColors) => {
  const scene = new Scene();
  const camera = new PerspectiveCamera(45, 1, 0.1, 50);
  camera.position.set(0, 0, 8);
  camera.lookAt(0, 0, 0);

  const renderer = new WebGLRenderer({
    canvas,
    alpha: true,
    antialias: true,
    preserveDrawingBuffer: true,
  });
  renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));

  const group = new Group();
  scene.add(group);

  const ringGeometry = new BufferGeometry();
  ringGeometry.setAttribute("position", new BufferAttribute(buildRing(RING_RADIUS), 3));
  const ringMaterial = new LineBasicMaterial({ transparent: true, opacity: 0.5 });
  group.add(new LineLoop(ringGeometry, ringMaterial));

  const { points: seasonPositions, spokes: spokePositions } = buildSeasonMarkers(RING_RADIUS);

  const spokeGeometry = new BufferGeometry();
  spokeGeometry.setAttribute("position", new BufferAttribute(spokePositions, 3));
  const spokeMaterial = new LineBasicMaterial({ transparent: true, opacity: 0.3 });
  group.add(new LineSegments(spokeGeometry, spokeMaterial));

  const seasonGeometry = new BufferGeometry();
  seasonGeometry.setAttribute("position", new BufferAttribute(seasonPositions, 3));
  const seasonMaterial = new PointsMaterial({
    size: 0.22,
    sizeAttenuation: true,
    transparent: true,
    opacity: 1,
  });
  group.add(new Points(seasonGeometry, seasonMaterial));

  return {
    resize(width, height) {
      camera.aspect = width / Math.max(1, height);
      camera.updateProjectionMatrix();
      renderer.setSize(width, height, false);
    },
    render() {
      const colors = getColors();
      ringMaterial.color.set(colors.accent);
      spokeMaterial.color.set(colors.accent2);
      seasonMaterial.color.set(colors.accent2);
      group.rotation.z += ROTATION_SPEED;
      renderer.render(scene, camera);
    },
    dispose() {
      ringGeometry.dispose();
      ringMaterial.dispose();
      spokeGeometry.dispose();
      spokeMaterial.dispose();
      seasonGeometry.dispose();
      seasonMaterial.dispose();
      renderer.dispose();
    },
  };
};
