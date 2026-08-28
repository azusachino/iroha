// Shared contract for the per-language ambient WebGL background. Each
// factory owns its own Three.js scene/camera/renderer; the host component
// (AmbientBackground.svelte) owns canvas lifecycle and never imports "three"
// directly -- see AmbientBackground.svelte for why.

export type AmbientColors = {
  accent: string;
  accent2: string;
  surface: string;
};

export type AmbientRenderer = {
  resize(width: number, height: number): void;
  render(): void;
  dispose(): void;
};

export type AmbientFactory = (
  canvas: HTMLCanvasElement,
  getColors: () => AmbientColors,
) => AmbientRenderer;

// Mirrors BarChart.svelte's own getComputedStyle(...).getPropertyValue(...)
// pattern rather than inventing a second way to read theme tokens.
export function readAmbientColors(el: Element): AmbientColors {
  const styles = getComputedStyle(el);
  return {
    accent: styles.getPropertyValue("--accent").trim() || "#5c8dff",
    accent2: styles.getPropertyValue("--accent-2").trim() || "#ff5c8a",
    surface: styles.getPropertyValue("--surface").trim() || "#171a21",
  };
}
