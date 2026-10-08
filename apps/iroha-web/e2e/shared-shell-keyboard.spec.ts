import {
  chromium,
  expect,
  test,
  type Page,
  type TestInfo,
} from "@playwright/test";
import { createHash } from "node:crypto";
import { mkdir, writeFile } from "node:fs/promises";
import { installPilotFixtures } from "./pilot-fixtures";

const modes = ["light", "dark"] as const;
const motions = ["no-preference", "reduce"] as const;
const compactWidths = [320, 390] as const;
const commandCount = 11;

function observeBrowser(page: Page) {
  const consoleErrors: string[] = [];
  const pageErrors: string[] = [];
  const apiRequests: { method: string; path: string }[] = [];
  const apiResponses: { method: string; path: string; status: number }[] = [];
  const apiFailures: string[] = [];
  page.on("console", (message) => {
    if (message.type() === "error") consoleErrors.push(message.text());
  });
  page.on("pageerror", (error) =>
    pageErrors.push(error.stack ?? error.message),
  );
  page.on("request", (request) => {
    const url = new URL(request.url());
    if (url.pathname.startsWith("/api/")) {
      apiRequests.push({
        method: request.method(),
        path: url.pathname + url.search,
      });
    }
  });
  page.on("response", (response) => {
    const url = new URL(response.url());
    if (url.pathname.startsWith("/api/")) {
      apiResponses.push({
        method: response.request().method(),
        path: url.pathname + url.search,
        status: response.status(),
      });
    }
  });
  page.on("requestfailed", (request) => {
    const url = new URL(request.url());
    if (url.pathname.startsWith("/api/")) {
      apiFailures.push(
        `${request.method()} ${url.pathname}${url.search}: ${request.failure()?.errorText}`,
      );
    }
  });
  return { consoleErrors, pageErrors, apiRequests, apiResponses, apiFailures };
}

async function tabUntilFocused(
  page: Page,
  target: ReturnType<Page["getByRole"]>,
) {
  for (let index = 0; index < 80; index++) {
    if (
      await target
        .evaluate((element) => document.activeElement === element)
        .catch(() => false)
    )
      return;
    await page.keyboard.press("Tab");
  }
  await expect(target).toBeFocused();
}

async function focusSnapshot(locator: ReturnType<Page["getByRole"]>) {
  return locator.evaluate((element) => {
    const style = getComputedStyle(element);
    const rect = element.getBoundingClientRect();
    return {
      active: document.activeElement === element,
      outlineStyle: style.outlineStyle,
      outlineWidth: parseFloat(style.outlineWidth),
      outlineColor: style.outlineColor,
      rect: {
        left: rect.left,
        right: rect.right,
        top: rect.top,
        bottom: rect.bottom,
        height: rect.height,
      },
      viewport: { width: innerWidth, height: innerHeight },
    };
  });
}

async function assertMenuFocus(locator: ReturnType<Page["getByRole"]>) {
  await expect(locator).toBeFocused();
  const focus = await focusSnapshot(locator);
  expect(focus.active).toBe(true);
  expect(focus.outlineStyle).not.toBe("none");
  expect(focus.outlineWidth).toBeGreaterThanOrEqual(2);
  expect(focus.outlineColor).not.toBe("transparent");
  expect(focus.outlineColor).not.toBe("rgba(0, 0, 0, 0)");
  expect(focus.rect.height).toBeGreaterThanOrEqual(24);
  expect(focus.rect.left).toBeGreaterThanOrEqual(-1);
  expect(focus.rect.right).toBeLessThanOrEqual(focus.viewport.width + 1);
  expect(focus.rect.top).toBeLessThan(focus.viewport.height);
  expect(focus.rect.bottom).toBeGreaterThan(0);
  return focus;
}

async function captureViewport(
  page: Page,
  info: TestInfo,
  name: string,
  details: Record<string, unknown>,
) {
  const before = await page.evaluate(() => ({
    width: innerWidth,
    height: innerHeight,
    dpr: devicePixelRatio,
    scrollX,
    scrollY,
  }));
  const session = await page.context().newCDPSession(page);
  const capture = await session.send("Page.captureScreenshot", {
    format: "png",
    captureBeyondViewport: false,
  });
  const png = Buffer.from(capture.data, "base64");
  const after = await page.evaluate(() => ({
    width: innerWidth,
    height: innerHeight,
    dpr: devicePixelRatio,
    scrollX,
    scrollY,
  }));
  await session.detach();
  expect(after).toEqual(before);
  const dimensions = {
    width: png.readUInt32BE(16),
    height: png.readUInt32BE(20),
  };
  expect(
    Math.abs(dimensions.width - before.width * before.dpr),
  ).toBeLessThanOrEqual(1);
  expect(
    Math.abs(dimensions.height - before.height * before.dpr),
  ).toBeLessThanOrEqual(1);
  const sha256 = createHash("sha256").update(png).digest("hex");
  await info.attach(name, { body: png, contentType: "image/png" });
  await info.attach(`${name}.json`, {
    body: JSON.stringify(
      { name, details, before, after, dimensions, sha256 },
      null,
      2,
    ),
    contentType: "application/json",
  });
  return { before, dimensions, sha256 };
}

