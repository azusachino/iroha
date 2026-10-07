// Bounded DOM diagnostics, not an axe/WCAG conformance audit. Canvas marks,
// non-constant-hue images and native browser zoom need other evidence.
export function measurePilotPage() {
  const visible = (el: Element) => {
    const s = getComputedStyle(el);
    const r = el.getBoundingClientRect();
    const closed = el.closest("details:not([open])");
    if (closed && !closed.querySelector(":scope > summary")?.contains(el))
      return false;
    if (
      !el.checkVisibility({
        checkOpacity: true,
        checkVisibilityCSS: true,
        contentVisibilityAuto: true,
      })
    )
      return false;
    return (
      r.width > 0 &&
      r.height > 0 &&
      s.visibility === "visible" &&
      s.display !== "none" &&
      !el.closest("[inert], [aria-hidden='true']")
    );
  };
  const name = (el: Element) =>
    el.getAttribute("aria-label") ||
    (el.getAttribute("aria-labelledby") || "")
      .split(/\s+/)
      .map((id) => document.getElementById(id)?.textContent || "")
      .join(" ")
      .trim() ||
    ("labels" in el
      ? Array.from((el as HTMLInputElement).labels || [])
          .map((label) => label.textContent)
          .join(" ")
      : "") ||
    el.textContent?.trim() ||
    el.getAttribute("title") ||
    "";
  const canvas = document.createElement("canvas");
  canvas.width = canvas.height = 1;
  const ctx = canvas.getContext("2d")!;
  const rgba = (color: string) => {
    ctx.clearRect(0, 0, 1, 1);
    ctx.fillStyle = color;
    ctx.fillRect(0, 0, 1, 1);
    const pixel = ctx.getImageData(0, 0, 1, 1).data;
    return [pixel[0], pixel[1], pixel[2], pixel[3] / 255];
  };
  const over = (front: number[], back: number[]) => {
    const alpha = front[3] + back[3] * (1 - front[3]);
    return alpha === 0
      ? [0, 0, 0, 0]
      : [0, 1, 2]
          .map(
            (i) =>
              (front[i] * front[3] + back[i] * back[3] * (1 - front[3])) /
              alpha,
          )
          .concat(alpha);
  };
  const luminance = (color: number[]) =>
    color
      .slice(0, 3)
      .map((v) => {
        const s = v / 255;
        return s <= 0.04045 ? s / 12.92 : ((s + 0.055) / 1.055) ** 2.4;
      })
      .reduce((sum, v, i) => sum + v * [0.2126, 0.7152, 0.0722][i], 0);
  const controls = Array.from(
    document.querySelectorAll(
      "button,a[href],input,select,textarea,summary,[tabindex='0']",
    ),
  )
    .filter(visible)
    .map((el) => {
      const r = el.getBoundingClientRect();
      const s = getComputedStyle(el);
      return {
        tag: el.tagName,
        name: name(el),
        width: r.width,
        height: r.height,
        x: r.x,
        right: r.right,
        clipped: r.right > innerWidth + 1 || r.left < -1,
        fontSize: s.fontSize,
        outline: s.outline,
        disabled: el.hasAttribute("disabled"),
      };
    });
  const backgroundAt = (element: Element) => {
    let backgrounds = [[0, 0, 0, 0]];
    for (let el: Element | null = element; el; el = el.parentElement) {
      const style = getComputedStyle(el);
      const opacity = Number(style.opacity);
      if (Number.isNaN(opacity) || opacity <= 0) continue;
      if (backgrounds.every((b) => b[3] === 1)) break;

      let layers = [rgba(style.backgroundColor)];
      if (style.backgroundImage !== "none") {
        const images: string[] = [];
        let depth = 0,
          start = 0;
        for (let index = 0; index < style.backgroundImage.length; index++) {
          const char = style.backgroundImage[index];
          if (char === "(") depth++;
          if (char === ")") depth--;
          if (char === "," && depth === 0) {
            images.push(style.backgroundImage.slice(start, index).trim());
            start = index + 1;
          }
        }
        images.push(style.backgroundImage.slice(start).trim());
        for (const image of images.reverse()) {
          if (image === "none") continue;
          const stops = image.match(
            /(?:rgba?|hsla?|color|hwb|lab|lch|oklab|oklch)\([^)]+\)|transparent/g,
          );
          if (!/^(linear|radial)-gradient\(/.test(image) || !stops) return null;
          const colors = stops.map(rgba);
          const solid = colors.filter((c) => c[3] > 0);
          if (
            solid.some((c) =>
              c.slice(0, 3).some((v, idx) => v !== solid[0][idx]),
            )
          )
            return null;
          layers = layers.flatMap((back) =>
            colors.map((front) => over(front, back)),
          );
        }
      }
      if (opacity < 1) {
        layers = layers.map((layer) => [
          layer[0],
          layer[1],
          layer[2],
          layer[3] * opacity,
        ]);
        backgrounds = backgrounds.map((bg) => [
          bg[0],
          bg[1],
          bg[2],
          bg[3] * opacity,
        ]);
      }
      backgrounds = backgrounds.flatMap((front) =>
        layers.map((back) => over(front, back)),
      );
    }
    return backgrounds.every((b) => b[3] === 1) ? backgrounds : null;
  };

  const textPairs: {
    text: string;
    ratio: number;
    required: number;
    foreground: number[];
    background: number[];
  }[] = [];
  let unmeasuredText = 0;
  const walker = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT);
  while (walker.nextNode()) {
    const node = walker.currentNode;
    const el = node.parentElement;
    const text = node.textContent?.trim();
    if (
      !el ||
      !text ||
      !visible(el) ||
      el.closest("script,style,option,canvas")
    )
      continue;

    const backgrounds = backgroundAt(el);
    if (!backgrounds || backgrounds.length === 0) {
      unmeasuredText++;
      continue;
    }

    let effectiveOpacity = 1;
    for (let anc: Element | null = el; anc; anc = anc.parentElement) {
      const op = Number(getComputedStyle(anc).opacity);
      if (!Number.isNaN(op)) effectiveOpacity *= op;
    }

    const style = getComputedStyle(el);
    const textRgba = rgba(style.color);
    textRgba[3] *= effectiveOpacity;

    const required =
      parseFloat(style.fontSize) >= 24 ||
      (parseFloat(style.fontSize) >= 18.5 && Number(style.fontWeight) >= 700)
        ? 3
        : 4.5;

    const candidates = backgrounds.map((bg) => {
      const foreground = over(textRgba, bg);
      const a = luminance(foreground);
      const b = luminance(bg);
      const ratio = (Math.max(a, b) + 0.05) / (Math.min(a, b) + 0.05);
      return { ratio, foreground, background: bg };
    });

    const worst = candidates.reduce(
      (min, c) => (c.ratio < min.ratio ? c : min),
      candidates[0],
    );

    textPairs.push({
      text: text.slice(0, 100),
      ratio: worst.ratio,
      required,
      foreground: worst.foreground,
      background: worst.background,
    });
  }
  return {
    viewportWidth: innerWidth,
    scrollWidth: document.documentElement.scrollWidth,
    bodyWidth: document.body.scrollWidth,
    mode: document.documentElement.dataset.theme,
    reducedMotion: matchMedia("(prefers-reduced-motion: reduce)").matches,
    mainCount: Array.from(document.querySelectorAll("main")).filter(visible)
      .length,
    headings: Array.from(document.querySelectorAll("h1,h2,h3"))
      .filter(visible)
      .map((el) => {
        const s = getComputedStyle(el);
        return {
          tag: el.tagName,
          text: el.textContent,
          size: s.fontSize,
          weight: s.fontWeight,
          lineHeight: s.lineHeight,
        };
      }),
    controls,
    textPairs,
    unmeasuredText,
    runningCssAnimations: document
      .getAnimations()
      .filter((a) => a.playState === "running").length,
    figures: Array.from(document.querySelectorAll("strong,td,dd"))
      .filter(visible)
      .slice(0, 30)
      .map((el) => ({
        text: el.textContent?.slice(0, 60),
        font: getComputedStyle(el).fontFamily,
        numeric: getComputedStyle(el).fontVariantNumeric,
      })),
  };
}
