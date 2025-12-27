<script lang="ts">
	import { Button } from '$lib/components/ui/button';
	import { X, Copy, Check, Share2, MessageCircle, Send } from '@lucide/svelte';
	import { shareEvent } from '$lib/api';
	import { goto } from '$app/navigation';

	let {
		eventId,
		eventTitle,
		open = $bindable(false),
		onClose
	}: {
		eventId: string;
		eventTitle: string;
		open?: boolean;
		onClose?: () => void;
	} = $props();

	let copied = $state(false);

	function getShareUrl() {
		return `${window.location.origin}/events/${eventId}`;
	}

	async function copyLink() {
		try {
			await navigator.clipboard.writeText(getShareUrl());
			copied = true;
			await shareEvent(eventId); // Track share
			setTimeout(() => (copied = false), 2000);
		} catch (err) {
			console.error('Failed to copy:', err);
		}
	}

	function shareToFacebook() {
		const url = encodeURIComponent(getShareUrl());
		window.open(
			`https://www.facebook.com/sharer/sharer.php?u=${url}`,
			'_blank',
			'width=600,height=400'
		);
		shareEvent(eventId);
	}

	function shareToTwitter() {
		const url = encodeURIComponent(getShareUrl());
		const text = encodeURIComponent(`Check out this event: ${eventTitle}`);
		window.open(
			`https://twitter.com/intent/tweet?url=${url}&text=${text}`,
			'_blank',
			'width=600,height=400'
		);
		shareEvent(eventId);
	}

	function shareToWhatsApp() {
		const url = encodeURIComponent(getShareUrl());
		const text = encodeURIComponent(`Check out this event: ${eventTitle} - ${getShareUrl()}`);
		window.open(`https://wa.me/?text=${text}`, '_blank');
		shareEvent(eventId);
	}

	async function shareViaMessage() {
		// Navigate to messages with prefilled event
		handleClose();
		goto(`/messages?share_event=${eventId}`);
		await shareEvent(eventId);
	}

	function handleClose() {
		open = false;
		onClose?.();
	}

	const shareOptions = [
		{ label: 'Facebook', icon: '📘', action: shareToFacebook },
		{ label: 'Twitter', icon: '🐦', action: shareToTwitter },
		{ label: 'WhatsApp', icon: '💬', action: shareToWhatsApp }
	];
</script>

{#if open}
	<!-- svelte-ignore a11y_click_events_have_key_events -->
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm"
		onclick={handleClose}
	>
		<!-- svelte-ignore a11y_click_events_have_key_events -->
		<!-- svelte-ignore a11y_no_static_element_interactions -->
		<div
			class="bg-card mx-4 w-full max-w-sm overflow-hidden rounded-2xl border border-white/10 shadow-2xl"
			onclick={(e) => e.stopPropagation()}
		>
			<!-- Header -->
			<div class="flex items-center justify-between border-b border-white/10 p-4">
				<div class="flex items-center gap-2">
					<Share2 size={20} class="text-primary" />
					<h2 class="text-lg font-bold">Share Event</h2>
				</div>
				<button
					class="text-muted-foreground hover:text-foreground rounded-full p-1 transition-colors"
					onclick={handleClose}
				>
					<X size={20} />
				</button>
			</div>

			<!-- Event Preview -->
			<div class="border-b border-white/10 p-4">
				<p class="text-muted-foreground text-sm">Sharing:</p>
				<p class="font-medium">{eventTitle}</p>
			</div>

			<!-- Share Options -->
			<div class="grid grid-cols-3 gap-4 p-4">
				{#each shareOptions as option}
					<button
						class="flex flex-col items-center gap-2 rounded-xl p-3 transition-colors hover:bg-white/10"
						onclick={option.action}
					>
						<span class="text-2xl">{option.icon}</span>
						<span class="text-xs font-medium">{option.label}</span>
					</button>
				{/each}
			</div>

			<div class="border-t border-white/10 px-4 py-3">
				<button
					class="flex w-full items-center gap-3 rounded-lg bg-white/5 px-4 py-3 transition-colors hover:bg-white/10"
					onclick={shareViaMessage}
				>
					<MessageCircle size={20} class="text-primary" />
					<span class="font-medium">Send via Message</span>
					<Send size={16} class="text-muted-foreground ml-auto" />
				</button>
			</div>

			<!-- Copy Link -->
			<div class="border-t border-white/10 p-4">
				<p class="text-muted-foreground mb-2 text-sm">Or copy link</p>
				<div class="flex gap-2">
					<input
						type="text"
						readonly
						value={getShareUrl()}
						class="bg-secondary/30 text-muted-foreground flex-1 rounded-lg border-none px-3 py-2 text-sm"
					/>
					<Button size="sm" onclick={copyLink} class="shrink-0 gap-2">
						{#if copied}
							<Check size={16} class="text-green-400" />
							Copied!
						{:else}
							<Copy size={16} />
							Copy
						{/if}
					</Button>
				</div>
			</div>
		</div>
	</div>
{/if}