async function exerciseKeyboardShell(
  page: Page,
  info: TestInfo,
  mode: (typeof modes)[number],
  prepared?: {
    fixture: Awaited<ReturnType<typeof installPilotFixtures>>;
    browser: ReturnType<typeof observeBrowser>;
  },
) {
  const browser = prepared?.browser ?? observeBrowser(page);
  const fixture =
    prepared?.fixture ?? (await installPilotFixtures(page, "overview"));
  if (!prepared) await page.goto("/overview");
  await expect(
    page.getByRole("button", { name: "Open command palette" }),
  ).toBeVisible();
  const theme = await page.locator("html").getAttribute("data-theme");
  expect(theme).toBe(mode);
  expect(
    await page.evaluate(
      () => matchMedia("(prefers-color-scheme: dark)").matches,
    ),
  ).toBe(mode === "dark");

  const summary = page.locator('summary[aria-label^="Account menu for"]');
  await tabUntilFocused(page, summary);
  await page.keyboard.press("Enter");
  await expect
    .poll(() =>
      summary.evaluate(
        (element) => (element.parentElement as HTMLDetailsElement).open,
      ),
    )
    .toBe(true);
  const menuItems = page.getByRole("menuitem");
  await expect(menuItems).toHaveCount(3);
  const accountFocus = [];
  for (let index = 0; index < 3; index++) {
    await page.keyboard.press("Tab");
    const item = menuItems.nth(index);
    const focus = await assertMenuFocus(item);
    accountFocus.push({
      index,
      name: (await item.innerText()).trim(),
      ...focus,
    });
    if (index === 0) {
      await captureViewport(page, info, `account-focused-first-${mode}.png`, {
        selectedName: accountFocus[index].name,
        rect: accountFocus[index].rect,
        focusOutline: {
          style: accountFocus[index].outlineStyle,
          width: accountFocus[index].outlineWidth,
          color: accountFocus[index].outlineColor,
        },
      });
    }
  }
  expect(accountFocus.map((item) => item.name)).toEqual([
    "Account settings",
    "Admin",
    "Log out",
  ]);
  await page.keyboard.press("Escape");
  await expect
    .poll(() =>
      summary.evaluate(
        (element) => (element.parentElement as HTMLDetailsElement).open,
      ),
    )
    .toBe(false);
  await expect(summary).toBeFocused();

  const opener = page.getByRole("button", { name: "Open command palette" });
  await tabUntilFocused(page, opener);
  const beforeUrl = page.url();
  const beforeScroll = await page.evaluate(() => ({ x: scrollX, y: scrollY }));
  await page.keyboard.press("Enter");
  const listbox = page.getByRole("listbox", { name: "Navigation commands" });
  await expect(listbox).toBeFocused();
  await expect(page.getByRole("option")).toHaveCount(commandCount);
  const options = page.getByRole("option");
  const selectedNames: string[] = [];
  const verifySelected = async (index: number) => {
    const option = options.nth(index);
    await expect(option).toHaveAttribute("aria-selected", "true");
    await expect(listbox).toHaveAttribute(
      "aria-activedescendant",
      `command-${index}`,
    );
    await expect(listbox).toBeFocused();
    await expect
      .poll(() =>
        option.evaluate((element) => {
          const optionRect = element.getBoundingClientRect();
          const listRect = element.parentElement!.getBoundingClientRect();
          return (
            optionRect.top >= listRect.top - 1 &&
            optionRect.bottom <= listRect.bottom + 1
          );
        }),
      )
      .toBe(true);
    expect(await page.evaluate(() => ({ x: scrollX, y: scrollY }))).toEqual(
      beforeScroll,
    );
    expect(page.url()).toBe(beforeUrl);
    selectedNames.push((await option.innerText()).trim());
  };
  await verifySelected(0);
  for (let index = 1; index < commandCount; index++) {
    await page.keyboard.press("ArrowDown");
    await verifySelected(index);
  }
  const lowerSelected = options.nth(commandCount - 1);
  await captureViewport(page, info, `palette-selected-lower-${mode}.png`, {
    selectedName: (await lowerSelected.innerText()).trim(),
    rect: await lowerSelected.boundingBox(),
    activeDescendant: await listbox.getAttribute("aria-activedescendant"),
  });
  await page.keyboard.press("ArrowDown");
  await verifySelected(0);
  for (let index = commandCount - 1; index >= 1; index--) {
    await page.keyboard.press("ArrowUp");
    await verifySelected(index);
  }
  await page.keyboard.press("ArrowUp");
  await verifySelected(0);
  await page.keyboard.press("ArrowUp");
  await verifySelected(commandCount - 1);
  await page.keyboard.press("ArrowDown");
  await verifySelected(0);
  await page.keyboard.press("Escape");
  await expect(
    page.getByRole("dialog", { name: "Command palette" }),
  ).toHaveCount(0);
  await expect(opener).toBeFocused();
  await captureViewport(page, info, `command-opener-restored-${mode}.png`, {
    focusedOpener: await opener.getAttribute("aria-label"),
    rect: await opener.boundingBox(),
  });
  expect(page.url()).toBe(beforeUrl);
  expect(fixture.unknown).toEqual([]);
  expect(browser.apiRequests.length).toBeGreaterThan(0);
  expect(browser.apiRequests.every((request) => request.method === "GET")).toBe(
    true,
  );
  expect(browser.apiResponses.length).toBe(browser.apiRequests.length);
  expect(
    browser.apiResponses.every(
      (response) => response.status >= 200 && response.status < 300,
    ),
  ).toBe(true);
  expect(browser.apiFailures).toEqual([]);
  expect(browser.consoleErrors).toEqual([]);
  expect(browser.pageErrors).toEqual([]);
  await info.attach(`shared-shell-observations-${mode}.json`, {
    body: JSON.stringify(
      {
        mode,
        theme,
        accountFocus,
        selectedNames,
        apiRequests: browser.apiRequests,
        apiResponses: browser.apiResponses,
        apiFailures: browser.apiFailures,
        consoleErrors: browser.consoleErrors,
        pageErrors: browser.pageErrors,
        unknownFixtures: fixture.unknown,
      },
      null,
      2,
    ),
    contentType: "application/json",
  });
}

