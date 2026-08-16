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
