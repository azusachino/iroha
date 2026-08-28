#!/usr/bin/env bun
// Screenshot themed routes with Playwright instead of agent-browser.
//
// agent-browser (this workstation's documented default web tool) cannot
// launch any Chrome build on this machine -- its own bundled binary, and
// the system google-chrome, both crash with the same
// GLIBC_ABI_GNU2_TLS/GLIBC_ABI_DT_X86_64_PLT mismatch against a
// Nix-provided alsa-lib pulled in globally, independent of LD_LIBRARY_PATH.
// See docs/runbooks/pitfalls/ in harus-workstation for the full trail.
// Playwright's own downloaded Chrome-for-Testing build does not hit this,
// so it's the working choice here -- not a default, a deliberate pick
// after agent-browser was verified broken in this sandbox.
//
// Usage: BASE=http://127.0.0.1:5173 THEME=atlas ROUTES=overview,expenses \
//   bun run scripts/visual-check.mjs
// Optional: CANVAS_SELECTOR="canvas.ambient-canvas" to also report WebGL
// draw-call counts and non-transparent pixel coverage for a canvas that's
// hard to eyeball in a screenshot (e.g. a faint ambient background) --
// headless Chromium's software compositor does not reliably paint WebGL
// canvas content into page screenshots in this sandbox even when the
// underlying GL buffer has real drawn content, so the pixel/draw-call
// counts are the trustworthy signal here, not the screenshot itself.

import { chromium } from "playwright";

// Must be cleared before Chromium launches (inherited by the spawned
// process); a Nix profile's alsa-lib on this machine's LD_LIBRARY_PATH is
// what triggers the crash above.
delete process.env.LD_LIBRARY_PATH;

const base = process.env.BASE || "http://127.0.0.1:5173";
const theme = process.env.THEME || "field-journal";
const routes = (process.env.ROUTES || "overview")
  .split(",")
  .map((r) => r.trim());
const out = process.env.OUT || ".visual-check";
const canvasSelector = process.env.CANVAS_SELECTOR;

const { mkdir } = await import("node:fs/promises");
await mkdir(out, { recursive: true });

const browser = await chromium.launch({
  headless: true,
  args: ["--no-sandbox"],
});
const page = await browser.newPage({ viewport: { width: 1280, height: 800 } });

const errors = [];
page.on("pageerror", (err) => errors.push(err.message));
page.on("console", (msg) => {
  if (msg.type() === "error") errors.push(msg.text());
});

await page.addInitScript(({ key, value }) => localStorage.setItem(key, value), {
  key: "iroha-design-language",
  value: theme,
});

for (const route of routes) {
  const path = route.replace(/^\/+/, "");
  const url = `${base.replace(/\/+$/, "")}/${path}`;
  errors.length = 0;
  await page.goto(url, { waitUntil: "networkidle" });
  await page.waitForTimeout(800);

  if (errors.length) {
    console.log(`errors on ${path}:`, errors);
  }

  const shotPath = `${out}/${theme}-${path.replaceAll("/", "-") || "root"}.png`;
  await page.screenshot({ path: shotPath, fullPage: true });
  console.log(`checked: ${path} -> ${shotPath}`);

  if (canvasSelector) {
    const info = await page.evaluate((selector) => {
      const canvas = document.querySelector(selector);
      if (!canvas) return { found: false };
      const gl = canvas.getContext("webgl2") || canvas.getContext("webgl");
      if (!gl) return { found: true, hasGlContext: false };
      const w = canvas.width;
      const h = canvas.height;
      const pixels = new Uint8Array(w * h * 4);
      gl.readPixels(0, 0, w, h, gl.RGBA, gl.UNSIGNED_BYTE, pixels);
      let nonTransparent = 0;
      for (let i = 3; i < pixels.length; i += 4)
        if (pixels[i] > 0) nonTransparent += 1;
      return {
        found: true,
        hasGlContext: true,
        width: w,
        height: h,
        nonTransparent,
      };
    }, canvasSelector);
    console.log(`canvas (${canvasSelector}) on ${path}:`, JSON.stringify(info));
  }
}

await browser.close();
