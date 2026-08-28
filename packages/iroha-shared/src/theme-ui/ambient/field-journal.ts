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

// A ruled margin, not a compass: two columns of short horizontal tick marks
// (a bullet-journal's date/note column down each page edge) drifting slowly
// downward and wrapping -- a page scrolling past, distinct from Atlas's
// planar drift or any rotation. Every sixth tick uses --accent-2 (the same
// "flagged note in a different ink" reading its own empty-mark and card
// accents already use), the rest --accent.
const TICK_SPACING = 0.5;
const COLUMN_HALF_HEIGHT = 8;
const COLUMN_X = 4.5;
const TICK_LENGTH = 0.3;
const FLAG_EVERY = 6;

function buildColumnSegments(x: number): { plain: Float32Array; flagged: Float32Array } {
  const plain: number[] = [];
  const flagged: number[] = [];
  let index = 0;
  for (let y = -COLUMN_HALF_HEIGHT; y <= COLUMN_HALF_HEIGHT; y += TICK_SPACING) {
    const target = index % FLAG_EVERY === 0 ? flagged : plain;
    target.push(x - TICK_LENGTH / 2, y, 0, x + TICK_LENGTH / 2, y, 0);
    index += 1;
  }
  return { plain: new Float32Array(plain), flagged: new Float32Array(flagged) };
}

export const createFieldJournalAmbientScene: AmbientFactory = (canvas, getColors) => {
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

  const left = buildColumnSegments(-COLUMN_X);
  const right = buildColumnSegments(COLUMN_X);

  const plainGeometry = new BufferGeometry();
  const plainPositions = new Float32Array(left.plain.length + right.plain.length);
  plainPositions.set(left.plain, 0);
  plainPositions.set(right.plain, left.plain.length);
  plainGeometry.setAttribute("position", new BufferAttribute(plainPositions, 3));
  const plainMaterial = new LineBasicMaterial({ transparent: true, opacity: 0.55 });
  group.add(new LineSegments(plainGeometry, plainMaterial));

  const flaggedGeometry = new BufferGeometry();
  const flaggedPositions = new Float32Array(left.flagged.length + right.flagged.length);
  flaggedPositions.set(left.flagged, 0);
  flaggedPositions.set(right.flagged, left.flagged.length);
  flaggedGeometry.setAttribute("position", new BufferAttribute(flaggedPositions, 3));
  const flaggedMaterial = new LineBasicMaterial({ transparent: true, opacity: 0.85 });
  group.add(new LineSegments(flaggedGeometry, flaggedMaterial));

  return {
    resize(width, height) {
      camera.aspect = width / Math.max(1, height);
      camera.updateProjectionMatrix();
      renderer.setSize(width, height, false);
    },
    render() {
      const colors = getColors();
      plainMaterial.color.set(colors.accent);
      flaggedMaterial.color.set(colors.accent2);
      group.position.y -= 0.006;
      if (group.position.y < -TICK_SPACING) group.position.y += TICK_SPACING;
      renderer.render(scene, camera);
    },
    dispose() {
      plainGeometry.dispose();
      plainMaterial.dispose();
      flaggedGeometry.dispose();
      flaggedMaterial.dispose();
      renderer.dispose();
    },
  };
};
