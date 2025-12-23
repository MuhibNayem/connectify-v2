	let userReactionEmoji: string | null = null;
	$: hasUserReacted = userReactionEmoji !== null;

	let isEditing = false;
	let editedContent = message.content;

	onMount(() => {
		// Check if current user has already reacted to this message
		if (currentUserId && message.reactions) {
			const userReaction = message.reactions.find((r) => r.user_id === currentUserId);
			if (userReaction) {
				userReactionEmoji = userReaction.emoji;
			}
		}

		const unsubscribe = websocketMessages.subscribe((event) => {
			if (!event?.data) return;

			// Handle MESSAGE_REACTION_UPDATE event
			if (event.type === 'MESSAGE_REACTION_UPDATE' && event.data.message_id === message.id) {
				const reactionEvent = event.data;

				if (reactionEvent.action === 'add') {
					// Add reaction if not already present for this user/emoji
					if (!message.reactions.some((r) => r.user_id === reactionEvent.user_id && r.emoji === reactionEvent.emoji)) {
						message.reactions = [
							...message.reactions,
							{ user_id: reactionEvent.user_id, emoji: reactionEvent.emoji, timestamp: reactionEvent.timestamp }
						];
					}
					// Update current user's reaction if it's their reaction
					if (reactionEvent.user_id === currentUserId) {
						userReactionEmoji = reactionEvent.emoji;
					}
				} else if (reactionEvent.action === 'remove') {
					// Remove reaction
					message.reactions = message.reactions.filter(
						(r) => !(r.user_id === reactionEvent.user_id && r.emoji === reactionEvent.emoji)
					);
					// Clear current user's reaction if it was theirs
					if (reactionEvent.user_id === currentUserId && userReactionEmoji === reactionEvent.emoji) {
						userReactionEmoji = null;
					}
				}
			}
		});

		return () => {
			unsubscribe();
		};
	});

	async function handleReaction(emoji: string) {
		if (!currentUserId) {
			alert('Please log in to react.');
			return;
		}

		try {
			if (hasUserReacted && userReactionEmoji === emoji) {
				// User is removing their existing reaction
				await apiRequest('DELETE', `/messages/${message.id}/react`, { emoji });
				// Optimistic update
				message.reactions = message.reactions.filter(
					(r) => !(r.user_id === currentUserId && r.emoji === emoji)
				);
				userReactionEmoji = null;
			} else {
				// User is adding a new reaction or changing their reaction
				// First, remove existing reaction if any
				if (hasUserReacted) {
					await apiRequest('DELETE', `/messages/${message.id}/react`, { emoji: userReactionEmoji });
					message.reactions = message.reactions.filter((r) => r.user_id !== currentUserId);
				}
				// Then add the new reaction
				await apiRequest('POST', `/messages/${message.id}/react`, { emoji });
				// Optimistic update
				message.reactions = [
					...message.reactions,
					{ user_id: currentUserId, emoji, timestamp: new Date().toISOString() }
				];
				userReactionEmoji = emoji;
			}
		} catch (e: any) {
			alert(`Failed to toggle reaction: ${e.message}`);
			console.error('Toggle reaction error:', e);
		}
	}

	async function handleEdit() {
		isEditing = true;
		editedContent = message.content; // Initialize with current content
	}

	async function handleSaveEdit() {
		if (editedContent.trim() === '' || editedContent === message.content) {
			isEditing = false;
			return;
		}

		const originalContent = message.content;
		const wasEdited = message.is_edited;

		// Optimistic update
		message.content = editedContent;
		message.is_edited = true;
		isEditing = false;

		try {
			await apiRequest('PUT', `/messages/${message.id}`, { content: editedContent });
			// No need to wait for response to update UI, we already did.
			// Only update metadata if needed from response, but content is key.
		} catch (e: any) {
			// Revert on failure
			message.content = originalContent;
			message.is_edited = wasEdited;
			isEditing = true;
			alert(`Failed to edit message: ${e.message}`);
			console.error('Edit message error:', e);
		}
	}

	function handleCancelEdit() {
		isEditing = false;
		editedContent = message.content;
	}

	async function handleDelete() {
		if (!confirm('Are you sure you want to delete this message?')) {
			return;
		}

		// Optimistic update
		const wasDeleted = message.is_deleted;
		message.is_deleted = true;
		
		try {
			await apiRequest('DELETE', `/messages/${message.id}`);
		} catch (e: any) {
			// Revert
			message.is_deleted = wasDeleted;
			alert(`Failed to delete message: ${e.message}`);
			console.error('Delete message error:', e);
		}
	}

	// Group reactions by emoji and count them
	$: groupedReactions = message.reactions?.reduce((acc, reaction) => {
		acc[reaction.emoji] = (acc[reaction.emoji] || 0) + 1;
		return acc;
	}, {} as { [key: string]: number });

	// Get unique emojis for display
	$: uniqueEmojis = Object.keys(groupedReactions || {});
