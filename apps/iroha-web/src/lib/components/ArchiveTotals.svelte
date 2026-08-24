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

  function display(value: number | null): string {
    return value == null ? "—" : value.toLocaleString();
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
  <div class="totals-grid">
    <a href="/motion">
      <strong>{display(activityCount)}</strong>
      <span>Activity sessions</span>
    </a>
    <a href="/night">
      <strong>{display(nightCount)}</strong>
      <span>Night sessions</span>
    </a>
    <a href="/library">
      <strong>{display(mediaCount)}</strong>
      <span>Media items</span>
    </a>
  </div>
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
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 0.65rem;
  }
  a {
    display: grid;
    gap: 0.2rem;
    padding: 0.8rem;
    border: 1px solid var(--border);
    border-radius: 9px;
    color: inherit;
    text-decoration: none;
  }
  a:hover,
  a:focus-visible {
    border-color: var(--accent);
    outline: none;
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
