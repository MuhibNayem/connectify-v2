<script lang="ts">
	import { callState, voiceCallService } from '$lib/stores/voice-call.svelte';
	import { Mic, MicOff, PhoneOff, User } from '@lucide/svelte';
	import { onMount } from 'svelte';
	import { draggable } from '$lib/actions/draggable';
	import Avatar from '$lib/components/ui/avatar/avatar.svelte';
	import AvatarFallback from '$lib/components/ui/avatar/avatar-fallback.svelte';
	import AvatarImage from '$lib/components/ui/avatar/avatar-image.svelte';
	import { getUserByID } from '$lib/api'; // Import API

	let isMuted = $state(false);
	let duration = $state(0);
	let interval: ReturnType<typeof setInterval>;
	let audioElement: HTMLAudioElement; // Audio element reference
	let localVideoElement: HTMLVideoElement;
	let remoteVideoElement: HTMLVideoElement;
	let userInfo = $state<{ full_name?: string; username: string; avatar?: string } | null>(null);

	// Derive connection status from store
	let status = $derived($callState.status);
	let targetId = $derived($callState.targetId || $callState.callerId);
	let remoteStream = $derived($callState.remoteStream);
	let localStream = $derived($callState.localStream);
	let callType = $derived($callState.callType);

	// Fetch user info when targetId is available, resiliently
	$effect(() => {
		if (targetId && targetId !== 'Unknown') {
			getUserByID(targetId)
				.then((data) => {
					if (data) {
						userInfo = {
							full_name: data.full_name,
							username: data.username,
							avatar: data.avatar
						};
					}
				})
				.catch((err) => {
					console.error('[CallInterface] Failed to fetch user info:', err);
					// Don't clear userInfo or crash, just log error.
				});
		}
	});

	function toggleMute() {
		isMuted = !isMuted;
		if ($callState.localStream) {
			$callState.localStream.getAudioTracks().forEach((track) => {
				track.enabled = !isMuted;
			});
		}
	}

	function endCall() {
		voiceCallService.endCall();
	}

	onMount(() => {
		interval = setInterval(() => {
			if (status === 'connected') {
				duration++;
			}
		}, 1000);

		return () => clearInterval(interval);
	});

	function formatDuration(seconds: number) {
		const mins = Math.floor(seconds / 60);
		const secs = seconds % 60;
		return `${mins.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`;
	}

	// Handle audio states
	$effect(() => {
		if (status === 'calling' && audioElement) {
			// Play Ringback tone (Soothing)
			audioElement.src = 'https://assets.mixkit.co/active_storage/sfx/2865/2865-preview.mp3';
			audioElement.loop = true;
			audioElement.volume = 0.2; // Lower volume to 20%
			audioElement.play().catch((e) => console.error('Error playing ringback:', e));
		} else if (status === 'connected' && audioElement) {
			// Stop ringing, switch to remote stream
			audioElement.pause();
			audioElement.srcObject = remoteStream;
			audioElement.loop = false;
			audioElement.volume = 1.0; // Reset volume for voice
			audioElement.play().catch((e) => console.error('Error playing remote audio:', e));
		}
	});

	// Handle video streams
	$effect(() => {
		if (callType === 'video') {
			if (localStream && localVideoElement) {
				localVideoElement.srcObject = localStream;
			}
			if (remoteStream && remoteVideoElement) {
				remoteVideoElement.srcObject = remoteStream;
			}
		}
	});
</script>

<div class="fixed inset-0 z-50 flex items-center justify-center bg-black/80 backdrop-blur-sm">
	<!-- Audio element for playback -->
	<audio bind:this={audioElement} autoplay playsinline controls={false} class="hidden"></audio>

	<div
		class="relative flex flex-col items-center gap-8 rounded-2xl border border-zinc-800 bg-zinc-900 shadow-2xl transition-all duration-200"
		class:w-full={callType === 'video'}
		class:h-full={callType === 'video'}
		class:max-w-none={callType === 'video'}
		class:p-0={callType === 'video'}
		class:game-mode={callType === 'video'}
		class:max-w-4xl={callType !== 'video'}
		class:p-8={callType !== 'video'}
		style={callType !== 'video'
			? 'resize: both; overflow: hidden; min-width: 320px; min-height: 480px;'
			: ''}
	>
		<div
			class="flex flex-col items-center gap-4 {callType === 'video'
				? 'absolute top-8 z-10 rounded-xl bg-black/50 p-4'
				: ''}"
		>
			{#if callType !== 'video'}
				<Avatar class="h-32 w-32 border-4 border-zinc-800 shadow-xl">
					<AvatarImage
						src={userInfo?.avatar || `https://api.dicebear.com/7.x/initials/svg?seed=${targetId}`}
						alt={userInfo?.username || targetId}
					/>
					<AvatarFallback><User size={48} /></AvatarFallback>
				</Avatar>
			{/if}
			<div class="text-center">
				<h2 class="text-2xl font-semibold text-white">
					{userInfo ? userInfo.full_name || userInfo.username : targetId || 'Unknown User'}
				</h2>
				<p class="text-zinc-400" class:text-white={callType === 'video'}>
					{#if status === 'connected'}
						{formatDuration(duration)}
					{:else if status === 'calling'}
						Calling...
					{:else}
						Connecting...
					{/if}
				</p>
			</div>
		</div>

		{#if callType === 'video'}
			<!-- Video Container Full Screen -->
			<div class="absolute inset-0 h-full w-full bg-black">
				<!-- Remote Video (Full Screen) -->
				<video
					bind:this={remoteVideoElement}
					autoplay
					playsinline
					class="h-full w-full object-cover"
				></video>

				<!-- Local Video (PIP) - Draggable -->
				<!-- svelte-ignore a11y_no_static_element_interactions -->
				<div
					use:draggable={{ bounds: document.body }}
					class="absolute bottom-24 right-4 z-20 h-48 w-72 cursor-grab overflow-hidden rounded-xl border-2 border-white/20 bg-black/50 shadow-2xl transition-colors hover:border-white hover:shadow-white/10 active:cursor-grabbing"
				>
					<video
						bind:this={localVideoElement}
						autoplay
						playsinline
						muted
						class="h-full w-full scale-x-[-1] transform object-cover"
					></video>
				</div>
			</div>
		{/if}

		<div
			class="mt-4 flex items-center gap-6"
			class:absolute={callType === 'video'}
			class:bottom-8={callType === 'video'}
			class:z-50={callType === 'video'}
		>
			<button
				onclick={toggleMute}
				class="group flex h-14 w-14 items-center justify-center rounded-full bg-zinc-800 transition-all hover:bg-zinc-700"
				aria-label={isMuted ? 'Unmute' : 'Mute'}
			>
				{#if isMuted}
					<MicOff class="h-6 w-6 text-red-400 transition-transform group-hover:scale-110" />
				{:else}
					<Mic class="h-6 w-6 text-white transition-transform group-hover:scale-110" />
				{/if}
			</button>

			<button
				onclick={endCall}
				class="group flex h-16 w-16 items-center justify-center rounded-full bg-red-500 shadow-lg shadow-red-500/20 transition-all hover:bg-red-600 hover:shadow-red-600/30"
				aria-label="End Call"
			>
				<PhoneOff class="h-8 w-8 text-white transition-transform group-hover:scale-110" />
			</button>
		</div>
	</div>
</div>
