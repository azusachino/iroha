<script lang="ts">
  import { page } from "$app/state";
  import { getMedia, type MediaDetail } from "$lib/api";
  import {
    cleanDescription,
    formatDate,
    formatProgressCount,
    mediaEventLabel,
    mediaWorkTotal,
    progressPercent,
  } from "$lib/format";
  import { heroTitleFontSize } from "$lib/hero-title";
  import { useTheme } from "$lib/themes/context.svelte";
  import ThemeRouteRenderer from "$lib/themes/ThemeRouteRenderer.svelte";
  import { hasThemeRoute } from "$lib/themes/registry";

  const HERO_TITLE_CLAMP = { minRem: 1.8, vw: 5, maxRem: 3.4 };

  let detail = $state<MediaDetail | null>(null);
  let loading = $state(true);
  let error = $state<string | null>(null);
  const theme = useTheme();

  // Re-fetch whenever the route param changes. SvelteKit reuses this component
  // across /library/[id] navigations, so onMount would fire only once and
  // clicking a related title would change the URL without reloading the page.
  $effect(() => {
    void load(page.params.id ?? "");
  });

  async function load(id: string) {
    loading = true;
    error = null;
    detail = null;
    if (!id) {
      error = "Missing media id";
      loading = false;
      return;
    }
    try {
      detail = await getMedia(id);
    } catch (cause) {
      error = cause instanceof Error ? cause.message : String(cause);
    } finally {
      loading = false;
    }
  }

  // The work's own episode/chapter count, independent of this user's
  // progress row -- known even for an ongoing series where progress.total
  // never is.
  function workTotal(): number | undefined {
    return mediaWorkTotal(
      detail?.item.media_type,
      detail?.item.episode_count,
      detail?.item.chapter_count,
    );
  }

  function progressValue(): number {
    const status = detail?.progress?.status ?? detail?.item.status;
    const position = detail?.progress?.position ?? detail?.item.position;
    const total = detail?.progress?.total ?? detail?.item.total;
    const percent =
      detail?.progress?.progress_percent ?? detail?.item.progress_percent;
    return progressPercent(status, position, total, percent, workTotal());
  }

  function progressCountLabel(): string {
    if (!detail?.progress) return "";
    return formatProgressCount(
      detail.progress.position,
      detail.progress.total,
      detail.progress.unit,
      detail.progress.status,
      workTotal(),
    );
  }

  // Whether a total is known or can be inferred (a completed item's
  // position stands in for its total, or the work's own episode/chapter
  // count is known) -- gates the percent readout and bar, which need an
  // actual total to mean anything, on the same basis
  // progressValue()/progressCountLabel() already use internally.
  function hasKnownTotal(): boolean {
    if (!detail?.progress) return false;
    if (detail.progress.total != null) return true;
    if (workTotal() != null) return true;
    return (
      detail.progress.status === "completed" && detail.progress.position != null
    );
  }

  function relationLabel(value: string): string {
    return value.replaceAll("_", " ");
  }

  function eventDate(value?: string): string {
    return value ? formatDate(value) : "Undated";
  }
</script>

<svelte:head>
  <title>{detail?.item.title ?? "Library"} · Library · iroha</title>
</svelte:head>

