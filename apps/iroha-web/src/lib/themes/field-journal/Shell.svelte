<script lang="ts">
  import type { Snippet } from "svelte";
  import { page } from "$app/state";

  let { children }: { children: Snippet } = $props();
  const links = [
    ["/", "Today"],
    ["/dashboard", "Overview"],
    ["/daily", "Patterns"],
    ["/activities", "Motion"],
    ["/sleep", "Night"],
    ["/media", "Library"],
  ];
</script>

<div class="field-journal-site">
  <header class="field-journal-masthead">
    <a class="field-journal-brand" href="/" aria-label="iroha field journal">
      <span class="field-journal-mark" aria-hidden="true">✽</span>
      <span>
        <strong>iroha</strong>
        <small>field journal</small>
      </span>
    </a>
    <nav aria-label="Field journal navigation">
      {#each links as [href, label]}
        <a
          class:active={page.url.pathname === href ||
            (href !== "/" && page.url.pathname.startsWith(href))}
          {href}>{label}</a
        >
      {/each}
    </nav>
    <span class="field-journal-status"
      >PRIVATE RECORD · {new Date().getFullYear()}</span
    >
  </header>

  <main class="field-journal-content">{@render children()}</main>

  <footer class="field-journal-footer">
    <span>iroha · sound + flower</span>
    <span>Notes remain traceable to imported evidence.</span>
  </footer>
</div>

<style>
  .field-journal-site {
    min-height: 100vh;
    background:
      radial-gradient(
        circle at 10% 0%,
        color-mix(in srgb, var(--accent) 12%, transparent),
        transparent 28rem
      ),
      var(--bg);
    color: var(--text);
  }

  .field-journal-masthead {
    position: sticky;
    top: 0;
    z-index: 20;
    display: grid;
    grid-template-columns: minmax(13rem, 1fr) auto minmax(13rem, 1fr);
    align-items: center;
    gap: 1.5rem;
    min-height: 4.75rem;
    padding: 0.8rem 2rem;
    border-bottom: 1px solid
      color-mix(in srgb, var(--accent) 35%, var(--border));
    background: color-mix(in srgb, var(--bg) 92%, transparent);
    backdrop-filter: blur(14px);
  }

  .field-journal-brand {
    display: inline-flex;
    align-items: center;
    gap: 0.7rem;
    color: var(--text);
    text-decoration: none;
  }

  .field-journal-mark {
    display: grid;
    width: 2.35rem;
    height: 2.35rem;
    place-items: center;
    border: 1px solid var(--accent);
    border-radius: 50% 50% 50% 0;
    color: var(--accent);
    font-family: Georgia, serif;
    font-size: 1.35rem;
    transform: rotate(-12deg);
  }

  .field-journal-brand strong,
  .field-journal-brand small {
    display: block;
  }

  .field-journal-brand strong {
    font-family: Georgia, serif;
    font-size: 1.15rem;
    letter-spacing: -0.04em;
  }

  .field-journal-brand small,
  .field-journal-status {
    color: var(--text-muted);
    font-size: 0.6rem;
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }

  .field-journal-masthead nav {
    display: flex;
    gap: 0.2rem;
  }

  .field-journal-masthead nav a {
    padding: 0.45rem 0.6rem;
    color: var(--text-muted);
    font-family: Georgia, serif;
    font-size: 0.82rem;
    text-decoration: none;
  }

  .field-journal-masthead nav a:hover,
  .field-journal-masthead nav a.active {
    color: var(--accent);
  }

  .field-journal-masthead nav a.active {
    text-decoration: underline;
    text-underline-offset: 0.35rem;
  }

  .field-journal-status {
    justify-self: end;
  }

  .field-journal-content {
    width: min(1120px, calc(100% - 3rem));
    margin: 0 auto;
    padding: 3rem 0 5rem;
  }

  .field-journal-footer {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    padding: 1rem 2rem;
    border-top: 1px solid var(--border);
    color: var(--text-muted);
    font-family: Georgia, serif;
    font-size: 0.75rem;
  }

  @media (max-width: 900px) {
    .field-journal-masthead {
      grid-template-columns: 1fr auto;
    }

    .field-journal-masthead nav {
      grid-column: 1 / -1;
      grid-row: 2;
      overflow-x: auto;
    }

    .field-journal-status {
      grid-column: 2;
      grid-row: 1;
    }
  }

  @media (max-width: 520px) {
    .field-journal-masthead {
      padding: 0.75rem 1rem;
    }

    .field-journal-content {
      width: min(100% - 2rem, 1120px);
      padding-top: 2rem;
    }

    .field-journal-footer {
      flex-direction: column;
      padding: 1rem;
    }
  }
</style>
