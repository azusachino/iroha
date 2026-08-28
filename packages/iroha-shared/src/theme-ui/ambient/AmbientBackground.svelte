<script lang="ts">
  // Canvas lifecycle only -- the actual Three.js scene lives behind a
  // dynamic import (see the $effect below) so this component, mounted on
  // every route from the root layout, never puts the "three" chunk on the
  // critical path. Per-theme scenes live in ./factories; a theme with no
  // entry there (Grapher, and any language not yet built) renders nothing.
  import { onMount } from "svelte";
  import { useTheme } from "../context.svelte";
  import { readAmbientColors, type AmbientRenderer } from "./renderer";

  const theme = useTheme();

  let canvasEl: HTMLCanvasElement;
  let renderer: AmbientRenderer | undefined;
  let frame: number | undefined;

  function prefersReducedMotion() {
    return window.matchMedia("(prefers-reduced-motion: reduce)").matches;
  }

  function stopLoop() {
    if (frame === undefined) return;
    cancelAnimationFrame(frame);
    frame = undefined;
  }

  function loop() {
    renderer?.render();
    frame = requestAnimationFrame(loop);
  }

  // Reduced motion is a state contract, not a slower loop: a preferring
  // client gets exactly one rendered frame and requestAnimationFrame is
  // never scheduled at all.
  function startLoop() {
    stopLoop();
    if (prefersReducedMotion() || document.hidden || !renderer) return;
    frame = requestAnimationFrame(loop);
  }

  function teardownRenderer() {
    stopLoop();
    renderer?.dispose();
    renderer = undefined;
  }

  onMount(() => {
    const resizeObserver = new ResizeObserver(() => {
      renderer?.resize(window.innerWidth, window.innerHeight);
      if (prefersReducedMotion()) renderer?.render();
    });
    resizeObserver.observe(document.documentElement);

    function onVisibilityChange() {
      if (document.hidden) stopLoop();
      else startLoop();
    }
    document.addEventListener("visibilitychange", onVisibilityChange);

    return () => {
      document.removeEventListener("visibilitychange", onVisibilityChange);
      resizeObserver.disconnect();
      teardownRenderer();
    };
  });

  $effect(() => {
    const language = theme.language();
    teardownRenderer();
    let cancelled = false;
    import("./factories").then(({ AMBIENT_FACTORIES }) => {
      if (cancelled || !canvasEl) return;
      const factory = AMBIENT_FACTORIES[language];
      if (!factory) return;
      renderer = factory(canvasEl, () => readAmbientColors(canvasEl));
      renderer.resize(window.innerWidth, window.innerHeight);
      if (prefersReducedMotion()) renderer.render();
      else startLoop();
    });
    return () => {
      cancelled = true;
    };
  });
</script>

<canvas bind:this={canvasEl} class="ambient-canvas" aria-hidden="true"></canvas>

<style>
  .ambient-canvas {
    position: fixed;
    inset: 0;
    z-index: -1;
    width: 100vw;
    height: 100vh;
    pointer-events: none;
  }

  @media (prefers-reduced-transparency: reduce) {
    .ambient-canvas {
      display: none;
    }
  }
</style>
