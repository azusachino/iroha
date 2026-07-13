<script lang="ts">
  import { page } from "$app/state";
  import { onMount } from "svelte";
  import { getMedia, type MediaDetail } from "$lib/api";

  let detail = $state<MediaDetail | null>(null);
  let loading = $state(true);
  let error = $state<string | null>(null);
  const mediaId = $derived(page.params.id ?? "");

  onMount(() => {
    void load();
  });

  async function load() {
    loading = true;
    error = null;
    if (!mediaId) {
      error = "Missing media id";
      loading = false;
      return;
    }
    try {
      detail = await getMedia(mediaId);
    } catch (cause) {
      error = cause instanceof Error ? cause.message : String(cause);
    } finally {
      loading = false;
    }
  }

  function progressValue(): number {
    if (!detail?.progress) return detail?.item.progress_percent ?? 0;
    if (detail.progress.progress_percent != null)
      return detail.progress.progress_percent;
    if (detail.progress.position != null && detail.progress.total) {
      return (detail.progress.position / detail.progress.total) * 100;
    }
    return 0;
  }

  function relationLabel(value: string): string {
    return value.replaceAll("_", " ");
  }

  function eventLabel(eventType: string): string {
    if (eventType === "list_state") return "Library snapshot";
    return eventType.replaceAll("_", " ");
  }

  function eventDate(value?: string): string {
    if (!value) return "Undated";
    return new Date(value).toLocaleDateString(undefined, {
      year: "numeric",
      month: "short",
      day: "numeric",
    });
  }
</script>

<svelte:head>
  <title>{detail?.item.title ?? "Media"} · iroha</title>
</svelte:head>

<section class="detail-shell">
  <p><a class="back-link" href="/media">← Back to media</a></p>

  {#if loading}
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
        <h1>{detail.item.title}</h1>
        {#if detail.work.original_title && detail.work.original_title !== detail.item.title}
          <p class="original-title">{detail.work.original_title}</p>
        {/if}
        <p class="work-description">
          {detail.work.description || "No description recorded."}
        </p>
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
              <strong>{Math.round(progressValue())}%</strong>
            </div>
            <div class="progress-track">
              <span
                style={`width: ${Math.min(Math.max(progressValue(), 0), 100)}%`}
              ></span>
            </div>
            <div class="progress-meta">
              <span
                >{detail.progress.position ?? 0}{detail.progress.total != null
                  ? ` / ${detail.progress.total}`
                  : ""}</span
              >
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
                      <strong>{eventLabel(event.event_type)}</strong><time
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
                    {#if event.note}<p class="event-note">{event.note}</p>{/if}
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
                  href={`/media/${relation.related_item_id}`}
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
    font-size: clamp(1.8rem, 5vw, 3.4rem);
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
