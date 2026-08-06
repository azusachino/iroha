// Media titles are frequently long, wordy light-novel titles (verified: a
// real synced title ran 100+ characters) with no upper bound from any
// provider. A CSS clamp() alone only reacts to viewport width, not content
// length, so a theme's normal hero font-size renders such a title as many
// lines of oversized text. heroTitleFontSize scales a theme's own clamp()
// down as title length grows, preserving each theme's relative sizing
// instead of imposing one shared size.
export interface HeroTitleClamp {
  minRem: number;
  vw: number;
  maxRem: number;
}

const LENGTH_SCALE_STEPS: Array<[minLength: number, scale: number]> = [
  [80, 0.42],
  [60, 0.55],
  [40, 0.7],
  [25, 0.85],
];

function scaleForLength(length: number): number {
  for (const [minLength, scale] of LENGTH_SCALE_STEPS) {
    if (length > minLength) return scale;
  }
  return 1;
}

export function heroTitleFontSize(title: string, base: HeroTitleClamp): string {
  const scale = scaleForLength(title.length);
  const min = (base.minRem * scale).toFixed(2);
  const vw = (base.vw * scale).toFixed(2);
  const max = (base.maxRem * scale).toFixed(2);
  return `clamp(${min}rem, ${vw}vw, ${max}rem)`;
}
