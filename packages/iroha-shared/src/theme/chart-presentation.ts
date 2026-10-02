// Canvas paint does not inherit new CSS colors after it has been drawn.
export function observeChartPresentation(render: () => void): () => void {
  const preferences = [
    window.matchMedia("(prefers-reduced-motion: reduce)"),
    window.matchMedia("(prefers-color-scheme: dark)"),
  ];
  for (const preference of preferences) preference.addEventListener("change", render);
  const observer = new MutationObserver(render);
  observer.observe(document.documentElement, {
    attributes: true,
    attributeFilter: ["data-theme", "data-language", "style"],
  });
  return () => {
    observer.disconnect();
    for (const preference of preferences) preference.removeEventListener("change", render);
  };
}
