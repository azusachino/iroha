<script lang="ts">
  import "./app.css";

  let { children } = $props();
</script>

<div class="shell">
  <div class="content">
    {@render children()}
  </div>
  <footer class="footer">
    Public projection of sanitized activity records · private daily, sleep, and
    media details stay out of this view.
  </footer>
</div>
