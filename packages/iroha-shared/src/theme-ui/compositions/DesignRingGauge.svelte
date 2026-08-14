<script lang="ts">
  import type { DesignTodayData } from "../../design-compositions";

  let {
    today,
    size = 168,
  }: {
    today: DesignTodayData;
    size?: number;
  } = $props();

  const ring = $derived(today.daily?.ring);
  const values = $derived([
    {
      value: ring?.move_kcal ?? 0,
      goal: ring?.move_goal_kcal ?? 1,
      color: "var(--ring-move, #ff5f57)",
    },
    {
      value: ring?.exercise_min ?? 0,
      goal: ring?.exercise_goal_min ?? 1,
      color: "var(--ring-exercise, #30d158)",
    },
    {
      value: ring?.stand_hours ?? 0,
      goal: ring?.stand_goal_hours ?? 1,
      color: "var(--ring-stand, #64d2ff)",
    },
  ]);
</script>

<svg
  class="ring-gauge"
  width={size}
  height={size}
  viewBox="0 0 100 100"
  role="img"
  aria-label="Daily activity rings"
>
  {#each values as ringValue, index}
    {@const radius = 39 - index * 10}
    {@const circumference = 2 * Math.PI * radius}
    {@const progress = Math.min(
      1,
      Math.max(0, ringValue.value / ringValue.goal),
    )}
    <circle class="ring-track" cx="50" cy="50" r={radius} stroke-width="6"
    ></circle>
    <circle
      class="ring-value"
      cx="50"
      cy="50"
      r={radius}
      stroke={ringValue.color}
      stroke-width="6"
      stroke-dasharray={`${circumference * progress} ${circumference}`}
      transform="rotate(-90 50 50)"
    ></circle>
  {/each}
  <text x="50" y="48" text-anchor="middle">IROHA</text>
  <text class="ring-caption" x="50" y="58" text-anchor="middle">DAY</text>
</svg>

<style>
  .ring-gauge {
    display: block;
    max-width: 100%;
    overflow: visible;
  }

  .ring-track,
  .ring-value {
    fill: none;
    stroke-linecap: round;
  }

  .ring-track {
    stroke: color-mix(in srgb, var(--text) 10%, transparent);
  }

  .ring-value {
    filter: drop-shadow(
      0 0 0.25rem color-mix(in srgb, currentcolor 38%, transparent)
    );
  }

  text {
    fill: var(--text);
    font-family: var(--font-mono, monospace);
    font-size: 0.24rem;
    font-weight: 700;
    letter-spacing: 0.14em;
  }

  .ring-caption {
    fill: var(--text-muted);
    font-size: 0.18rem;
    font-weight: 500;
  }
</style>
