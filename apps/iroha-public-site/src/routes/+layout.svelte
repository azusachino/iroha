<script lang="ts">
  import "./app.css";
  import { site } from "$lib/site";

  let { children } = $props();
</script>

<svelte:head>
  <meta name="description" content={site.description} />
</svelte:head>

<div class="shell">
  <div class="content">
    {@render children()}
  </div>
  <footer class="footer">
    <span>{site.name} {site.byline}</span>
    <span>
      Public projection of sanitized activity records · private daily, sleep,
      and media details stay out of this view.
    </span>
    <a href={site.repositoryUrl}>View source on GitHub</a>
  </footer>
</div>
