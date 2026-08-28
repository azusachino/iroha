<script lang="ts">
  import { sourceBrand } from "../domain/source";

  let { source }: { source?: string | null } = $props();
  const brand = $derived(sourceBrand(source));
</script>

<span
  class={`source-badge source-${brand.id}`}
  title={brand.raw && brand.raw !== brand.label ? `Raw source: ${brand.raw}` : brand.label}
  aria-label={`Source: ${brand.label}`}
>
  <span class="source-mark" aria-hidden="true">{brand.mark}</span>
  <span>{brand.label}</span>
</span>

<style>
  .source-badge {
    display: inline-flex;
    width: fit-content;
    align-items: center;
    gap: 0.4rem;
    min-height: 1.7rem;
    padding: 0.2rem 0.55rem 0.2rem 0.28rem;
    border: 1px solid color-mix(in srgb, var(--source-color) 42%, var(--border));
    border-radius: var(--radius);
    background: color-mix(in srgb, var(--source-color) 12%, var(--surface));
    color: var(--text);
    font-size: 0.75rem;
    font-weight: 700;
    white-space: nowrap;
  }

  .source-mark {
    display: grid;
    width: 1.2rem;
    height: 1.2rem;
    place-items: center;
    border-radius: 50%;
    background: var(--source-color);
    color: #fff;
    font-size: 0.72rem;
    font-weight: 800;
    line-height: 1;
  }

  .source-apple-health,
  .source-apple-watch {
    --source-color: #ff375f;
  }

  .source-garmin {
    --source-color: #008c95;
  }

  .source-strava {
    --source-color: #fc4c02;
  }

  .source-fitbit {
    --source-color: #00b0b9;
  }

  .source-oura {
    --source-color: #9d8062;
  }

  .source-unknown {
    --source-color: var(--accent);
  }
</style>
