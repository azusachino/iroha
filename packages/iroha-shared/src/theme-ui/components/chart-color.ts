// Canvas chart renderers need a resolved paint value, not a CSS var() reference.
export function resolveColor(
  value: string | undefined,
  styles: Pick<CSSStyleDeclaration, "getPropertyValue">,
  fallback: string,
): string {
  if (!value) return fallback;
  const match = value.match(/^var\((--[\w-]+)\)$/);
  return match ? styles.getPropertyValue(match[1]).trim() || fallback : value;
}
