<script lang="ts">
  import type { MediaRow } from "../../domain/media";
  import {
    formatProgressCount,
    mediaTypeColor,
    mediaTypeLabel,
    progressPercent,
  } from "../../domain/media";
  import type { DesignLanguage } from "../../theme/themes";

  let {
    item,
    theme,
  }: {
    item: MediaRow;
    theme: DesignLanguage;
  } = $props();

  const title = $derived(item.native_title || item.title);
  const percent = $derived(
    progressPercent(
      item.status,
      item.position,
      item.total,
      item.progress_percent,
    ),
  );
</script>

<a
  class="media-asset-card"
  data-theme={theme}
  href={`/library/${item.id}`}
  aria-label={title}
>
  <span class="asset-cover">
    {#if item.cover_image_url}
      <img src={item.cover_image_url} alt="" loading="lazy" />
    {:else}
      <span class="asset-placeholder" aria-hidden="true">{title.slice(0, 1)}</span>
    {/if}
  </span>

  <strong class="asset-title">{title}</strong>
  <span class="asset-type">
    <i
      class="asset-type-dot"
      style={`background:${mediaTypeColor(item.media_type)}`}
    ></i>
    {mediaTypeLabel(item.media_type)}
  </span>
  <small class="asset-meta"
    >{item.status || "unknown"} · {formatProgressCount(
      item.position,
      item.total,
      item.unit,
      item.status,
    )}</small
  >

  <span class="asset-progress" aria-label={`${percent}% progress`}>
    <i style={`width: ${percent}%`}></i>
  </span>
</a>

<style>
  .media-asset-card {
    position: relative;
    display: grid;
    gap: 0.35rem;
    min-width: 0;
    color: var(--text);
    text-decoration: none;
  }

  .asset-cover {
    position: relative;
    display: block;
    width: 100%;
    aspect-ratio: 3 / 4;
    overflow: hidden;
    background: color-mix(in srgb, var(--accent) 12%, var(--surface));
  }

  .asset-cover img,
  .asset-placeholder {
    display: block;
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  .asset-placeholder {
    display: grid;
    place-items: center;
    color: var(--accent);
    font-size: 2.2rem;
    font-weight: 700;
  }

  .asset-title,
  .asset-meta,
  .asset-type {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .asset-title {
    font-size: 0.88rem;
    font-weight: 600;
  }

  .asset-type,
  .asset-meta {
    color: var(--text-muted);
    font-size: 0.65rem;
  }

  .asset-type {
    display: flex;
    align-items: center;
    gap: 0.3rem;
  }

  .asset-type-dot {
    width: 6px;
    height: 6px;
    flex: 0 0 auto;
    border-radius: 50%;
  }

  .asset-progress,

  .asset-progress i,



  .media-asset-card[data-theme="atlas"] .asset-cover {
    border-radius: calc(var(--radius) * 0.5);
  }

  .media-asset-card[data-theme="atlas"] .asset-type,
  .media-asset-card[data-theme="atlas"] .asset-meta {
    font-family: var(--font-mono);
  }

  .media-asset-card[data-theme="field-journal"] {
    gap: 0.4rem;
    padding: 0.6rem;
    border: 1px solid var(--border);
    background: color-mix(in srgb, var(--surface-1) 92%, transparent);
  }

  .media-asset-card[data-theme="field-journal"] .asset-title {
    font-family: var(--font-serif);
    font-weight: 400;
  }

  .media-asset-card[data-theme="field-journal"] .asset-type,
  .media-asset-card[data-theme="field-journal"] .asset-meta {
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }

  .media-asset-card[data-theme="phenology"] {
    gap: 0.5rem;
  }

  .media-asset-card[data-theme="phenology"] .asset-cover {
    border-radius: var(--radius);
  }

  .media-asset-card[data-theme="phenology"] .asset-title {
    font-style: italic;
    font-weight: 400;
  }

  .media-asset-card[data-theme="cadence"] .asset-cover {
    border-radius: calc(var(--radius) * 0.6);
  }

  .media-asset-card[data-theme="cadence"] .asset-title {
    font-size: 0.86rem;
  }

  .media-asset-card[data-theme="archive"] .asset-cover {
    border-radius: var(--radius);
  }

  .media-asset-card[data-theme="archive"] .asset-title {
    font-family: var(--font-serif);
    font-weight: 700;
  }

  .media-asset-card[data-theme="archive"] .asset-type,
  .media-asset-card[data-theme="archive"] .asset-meta {
    font-family: var(--font-mono);
  }


  .media-asset-card[data-theme="grapher"] {
    grid-template-columns: 5rem minmax(0, 1fr);
    grid-template-areas:
      "cover title"
      "cover type"
      "cover meta"
      "cover progress";
    column-gap: 0.75rem;
    row-gap: 0.25rem;
    border-top: 1px solid var(--border);
    padding: 0.7rem 0;
  }

  .media-asset-card[data-theme="grapher"] .asset-cover {
    grid-area: cover;
    align-self: stretch;
    min-height: 6.7rem;
  }

  .media-asset-card[data-theme="grapher"] .asset-title {
    grid-area: title;
    align-self: end;
  }

  .media-asset-card[data-theme="grapher"] .asset-type {
    grid-area: type;
  }

  .media-asset-card[data-theme="grapher"] .asset-meta {
    grid-area: meta;
  }

  .media-asset-card[data-theme="grapher"] .asset-progress {
    grid-area: progress;
    align-self: end;
  }

  .media-asset-card[data-theme="grapher"]:hover .asset-title {
    color: var(--accent);
  }

  @media (max-width: 768px) {
    .media-asset-card[data-theme="grapher"] {
      grid-template-columns: 4.5rem minmax(0, 1fr);
    }
  }
</style>
