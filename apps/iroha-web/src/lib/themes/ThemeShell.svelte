<script lang="ts">
  import type { Snippet } from "svelte";

  export type ShellVariant = "atlas" | "phenology" | "sound-map" | "archive";

  let { variant, children }: { variant: ShellVariant; children: Snippet } =
    $props();

  const content = {
    atlas: {
      label: "Iroha Atlas",
      note: "places, routes, distance",
      mark: "N",
    },
    phenology: {
      label: "Iroha Phenology",
      note: "sleep, seasons, recovery",
      mark: "◒",
    },
    "sound-map": {
      label: "Iroha Sound Map",
      note: "rhythm, cadence, intensity",
      mark: "≈",
    },
    archive: {
      label: "Iroha Archive",
      note: "media, history, provenance",
      mark: "№",
    },
  } as const;
  const copy = $derived(content[variant]);
</script>

<div class={`theme-preview theme-preview-${variant}`}>
  <header class="theme-preview-heading">
    <div class="theme-preview-brand">
      <span class="theme-preview-mark" aria-hidden="true">{copy.mark}</span>
      <div>
        <p>iroha / design language</p>
        <strong>{copy.label}</strong>
      </div>
    </div>
    <span class="theme-preview-note">{copy.note}</span>
  </header>
  <div class="theme-preview-body">{@render children()}</div>
  <footer class="theme-preview-footer">
    <span>private record</span>
    <span>visual language preview</span>
  </footer>
</div>

<style>
  .theme-preview {
    min-height: 100vh;
    color: var(--text);
  }

  .theme-preview-heading {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 2rem;
    width: min(1200px, calc(100% - 3rem));
    margin: 0 auto;
    padding: 2rem 0 1.25rem;
    border-bottom: 1px solid var(--border);
  }

  .theme-preview-brand {
    display: flex;
    align-items: center;
    gap: 0.8rem;
  }

  .theme-preview-mark {
    display: grid;
    width: 2.75rem;
    height: 2.75rem;
    place-items: center;
    border: 1px solid currentColor;
    font-family: Georgia, serif;
    font-size: 1.35rem;
  }

  .theme-preview-brand p,
  .theme-preview-brand strong {
    display: block;
    margin: 0;
  }

  .theme-preview-brand p,
  .theme-preview-note,
  .theme-preview-footer {
    color: var(--text-muted);
    font-size: 0.65rem;
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }

  .theme-preview-brand strong {
    margin-top: 0.25rem;
    font-size: 1rem;
  }

  .theme-preview-body {
    width: min(1200px, calc(100% - 3rem));
    margin: 0 auto;
    padding: 2.5rem 0 4rem;
  }

  .theme-preview-footer {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    padding: 1rem max(1rem, calc((100% - 1200px) / 2));
    border-top: 1px solid var(--border);
  }

  .theme-preview-atlas {
    --accent: #287d94;
    --bg: #eef4f1;
    --surface-1: #f8fbf8;
    --text: #18343a;
    --text-muted: #5f787c;
    background-color: var(--bg);
    background-image:
      linear-gradient(
        color-mix(in srgb, var(--accent) 10%, transparent) 1px,
        transparent 1px
      ),
      linear-gradient(
        90deg,
        color-mix(in srgb, var(--accent) 10%, transparent) 1px,
        transparent 1px
      );
    background-size: 2rem 2rem;
    font-family: "Avenir Next", sans-serif;
  }

  .theme-preview-atlas .theme-preview-mark {
    border-radius: 50%;
  }

  .theme-preview-phenology {
    --accent: #9c5c43;
    --bg: #f4eee5;
    --surface-1: #fffaf1;
    --text: #3f3029;
    --text-muted: #866f61;
    background:
      radial-gradient(circle at 80% 8%, #e4c7a5 0 0.25rem, transparent 0.3rem),
      var(--bg);
    font-family: "Iowan Old Style", Georgia, serif;
  }

  .theme-preview-phenology .theme-preview-mark {
    border-radius: 50% 50% 0 50%;
    transform: rotate(-22deg);
  }

  .theme-preview-sound-map {
    --accent: #70e6c0;
    --bg: #0c1519;
    --surface-1: #112328;
    --text: #e2fff4;
    --text-muted: #7da59b;
    background: var(--bg);
    font-family: "IBM Plex Mono", "SFMono-Regular", monospace;
  }

  .theme-preview-sound-map .theme-preview-heading,
  .theme-preview-sound-map .theme-preview-footer {
    border-color: color-mix(in srgb, var(--accent) 30%, transparent);
  }

  .theme-preview-sound-map .theme-preview-mark {
    border-width: 0 0 2px;
    color: var(--accent);
  }

  .theme-preview-archive {
    --accent: #9b4e32;
    --bg: #e8dfca;
    --surface-1: #f4ecd9;
    --text: #33271f;
    --text-muted: #796656;
    background:
      repeating-linear-gradient(
        0deg,
        color-mix(in srgb, var(--accent) 4%, transparent) 0 1px,
        transparent 1px 4px
      ),
      var(--bg);
    font-family: Georgia, "Times New Roman", serif;
  }

  .theme-preview-archive .theme-preview-heading {
    border-bottom: 3px double var(--accent);
  }

  .theme-preview-archive .theme-preview-mark {
    border-style: double;
  }

  @media (max-width: 640px) {
    .theme-preview-heading {
      display: block;
      width: min(100% - 2rem, 1200px);
    }

    .theme-preview-note {
      display: block;
      margin-top: 1rem;
    }

    .theme-preview-body {
      width: min(100% - 2rem, 1200px);
      padding-top: 2rem;
    }

    .theme-preview-footer {
      flex-direction: column;
    }
  }
</style>
