<script lang="ts">
	import { callState, voiceCallService } from '$lib/stores/voice-call.svelte';
	import { Phone, PhoneOff, User, Video } from '@lucide/svelte';
	import Avatar from '$lib/components/ui/avatar/avatar.svelte';
	import AvatarImage from '$lib/components/ui/avatar/avatar-image.svelte';
	import AvatarFallback from '$lib/components/ui/avatar/avatar-fallback.svelte';

	import { getUserByID } from '$lib/api';

	let callerId = $derived($callState.callerId || 'Unknown');
	let callType = $derived($callState.callType);
	let userInfo = $state<{ full_name?: string; username: string; avatar?: string } | null>(null);

	$effect(() => {
		if (callerId && callerId !== 'Unknown') {
			getUserByID(callerId)
				.then((data) => {
					userInfo = {
						full_name: data.full_name,
						username: data.username,
						avatar: data.avatar
					};
				})
				.catch((e) => console.error('Failed to fetch caller info:', e));
		}
	});

	function acceptCall() {
		voiceCallService.acceptCall();
	}

	function rejectCall() {
		voiceCallService.rejectCall();
	}

	let audioElement: HTMLAudioElement;

	$effect(() => {
		if (audioElement) {
			// Play Ringtone (Soothing)
			audioElement.src = 'https://assets.mixkit.co/active_storage/sfx/2865/2865-preview.mp3';
			audioElement.loop = true;
			audioElement.volume = 0.2; // Lower volume to 20%
			audioElement.play().catch((e) => console.error('Error playing ringtone:', e));
		}
	});
</script>

<div class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
	<audio bind:this={audioElement} class="hidden"></audio>
	<div
		class="animate-in fade-in zoom-in-95 flex w-full max-w-sm flex-col items-center gap-6 rounded-2xl border border-zinc-800 bg-zinc-900 p-8 shadow-2xl duration-300"
	>
		<div class="flex flex-col items-center gap-3">
			<Avatar class="h-24 w-24 animate-pulse border-2 border-zinc-700">
				<AvatarImage
					src={userInfo?.avatar || `https://api.dicebear.com/7.x/initials/svg?seed=${callerId}`}
					alt={userInfo?.username || callerId}
				/>
				<AvatarFallback><User size={32} /></AvatarFallback>
			</Avatar>
			<div class="text-center">
				<h2 class="text-2xl font-semibold text-white">
					{userInfo ? userInfo.full_name || userInfo.username : callerId || 'Unknown User'}
				</h2>
				<div class="mt-1 flex items-center justify-center gap-2 text-zinc-400">
					{#if callType === 'video'}
						<Video size={16} />
						<span>Incoming Video Call...</span>
					{:else}
						<Phone size={16} />
						<span>Incoming Voice Call...</span>
					{/if}
				</div>
			</div>
		</div>

		<div class="mt-2 flex items-center gap-8">
			<button
				onclick={rejectCall}
				class="group flex flex-col items-center gap-2"
				aria-label="Decline Call"
			>
				<div
					class="flex h-14 w-14 items-center justify-center rounded-full bg-red-500 shadow-lg shadow-red-500/20 transition-transform group-hover:scale-110"
				>
					<PhoneOff class="h-6 w-6 text-white" />
				</div>
				<span class="text-xs font-medium text-zinc-400 group-hover:text-red-400">Decline</span>
			</button>

			<button
				onclick={acceptCall}
				class="group flex flex-col items-center gap-2"
				aria-label="Accept Call"
			>
				<div
					class="flex h-14 w-14 animate-bounce items-center justify-center rounded-full bg-green-500 shadow-lg shadow-green-500/20 transition-transform group-hover:scale-110"
				>
					<Phone class="h-6 w-6 fill-current text-white" />
				</div>
				<span class="text-xs font-medium text-zinc-400 group-hover:text-green-400">Accept</span>
			</button>
		</div>
	</div>
</div>
