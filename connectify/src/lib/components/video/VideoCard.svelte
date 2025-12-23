<script lang="ts">
	import { Avatar, AvatarFallback, AvatarImage } from '$lib/components/ui/avatar';
	import { Button } from '$lib/components/ui/button';
	import { ThumbsUp, MessageCircle, Share2, MoreHorizontal } from '@lucide/svelte';

	let { video } = $props();

	let isPlaying = $state(false);
</script>

<div class="glass-card bg-card mb-4 overflow-hidden rounded-xl">
	<!-- Header -->
	<div class="flex items-center justify-between p-4">
		<div class="flex items-center space-x-3">
			<Avatar class="h-10 w-10">
				<AvatarImage src={video.author.avatar} />
				<AvatarFallback>{video.author.name[0]}</AvatarFallback>
			</Avatar>
			<div>
				<h3 class="text-foreground cursor-pointer font-semibold hover:underline">
					{video.author.name}
				</h3>
				<span class="text-muted-foreground text-xs"
					>{video.date} • <span class="font-medium text-blue-500">Follow</span></span
				>
			</div>
		</div>
		<Button variant="ghost" size="icon" class="rounded-full">
			<MoreHorizontal size={20} />
		</Button>
	</div>

	<!-- Title/Description -->
	<div class="px-4 pb-3">
		<p class="text-sm font-medium leading-normal">{video.title}</p>
	</div>

	<!-- Video Player (Mock/Real) -->
	<div class="relative aspect-video w-full bg-black">
		<video
			src={video.video_url}
			class="h-full w-full object-contain"
			controls
			poster={video.thumbnail}
		>
			<track kind="captions" />
		</video>
	</div>

	<!-- Stats & Actions -->
	<div class="p-4">
		<div class="text-muted-foreground mb-3 flex items-center justify-between text-xs">
			<div class="flex items-center gap-1">
				<div class="flex -space-x-1">
					<div
						class="flex h-4 w-4 items-center justify-center rounded-full bg-blue-500 text-[10px] text-white"
					>
						<ThumbsUp size={10} fill="white" />
					</div>
				</div>
				<span>{video.stats?.likes || '1.2K'}</span>
			</div>
			<div class="flex gap-3">
				<span>{video.stats?.comments || '240'} comments</span>
				<span>{video.stats?.shares || '56'} shares</span>
				<span>{video.views} views</span>
			</div>
		</div>

		<hr class="mb-2 border-white/10" />

		<div class="flex items-center justify-between">
			<Button variant="ghost" class="text-muted-foreground hover:text-foreground flex-1 gap-2">
				<ThumbsUp size={20} /> Like
			</Button>
			<Button variant="ghost" class="text-muted-foreground hover:text-foreground flex-1 gap-2">
				<MessageCircle size={20} /> Comment
			</Button>
			<Button variant="ghost" class="text-muted-foreground hover:text-foreground flex-1 gap-2">
				<Share2 size={20} /> Share
			</Button>
		</div>
	</div>
</div>
