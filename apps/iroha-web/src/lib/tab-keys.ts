/** Arrow/Home/End navigation for the WAI-ARIA tabs pattern (automatic activation). */
export function nextTabId<T extends string>(
  ids: readonly T[],
  current: T,
  key: string,
): T | null {
  const index = ids.indexOf(current);
  if (index < 0 || ids.length === 0) return null;
  switch (key) {
    case "ArrowRight":
    case "ArrowDown":
      return ids[(index + 1) % ids.length];
    case "ArrowLeft":
    case "ArrowUp":
      return ids[(index - 1 + ids.length) % ids.length];
    case "Home":
      return ids[0];
    case "End":
      return ids[ids.length - 1];
    default:
      return null;
  }
}
