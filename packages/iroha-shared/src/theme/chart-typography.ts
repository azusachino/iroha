// ECharts needs CSS pixels; the canonical role itself remains a rem token.
export function chartFontSize(styles: CSSStyleDeclaration): number {
  return Number.parseFloat(styles.getPropertyValue("--type-caption")) *
    Number.parseFloat(getComputedStyle(document.documentElement).fontSize);
}
