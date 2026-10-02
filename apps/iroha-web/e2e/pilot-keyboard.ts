import { expect, type Page } from "@playwright/test";

// Real Tab/Enter also works at native Chrome zoom, where pointer coordinates
// are not a reliable substitute for the keyboard journey under measurement.
export async function openNavigationByKeyboard(page: Page, name: string) {
  const menu = page
    .getByRole("navigation", { name: "Primary navigation" })
    .getByRole("group")
    .filter({ has: page.getByText(name, { exact: true }) });
  const label = menu.getByText(name, { exact: true });
  const focused = () =>
    label.evaluate(
      (element) => element.closest("summary") === document.activeElement,
    );
  for (let step = 0; step < 80; step++) {
    await page.keyboard.press("Tab");
    if (await focused()) break;
  }
  await expect.poll(focused).toBe(true);
  await page.keyboard.press("Enter");
  await expect(menu).toHaveAttribute("open", "");
  await expect
    .poll(() =>
      menu.evaluate((element) => {
        const box = element.querySelector("a")!.getBoundingClientRect();
        return (
          box.left >= 0 &&
          box.right <= innerWidth + 1 &&
          box.top >= 0 &&
          box.bottom <= innerHeight
        );
      }),
    )
    .toBe(true);
}
