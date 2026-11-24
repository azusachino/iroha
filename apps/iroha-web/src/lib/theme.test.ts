import { afterEach, describe, expect, it, vi } from "vitest";
import {
  applyTheme,
  cycleTheme,
  getEffectiveTheme,
  getThemePreference,
  initThemeWatcher,
  themeState,
} from "./theme";

describe("theme state and meta synchronization", () => {
  const originalDocument = globalThis.document;
  const originalWindow = globalThis.window;
  const originalLocalStorage = globalThis.localStorage;

  afterEach(() => {
    Object.defineProperty(globalThis, "document", {
      value: originalDocument,
      configurable: true,
    });
    Object.defineProperty(globalThis, "window", {
      value: originalWindow,
      configurable: true,
    });
    Object.defineProperty(globalThis, "localStorage", {
      value: originalLocalStorage,
      configurable: true,
    });
    vi.restoreAllMocks();
  });

  function installFakeDom(systemLight = false) {
    const storage = new Map<string, string>();
    const colorSchemeMeta = { content: "light dark" };
    const themeColorMeta = { content: "#0b0f19" };
    const favicon = { href: "/favicon.svg?v=3" };
    const dataset: Record<string, string> = {};
    const windowListeners = new Map<string, ((event: Event) => void)[]>();
    const mediaListeners: (() => void)[] = [];
    let isLight = systemLight;

    Object.defineProperty(globalThis, "localStorage", {
      value: {
        getItem: (key: string) => storage.get(key) ?? null,
        setItem: (key: string, value: string) => storage.set(key, value),
        removeItem: (key: string) => storage.delete(key),
      },
      configurable: true,
    });

    Object.defineProperty(globalThis, "document", {
      value: {
        documentElement: { dataset },
        querySelector: (selector: string) => {
          if (selector === 'meta[name="color-scheme"]') return colorSchemeMeta;
          if (selector === 'meta[name="theme-color"]') return themeColorMeta;
          if (selector === "#theme-favicon") return favicon;
          return null;
        },
      },
      configurable: true,
    });

    Object.defineProperty(globalThis, "window", {
      value: {
        matchMedia: () => ({
          get matches() {
            return isLight;
          },
          addEventListener: (_: string, fn: () => void) =>
            mediaListeners.push(fn),
          removeEventListener: (_: string, fn: () => void) => {
            const index = mediaListeners.indexOf(fn);
            if (index >= 0) mediaListeners.splice(index, 1);
          },
        }),
        addEventListener: (type: string, fn: (event: Event) => void) => {
          const list = windowListeners.get(type) ?? [];
          list.push(fn);
          windowListeners.set(type, list);
        },
        removeEventListener: (type: string, fn: (event: Event) => void) => {
          const list = windowListeners.get(type) ?? [];
          const index = list.indexOf(fn);
          if (index >= 0) list.splice(index, 1);
        },
      },
      configurable: true,
    });

    return {
      storage,
      colorSchemeMeta,
      themeColorMeta,
      favicon,
      dataset,
      setSystemLight: (next: boolean) => {
        isLight = next;
        for (const fn of mediaListeners) fn();
      },
      dispatchStorage: (key: string | null) => {
        for (const fn of windowListeners.get("storage") ?? []) {
          fn({ key } as StorageEvent);
        }
      },
    };
  }

  it("synchronizes color-scheme meta, theme-color meta, dataset, and reactive state", () => {
    const dom = installFakeDom(false);

    expect(applyTheme("light")).toBe("light");
    expect(dom.colorSchemeMeta.content).toBe("light");
    expect(dom.themeColorMeta.content).toBe("#f7f6f2");
    expect(dom.dataset.theme).toBe("light");
    expect(dom.dataset.themePreference).toBe("light");
    expect(themeState.preference).toBe("light");
    expect(themeState.effective).toBe("light");
    expect(dom.storage.get("iroha-theme")).toBe("light");

    expect(cycleTheme()).toBe("dark");
    expect(dom.colorSchemeMeta.content).toBe("dark");
    expect(dom.themeColorMeta.content).toBe("#0b0f19");
    expect(themeState.preference).toBe("dark");
    expect(themeState.effective).toBe("dark");

    expect(cycleTheme()).toBe("auto");
    expect(dom.colorSchemeMeta.content).toBe("light dark");
    expect(dom.themeColorMeta.content).toBe("#0b0f19");
    expect(dom.storage.has("iroha-theme")).toBe(false);
    expect(getThemePreference()).toBe("auto");
    expect(getEffectiveTheme()).toBe("dark");
  });

  it("syncs across tabs via storage events and keeps explicit overrides on OS changes", () => {
    const dom = installFakeDom(false);
    const stop = initThemeWatcher();

    // System preference flip updates auto mode.
    dom.setSystemLight(true);
    expect(themeState.effective).toBe("light");
    expect(dom.themeColorMeta.content).toBe("#f7f6f2");
    expect(dom.colorSchemeMeta.content).toBe("light dark");

    // Cross-tab storage write switches to pinned dark without mutating storage.
    dom.storage.set("iroha-theme", "dark");
    dom.dispatchStorage("iroha-theme");
    expect(themeState.preference).toBe("dark");
    expect(themeState.effective).toBe("dark");
    expect(dom.colorSchemeMeta.content).toBe("dark");

    // OS switching to dark and back to light does not clear the pinned override.
    dom.setSystemLight(false);
    dom.setSystemLight(true);
    expect(dom.storage.get("iroha-theme")).toBe("dark");
    expect(themeState.preference).toBe("dark");

    stop();
  });
});
