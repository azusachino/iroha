<script lang="ts">
	import DomainTile from '$lib/components/DomainTile.svelte';
</script>

<section class="dashboard-shell">
	<div class="dashboard-heading">
		<h1>Dashboard</h1>
		<p class="muted">Personal data cockpit</p>
	</div>
	<div class="domain-grid">
		<DomainTile name="Activity" stat="Runs, walks, rides, and routes" href="/activities" state="active" />
		<DomainTile name="Media" stat="Reading and watching history" state="soon" />
		<DomainTile name="Sleep" stat="Recovery and sleep sessions" state="soon" />
	</div>
</section>

<style>
	.dashboard-shell {
		display: grid;
		gap: 1rem;
	}

	.dashboard-heading {
		display: flex;
		align-items: end;
		justify-content: space-between;
		gap: 1rem;
	}

	.dashboard-heading p {
		margin: 0;
	}

	.domain-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(13rem, 1fr));
		gap: 1rem;
	}
</style>