<section class="detail-shell">
  <p><a class="back-link" href="/library">← Back to Library</a></p>
  {#if hasThemeRoute(theme.definition(), "media-detail") && !loading && !error && detail}
    <ThemeRouteRenderer
      route="media-detail"
      props={{ detail, progress: progressValue() }}
    />
  {:else if loading}
    <p class="muted">Loading item…</p>
  {:else if error}
    <p class="error">Failed to load item: {error}</p>
  {:else if detail}
    <section class="hero tile">
      {#if detail.item.cover_image_url}
        <img class="hero-cover" src={detail.item.cover_image_url} alt="" />
      {:else}
        <div class="hero-cover cover-placeholder" aria-hidden="true">
          {detail.item.title.slice(0, 1)}
        </div>
      {/if}
      <div class="hero-copy">
        <p class="eyebrow">{detail.item.media_type.replaceAll("_", " ")}</p>
        <h1
          style:font-size={heroTitleFontSize(
            detail.item.native_title || detail.item.title,
            HERO_TITLE_CLAMP,
          )}
        >
          {detail.item.native_title || detail.item.title}
        </h1>
        {#if detail.item.native_title && detail.item.native_title !== detail.item.title}
          <p class="original-title">{detail.item.title}</p>
        {/if}
        {#if detail.work.description}
          <p class="work-description">
            {cleanDescription(detail.work.description)}
          </p>
        {/if}
        <div class="hero-meta">
          <span>{detail.item.status?.replaceAll("_", " ") ?? "Untracked"}</span>
          {#if detail.item.rating != null}<span class="score"
              >{detail.item.rating.toFixed(1)} / 10</span
            >{/if}
          {#if detail.work.first_release_date}<span
              >{detail.work.first_release_date.slice(0, 4)}</span
            >{/if}
        </div>
      </div>
    </section>

    <div class="detail-grid">
      <div class="main-column">
        {#if detail.progress}
          <section class="progress-panel tile">
            <div class="section-heading">
              <div>
                <p class="eyebrow">Progress</p>
                <h2>{detail.progress.unit || "Current position"}</h2>
              </div>
              {#if hasKnownTotal()}
                <strong>{Math.round(progressValue())}%</strong>
              {/if}
            </div>
            {#if hasKnownTotal()}
              <div class="progress-track">
                <span
                  style={`width: ${Math.min(Math.max(progressValue(), 0), 100)}%`}
                ></span>
              </div>
            {/if}
            <div class="progress-meta">
              <span>{progressCountLabel()}</span>
              <span
                >{detail.progress.play_count
                  ? `${detail.progress.play_count} replays`
                  : ""}</span
              >
            </div>
          </section>
        {/if}

        <section class="timeline-section">
          <div class="section-heading">
            <div>
              <p class="eyebrow">History</p>
              <h2>Events</h2>
            </div>
            <span class="muted">{detail.events.length}</span>
          </div>
          {#if detail.events.length}
            <ol class="timeline">
              {#each detail.events as event (event.id)}
                <li>
                  <span class="timeline-dot" aria-hidden="true"></span>
                  <div class="timeline-card tile">
                    <div class="event-head">
                      <strong>{mediaEventLabel(event.event_type)}</strong><time
                        >{eventDate(event.event_at)}</time
                      >
                    </div>
                    <div class="event-detail">
                      {#if event.progress_percent != null}<span
                          >{Math.round(event.progress_percent)}% complete</span
                        >{/if}
                      {#if event.position != null}<span
                          >{event.position}{event.total != null
                            ? ` / ${event.total}`
                            : ""}
                          {event.unit}</span
                        >{/if}
                      {#if event.rating != null}<span class="score"
                          >Rated {event.rating.toFixed(1)} / 10</span
                        >{/if}
                    </div>
                    {#if event.note}<p class="event-note">
                        {event.note}
                      </p>{/if}
                  </div>
                </li>
              {/each}
            </ol>
          {:else}
            <div class="empty-panel tile">No event history recorded.</div>
          {/if}
        </section>
      </div>

      <aside class="side-column">
        {#if detail.creators.length}
          <section class="side-panel tile">
            <p class="eyebrow">Credits</p>
            <h2>People</h2>
            <ul class="creator-list">
              {#each detail.creators as creator (creator.id)}<li>
                  <span>{creator.name}</span><small>{creator.role}</small>
                </li>{/each}
            </ul>
          </section>
        {/if}
        {#if detail.relations.length}
          <section class="side-panel tile">
            <p class="eyebrow">Connections</p>
            <h2>Related</h2>
            <div class="relation-list">
              {#each detail.relations as relation (relation.id)}<a
                  class="relation-card"
                  href={`/library/${relation.related_item_id}`}
                  >{#if relation.cover_image_url}<img
                      src={relation.cover_image_url}
                      alt=""
                    />{:else}<div
                      class="relation-placeholder"
                      aria-hidden="true"
                    >
                      {relation.related_title.slice(0, 1)}
                    </div>{/if}<span
                    ><strong>{relation.related_title}</strong><small
                      >{relationLabel(relation.relation_type)}</small
                    ></span
                  ></a
                >{/each}
            </div>
          </section>
        {/if}
      </aside>
    </div>
  {/if}
</section>

<style>
  .detail-shell {
    display: grid;
    gap: 1.25rem;
  }
  .back-link {
    color: var(--text-muted);
    font-size: 0.85rem;
  }
  .hero {
    padding: 1.25rem;
    display: grid;
    grid-template-columns: 10rem 1fr;
    gap: 1.5rem;
  }
  .hero-cover {
    width: 10rem;
    aspect-ratio: 2 / 3;
    object-fit: cover;
    background: var(--surface-2);
  }
  .cover-placeholder,
  .relation-placeholder {
    display: grid;
    place-items: center;
    border: 1px solid var(--border);
    background: linear-gradient(145deg, var(--surface-2), var(--surface));
    color: var(--accent);
    font-size: 2.5rem;
    font-weight: 800;
  }
  .hero-copy {
    align-self: center;
    display: grid;
    gap: 0.75rem;
  }
  h1,
  h2,
  p {
    margin: 0;
  }
  h1 {
    max-width: 38rem;
    line-height: 1.02;
    letter-spacing: -0.045em;
  }
  h2 {
    font-size: 1rem;
  }
  .eyebrow {
    color: var(--text-muted);
    font-size: 0.68rem;
    font-weight: 750;
    letter-spacing: 0.09em;
    text-transform: uppercase;
  }
  .original-title,
  .muted {
    color: var(--text-muted);
  }
  .work-description {
    max-width: 55rem;
    color: var(--text-muted);
    line-height: 1.55;
    white-space: pre-line;
  }
  .hero-meta,
  .progress-meta,
  .event-detail {
    display: flex;
    flex-wrap: wrap;
    gap: 0.65rem 1rem;
    color: var(--text-muted);
    font-size: 0.8rem;
  }
  .score {
    color: var(--accent);
    font-weight: 750;
  }
  .detail-grid {
    display: grid;
    grid-template-columns: minmax(0, 1.5fr) minmax(15rem, 0.7fr);
    gap: 1.5rem;
    align-items: start;
  }
  .main-column,
  .side-column,
  .timeline-section {
    display: grid;
    gap: 1rem;
  }
  .progress-panel,
  .side-panel {
    padding: 1rem;
    display: grid;
    gap: 0.9rem;
  }
  .section-heading {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    gap: 1rem;
  }
  .section-heading strong {
    color: var(--accent);
    font-size: 1.3rem;
  }
  .progress-track {
    height: 0.6rem;
    overflow: hidden;
    border-radius: 999px;
    background: var(--surface-2);
  }
  .progress-track span {
    display: block;
    height: 100%;
    border-radius: inherit;
    background: linear-gradient(90deg, var(--accent), var(--accent-2));
  }
  .timeline {
    position: relative;
    display: grid;
    gap: 0.8rem;
    margin: 0;
    padding: 0 0 0 1.25rem;
    list-style: none;
  }
  .timeline::before {
    content: "";
    position: absolute;
    top: 0.4rem;
    bottom: 0.4rem;
    left: 0.3rem;
    width: 1px;
    background: var(--border);
  }
  .timeline li {
    position: relative;
  }
  .timeline-dot {
    position: absolute;
    top: 1.2rem;
    left: -1.2rem;
    width: 0.55rem;
    height: 0.55rem;
    border: 2px solid var(--bg);
    border-radius: 50%;
    background: var(--accent);
    box-shadow: 0 0 0 1px var(--accent);
  }
  .timeline-card {
    padding: 0.9rem 1rem;
    display: grid;
    gap: 0.6rem;
  }
  .event-head {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
  }
  time,
  .event-note {
    color: var(--text-muted);
    font-size: 0.78rem;
  }
  .event-note {
    line-height: 1.45;
  }
  .creator-list {
    display: grid;
    gap: 0.7rem;
    margin: 0;
    padding: 0;
    list-style: none;
  }
  .creator-list li {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    font-size: 0.82rem;
  }
  .creator-list small,
  .relation-card small {
    color: var(--text-muted);
  }
  .relation-list {
    display: grid;
    gap: 0.65rem;
  }
  .relation-card {
    display: grid;
    grid-template-columns: 2.8rem 1fr;
    gap: 0.65rem;
    align-items: center;
    color: var(--text);
  }
  .relation-card img,
  .relation-placeholder {
    width: 2.8rem;
    aspect-ratio: 2 / 3;
    object-fit: cover;
  }
  .relation-card span {
    min-width: 0;
    display: grid;
    gap: 0.22rem;
  }
  .relation-card strong {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-size: 0.8rem;
  }
  .empty-panel {
    padding: 1rem;
    color: var(--text-muted);
  }
  .error {
    color: var(--danger);
  }
  @media (max-width: 720px) {
    .hero {
      grid-template-columns: 6.5rem 1fr;
      gap: 1rem;
      padding: 0.9rem;
    }
    .hero-cover {
      width: 6.5rem;
    }
    .detail-grid {
      grid-template-columns: 1fr;
    }
  }
</style>
