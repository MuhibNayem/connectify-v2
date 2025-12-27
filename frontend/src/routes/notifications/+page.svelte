<script lang="ts">
	import NotificationList from '$lib/components/notifications/NotificationList.svelte';
</script>

<div class="min-h-screen bg-transparent pb-12 pt-4 font-sans">
	<div class="mx-auto max-w-2xl">
		<div class="glass-panel w-full overflow-hidden rounded-xl border border-white/10 p-0">
			<NotificationList mode="full" />
		</div>
	</div>
</div>
