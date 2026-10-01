// Inspect the already-loaded dev-server module through ECharts' public API.
// Do not import a second copy or infer canvas data from an ARIA label.
export async function inspectPilotCharts() {
  const entry = performance
    .getEntriesByType("resource")
    .find((resource) => /\/echarts_core\.js(?:\?|$)/.test(resource.name));
  if (!entry)
    return { verified: false as const, reason: "No loaded ECharts core URL" };
  const core = await import(entry.name);
  const charts = Array.from(document.querySelectorAll('[role="img"]')).flatMap(
    (el) => {
      const chart = core.getInstanceByDom(el);
      if (!chart) return [];
      const option = chart.getOption();
      const series: {
        name?: string;
        data?: unknown[];
        showSymbol?: boolean;
      }[] = option.series ?? [];
      return [
        {
          label: el.getAttribute("aria-label"),
          series: series.map((series) => ({
            name: series.name,
            data: series.data,
            showSymbol: series.showSymbol,
          })),
          animation: option.animation,
          animationDuration: option.animationDuration,
          xAxis: (option.xAxis ?? []).map(
            (axis: { name?: string; data?: unknown[] }) => ({
              name: axis.name,
              data: axis.data,
            }),
          ),
          yAxis: (option.yAxis ?? []).map(
            (axis: { name?: string; min?: number; max?: number }) => ({
              name: axis.name,
              min: axis.min,
              max: axis.max,
            }),
          ),
        },
      ];
    },
  );
  return { verified: true as const, module: entry.name, charts };
}
