<script lang="ts">
  import { untrack } from "svelte";
  import { tweened } from "svelte/motion";

  // Renders bare text (no wrapping element) so a caller can drop it into an
  // existing `<strong class="large-value">`/`<small>` layout unchanged.
  let {
    value,
    digits = 0,
  }: {
    value: number | null | undefined;
    digits?: number;
  } = $props();

  function isFiniteNumber(input: unknown): input is number {
    return typeof input === "number" && Number.isFinite(input);
  }

  const reducedMotion =
    typeof window !== "undefined" &&
    window.matchMedia("(prefers-reduced-motion: reduce)").matches;

  // Mirrors --motion-data-update (themes.css); tweened's duration is a JS
  // number, not the CSS custom property directly. Both reads below are
  // deliberately one-time seeds -- every later update flows through the
  // $effect below, not through re-reading `value` here.
  const tween = tweened(
    untrack(() => (isFiniteNumber(value) ? value : 0)),
    { duration: reducedMotion ? 0 : 600 },
  );

  let previous: number | null | undefined = untrack(() => value);

  $effect(() => {
    // A null/non-finite endpoint on either side is "no data", not zero --
    // jump instead of tweening through a number that was never observed.
    if (isFiniteNumber(value)) {
      if (isFiniteNumber(previous)) {
        void tween.set(value);
      } else {
        void tween.set(value, { duration: 0 });
      }
    }
    previous = value;
  });

  const display = $derived(
    isFiniteNumber(value)
      ? $tween.toLocaleString(undefined, {
          minimumFractionDigits: digits,
          maximumFractionDigits: digits,
        })
      : "—",
  );
</script>

{display}
