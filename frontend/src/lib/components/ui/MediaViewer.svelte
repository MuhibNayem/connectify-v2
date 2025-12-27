<script lang="ts">
	import { X, ChevronLeft, ChevronRight } from '@lucide/svelte';
	import { fade, scale } from 'svelte/transition';
	import { onMount, onDestroy } from 'svelte';

	interface MediaItem {
		type: 'image' | 'video';
		url: string;
		thumbnail?: string;
	}

	let {
		open = false,
		media = [],
		initialIndex = 0,
		onClose,
		onReachEnd
	} = $props<{
		open: boolean;
		media: MediaItem[];
		initialIndex?: number;
		onClose: () => void;
		onReachEnd?: () => void;
	}>();

	let currentIndex = $state(initialIndex);

	$effect(() => {
		if (open) {
			currentIndex = initialIndex;
			document.body.style.overflow = 'hidden';
		} else {
			document.body.style.overflow = '';
		}
	});

	onMount(() => {
		window.addEventListener('keydown', handleKeydown);
	});

	onDestroy(() => {
		if (typeof window !== 'undefined') {
			window.removeEventListener('keydown', handleKeydown);
			document.body.style.overflow = '';
		}
	});

	function handleKeydown(e: KeyboardEvent) {
		if (!open) return;
		if (e.key === 'Escape') onClose();
		if (e.key === 'ArrowLeft') prev();
		if (e.key === 'ArrowRight') next();
	}

	function next() {
		currentIndex = (currentIndex + 1) % media.length;
		// Pre-load more when we get close to the end (5 items remaining)
		if (currentIndex >= media.length - 5 && onReachEnd) {
			onReachEnd();
		}
	}

	function prev() {
		// Prevent wrapping to end if we want strict pagination, but for now standard wrapping is fine
		currentIndex = (currentIndex - 1 + media.length) % media.length;
	}

	function handleBackdropClick(e: MouseEvent) {
		if (e.target === e.currentTarget) {
			onClose();
		}
	}
</script>

{#if open}
	<div
		class="fixed inset-0 z-[100] flex items-center justify-center bg-black/90 backdrop-blur-md"
		transition:fade={{ duration: 200 }}
		onclick={handleBackdropClick}
		role="dialog"
		aria-modal="true"
	>
		<!-- Close Button -->
		<button
			class="absolute right-4 top-4 z-[101] rounded-full p-2 text-white/70 transition hover:bg-white/10 hover:text-white"
			onclick={onClose}
		>
			<X size={32} />
		</button>

		<!-- Navigation -->
		{#if media.length > 1}
			<button
				class="absolute left-4 z-[101] rounded-full p-3 text-white/70 transition hover:bg-white/10 hover:text-white"
				onclick={(e) => {
					e.stopPropagation();
					prev();
				}}
			>
				<ChevronLeft size={40} />
			</button>

			<button
				class="absolute right-4 z-[101] rounded-full p-3 text-white/70 transition hover:bg-white/10 hover:text-white"
				onclick={(e) => {
					e.stopPropagation();
					next();
				}}
			>
				<ChevronRight size={40} />
			</button>
		{/if}

		<!-- Content -->
		<div class="relative flex h-full w-full items-center justify-center p-4 md:p-12">
			{#key currentIndex}
				<div
					class="relative max-h-full max-w-full"
					transition:scale={{ start: 0.9, duration: 200 }}
				>
					{#if media[currentIndex].type === 'image'}
						<img
							src={media[currentIndex].url}
							alt="Media"
							class="max-h-[85vh] max-w-full rounded-lg object-contain shadow-2xl"
						/>
					{:else if media[currentIndex].type === 'video'}
						<video
							src={media[currentIndex].url}
							controls
							autoplay
							class="max-h-[85vh] max-w-full rounded-lg shadow-2xl"
						>
							<track kind="captions" />
						</video>
					{/if}
				</div>
			{/key}

			<!-- Counter -->
			{#if media.length > 1}
				<div
					class="absolute bottom-6 left-1/2 -translate-x-1/2 rounded-full bg-black/50 px-4 py-1 text-sm text-white backdrop-blur"
				>
					{currentIndex + 1} / {media.length}
				</div>
			{/if}
		</div>
	</div>
{/if}
