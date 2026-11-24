/**
 * Progressive-enhancement light-dismiss fallback for browsers without native
 * `<dialog closedby="any">` support (e.g. Safari).
 *
 * Tests pointer coordinates against the dialog's bounding box on both
 * pointerdown and pointerup so clicks on dialog padding or text-selection
 * drags ending on the backdrop do not dismiss the dialog.
 */
export function supportsNativeClosedBy(): boolean {
  return (
    typeof HTMLDialogElement !== "undefined" &&
    "closedBy" in HTMLDialogElement.prototype
  );
}

export function isPointerOutsideRect(
  clientX: number,
  clientY: number,
  rect: Pick<DOMRect, "left" | "right" | "top" | "bottom">,
): boolean {
  return (
    clientX < rect.left ||
    clientX > rect.right ||
    clientY < rect.top ||
    clientY > rect.bottom
  );
}

export function attachDialogLightDismiss(
  dialog: HTMLDialogElement,
  nativeSupported: boolean = supportsNativeClosedBy(),
): () => void {
  if (nativeSupported) {
    return () => {};
  }

  let pressedOutside = false;

  const onPointerDown = (event: PointerEvent) => {
    pressedOutside = isPointerOutsideRect(
      event.clientX,
      event.clientY,
      dialog.getBoundingClientRect(),
    );
  };

  const onPointerUp = (event: PointerEvent) => {
    const releasedOutside = isPointerOutsideRect(
      event.clientX,
      event.clientY,
      dialog.getBoundingClientRect(),
    );
    if (pressedOutside && releasedOutside && dialog.open) {
      dialog.close();
    }
    pressedOutside = false;
  };

  dialog.addEventListener("pointerdown", onPointerDown);
  dialog.addEventListener("pointerup", onPointerUp);

  return () => {
    dialog.removeEventListener("pointerdown", onPointerDown);
    dialog.removeEventListener("pointerup", onPointerUp);
  };
}

export function dialogLightDismiss(dialog: HTMLDialogElement): {
  destroy: () => void;
} {
  const destroy = attachDialogLightDismiss(dialog);
  return { destroy };
}
