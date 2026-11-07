<script lang="ts">
  let {
    size = 22,
    class: className = "brand-mark",
  }: {
    size?: number;
    class?: string;
  } = $props();
</script>

<svg
  xmlns="http://www.w3.org/2000/svg"
  viewBox="0 0 64 64"
  width={size}
  height={size}
  class={className}
  aria-hidden="true"
  role="img"
>
  <defs>
    <radialGradient
      id="brand-mark-halo"
      cx="0"
      cy="0"
      r="1"
      gradientTransform="translate(32 32) scale(24)"
    >
      <stop stop-color="var(--accent, #00d2b4)" stop-opacity="0.28" />
      <stop stop-color="var(--accent, #00d2b4)" stop-opacity="0" />
    </radialGradient>
  </defs>

  <!-- Container field with rounded corners -->
  <rect width="64" height="64" rx="17" class="mark-field" />

  <!-- Subtle glow halo -->
  <circle cx="32" cy="32" r="23" fill="url(#brand-mark-halo)" />

  <!-- Orbit boundary ring -->
  <circle
    cx="32"
    cy="32"
    r="21"
    fill="none"
    class="mark-orbit"
    stroke-width="1.2"
  />

  <!-- Four data-bloom petals -->
  <!-- North (Primary Accent) -->
  <path
    d="M32 31C25.5 27.3 23.9 20.1 26.7 13.8 28.3 10.2 31 8 32 8c1 0 3.7 2.2 5.3 5.8 2.8 6.3 1.2 13.5-5.3 17.2Z"
    class="petal petal-n"
  />
  <!-- East (Secondary Accent) -->
  <path
    d="M33 32C36.7 25.5 43.9 23.9 50.2 26.7 53.8 28.3 56 31 56 32c0 1-2.2 3.7-5.8 5.3-6.3 2.8-13.5 1.2-17.2-5.3Z"
    class="petal petal-e"
  />
  <!-- South (Tertiary Accent) -->
  <path
    d="M32 33C38.5 36.7 40.1 43.9 37.3 50.2 35.7 53.8 33 56 32 56c-1 0-3.7-2.2-5.3-5.8-2.8-6.3-1.2-13.5 5.3-17.2Z"
    class="petal petal-s"
  />
  <!-- West (Quaternary Accent) -->
  <path
    d="M31 32C27.3 38.5 20.1 40.1 13.8 37.3 10.2 35.7 8 33 8 32c0-1 2.2-3.7 5.8-5.3 6.3-2.8 13.5-1.2 17.2 5.3Z"
    class="petal petal-w"
  />

  <!-- Central core hub -->
  <circle cx="32" cy="32" r="7.2" class="mark-hub" stroke-width="1.6" />
  <circle cx="32" cy="32" r="2.8" class="mark-core" />
</svg>

<style>
  svg {
    display: inline-block;
    vertical-align: middle;
    flex-shrink: 0;
  }

  .mark-field {
    fill: color-mix(
      in srgb,
      var(--surface-2, #1e293b) 80%,
      var(--surface, #111827)
    );
    stroke: color-mix(
      in srgb,
      var(--accent, #00d2b4) 22%,
      var(--border, #233146)
    );
    stroke-width: 1.5;
  }

  .mark-orbit {
    stroke: color-mix(in srgb, var(--text, #e6e8ec) 16%, transparent);
  }

  .petal-n {
    fill: var(--accent, #00d2b4);
  }

  .petal-e {
    fill: var(--accent-2, #ff5c8a);
  }

  .petal-s {
    fill: var(--mark-violet, var(--accent, #8d78ed));
  }

  .petal-w {
    fill: var(--mark-amber, var(--accent-2, #e9b84e));
  }

  .mark-hub {
    fill: var(--surface, #111827);
    stroke: var(--text, #e6e8ec);
  }

  .mark-core {
    fill: var(--accent, #00d2b4);
  }
</style>
