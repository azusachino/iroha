// Bounded DOM diagnostics, not an axe/WCAG conformance audit. Canvas marks,
// gradients/images, ancestor opacity and native browser zoom need other evidence.
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
    let background = [0, 0, 0, 0];
    let supported = true;
    for (
      let ancestor: Element | null = el;
      ancestor;
      ancestor = ancestor.parentElement
    ) {
      const style = getComputedStyle(ancestor);
      if (
        (background[3] < 1 && style.backgroundImage !== "none") ||
        Number(style.opacity) !== 1
      ) {
        supported = false;
        break;
      }
      if (background[3] < 1)
        background = over(background, rgba(style.backgroundColor));
    }
    if (!supported || background[3] < 1) {
      unmeasuredText++;
      continue;
    }
    const style = getComputedStyle(el);
    const foreground = over(rgba(style.color), background);
    const a = luminance(foreground),
      b = luminance(background);
    const required =
      parseFloat(style.fontSize) >= 24 ||
      (parseFloat(style.fontSize) >= 18.5 && Number(style.fontWeight) >= 700)
        ? 3
        : 4.5;
    textPairs.push({
      text: text.slice(0, 100),
      ratio: (Math.max(a, b) + 0.05) / (Math.min(a, b) + 0.05),
      required,
      foreground,
      background,
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
