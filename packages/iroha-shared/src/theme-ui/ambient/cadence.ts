import {
  BufferAttribute,
  BufferGeometry,
  Group,
  LineBasicMaterial,
  LineSegments,
  PerspectiveCamera,
  Scene,
  WebGLRenderer,
} from "three";
import type { AmbientFactory } from "./renderer";

// A level-meter rack, not a waveform: independent vertical bars, each
// pulsing on its own fixed phase/frequency computed from its index --
// deterministic, never driven by any audio input, matching Cadence's own
// lens copy ("Do not pretend expense data is audio"). Every fourth bar
// reads --accent-2 (a channel standing out from the rack), the rest
// --accent.
const BAR_COUNT = 24;
const BAR_SPACING = 0.36;
const MIN_HEIGHT = 0.4;
const MAX_HEIGHT = 3.2;
const ACCENT2_EVERY = 4;

function barPhase(index: number): number {
  return (((index * 47) % BAR_COUNT) / BAR_COUNT) * Math.PI * 2;
}

function barFrequency(index: number): number {
  return 0.6 + ((index * 13) % 7) * 0.08;
}

type BarSet = {
  indices: number[];
  positions: Float32Array;
  geometry: BufferGeometry;
  material: LineBasicMaterial;
};

function buildBarSet(indices: number[]): BarSet {
  const positions = new Float32Array(indices.length * 2 * 3);
  const geometry = new BufferGeometry();
  geometry.setAttribute("position", new BufferAttribute(positions, 3));
  const material = new LineBasicMaterial({ transparent: true, opacity: 0.8 });
  return { indices, positions, geometry, material };
}

function updateBarSet(set: BarSet, time: number): void {
  const half = (BAR_COUNT - 1) / 2;
  set.indices.forEach((barIndex, slot) => {
    const x = (barIndex - half) * BAR_SPACING;
    const wave = 0.5 + 0.5 * Math.sin(time * barFrequency(barIndex) + barPhase(barIndex));
    const height = MIN_HEIGHT + (MAX_HEIGHT - MIN_HEIGHT) * wave;
    const base = slot * 6;
    set.positions[base] = x;
    set.positions[base + 1] = -height / 2;
    set.positions[base + 2] = 0;
    set.positions[base + 3] = x;
    set.positions[base + 4] = height / 2;
    set.positions[base + 5] = 0;
  });
  set.geometry.attributes.position.needsUpdate = true;
}

export const createCadenceAmbientScene: AmbientFactory = (canvas, getColors) => {
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

  const accentIndices: number[] = [];
  const accent2Indices: number[] = [];
  for (let i = 0; i < BAR_COUNT; i += 1) {
    (i % ACCENT2_EVERY === 0 ? accent2Indices : accentIndices).push(i);
  }

  const accentSet = buildBarSet(accentIndices);
  const accent2Set = buildBarSet(accent2Indices);
  group.add(new LineSegments(accentSet.geometry, accentSet.material));
  group.add(new LineSegments(accent2Set.geometry, accent2Set.material));

  let time = 0;

  return {
    resize(width, height) {
      camera.aspect = width / Math.max(1, height);
      camera.updateProjectionMatrix();
      renderer.setSize(width, height, false);
    },
    render() {
      const colors = getColors();
      accentSet.material.color.set(colors.accent);
      accent2Set.material.color.set(colors.accent2);
      time += 0.02;
      updateBarSet(accentSet, time);
      updateBarSet(accent2Set, time);
      renderer.render(scene, camera);
    },
    dispose() {
      accentSet.geometry.dispose();
      accentSet.material.dispose();
      accent2Set.geometry.dispose();
      accent2Set.material.dispose();
      renderer.dispose();
    },
  };
};
