import { expect, test } from "@playwright/test";
import { PUBLIC_BASE_URL } from "../playwright.config";
import { installPilotFixtures } from "./pilot-fixtures";
import { openNavigationByKeyboard } from "./pilot-keyboard";

for (const pilot of ["overview", "expenses", "metrics", "public"] as const) {
  for (const mode of ["light", "dark"] as const) {
    for (const width of [320, 768, 1280]) {
      for (const navigation of pilot === "public"
        ? [null]
        : ["Domains", "Analyze", "More"]) {
        test(`${pilot} ${mode} ${width} chrome wraps and keeps ${navigation ?? "public"} navigation keyboard reachable`, async ({
          page,
        }, info) => {
          await page.setViewportSize({ width, height: 900 });
          await page.emulateMedia({
            colorScheme: mode,
            reducedMotion: "reduce",
          });
          const fixture = await installPilotFixtures(page, pilot);
          await page.goto(
            pilot === "public"
              ? PUBLIC_BASE_URL
              : `/${pilot}${pilot === "expenses" ? "?date=2026-08&currency=JPY" : ""}`,
          );
          await expect(page.getByRole("img").first()).toBeVisible();
          const title =
            pilot === "overview"
              ? "Overview"
              : pilot === "expenses"
                ? "Expenses"
                : pilot === "metrics"
                  ? "Metrics"
                  : "harus track";
          const header = page.getByRole("region", {
            name: `${title} header`,
            exact: true,
          });
          await expect(header).toBeVisible();
          await expect(page.getByRole("heading", { level: 1 })).toHaveCount(1);
          if (width === 1280)
            expect((await header.boundingBox())!.height).toBeLessThanOrEqual(
              128,
            );
          await info.attach("chrome.png", {
            body: await page.screenshot({ fullPage: true }),
            contentType: "image/png",
          });
          // Stress the actual mounted header CSS without adding a production-only test route.
          if (width === 320) {
            const heading = header.getByRole("heading", { level: 1 });
            await heading.evaluate((element) => {
              element.textContent =
                "A long route title with meaningful words that must remain readable ".repeat(
                  4,
                );
            });
            const geometry = await header.evaluate((element) => {
              const title = element
                .querySelector("h1")!
                .getBoundingClientRect();
              const context = element
                .querySelector("p")!
                .getBoundingClientRect();
              return {
                titleBottom: title.bottom,
                contextTop: context.top,
                left: title.left,
                right: title.right,
                width: innerWidth,
              };
            });
            expect(geometry.contextTop).toBeGreaterThanOrEqual(
              geometry.titleBottom,
            );
            expect(geometry.left).toBeGreaterThanOrEqual(0);
            expect(geometry.right).toBeLessThanOrEqual(geometry.width);
          }
          if (navigation) await openNavigationByKeyboard(page, navigation);
          const targets = await page.evaluate(() => {
            const elements = Array.from(
              document.querySelectorAll<HTMLElement>(
                "a[href],button,input,select,textarea,summary,[tabindex]",
              ),
            ).filter(
              (element) =>
                element.tabIndex >= 0 &&
                !element.matches(":disabled") &&
                !element.closest("[inert],[aria-hidden='true']") &&
                element.getClientRects().length &&
                getComputedStyle(element).visibility !== "hidden" &&
                (!element.closest("details:not([open])") ||
                  element
                    .closest("details:not([open])")
                    ?.querySelector("summary") === element),
            );
            elements.forEach((element, index) => {
              element.dataset.chromeFocus = String(index);
            });
            (document.activeElement as HTMLElement)?.blur();
            return elements.map((_, index) => String(index));
          });
          const visited = new Set<string>();
          for (let index = 0; index < targets.length * 2 + 4; index++) {
            await page.keyboard.press("Tab");
            const stop = await page.evaluate(() => {
              const element = document.activeElement as HTMLElement;
              const box = element.getBoundingClientRect();
              return {
                id: element.dataset.chromeFocus,
                name: element.textContent?.trim(),
                left: box.left,
                right: box.right,
                top: box.top,
                bottom: box.bottom,
                width: innerWidth,
                height: innerHeight,
              };
            });
            if (stop.id === undefined) continue;
            visited.add(stop.id);
            if (stop.left < -1) {
              await info.attach("offscreen-navigation.json", {
                body: JSON.stringify(
                  await page.evaluate(() => {
                    const active = document.activeElement as HTMLElement;
                    const menu = active.closest("details");
                    const popover = active.closest(
                      ".navigation-popover",
                    ) as HTMLElement | null;
                    return {
                      active: active.outerHTML,
                      open: menu?.open,
                      style: popover?.getAttribute("style"),
                      display: popover
                        ? getComputedStyle(popover).display
                        : null,
                      menu: menu?.outerHTML,
                    };
                  }),
                ),
                contentType: "application/json",
              });
            }
            expect(stop.left, stop.name).toBeGreaterThanOrEqual(-1);
            expect(stop.right, stop.name).toBeLessThanOrEqual(stop.width + 1);
            expect(stop.top, stop.name).toBeLessThan(stop.height);
            expect(stop.bottom, stop.name).toBeGreaterThan(0);
            if (visited.size === targets.length) break;
          }
          expect([...visited].sort()).toEqual(targets.sort());
          expect(
            await page.evaluate(() => document.documentElement.scrollWidth),
          ).toBeLessThanOrEqual(width + 1);
          expect(fixture.unknown).toEqual([]);
        });
      }
    }
  }
}

test("Floating navigation does not hide a keyboard-focused link when the pointer leaves", async ({
  page,
}) => {
  await page.setViewportSize({ width: 320, height: 900 });
  await page.clock.install();
  await installPilotFixtures(page, "overview");
  await page.goto("/overview");
  await expect(page.getByRole("img").first()).toBeVisible();
  await openNavigationByKeyboard(page, "More");
  await page.keyboard.press("Tab");
  const link = page.getByRole("link", { name: /To-go/ });
  await expect(link).toBeFocused();
  const box = (await link.boundingBox())!;
  await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2);
  await page.mouse.move(319, 899);
  await page.clock.runFor(200);
  await expect(
    page
      .getByRole("navigation", { name: "Primary navigation" })
      .getByRole("group")
      .filter({ has: page.getByText("More", { exact: true }) }),
  ).toHaveAttribute("open", "");
  await expect(link).toBeFocused();
});
