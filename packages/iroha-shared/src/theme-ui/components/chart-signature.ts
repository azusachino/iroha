/**
 * Identity of a chart's inputs by content. Charts redraw when this changes, not
 * when a parent re-renders and hands down structurally identical arrays; that
 * happens on every prop change when the props travel through spread props, and
 * a redraw clears the canvas and replays the entry animation.
 */
export function chartSignature(...inputs: unknown[]): string {
  return JSON.stringify(inputs);
}
