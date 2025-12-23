<script lang="ts">
	import { lightbox } from '$lib/stores/lightbox.svelte';
	import { fade, scale } from 'svelte/transition';
	import { cubicOut } from 'svelte/easing';
	import { onMount } from 'svelte';

	function handleKeydown(e: KeyboardEvent) {
		if (!lightbox.isOpen) return;

		switch (e.key) {
			case 'Escape':
				lightbox.close();
				break;
			case 'ArrowRight':
				lightbox.next();
				break;
			case 'ArrowLeft':
				lightbox.prev();
				break;
		}
	}
</script>

<svelte:window on:keydown={handleKeydown} />

{#if lightbox.isOpen && lightbox.currentItem}
	<!-- Backdrop -->
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/90 backdrop-blur-sm"
		transition:fade={{ duration: 200 }}
		on:click|self={() => lightbox.close()}
		role="dialog"
		aria-modal="true"
	>
		<!-- Close Button -->
		<button
			class="absolute right-4 top-4 z-50 rounded-full bg-white/10 p-2 text-white transition-colors hover:bg-white/20"
			on:click={() => lightbox.close()}
		>
			<svg
				xmlns="http://www.w3.org/2000/svg"
				class="h-6 w-6"
				fill="none"
				viewBox="0 0 24 24"
				stroke="currentColor"
			>
				<path
					stroke-linecap="round"
					stroke-linejoin="round"
					stroke-width="2"
					d="M6 18L18 6M6 6l12 12"
				/>
			</svg>
		</button>

		<!-- Content -->
		<div
			class="relative flex h-full w-full items-center justify-center p-4 md:p-8"
			transition:scale={{ duration: 300, start: 0.9, easing: cubicOut, opacity: 0 }}
		>
			<!-- Prev Button -->
			{#if lightbox.currentIndex > 0}
				<button
					class="absolute left-4 top-1/2 z-50 -translate-y-1/2 rounded-full bg-white/10 p-3 text-white backdrop-blur-md transition-all hover:scale-110 hover:bg-white/20"
					on:click|stopPropagation={() => lightbox.prev()}
				>
					<svg
						xmlns="http://www.w3.org/2000/svg"
						class="h-6 w-6"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M15 19l-7-7 7-7"
						/>
					</svg>
				</button>
			{/if}

			<!-- Media Item -->
			{#key lightbox.currentIndex}
				<div
					class="relative max-h-full max-w-full"
					in:scale={{ start: 0.95, duration: 300, easing: cubicOut }}
				>
					{#if lightbox.currentItem.type === 'video'}
						<!-- svelte-ignore a11y-media-has-caption -->
						<video
							src={lightbox.currentItem.url}
							controls
							autoplay
							class="max-h-[85vh] max-w-full rounded-lg shadow-2xl"
						></video>
					{:else}
						<img
							src={lightbox.currentItem.url}
							alt={lightbox.currentItem.name || 'Media'}
							class="max-h-[85vh] max-w-full rounded-lg object-contain shadow-2xl"
						/>
					{/if}
					<!-- Caption/Counter -->
					<div class="absolute -bottom-10 left-0 right-0 text-center font-medium text-white/80">
						{lightbox.currentIndex + 1} / {lightbox.media.length}
					</div>
				</div>
			{/key}

			<!-- Next Button -->
			{#if lightbox.currentIndex < lightbox.media.length - 1}
				<button
					class="absolute right-4 top-1/2 z-50 -translate-y-1/2 rounded-full bg-white/10 p-3 text-white backdrop-blur-md transition-all hover:scale-110 hover:bg-white/20"
					on:click|stopPropagation={() => lightbox.next()}
				>
					<svg
						xmlns="http://www.w3.org/2000/svg"
						class="h-6 w-6"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M9 5l7 7-7 7"
						/>
					</svg>
				</button>
			{/if}
		</div>
	</div>
{/if}
