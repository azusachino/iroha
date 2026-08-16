<script lang="ts">
  import type { MediaChange } from "../../domain/media";
  import { formatDateOnly } from "../../format/format";

  let {
    updates,
    onOpenMedia,
  }: {
    updates: MediaChange[];
    onOpenMedia?: (id: string) => void;
  } = $props();

  function basisLabel(update: MediaChange): string {
    if (update.time_basis === "provider_activity") return "Provider activity";
    if (update.time_basis === "source_date") return "Source date";
    return update.time_basis.replaceAll("_", " ");
  }

  function updateLabel(update: MediaChange): string {
    const kind =
      update.change_kind === "provider_activity"
        ? "Activity recorded"
        : update.progress_percent != null || update.position != null
          ? "Reading progress"
          : update.status
            ? `Status · ${update.status.replaceAll("_", " ")}`
            : "Library updated";
    const position =
      update.position != null
        ? `${update.position}${update.total != null ? ` / ${update.total}` : ""}${update.unit ? ` ${update.unit}` : ""}`
        : "";
    return [kind, position, update.note].filter(Boolean).join(" · ");
  }

  function dateLabel(update: MediaChange): string {
    return update.effective_on ?? formatDateOnly(update.effective_at);
  }
</script>

<ul class="media-update-list">
  {#each updates as update (update.id)}
    <li>
      {#if onOpenMedia}
        <button
          class="media-update-row"
          type="button"
          onclick={() => onOpenMedia?.(update.media_id)}
        >
          <span class="media-update-mark" aria-hidden="true"></span>
          <span class="media-update-copy">
            <strong>{update.native_title || update.title}</strong>
            <span>{updateLabel(update)} · {basisLabel(update)}</span>
          </span>
          <span class="media-update-date">{dateLabel(update)}</span>
        </button>
      {:else}
        <div class="media-update-row static">
          <span class="media-update-mark" aria-hidden="true"></span>
          <span class="media-update-copy">
            <strong>{update.native_title || update.title}</strong>
            <span>{updateLabel(update)} · {basisLabel(update)}</span>
          </span>
          <span class="media-update-date">{dateLabel(update)}</span>
        </div>
      {/if}
    </li>
  {/each}
</ul>

<style>
  .media-update-list {
    display: grid;
    gap: 0.35rem;
    margin: 0;
    padding: 0;
    list-style: none;
  }
  .media-update-row {
    display: grid;
    grid-template-columns: auto minmax(0, 1fr) auto;
    gap: 0.65rem;
    align-items: center;
    width: 100%;
    padding: 0.55rem 0.65rem;
    border: 1px solid color-mix(in srgb, var(--accent) 18%, var(--border));
    border-radius: var(--radius, 0.5rem);
    background: color-mix(in srgb, var(--accent) 5%, var(--surface));
    color: var(--text);
    font: inherit;
    text-align: left;
    cursor: pointer;
  }
  .media-update-row:hover {
    border-color: color-mix(in srgb, var(--accent) 52%, var(--border));
  }
  .media-update-row.static {
    cursor: default;
  }
  .media-update-mark {
    width: 0.48rem;
    height: 0.48rem;
    border-radius: 50%;
    background: var(--accent);
    box-shadow: 0 0 0 0.22rem color-mix(in srgb, var(--accent) 14%, transparent);
  }
  .media-update-copy {
    display: grid;
    min-width: 0;
    gap: 0.12rem;
  }
  .media-update-copy strong,
  .media-update-copy span,
  .media-update-date {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .media-update-copy strong {
    font-size: 0.78rem;
  }
  .media-update-copy span,
  .media-update-date {
    color: var(--text-muted);
    font-size: 0.68rem;
  }
  .media-update-date {
    font-variant-numeric: tabular-nums;
  }
  @media (max-width: 38rem) {
    .media-update-row {
      grid-template-columns: auto minmax(0, 1fr);
    }
    .media-update-date {
      grid-column: 2;
    }
  }
</style>
