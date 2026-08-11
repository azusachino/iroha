<script lang="ts">
  import { onMount } from "svelte";

  type Theme = "light" | "dark";

  const STORAGE_KEY = "iroha-public-theme";
  let theme = $state<Theme>("dark");

  function applyTheme(next: Theme) {
    theme = next;
    document.documentElement.dataset.theme = next;
    localStorage.setItem(STORAGE_KEY, next);
    document
      .querySelector('meta[name="theme-color"]')
      ?.setAttribute("content", next === "dark" ? "#0f1115" : "#f5f7fa");
  }

  onMount(() => {
    const stored = localStorage.getItem(STORAGE_KEY);
    const preferred = window.matchMedia("(prefers-color-scheme: light)").matches
      ? "light"
      : "dark";
    applyTheme(stored === "light" || stored === "dark" ? stored : preferred);
  });
</script>

<div class="theme-toggle" role="group" aria-label="Color theme">
  <button
    type="button"
    class:active={theme === "light"}
    aria-pressed={theme === "light"}
    onclick={() => applyTheme("light")}
  >
    Light
  </button>
  <button
    type="button"
    class:active={theme === "dark"}
    aria-pressed={theme === "dark"}
    onclick={() => applyTheme("dark")}
  >
    Dark
  </button>
</div>

<style>
  .theme-toggle {
    display: inline-flex;
    gap: 0.2rem;
    padding: 0.2rem;
    border: 1px solid var(--border);
    border-radius: 999px;
    background: var(--surface-2);
  }

  button {
    border: 0;
    border-radius: 999px;
    padding: 0.35rem 0.65rem;
    background: transparent;
    color: var(--text-muted);
    cursor: pointer;
    font-size: 0.75rem;
    font-weight: 700;
  }

  button.active {
    background: var(--surface);
    color: var(--text);
    box-shadow: 0 1px 3px rgb(0 0 0 / 0.14);
  }
</style>
