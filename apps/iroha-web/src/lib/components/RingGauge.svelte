<script lang="ts">
  // Apple-style concentric activity rings. Form = progress-to-goal, so the
  // honest mark is a radial gauge. Arcs use the validated MARK-tier trio
  // (Move=magenta, Exercise=amber, Stand=teal, CVD-safe); the neon glow is
  // chrome only and every arc carries a numeric label (legend), so meaning
  // never rests on hue. pathLength=100 lets us set the sweep as a plain %.
  export interface Ring {
    label: string;
    value: number;
    goal: number;
    unit: string;
    color: string;
  }

  let { rings, size = 132 }: { rings: Ring[]; size?: number } = $props();

  const center = $derived(size / 2);
  const stroke = 11;
  const gap = 4;

  function pct(r: Ring): number {
    if (!Number.isFinite(r.value) || !Number.isFinite(r.goal) || r.goal <= 0)
      return 0;
    return Math.min(100, (r.value / r.goal) * 100);
  }

  function display(value: number): string {
    return Number.isFinite(value) ? Math.round(value).toLocaleString() : "—";
  }

  // Outermost ring first; each inner ring steps in by stroke+gap.
  const geom = $derived(
    rings.map((r, i) => ({
      ring: r,
      radius: center - stroke / 2 - i * (stroke + gap),
      pct: pct(r),
      closed: pct(r) >= 100,
    })),
  );
</script>

<div class="ring-gauge" style={`--gsize:${size}px`}>
  <svg
    viewBox={`0 0 ${size} ${size}`}
    width={size}
    height={size}
    role="img"
    aria-label="Activity rings"
  >
    {#each geom as g}
      <circle
        class="track"
        cx={center}
        cy={center}
        r={g.radius}
        stroke-width={stroke}
      />
    {/each}
    {#each geom as g}
      <circle
        class="arc"
        class:closed={g.closed}
        cx={center}
        cy={center}
        r={g.radius}
        stroke-width={stroke}
        stroke={g.ring.color}
        pathLength="100"
        stroke-dasharray={`${g.pct} 100`}
        stroke-linecap="round"
        transform={`rotate(-90 ${center} ${center})`}
      />
    {/each}
  </svg>
  <ul class="legend">
    {#each geom as g}
      <li>
        <span class="dot" style={`background:${g.ring.color}`}></span>
        <span class="lbl">{g.ring.label}</span>
        <span class="val">
          {display(g.ring.value)}<span class="goal"
            >/{display(g.ring.goal)} {g.ring.unit}</span
          >
          {#if g.closed}<span class="check" aria-label="goal met">✓</span>{/if}
        </span>
      </li>
    {/each}
  </ul>
</div>

<style>
  .ring-gauge {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 1.1rem;
  }
  svg {
    flex: 0 0 auto;
  }
  .track {
    fill: none;
    stroke: color-mix(in srgb, var(--text-muted) 22%, transparent);
  }
  .arc {
    fill: none;
    transition: stroke-dasharray 0.6s ease;
  }
  .legend {
    list-style: none;
    margin: 0;
    padding: 0;
    display: grid;
    gap: 0.5rem;
    min-width: 0;
    flex: 1 1 8rem;
  }
  .legend li {
    display: grid;
    grid-template-columns: auto auto 1fr;
    align-items: baseline;
    gap: 0.4rem 0.5rem;
  }
  .dot {
    width: 9px;
    height: 9px;
    border-radius: 50%;
    align-self: center;
  }
  .lbl {
    font-size: 0.82rem;
    color: var(--text-muted);
  }
  .val {
    font-size: 0.92rem;
    font-weight: 650;
    color: var(--text);
    white-space: nowrap;
  }
  .goal {
    font-weight: 400;
    color: var(--text-muted);
  }
  .check {
    color: var(--accent);
    font-weight: 700;
    margin-left: 0.15rem;
  }
</style>
