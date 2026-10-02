import { chromium, expect, test } from "@playwright/test";
import { mkdir, writeFile } from "node:fs/promises";
import { PUBLIC_BASE_URL } from "../playwright.config";
import { installPilotFixtures } from "./pilot-fixtures";

// Chrome's own tab zoom, not CSS zoom or CDP compositor/pinch scaling.
for (const pilot of ["overview", "expenses", "metrics", "public"] as const) {
  for (const mode of ["light", "dark"] as const) {
    test(`${pilot} ${mode} native 200 percent zoom reflows to 320 CSS pixels`, async ({
      baseURL,
    }, info) => {
      const extension = info.outputPath("zoom-extension");
      await mkdir(extension, { recursive: true });
      await writeFile(
        `${extension}/manifest.json`,
        JSON.stringify({
          manifest_version: 3,
          name: "Iroha native zoom fixture",
          version: "1.0",
          permissions: ["tabs"],
          background: { service_worker: "worker.js" },
        }),
      );
      await writeFile(
        `${extension}/worker.js`,
        "chrome.runtime.onInstalled.addListener(() => {});",
      );
      const context = await chromium.launchPersistentContext(
        info.outputPath("profile"),
        {
          channel: "chromium",
          headless: true,
          viewport: null,
          deviceScaleFactor: undefined,
          ignoreDefaultArgs: ["--disable-extensions"],
          args: [
            `--disable-extensions-except=${extension}`,
            `--load-extension=${extension}`,
            "--window-size=640,1000",
          ],
        },
      );
      try {
        const page = await context.newPage();
        await page.emulateMedia({ colorScheme: mode, reducedMotion: "reduce" });
        const fixture = await installPilotFixtures(page, pilot);
        const errors: string[] = [];
        page.on("pageerror", (error) =>
          errors.push(error.stack ?? error.message),
        );
        const base = pilot === "public" ? PUBLIC_BASE_URL : baseURL;
        if (!base) throw new Error("Playwright baseURL is not configured");
        await page.goto(
          pilot === "public"
            ? base
            : `${base}/${pilot}${pilot === "expenses" ? "?date=2026-08&currency=JPY" : ""}`,
        );
        await expect(page.getByRole("img").first()).toBeVisible();
        const before = await page.evaluate(() => ({
          width: innerWidth,
          dpr: devicePixelRatio,
        }));
        const worker =
          context.serviceWorkers()[0] ??
          (await context.waitForEvent("serviceworker"));
        const zoom = await worker.evaluate(`new Promise((resolve, reject) => {
          chrome.tabs.query({}, tabs => {
            const tab = tabs.find(tab => tab.url === ${JSON.stringify(page.url())});
            if (!tab) return reject(new Error('Fixture tab not found'));
            chrome.tabs.setZoom(tab.id, 2, () => {
              if (chrome.runtime.lastError) return reject(new Error(chrome.runtime.lastError.message));
              chrome.tabs.getZoom(tab.id, resolve);
            });
          });
        })`);
        expect(zoom).toBe(2);
        await expect.poll(() => page.evaluate(() => innerWidth)).toBe(320);
        await expect
          .poll(() =>
            page.evaluate(
              () => document.documentElement.scrollWidth <= innerWidth + 1,
            ),
          )
          .toBe(true);
        const measured = await page.evaluate(() => ({
          width: innerWidth,
          dpr: devicePixelRatio,
          document: document.documentElement.scrollWidth,
          cssZoom: getComputedStyle(document.documentElement).zoom,
          overflow: Array.from(document.querySelectorAll("body *"))
            .filter((el) => el.getBoundingClientRect().width > innerWidth)
            .slice(0, 12)
            .map((el) => ({
              tag: el.tagName,
              className: el.className,
              width: el.getBoundingClientRect().width,
              minWidth: getComputedStyle(el).minWidth,
            })),
        }));
        await info.attach("zoom-layout.json", {
          body: JSON.stringify(measured),
          contentType: "application/json",
        });
        expect(before.width).toBe(640);
        expect(measured.dpr).toBe(before.dpr * 2);
        expect(measured.document).toBeLessThanOrEqual(measured.width + 1);
        expect(measured.cssZoom).toBe("1");
        await info.attach("native-zoom.json", {
          body: JSON.stringify({ pilot, mode, zoom, before, measured }),
          contentType: "application/json",
        });
        // A full-page screenshot with viewport:null temporarily resizes the
        // browser, changing the native zoom layout under measurement.
        const cdp = await context.newCDPSession(page);
        const capture = await cdp.send("Page.captureScreenshot", {
          format: "png",
          captureBeyondViewport: false,
        });
        await info.attach("native-zoom.png", {
          body: Buffer.from(capture.data, "base64"),
          contentType: "image/png",
        });
        expect(await page.evaluate(() => innerWidth)).toBe(320);
        expect(errors).toEqual([]);
        expect(fixture.unknown).toEqual([]);
      } finally {
        await context.close();
      }
    });
  }
}