for (const mode of modes) {
  for (const reducedMotion of motions) {
    for (const width of compactWidths) {
      test(`shared shell keyboard behavior ${width}x844 ${mode} ${reducedMotion}`, async ({
        page,
      }, info) => {
        await page.setViewportSize({ width, height: 844 });
        await page.emulateMedia({ colorScheme: mode, reducedMotion });
        await exerciseKeyboardShell(page, info, mode);
      });
    }
  }

  test(`shared shell keyboard behavior native tab zoom 200 percent ${mode}`, async ({
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
        reducedMotion: "reduce",
        colorScheme: mode,
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
      const base = baseURL;
      if (!base) throw new Error("Playwright baseURL is not configured");
      const fixture = await installPilotFixtures(page, "overview");
      const browser = observeBrowser(page);
      await page.goto(`${base}/overview`);
      await expect(
        page.getByRole("button", { name: "Open command palette" }),
      ).toBeVisible();
      const before = await page.evaluate(() => ({
        width: innerWidth,
        height: innerHeight,
        dpr: devicePixelRatio,
      }));
      expect(before.width).toBe(640);
      const worker =
        context.serviceWorkers()[0] ??
        (await context.waitForEvent("serviceworker"));
      const zoom = await worker.evaluate(`new Promise((resolve, reject) => {
        chrome.tabs.query({}, tabs => {
          const tab = tabs.find(candidate => candidate.url === ${JSON.stringify(page.url())});
          if (!tab) return reject(new Error('Fixture tab not found'));
          chrome.tabs.setZoom(tab.id, 2, () => {
            if (chrome.runtime.lastError) return reject(new Error(chrome.runtime.lastError.message));
            chrome.tabs.getZoom(tab.id, resolve);
          });
        });
      })`);
      expect(zoom).toBe(2);
      await expect.poll(() => page.evaluate(() => innerWidth)).toBe(320);
      const afterZoom = await page.evaluate(() => ({
        width: innerWidth,
        height: innerHeight,
        dpr: devicePixelRatio,
        cssZoom: getComputedStyle(document.documentElement).zoom,
      }));
      expect(afterZoom.width).toBe(320);
      expect(afterZoom.dpr).toBe(before.dpr * 2);
      expect(afterZoom.cssZoom).toBe("1");
      await info.attach(`native-zoom-${mode}.json`, {
        body: JSON.stringify({ mode, zoom, before, afterZoom }, null, 2),
        contentType: "application/json",
      });
      await exerciseKeyboardShell(page, info, mode, { fixture, browser });
      expect(await page.evaluate(() => innerWidth)).toBe(320);
      expect(await page.evaluate(() => devicePixelRatio)).toBe(afterZoom.dpr);
      expect(fixture.unknown).toEqual([]);
      expect(browser.apiFailures).toEqual([]);
    } finally {
      await context.close();
    }
  });
}
