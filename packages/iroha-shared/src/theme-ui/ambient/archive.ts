import {
  BufferAttribute,
  BufferGeometry,
  Color,
  Group,
  LineBasicMaterial,
  LineSegments,
  PerspectiveCamera,
  Scene,
  WebGLRenderer,
} from "three";
import type { AmbientFactory } from "./renderer";

// A card-catalog drawer, not a filmstrip: a row of vertical card-edge ticks
// with a soft brightness sweep travelling left to right and wrapping, as if
// a hand were flipping past the drawer. Cards fade toward --surface rather
// than to black -- a fade into the catalog's own background reads as
// "unlit", not "gone". Every fifth card baselines to --accent-2 (a tabbed
// divider standing out from the row), the rest --accent.
const CARD_COUNT = 40;
const CARD_SPACING = 0.24;
const CARD_HEIGHT = 1.6;
const SWEEP_RADIUS = 2.2;
const SWEEP_SPEED = 0.01;
const ACCENT2_EVERY = 5;

export const createArchiveAmbientScene: AmbientFactory = (canvas, getColors) => {
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

  const half = (CARD_COUNT - 1) / 2;
  const halfWidth = half * CARD_SPACING;
  const xPositions = new Float32Array(CARD_COUNT);
  for (let i = 0; i < CARD_COUNT; i += 1) {
    xPositions[i] = (i - half) * CARD_SPACING;
  }

  const positions = new Float32Array(CARD_COUNT * 2 * 3);
  for (let i = 0; i < CARD_COUNT; i += 1) {
    const x = xPositions[i];
    const base = i * 6;
    positions[base] = x;
    positions[base + 1] = -CARD_HEIGHT / 2;
    positions[base + 2] = 0;
    positions[base + 3] = x;
    positions[base + 4] = CARD_HEIGHT / 2;
    positions[base + 5] = 0;
  }

  const colors = new Float32Array(CARD_COUNT * 2 * 3);
  const geometry = new BufferGeometry();
  geometry.setAttribute("position", new BufferAttribute(positions, 3));
  const colorAttribute = new BufferAttribute(colors, 3);
  geometry.setAttribute("color", colorAttribute);
  const material = new LineBasicMaterial({
    vertexColors: true,
    transparent: true,
    opacity: 0.9,
  });
  group.add(new LineSegments(geometry, material));

  const dim = new Color();
  const baseline = new Color();
  const mixed = new Color();
  let sweepX = -halfWidth;

  return {
    resize(width, height) {
      camera.aspect = width / Math.max(1, height);
      camera.updateProjectionMatrix();
      renderer.setSize(width, height, false);
    },
    render() {
      const themeColors = getColors();
      dim.set(themeColors.surface);
      for (let i = 0; i < CARD_COUNT; i += 1) {
        baseline.set(i % ACCENT2_EVERY === 0 ? themeColors.accent2 : themeColors.accent);
        const distance = Math.abs(xPositions[i] - sweepX);
        const brightness = Math.max(0, 1 - distance / SWEEP_RADIUS);
        mixed.copy(dim).lerp(baseline, brightness);
        const base = i * 6;
        colors[base] = mixed.r;
        colors[base + 1] = mixed.g;
        colors[base + 2] = mixed.b;
        colors[base + 3] = mixed.r;
        colors[base + 4] = mixed.g;
        colors[base + 5] = mixed.b;
      }
      colorAttribute.needsUpdate = true;

      sweepX += SWEEP_SPEED;
      if (sweepX > halfWidth + SWEEP_RADIUS) sweepX = -halfWidth - SWEEP_RADIUS;

      renderer.render(scene, camera);
    },
    dispose() {
      geometry.dispose();
      material.dispose();
      renderer.dispose();
    },
  };
};
