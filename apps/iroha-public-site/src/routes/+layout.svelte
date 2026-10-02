<script lang="ts">
  import { onMount } from "svelte";
  import "./app.css";
  import { site } from "$lib/site";

  let { children } = $props();

  // Remove bootstrap feedback only when the success or error shell mounts.
  onMount(() => document.getElementById("public-bootstrap-status")?.remove());
</script>

<svelte:head>
  <meta name="description" content={site.description} />
</svelte:head>

<div class="shell">
  <main class="content">
    {@render children()}
  </main>
  <footer class="footer">
    <span>{site.name} {site.byline} · v{site.version}</span>
    <span>
      Public projection of sanitized activity records · private daily, sleep,
      and media details stay out of this view.
    </span>
    <a href={site.repositoryUrl}>View source on GitHub</a>
  </footer>
</div>
