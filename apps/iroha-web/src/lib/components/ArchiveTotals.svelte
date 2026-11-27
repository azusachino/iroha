<script lang="ts">
  let {
    activityCount,
    nightCount,
    mediaCount,
  }: {
    activityCount: number | null;
    nightCount: number | null;
    mediaCount: number | null;
  } = $props();

  const entries = $derived([
    { label: "Activity sessions", count: activityCount, href: "/motion" },
    { label: "Night sessions", count: nightCount, href: "/night" },
    { label: "Media items", count: mediaCount, href: "/library" },
  ]);

  const populated = $derived(
    entries.filter(
      (entry): entry is { label: string; count: number; href: string } =>
        entry.count != null && entry.count > 0,
    ),
  );

  const emptyEntries = $derived(
    entries.filter((entry) => entry.count == null || entry.count === 0),
  );

  function formatEmptyList(items: { label: string }[]): string {
    const names = items.map((item) => item.label.toLowerCase());
    if (names.length === 1) return names[0];
    if (names.length === 2) return `${names[0]} or ${names[1]}`;
    return `${names.slice(0, -1).join(", ")}, or ${names[names.length - 1]}`;
  }
</script>

<section class="archive-totals tile" aria-label="Across the archive">
  <div class="totals-heading">
    <div>
      <p class="eyebrow">Across the archive</p>
      <p class="totals-title">Every record has a place.</p>
    </div>
    <span>All imported records, not only movement.</span>
  </div>
  {#if populated.length > 0}
    <div class="totals-grid">
      {#each populated as entry (entry.href)}
        <a href={entry.href}>
          <strong>{entry.count.toLocaleString()}</strong>
          <span>{entry.label}</span>
        </a>
      {/each}
    </div>
  {/if}
  {#if emptyEntries.length > 0}
    <p class="totals-empty">
      No records yet for {formatEmptyList(emptyEntries)}.
    </p>
  {/if}
</section>

<style>
  .archive-totals {
    display: grid;
    gap: 1rem;
    padding: 1rem 1.1rem;
  }
  .totals-heading {
    display: flex;
    align-items: end;
    justify-content: space-between;
    gap: 1rem;
  }
  .eyebrow {
    margin: 0 0 0.3rem;
    color: var(--accent);
    font-size: 0.68rem;
    font-weight: 750;
    letter-spacing: 0.11em;
    text-transform: uppercase;
  }
  .totals-title {
    margin: 0;
    font-size: 1.05rem;
    font-weight: 700;
  }
  .totals-heading > span {
    color: var(--text-muted);
    font-size: 0.75rem;
  }
  .totals-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(min(100%, 12rem), 1fr));
    gap: 0.65rem;
  }
  .totals-empty {
    margin: 0;
    color: var(--text-muted);
    font-size: var(--type-caption);
  }
  a {
    display: grid;
    gap: 0.2rem;
    padding: 0.8rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    color: inherit;
    text-decoration: none;
  }
  a:hover,
  a:focus-visible {
    border-color: var(--accent);
  }
  a:focus-visible {
    outline: 2px solid var(--color-focus);
    outline-offset: 2px;
  }
  strong {
    color: var(--text);
    font-size: 1.35rem;
  }
  a span {
    color: var(--text-muted);
    font-size: 0.72rem;
  }
  @media (max-width: 640px) {
    .totals-heading {
      align-items: start;
      flex-direction: column;
    }
  }
</style>
