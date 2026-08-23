<script lang="ts">
  import { ArrowRight, BookOpen } from "@lucide/svelte";
  import { PAGE_PERIOD_DEFAULTS } from "@iroha/shared/format/period";
  import { THEME_DEFINITIONS } from "$lib/themes/registry";
  import { useTheme } from "$lib/themes/context.svelte";

  type PageGuide = {
    name: string;
    href: string;
    question: string;
    chart: string;
    detail: string;
    period: string;
  };

  const pageGuides: PageGuide[] = [
    {
      name: "Today",
      href: "/",
      question: "What did today contain?",
      chart: "Daily signal, rings, and a chronological cross-domain read.",
      detail:
        "A date-level cockpit linking movement, night, health, and media.",
      period: "Current day",
    },
    {
      name: "Overview",
      href: "/overview",
      question: "What is the latest shape of my data?",
      chart: "Cross-domain headline cards and the latest meaningful records.",
      detail: "A doorway into domains, not a second data store or a scorecard.",
      period: "Latest available day",
    },
    {
      name: "Motion",
      href: "/motion",
      question: "How did I move, and how is it changing?",
      chart: "Activity totals, trend comparison, and an ECharts metric view.",
      detail:
        "Rows open on click to route, laps, source, and exact activity facts.",
      period: PAGE_PERIOD_DEFAULTS.motion,
    },
    {
      name: "Night",
      href: "/night",
      question: "How did the night unfold?",
      chart: "Sleep architecture, monthly comparison, and stage composition.",
      detail:
        "Click any row or date to inspect every canonical sleep-stage interval.",
      period: PAGE_PERIOD_DEFAULTS.night,
    },
    {
      name: "Library",
      href: "/library",
      question: "What have I collected and returned to?",
      chart: "Collection coverage, progress, and history across media types.",
      detail: "A lifetime catalogue with item-level history and provenance.",
      period: PAGE_PERIOD_DEFAULTS.library,
    },
    {
      name: "Expenses",
      href: "/expenses",
      question: "Where did the month go?",
      chart:
        "Category color, daily flow, currency totals, and monthly comparison.",
      detail:
        "The canonical expense ledger is read-only here; mutations use the API.",
      period: PAGE_PERIOD_DEFAULTS.expenses,
    },
    {
      name: "Patterns",
      href: "/patterns",
      question: "What repeats across my days?",
      chart: "Small multiples, rings, and daily/monthly/yearly aggregation.",
      detail:
        "Missing observations stay missing; a gap is not silently turned into zero.",
      period: PAGE_PERIOD_DEFAULTS.patterns,
    },
    {
      name: "Reports",
      href: "/reports",
      question: "What changed this month?",
      chart: "Comparison first: trend, incline/decline, and domain charts.",
      detail: "The exact rows and provenance sit below the visual summary.",
      period: PAGE_PERIOD_DEFAULTS.reports,
    },
    {
      name: "Metrics",
      href: "/metrics",
      question: "How do I inspect one metric precisely?",
      chart: "Metric series, comparison, annotations, and exportable values.",
      detail:
        "A catalog explorer for canonical and derived metrics with units intact.",
      period: PAGE_PERIOD_DEFAULTS.metrics,
    },
    {
      name: "To-go",
      href: "/to-go",
      question: "What needs attention next?",
      chart: "Tasks and background work, separate from domain evidence.",
      detail:
        "Operational convenience; it does not own personal domain records.",
      period: "No period",
    },
    {
      name: "Admin",
      href: "/admin",
      question: "Is the cockpit healthy and what is it doing?",
      chart: "Health, metric catalog, and grouped execution kinds.",
      detail:
        "Read-only operations view; repeated executions are summarized by kind.",
      period: "No period",
    },
    {
      name: "Design",
      href: "/design",
      question: "What does this data language make visible?",
      chart:
        "Registered theme specimens using the same canonical sample payload.",
      detail:
        "A working design surface, not a gallery detached from the application.",
      period: "No period",
    },
  ];

  const principles = [
    [
      "Canonical first",
      "Every source is normalized into Iroha records with canonical dates, units, and provenance.",
    ],
    [
      "Chart first",
      "A visual answers the page question before the exact ledger and export details.",
    ],
    [
      "Evidence below",
      "Rows, intervals, source brands, and raw references remain available for verification.",
    ],
    [
      "Gaps are data",
      "Unknown, unavailable, and zero are different states and are rendered differently.",
    ],
  ];

  const theme = useTheme();
