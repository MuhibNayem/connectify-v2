<script lang="ts">
	import { Button } from '$lib/components/ui/button';
	import { Check, Star, X, ChevronDown, Loader2 } from '@lucide/svelte';
	import { rsvpEvent, type RSVPStatus } from '$lib/api';

	let {
		eventId,
		currentStatus,
		onStatusChange,
		size = 'default',
		variant = 'default'
	}: {
		eventId: string;
		currentStatus?: RSVPStatus;
		onStatusChange?: (status: RSVPStatus) => void;
		size?: 'default' | 'sm' | 'lg';
		variant?: 'default' | 'hero';
	} = $props();

	let loading = $state(false);
	let showDropdown = $state(false);

	const statuses: { value: RSVPStatus; label: string; icon: typeof Check }[] = [
		{ value: 'going', label: 'Going', icon: Check },
		{ value: 'interested', label: 'Interested', icon: Star },
		{ value: 'not_going', label: 'Not Going', icon: X }
	];

	async function handleRSVP(status: RSVPStatus) {
		if (loading) return;
		loading = true;
		showDropdown = false;

		try {
			await rsvpEvent(eventId, status);
			onStatusChange?.(status);
		} catch (err) {
			console.error('Failed to RSVP:', err);
		} finally {
			loading = false;
		}
	}

	function getButtonLabel() {
		if (!currentStatus) return 'RSVP';
		const status = statuses.find((s) => s.value === currentStatus);
		return status?.label || 'RSVP';
	}

	function getButtonClass() {
		if (variant === 'hero') {
			if (currentStatus === 'going') return 'bg-green-600 hover:bg-green-700 text-white';
			if (currentStatus === 'interested') return 'bg-yellow-600 hover:bg-yellow-700 text-white';
			return 'bg-primary hover:bg-primary/90';
		}
		if (currentStatus === 'going')
			return 'bg-green-600/20 text-green-400 border-green-500/30 hover:bg-green-600/30';
		if (currentStatus === 'interested')
			return 'bg-yellow-600/20 text-yellow-400 border-yellow-500/30 hover:bg-yellow-600/30';
		return '';
	}

	function getCurrentIcon() {
		if (currentStatus === 'going') return Check;
		if (currentStatus === 'interested') return Star;
		return null;
	}
</script>

<div class="relative">
	<!-- svelte-ignore a11y_click_events_have_key_events -->
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<div onclick={() => (showDropdown = !showDropdown)}>
		<Button {size} class="gap-2 {getButtonClass()}" disabled={loading}>
			{#if loading}
				<Loader2 class="h-4 w-4 animate-spin" />
			{:else if getCurrentIcon()}
				<svelte:component this={getCurrentIcon()} size={16} />
			{/if}
			{getButtonLabel()}
			<ChevronDown size={14} />
		</Button>
	</div>

	{#if showDropdown}
		<!-- svelte-ignore a11y_click_events_have_key_events -->
		<!-- svelte-ignore a11y_no_static_element_interactions -->
		<div
			class="bg-background/95 absolute left-0 top-full z-50 mt-1 w-40 overflow-hidden rounded-lg border border-white/10 shadow-xl backdrop-blur-lg"
			onclick={(e) => e.stopPropagation()}
		>
			{#each statuses as status}
				<button
					class="flex w-full items-center gap-3 px-4 py-2.5 text-sm transition-colors hover:bg-white/10 {currentStatus ===
					status.value
						? 'text-primary bg-white/5'
						: 'text-foreground'}"
					onclick={() => handleRSVP(status.value)}
				>
					<status.icon size={16} />
					{status.label}
					{#if currentStatus === status.value}
						<Check size={14} class="ml-auto" />
					{/if}
				</button>
			{/each}
		</div>

		<!-- Backdrop to close dropdown -->
		<!-- svelte-ignore a11y_click_events_have_key_events -->
		<!-- svelte-ignore a11y_no_static_element_interactions -->
		<div class="fixed inset-0 z-40" onclick={() => (showDropdown = false)}></div>
	{/if}
</div>
