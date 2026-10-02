// Measures resolved paint over solid surfaces and constant-hue gradient bounds,
// not pixel geometry or color-vision conformance. Other paint fails explicitly.
export async function measureRenderedContrast(includeText = false) {
  const canvas = document.createElement("canvas");
  canvas.width = canvas.height = 1;
  const context = canvas.getContext("2d")!;
  const rgba = (paint: string) => {
    context.clearRect(0, 0, 1, 1);
    context.fillStyle = paint;
    context.fillRect(0, 0, 1, 1);
    const pixel = context.getImageData(0, 0, 1, 1).data;
    return [pixel[0], pixel[1], pixel[2], pixel[3] / 255];
  };
  const over = (front: number[], back: number[]) => {
    const alpha = front[3] + back[3] * (1 - front[3]);
    return alpha
      ? [0, 1, 2]
          .map(
            (i) =>
              (front[i] * front[3] + back[i] * back[3] * (1 - front[3])) /
              alpha,
          )
          .concat(alpha)
      : [0, 0, 0, 0];
  };
  const backgroundAt = (element: Element | null) => {
    let backgrounds = [[0, 0, 0, 0]];
    for (let el = element; el; el = el.parentElement) {
      const style = getComputedStyle(el);
      if (Number(style.opacity) !== 1) return null;
      if (backgrounds.every((background) => background[3] === 1)) continue;
      let layers = [rgba(style.backgroundColor)];
      if (style.backgroundImage !== "none") {
        // Bound the existing constant-hue linear/radial overlays independently.
        // Colored-to-colored interpolation, images and other paint stay unsupported.
        const images = [];
        let depth = 0,
          start = 0;
        for (let index = 0; index < style.backgroundImage.length; index++) {
          const character = style.backgroundImage[index];
          if (character === "(") depth++;
          if (character === ")") depth--;
          if (character === "," && depth === 0) {
            images.push(style.backgroundImage.slice(start, index).trim());
            start = index + 1;
          }
        }
        images.push(style.backgroundImage.slice(start).trim());
        for (const image of images.reverse()) {
          if (image === "none") continue;
          const stops = image.match(/rgba?\([^)]+\)/g);
          if (!/^(linear|radial)-gradient\(/.test(image) || !stops) return null;
          const colors = stops.map(rgba);
          const solid = colors.filter((color) => color[3] > 0);
          if (
            solid.some((color) =>
              color
                .slice(0, 3)
                .some((value, index) => value !== solid[0][index]),
            )
          )
            return null;
          layers = layers.flatMap((back) =>
            colors.map((front) => over(front, back)),
          );
        }
      }
      backgrounds = backgrounds.flatMap((front) =>
        layers.map((back) => over(front, back)),
      );
    }
    return backgrounds.every((background) => background[3] === 1)
      ? backgrounds
      : null;
  };
  const luminance = (color: number[]) =>
    color
      .slice(0, 3)
      .map((channel) => {
        const value = channel / 255;
        return value <= 0.04045
          ? value / 12.92
          : ((value + 0.055) / 1.055) ** 2.4;
      })
      .reduce(
        (sum, value, index) => sum + value * [0.2126, 0.7152, 0.0722][index],
        0,
      );
  const contrast = (paint: unknown, backgrounds: number[][] | null) => {
    if (
      typeof paint !== "string" ||
      !CSS.supports("color", paint) ||
      !backgrounds
    )
      return null;
    return Math.min(
      ...backgrounds.map((background) => {
        const front = luminance(over(rgba(paint), background));
        const back = luminance(background);
        return (Math.max(front, back) + 0.05) / (Math.min(front, back) + 0.05);
      }),
    );
  };
  const focused = document.activeElement;
  const focusStyle = focused ? getComputedStyle(focused) : null;
  const focus = focusStyle
    ? {
        color: focusStyle.outlineColor,
        background: backgroundAt(focused!.parentElement),
        ancestors: (() => {
          const styles = [];
          for (let el = focused!.parentElement; el; el = el.parentElement) {
            const style = getComputedStyle(el);
            styles.push({
              tag: el.tagName,
              background: style.backgroundColor,
              image: style.backgroundImage,
              opacity: style.opacity,
            });
          }
          return styles;
        })(),
        ratio: contrast(
          focusStyle.outlineColor,
          backgroundAt(focused!.parentElement),
        ),
      }
    : null;
  const entry = performance
    .getEntriesByType("resource")
    .find((resource) => /\/echarts_core\.js(?:\?|$)/.test(resource.name));
  const marks: {
    chart: string | null;
    series: string;
    color: unknown;
    ratio: number | null;
  }[] = [];
  if (entry) {
    const core = await import(entry.name);
    for (const el of document.querySelectorAll('[role="img"]')) {
      const chart = core.getInstanceByDom(el);
      if (!chart) continue;
      const option = chart.getOption();
      const background = backgroundAt(el);
      for (const [seriesIndex, series] of (option.series ?? []).entries()) {
        for (const [dataIndex, datum] of (series.data ?? []).entries()) {
          const value =
            typeof datum === "object" && datum !== null ? datum.value : datum;
          if (!Number.isFinite(value)) continue;
          const color = chart.getVisual({ seriesIndex, dataIndex }, "color");
          marks.push({
            chart: el.getAttribute("aria-label"),
            series: series.name,
            color,
            ratio: contrast(color, background),
          });
        }
      }
    }
  }
  const textPairs = [];
  const walker = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT);
  while (includeText && walker.nextNode()) {
    const el = walker.currentNode.parentElement;
    const text = walker.currentNode.textContent?.trim();
    if (
      !el ||
      !text ||
      el.closest("script,style,option,canvas,[inert],[aria-hidden='true']")
    )
      continue;
    const closed = el.closest("details:not([open])");
    if (closed && !closed.querySelector(":scope > summary")?.contains(el))
      continue;
    if (!el.checkVisibility({ checkOpacity: true, checkVisibilityCSS: true }))
      continue;
    const style = getComputedStyle(el);
    const required =
      parseFloat(style.fontSize) >= 24 ||
      (parseFloat(style.fontSize) >= 18.5 && Number(style.fontWeight) >= 700)
        ? 3
        : 4.5;
    textPairs.push({
      text: text.slice(0, 100),
      color: style.color,
      ratio: contrast(style.color, backgroundAt(el)),
      required,
    });
  }
  return { focus, marks, textPairs };
}