</script>

<svelte:head><title>Manual · iroha</title></svelte:head>

<section class="manual-page" data-theme={theme.definition().identity.id}>
  <header class="manual-hero">
    <div>
      <p class="eyebrow"><BookOpen size={14} /> Iroha manual</p>
      <h1>How to read your world.</h1>
      <p class="hero-copy">
        Iroha is a personal data cockpit. It keeps the canonical record in one
        place, then gives each question a visual language, a period, and a path
        back to exact evidence.
      </p>
    </div>
    <div class="hero-mark" aria-hidden="true">
      <span>01</span><strong>read</strong><small
        >the shape, then the record</small
      >
    </div>
  </header>

  <section class="principle-grid" aria-label="Iroha principles">
    {#each principles as principle}
      <article class="principle-card">
        <span class="index">0{principles.indexOf(principle) + 1}</span>
        <h2>{principle[0]}</h2>
        <p>{principle[1]}</p>
      </article>
    {/each}
  </section>

  <section
    class="manual-section composition"
    aria-labelledby="composition-title"
  >
    <header class="section-heading">
      <div>
        <p class="eyebrow">The reading order</p>
        <h2 id="composition-title">One record, four views.</h2>
      </div>
      <p>
        The same canonical data may be collected from Apple Health, Garmin, a
        file import, or an API call. The source changes; the cockpit contract
        does not.
      </p>
    </header>
    <div class="reading-flow" aria-label="Iroha data representation flow">
      <article>
        <span>01</span><strong>Source</strong>
        <p>Imported evidence and its original provider.</p>
      </article>
      <span class="flow-arrow" aria-hidden="true"><ArrowRight /></span>
      <article>
        <span>02</span><strong>Record</strong>
        <p>Canonical date, units, category, and provenance.</p>
      </article>
      <span class="flow-arrow" aria-hidden="true"><ArrowRight /></span>
      <article>
        <span>03</span><strong>Chart</strong>
        <p>Aggregation and comparison answer the page question.</p>
      </article>
      <span class="flow-arrow" aria-hidden="true"><ArrowRight /></span>
      <article>
        <span>04</span><strong>Evidence</strong>
        <p>Click the line, date, or row for the exact detail.</p>
      </article>
    </div>
  </section>

  <section class="manual-section" aria-labelledby="pages-title">
    <header class="section-heading">
      <div>
        <p class="eyebrow">Page map</p>
        <h2 id="pages-title">Every page has one job.</h2>
      </div>
      <p>
        Use the chart to orient yourself, then open the record when you need to
        verify it.
      </p>
    </header>
    <div class="page-grid">
      {#each pageGuides as pageGuide}
        <a class="page-card" href={pageGuide.href}>
          <div class="page-card-head">
            <span>{pageGuide.name}</span><ArrowRight size={15} />
          </div>
          <h3>{pageGuide.question}</h3>
          <p class="page-chart">{pageGuide.chart}</p>
          <p>{pageGuide.detail}</p>
          <small><b>Default scope</b> {pageGuide.period}</small>
        </a>
      {/each}
    </div>
  </section>

  <section class="manual-section" aria-labelledby="interaction-title">
    <header class="section-heading">
      <div>
        <p class="eyebrow">Interaction contract</p>
        <h2 id="interaction-title">The cockpit should feel predictable.</h2>
      </div>
    </header>
    <div class="contract-grid">
      <article>
        <strong>Click the visual row</strong>
        <p>
          Lines, dates, bars, and ledger rows open their detail. A separate
          “detail” button is unnecessary chrome.
        </p>
      </article>
      <article>
        <strong>Use ← →</strong>
        <p>
          Move to the previous or next canonical day or month without reopening
          the picker.
        </p>
      </article>
      <article>
        <strong>Read the URL</strong>
        <p>
          Period and filter state are canonical query parameters, so refresh and
          sharing preserve the view.
        </p>
      </article>
      <article>
        <strong>Trust the badge</strong>
        <p>
          Provider brands are mapped from source identifiers; the raw source
          remains available as provenance.
        </p>
      </article>
    </div>
  </section>

  <section class="manual-section themes" aria-labelledby="themes-title">
    <header class="section-heading">
      <div>
        <p class="eyebrow">Theme registry</p>
        <h2 id="themes-title">Six ways to see the same truth.</h2>
      </div>
      <p>
        The Design page changes this same selection and renders the real
        registered theme components.
      </p>
    </header>
    <div class="theme-grid">
      {#each THEME_DEFINITIONS as definition}
        <button
          type="button"
          style={"--theme-color:" + definition.identity.swatch}
          class:active={theme.language() === definition.identity.id}
          class={`theme-card ${definition.identity.id}`}
          onclick={() => theme.select(definition.identity.id)}
        >
          <span class="theme-swatch" aria-hidden="true"
            >{definition.identity.mark}</span
          >
          <span class="theme-card-copy">
            <strong>{definition.identity.label}</strong>
            <small>{definition.identity.hint}</small>
            <p>{definition.identity.description}</p>
            <span class="theme-lens"
              ><b>Expenses</b>
              {definition.identity.lenses.expenses.lead}</span
            >
            <span class="theme-lens"
              ><b>Reports</b>
              {definition.identity.lenses.reports.lead}</span
            >
          </span>
        </button>
      {/each}
    </div>
  </section>

  <footer class="manual-footer">
    <span
      >Canonical dates use <code>yyyy-MM</code> for periods and
      <code>yyyy-MM-dd</code> for days.</span
    >
    <a href="/design">Open the theme workshop <ArrowRight size={15} /></a>
  </footer>
</section>

<style>
  .manual-page {
    display: grid;
    gap: 2rem;
    padding-bottom: 3rem;
  }

  .manual-hero,
  .section-heading,
  .manual-footer {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 2rem;
  }

  .manual-hero {
    padding: 1rem 0 0;
  }

  .manual-hero h1,
  .section-heading h2 {
    margin: 0;
    letter-spacing: -0.045em;
  }

  .manual-hero h1 {
    max-width: 38rem;
    font-size: clamp(2.5rem, 7vw, 5.8rem);
    line-height: 0.98;
  }

  .eyebrow {
    display: inline-flex;
    align-items: center;
    gap: 0.4rem;
    margin: 0 0 0.55rem;
    color: var(--accent);
    font-size: 0.7rem;
    font-weight: 800;
    letter-spacing: 0.11em;
    text-transform: uppercase;
  }

  .hero-copy,
  .section-heading > p {
    max-width: 40rem;
    margin: 0.9rem 0 0;
    color: var(--text-muted);
    font-size: 1rem;
    line-height: 1.7;
  }

  .hero-mark {
    display: grid;
    min-width: 11rem;
    padding: 1rem;
    border: 1px solid var(--border);
    border-radius: 1rem;
    background: linear-gradient(
      135deg,
      var(--surface),
      color-mix(in srgb, var(--accent) 10%, var(--surface))
    );
    color: var(--text-muted);
    text-align: right;
  }

  .hero-mark span {
    color: var(--accent);
    font-size: 0.7rem;
    letter-spacing: 0.1em;
  }

  .hero-mark strong {
    color: var(--text);
    font-size: 2rem;
    letter-spacing: -0.06em;
  }

  .hero-mark small {
    font-size: 0.7rem;
  }

  .principle-grid,
  .page-grid,
  .contract-grid,
  .theme-grid {
    display: grid;
    gap: 0.8rem;
  }

  .principle-grid {
    grid-template-columns: repeat(4, 1fr);
  }

  .principle-card,
  .page-card,
  .contract-grid article,
  .theme-card,
  .reading-flow article {
    border: 1px solid var(--border);
    border-radius: 0.85rem;
    background: color-mix(in srgb, var(--surface) 92%, var(--accent) 8%);
  }

  .principle-card {
    padding: 1rem;
  }
  .principle-card .index {
    color: var(--accent);
    font: 700 0.72rem/1 monospace;
  }
  .principle-card h2 {
    margin: 1.4rem 0 0.45rem;
    font-size: 1rem;
  }
  .principle-card p,
  .page-card p,
  .contract-grid p,
  .reading-flow p,
  .theme-card p {
    margin: 0;
    color: var(--text-muted);
    font-size: 0.84rem;
    line-height: 1.55;
  }

  .manual-section {
    display: grid;
    gap: 1rem;
  }
  .section-heading {
    align-items: end;
  }
  .section-heading h2 {
    font-size: clamp(1.6rem, 3vw, 2.4rem);
  }
  .section-heading > p {
    max-width: 32rem;
    margin: 0;
  }

  .reading-flow {
    display: grid;
    grid-template-columns: 1fr auto 1fr auto 1fr auto 1fr;
    align-items: center;
    gap: 0.65rem;
  }

  .reading-flow article {
    min-height: 8rem;
    padding: 1rem;
  }
  .reading-flow article > span {
    color: var(--accent);
    font: 700 0.72rem/1 monospace;
  }
  .reading-flow strong {
    display: block;
    margin: 1.1rem 0 0.35rem;
    font-size: 1.05rem;
  }
  .flow-arrow {
    color: var(--accent);
  }

  .page-grid {
    grid-template-columns: repeat(3, 1fr);
  }
  .page-card {
    display: grid;
    gap: 0.65rem;
    padding: 1rem;
    color: inherit;
    text-decoration: none;
    transition:
      transform 160ms ease,
      border-color 160ms ease;
  }
  .page-card:hover,
  .page-card:focus-visible {
    border-color: var(--accent);
    transform: translateY(-2px);
  }
  .page-card-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    color: var(--accent);
    font-size: 0.78rem;
    font-weight: 800;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }
  .page-card h3 {
    margin: 0.35rem 0 0;
    font-size: 1.1rem;
    letter-spacing: -0.02em;
  }
  .page-card .page-chart {
    color: var(--text);
  }
  .page-card small {
    margin-top: 0.55rem;
    color: var(--text-muted);
    font-size: 0.72rem;
  }
  .page-card small b {
    color: var(--accent);
    font-weight: 750;
  }

  .contract-grid {
    grid-template-columns: repeat(4, 1fr);
  }
  .contract-grid article {
    padding: 1rem;
  }
  .contract-grid strong {
    display: block;
    margin-bottom: 0.5rem;
    font-size: 0.95rem;
  }

  .theme-grid {
    grid-template-columns: repeat(3, 1fr);
  }
  .theme-card {
    display: flex;
    align-items: flex-start;
    gap: 0.8rem;
    padding: 1rem;
    color: inherit;
    text-align: left;
    cursor: pointer;
  }
  .theme-card:hover,
  .theme-card.active {
    border-color: var(--accent);
    box-shadow: 0 0 0 1px color-mix(in srgb, var(--accent) 30%, transparent);
  }
  .theme-swatch {
    display: grid;
    flex: 0 0 auto;
    width: 2.25rem;
    height: 2.25rem;
    place-items: center;
    border-radius: 0.65rem;
    background: var(--theme-color, var(--accent));
    box-shadow: inset 0 0 0 5px color-mix(in srgb, white 15%, transparent);
    color: var(--brand-ink, white);
    font-family: var(--font-serif, var(--font-mono, monospace));
    font-size: 1.1rem;
    font-weight: 700;
  }
  .theme-card-copy {
    display: grid;
    gap: 0.25rem;
  }
  .theme-card-copy strong {
    font-size: 0.95rem;
  }
  .theme-card-copy small {
    color: var(--accent);
    font-size: 0.72rem;
    text-transform: uppercase;
  }
  .theme-card-copy p {
    margin-top: 0.25rem;
  }
  .theme-lens {
    display: block;
    margin-top: 0.35rem;
    color: var(--text-muted);
    font-size: 0.72rem;
    line-height: 1.4;
  }
  .theme-lens b {
    margin-right: 0.25rem;
    color: var(--text);
  }
  .manual-footer {
    align-items: center;
    padding: 1rem 0 0;
    border-top: 1px solid var(--border);
    color: var(--text-muted);
    font-size: 0.78rem;
  }
  .manual-footer a {
    display: inline-flex;
    align-items: center;
    gap: 0.4rem;
    min-height: var(--control-target-min);
    color: var(--accent);
    font-weight: 750;
    text-decoration: none;
  }
  code {
    padding: 0.1rem 0.3rem;
    border-radius: 0.25rem;
    background: var(--surface-strong);
    color: var(--text);
    font-family: monospace;
  }

  @media (max-width: 1024px) {
    .principle-grid,
    .contract-grid {
      grid-template-columns: repeat(2, 1fr);
    }
    .page-grid,
    .theme-grid {
      grid-template-columns: repeat(2, 1fr);
    }
    .reading-flow {
      grid-template-columns: repeat(2, 1fr);
    }
    .flow-arrow {
      display: none;
    }
  }

  @media (max-width: 640px) {
    .manual-hero,
    .section-heading,
    .manual-footer {
      display: grid;
    }
    .hero-mark {
      justify-self: start;
      text-align: left;
    }
    .principle-grid,
    .page-grid,
    .contract-grid,
    .theme-grid,
    .reading-flow {
      grid-template-columns: 1fr;
    }
  }
</style>
