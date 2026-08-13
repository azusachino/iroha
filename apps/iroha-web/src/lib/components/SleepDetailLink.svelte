<script lang="ts">
  let { id, label = "Open" }: { id: string; label?: string } = $props();
</script>

<a class="sleep-detail-link" href={`/night/${id}`}>{label}</a>

<style>
  .sleep-detail-link {
    color: var(--accent);
    font-size: 0.72rem;
    font-weight: 700;
    white-space: nowrap;
  }

  .sleep-detail-link:hover,
  .sleep-detail-link:focus-visible {
    color: var(--text);
  }
</style>