</script>

<div class="flex items-start space-x-3 p-4">
	<Avatar class="h-8 w-8">
		<AvatarImage src={message.sender?.avatar || 'https://github.com/shadcn.png'} alt={message.sender?.username} />
		<AvatarFallback>{message.sender?.username?.charAt(0).toUpperCase()}</AvatarFallback>
	</Avatar>
	<div class="flex-1">
		<div class="flex items-baseline space-x-2">
			<p class="font-semibold text-gray-900">{message.sender?.full_name || message.sender?.username}</p>
			<p class="text-xs text-gray-500">
				{formatDistanceToNow(new Date(message.created_at), { addSuffix: true })}
				{#if message.is_edited}
					<span class="ml-1 text-gray-400">(Edited)</span>
				{/if}
				{#if message.seen_by && currentUserId && message.seen_by.includes(currentUserId)}
					<span class="ml-1 text-blue-500">✓✓</span>
				{/if}
			</p>
		</div>
		{#if isEditing}
			<textarea
				bind:value={editedContent}
				class="w-full p-2 border rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
				rows="3"
			></textarea>
			<div class="mt-2 flex space-x-2">
				<Button size="sm" on:click={handleSaveEdit}>Save</Button>
				<Button variant="ghost" size="sm" on:click={handleCancelEdit}>Cancel</Button>
			</div>
		{:else}
			<p class="leading-relaxed text-gray-800">{message.content}</p>
		{/if}

		{#if currentUserId === message.sender_id && !message.is_deleted}
			<div class="mt-2 flex space-x-2">
				<Button variant="ghost" size="sm" on:click={handleEdit}>Edit</Button>
				<Button variant="ghost" size="sm" on:click={handleDelete}>Delete</Button>
			</div>
		{/if}

		{#if uniqueEmojis.length > 0}
			<div class="mt-2 flex space-x-2">
				{#each uniqueEmojis as emoji (emoji)}
					<div
						class="flex items-center space-x-1 rounded-full bg-gray-100 px-2 py-1 text-xs text-gray-600"
					>
						<span>{emoji}</span>
						<span>{groupedReactions[emoji]}</span>
					</div>
				{/each}
			</div>
		{/if}

		<div class="mt-2 flex space-x-2">
			<Button variant="ghost" size="sm" on:click={() => handleReaction('👍')}>
				👍 {userReactionEmoji === '👍' ? 'Unlike' : 'Like'}
			</Button>
			<Button variant="ghost" size="sm" on:click={() => handleReaction('❤️')}>
				❤️ {userReactionEmoji === '❤️' ? 'Unlove' : 'Love'}
			</Button>
			<Button variant="ghost" size="sm" on:click={() => handleReaction('😂')}>
				😂 {userReactionEmoji === '😂' ? 'Unlaugh' : 'Laugh'}
			</Button>
			<Button variant="ghost" size="sm" on:click={() => dispatch('replyMessage', message)}>
				Reply
			</Button>
		</div>
	</div>
</div>
