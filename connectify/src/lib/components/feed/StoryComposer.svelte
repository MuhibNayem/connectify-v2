<script lang="ts">
	import { Button } from '$lib/components/ui/button';
	import { X, Globe, Users, Lock, ChevronDown, UserX, UserCheck } from '@lucide/svelte';
	import { auth } from '$lib/stores/auth.svelte';
	import FriendSelectorModal from './FriendSelectorModal.svelte';

	let { file, mediaType, activeTab, onClose, onPost } = $props<{
		file: File;
		mediaType: 'image' | 'video';
		activeTab: 'stories' | 'reels';
		onClose: () => void;
		onPost: (privacy: string, allowed: string[], blocked: string[]) => void;
	}>();

	let previewUrl = URL.createObjectURL(file);
	let privacy = $state<'PUBLIC' | 'FRIENDS' | 'ONLY_ME' | 'CUSTOM' | 'FRIENDS_EXCEPT'>('FRIENDS');

	// Privacy View State
	let showPrivacyMenu = $state(false);

	// Friend Selection State
	let showFriendSelector = $state(false);
	let selectorMode = $state<'CUSTOM' | 'FRIENDS_EXCEPT'>('CUSTOM');
	let allowedViewers = $state<string[]>([]);
	let blockedViewers = $state<string[]>([]);

	function handlePrivacySelect(type: typeof privacy) {
		privacy = type;
		showPrivacyMenu = false;

		if (type === 'CUSTOM') {
			selectorMode = 'CUSTOM';
			showFriendSelector = true;
		} else if (type === 'FRIENDS_EXCEPT') {
			selectorMode = 'FRIENDS_EXCEPT';
			showFriendSelector = true;
		}
	}

	function handleFriendSelectionSave(ids: string[]) {
		if (selectorMode === 'CUSTOM') {
			allowedViewers = ids;
		} else {
			blockedViewers = ids;
		}
		showFriendSelector = false;
	}

	function handlePost() {
		onPost(privacy, allowedViewers, blockedViewers);
	}

	function getPrivacyIcon(p: string) {
		switch (p) {
			case 'PUBLIC':
				return Globe;
			case 'FRIENDS':
				return Users;
			case 'ONLY_ME':
				return Lock;
			case 'CUSTOM':
				return UserCheck;
			case 'FRIENDS_EXCEPT':
				return UserX;
			default:
				return Users;
		}
	}

	function getPrivacyLabel(p: string) {
		switch (p) {
			case 'PUBLIC':
				return 'Public';
			case 'FRIENDS':
				return 'Friends';
			case 'ONLY_ME':
				return 'Only Me';
			case 'CUSTOM':
				return 'Specific Friends';
			case 'FRIENDS_EXCEPT':
				return 'Friends Except...';
			default:
				return 'Friends';
		}
	}

	const PreviewIcon = $derived(getPrivacyIcon(privacy));
</script>

<div class="fixed inset-0 z-40 flex flex-col items-center justify-center bg-black">
	<!-- Top Bar -->
	<div
		class="absolute left-0 right-0 top-0 z-50 flex items-center justify-between bg-gradient-to-b from-black/80 to-transparent p-4"
	>
		<button
			onclick={onClose}
			class="rounded-full p-2 text-white transition-colors hover:bg-white/10"
		>
			<X size={24} />
		</button>
		<div class="font-semibold text-white">Preview</div>
		<div class="w-10"></div>
		<!-- Spacer for center alignment -->
	</div>

	<!-- Main Preview Area -->
	<div class="relative flex h-full w-full items-center justify-center p-4 pb-24">
		<!-- Phone-like container for vertical stories -->
		<div
			class="relative aspect-[9/16] w-full max-w-sm overflow-hidden rounded-2xl border border-white/10 bg-black shadow-2xl"
		>
			{#if mediaType === 'video'}
				<video src={previewUrl} class="h-full w-full object-cover" autoplay muted loop playsinline
				></video>
			{:else}
				<img src={previewUrl} alt="Preview" class="h-full w-full object-cover" />
			{/if}

			<!-- Privacy Selector Overlay (Bottom Left) -->
			{#if activeTab === 'stories'}
				<div class="absolute bottom-4 left-4 z-20">
					<div class="relative">
						<button
							onclick={() => (showPrivacyMenu = !showPrivacyMenu)}
							class="flex items-center gap-2 rounded-full border border-white/20 bg-black/50 px-3 py-1.5 text-sm font-medium text-white backdrop-blur-md transition-colors hover:bg-black/70"
						>
							<PreviewIcon size={14} />
							<span>{getPrivacyLabel(privacy)}</span>
							<ChevronDown size={14} class="opacity-70" />
						</button>

						{#if showPrivacyMenu}
							<div
								class="animate-in fade-in slide-in-from-bottom-2 absolute bottom-full left-0 mb-2 w-48 overflow-hidden rounded-xl border border-white/10 bg-[#1c1c1e] shadow-xl"
							>
								<div class="flex flex-col p-1">
									{#each ['PUBLIC', 'FRIENDS', 'FRIENDS_EXCEPT', 'CUSTOM', 'ONLY_ME'] as option}
										{@const Icon = getPrivacyIcon(option)}
										<button
											onclick={() => handlePrivacySelect(option as any)}
											class="flex w-full items-center gap-3 rounded-lg p-2.5 text-left text-sm transition-colors hover:bg-white/10 {privacy ===
											option
												? 'bg-primary/20 text-primary'
												: 'text-white'}"
										>
											<Icon size={16} />
											<span>{getPrivacyLabel(option)}</span>
										</button>
									{/each}
								</div>
							</div>
						{/if}
					</div>
				</div>
			{/if}
		</div>
	</div>

	<!-- Bottom Bar (Actions) -->
	<div
		class="absolute bottom-0 left-0 right-0 flex items-center justify-end gap-4 bg-gradient-to-t from-black/90 to-transparent p-4"
	>
		<Button onclick={handlePost} class="shadow-primary/25 rounded-full px-8 font-bold shadow-lg">
			Share {activeTab === 'stories' ? 'Story' : 'Reel'}
		</Button>
	</div>

	<!-- Friend Selection Modal -->
	{#if showFriendSelector}
		<FriendSelectorModal
			exclude={selectorMode === 'FRIENDS_EXCEPT'}
			initialSelected={selectorMode === 'CUSTOM' ? allowedViewers : blockedViewers}
			onSave={handleFriendSelectionSave}
			onClose={() => (showFriendSelector = false)}
		/>
	{/if}
</div>
