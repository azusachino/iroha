<script lang="ts">
  import type { ShellThemeProps } from "../../shell-view";
  let { children, theme }: ShellThemeProps = $props();
</script>

<div class="mix-site" data-theme={theme}>
  <div class="mix-content">{@render children()}</div>
  <footer class="mix-footer">
    <span class="mix-meter" aria-hidden="true">
      <i></i><i></i><i></i><i></i><i></i><i></i><i></i><i></i>
    </span>
    <span>iroha sound-map · rhythm, cadence, intensity</span>
    <span>every level traces back to an imported source</span>
  </footer>
</div>

<style>
  .mix-site {
    position: relative;
    min-height: 100vh;
    background:
      repeating-linear-gradient(
        0deg,
        color-mix(in srgb, var(--accent) 5%, transparent) 0 1px,
        transparent 1px 2.5rem
      ),
      radial-gradient(
        circle at 88% 2%,
        color-mix(in srgb, var(--accent) 12%, transparent),
        transparent 26rem
      ),
      radial-gradient(
        circle at 4% 96%,
        color-mix(in srgb, var(--accent-2) 10%, transparent),
        transparent 24rem
      ),
      var(--bg);
    color: var(--text);
    font-family: var(--font-mono);
  }

  .mix-content {
    width: min(1160px, calc(100% - 3rem));
    margin: 0 auto;
    padding: 3rem 0 4rem;
  }

  .mix-footer {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
    border-top: 1px solid var(--border);
    padding: 1rem max(1rem, calc((100% - 1160px) / 2));
    background: color-mix(in srgb, var(--bg) 88%, transparent);
    color: var(--text-muted);
    font-size: 0.65rem;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }

  .mix-meter {
    display: inline-flex;
    align-items: flex-end;
    gap: 0.15rem;
    height: 0.9rem;
  }

  .mix-meter i {
    display: block;
    width: 0.16rem;
    background: var(--accent);
    opacity: 0.8;
  }

  .mix-meter i:nth-child(1) {
    height: 30%;
  }
  .mix-meter i:nth-child(2) {
    height: 55%;
  }
  .mix-meter i:nth-child(3) {
    height: 80%;
  }
  .mix-meter i:nth-child(4) {
    height: 100%;
    background: var(--accent-2);
  }
  .mix-meter i:nth-child(5) {
    height: 65%;
  }
  .mix-meter i:nth-child(6) {
    height: 90%;
    background: var(--accent-2);
  }
  .mix-meter i:nth-child(7) {
    height: 45%;
  }
  .mix-meter i:nth-child(8) {
    height: 22%;
  }

  @media (max-width: 640px) {
    .mix-content {
      width: min(100% - 2rem, 1160px);
      padding-top: 2rem;
    }

    .mix-footer {
      flex-direction: column;
      align-items: flex-start;
      gap: 0.4rem;
      padding: 1rem;
    }
  }
</style>
