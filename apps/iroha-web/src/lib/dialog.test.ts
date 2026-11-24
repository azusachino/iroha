import { describe, expect, it, vi } from "vitest";
import { attachDialogLightDismiss, isPointerOutsideRect } from "./dialog";

describe("dialog light-dismiss fallback", () => {
  it("detects pointer coordinates outside a rect", () => {
    const rect = { left: 100, right: 300, top: 80, bottom: 240 };
    expect(isPointerOutsideRect(50, 150, rect)).toBe(true);
    expect(isPointerOutsideRect(350, 150, rect)).toBe(true);
    expect(isPointerOutsideRect(200, 40, rect)).toBe(true);
    expect(isPointerOutsideRect(200, 260, rect)).toBe(true);
    expect(isPointerOutsideRect(100, 80, rect)).toBe(false);
    expect(isPointerOutsideRect(200, 150, rect)).toBe(false);
  });

  it("closes only when both pointerdown and pointerup occur outside", () => {
    const listeners = new Map<string, (event: PointerEvent) => void>();
    const close = vi.fn();
    const fakeDialog = {
      open: true,
      close,
      getBoundingClientRect: () => ({
        left: 100,
        right: 300,
        top: 80,
        bottom: 240,
      }),
      addEventListener: (type: string, fn: (event: PointerEvent) => void) => {
        listeners.set(type, fn);
      },
      removeEventListener: (type: string) => {
        listeners.delete(type);
      },
    } as unknown as HTMLDialogElement;

    const detach = attachDialogLightDismiss(fakeDialog, false);

    // Drag starting inside and ending outside must NOT dismiss.
    listeners.get("pointerdown")?.({
      clientX: 150,
      clientY: 120,
    } as PointerEvent);
    listeners.get("pointerup")?.({ clientX: 20, clientY: 20 } as PointerEvent);
    expect(close).not.toHaveBeenCalled();

    // Press and release inside must NOT dismiss.
    listeners.get("pointerdown")?.({
      clientX: 150,
      clientY: 120,
    } as PointerEvent);
    listeners.get("pointerup")?.({
      clientX: 160,
      clientY: 130,
    } as PointerEvent);
    expect(close).not.toHaveBeenCalled();

    // Press and release outside dismisses an open dialog.
    listeners.get("pointerdown")?.({
      clientX: 10,
      clientY: 10,
    } as PointerEvent);
    listeners.get("pointerup")?.({ clientX: 12, clientY: 12 } as PointerEvent);
    expect(close).toHaveBeenCalledTimes(1);

    detach();
    expect(listeners.size).toBe(0);
  });

  it("is a no-op when native closedBy is supported", () => {
    const addEventListener = vi.fn();
    const fakeDialog = { addEventListener } as unknown as HTMLDialogElement;
    const detach = attachDialogLightDismiss(fakeDialog, true);
    expect(addEventListener).not.toHaveBeenCalled();
    detach();
  });
});
