// Inspect the already-loaded module or DOM/window instances through ECharts' public API.
// Do not import a second copy or infer canvas data from an ARIA label.
export async function inspectPilotCharts() {
  let core: { getInstanceByDom: (el: Element) => any } | null =
    (
      window as unknown as {
        __echarts_core__?: { getInstanceByDom: (el: Element) => any };
      }
    ).__echarts_core__ ??
    ((
      window as unknown as { __echarts_getInstanceByDom?: (el: Element) => any }
    ).__echarts_getInstanceByDom
      ? {
          getInstanceByDom: (
            window as unknown as {
              __echarts_getInstanceByDom: (el: Element) => any;
            }
          ).__echarts_getInstanceByDom,
        }
      : null);

  const entry = performance
    .getEntriesByType("resource")
    .find((resource) => /\/echarts_core\.js(?:\?|$)/.test(resource.name));

  if (!core && entry) {
    core = await import(entry.name);
  }

  const getChart = (el: Element) => {
    return (
      (el as unknown as { __echarts_instance__?: any }).__echarts_instance__ ??
      (el.querySelector("canvas") as unknown as { __echarts_instance__?: any })
        ?.__echarts_instance__ ??
      core?.getInstanceByDom(el) ??
      (
        window as unknown as {
          __echarts_getInstanceByDom?: (node: Element) => any;
        }
      ).__echarts_getInstanceByDom?.(el) ??
      null
    );
  };

  const imgElements = Array.from(
    document.querySelectorAll('[role="img"], [_echarts_instance_]'),
  );

  const hasChartInstance = imgElements.some((el) => getChart(el));
  if (!core && !hasChartInstance && !entry)
    return { verified: false as const, reason: "No loaded ECharts core URL" };

  const charts = imgElements.flatMap((el) => {
    const chart = getChart(el);
    if (!chart) return [];
    const option = chart.getOption();
    const axes = (
      values: {
        type?: string;
        name?: string;
        data?: unknown[];
        min?: number;
        max?: number;
        interval?: number;
        axisLabel?: {
          hideOverlap?: boolean;
          fontSize?: number;
          formatter?: (value: number) => string;
        };
      }[],
    ) =>
      values.map((axis) => ({
        type: axis.type,
        name: axis.name,
        data: axis.data,
        min: axis.min,
        max: axis.max,
        interval: axis.interval,
        tickLabels:
          Number.isFinite(axis.min) &&
          Number.isFinite(axis.max) &&
          Number.isFinite(axis.interval) &&
          axis.interval! > 0
            ? Array.from(
                {
                  length: Math.min(
                    100,
                    Math.round((axis.max! - axis.min!) / axis.interval!) + 1,
                  ),
                },
                (_, index) => {
                  const value = axis.min! + index * axis.interval!;
                  return typeof axis.axisLabel?.formatter === "function"
                    ? axis.axisLabel.formatter(value)
                    : String(value);
                },
              )
            : [],
        hideOverlap: axis.axisLabel?.hideOverlap,
        fontSize: axis.axisLabel?.fontSize,
      }));
    const series: {
      name?: string;
      data?: unknown[];
      showSymbol?: boolean;
      showAllSymbol?: boolean | "auto";
      connectNulls?: boolean;
    }[] = option.series ?? [];
    return [
      {
        label: el.getAttribute("aria-label"),
        series: series.map((series, seriesIndex) => ({
          name: series.name,
          data: series.data,
          showSymbol: series.showSymbol,
          showAllSymbol: series.showAllSymbol,
          connectNulls: series.connectNulls,
          color: chart.getVisual({ seriesIndex }, "color"),
          symbolSizes: series.data?.map((_, dataIndex) =>
            chart.getVisual({ seriesIndex, dataIndex }, "symbolSize"),
          ),
        })),
        animation: option.animation,
        animationDuration: option.animationDuration,
        grid: (option.grid ?? []).map((grid: { bottom?: number | string }) => ({
          bottom: grid.bottom,
        })),
        xAxis: axes(option.xAxis ?? []),
        yAxis: axes(option.yAxis ?? []),
      },
    ];
  });
  return { verified: true as const, module: entry?.name ?? "runtime", charts };
}
