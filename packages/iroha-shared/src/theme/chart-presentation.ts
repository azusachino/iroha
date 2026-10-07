import { registerPostInit, getInstanceByDom } from "echarts/core";

// Ensure chart instances are attached to DOM and discoverable in both dev and production
if (typeof window !== "undefined") {
  (window as unknown as { __echarts_core__?: unknown }).__echarts_core__ = {
    getInstanceByDom,
  };
  (
    window as unknown as {
      __echarts_getInstanceByDom?: typeof getInstanceByDom;
    }
  ).__echarts_getInstanceByDom = getInstanceByDom;
  try {
    registerPostInit((chart) => {
      const dom = chart.getDom();
      if (dom) {
        (dom as unknown as { __echarts_instance__?: unknown }).__echarts_instance__ =
          chart;
        const canvas = dom.querySelector("canvas");
        if (canvas) {
          (
            canvas as unknown as { __echarts_instance__?: unknown }
          ).__echarts_instance__ = chart;
        }
      }
    });
  } catch (error) {
    void error;
  }
}

export function attachChartInstance(
  dom: HTMLElement | null | undefined,
  chart: unknown,
): void {
  if (dom && chart) {
    (dom as unknown as { __echarts_instance__?: unknown }).__echarts_instance__ =
      chart;
    const canvas = dom.querySelector("canvas");
    if (canvas) {
      (
        canvas as unknown as { __echarts_instance__?: unknown }
      ).__echarts_instance__ = chart;
    }
  }
}

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
