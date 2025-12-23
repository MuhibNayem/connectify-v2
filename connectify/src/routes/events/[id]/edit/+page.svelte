<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import EventsSidebar from '$lib/components/events/EventsSidebar.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { Textarea } from '$lib/components/ui/textarea';
	import { Label } from '$lib/components/ui/label';
	import { ArrowLeft, Loader2, Save, Upload, X, Globe, MapPin } from '@lucide/svelte';
	import {
		getEvent,
		updateEvent,
		uploadFiles,
		type Event,
		type UpdateEventRequest,
		type EventPrivacy
	} from '$lib/api';

	let event: Event | null = $state(null);
	let loading = $state(true);
	let saving = $state(false);
	let error = $state('');

	// Form state
	let title = $state('');
	let description = $state('');
	let startDate = $state('');
	let startTime = $state('');
	let endDate = $state('');
	let endTime = $state('');
	let location = $state('');
	let isOnline = $state(false);
	let privacy = $state<EventPrivacy>('public');
	let category = $state('');
	let coverImage = $state('');
	let coverImageFile: File | null = $state(null);

	const categories = [
		'Music',
		'Sports',
		'Business',
		'Tech',
		'Art',
		'Food',
		'Community',
		'Gaming',
		'Education',
		'Other'
	];

	onMount(async () => {
		try {
			const id = $page.params.id;
			if (id) {
				event = await getEvent(id);

				if (!event.is_host) {
					goto(`/events/${id}`);
					return;
				}

				// Populate form
				title = event.title;
				description = event.description || '';
				location = event.location || '';
				isOnline = event.is_online;
				privacy = event.privacy;
				category = event.category || '';
				coverImage = event.cover_image || '';

				// Parse dates
				const start = new Date(event.start_date);
				startDate = start.toISOString().split('T')[0];
				startTime = start.toTimeString().slice(0, 5);

				if (event.end_date) {
					const end = new Date(event.end_date);
					endDate = end.toISOString().split('T')[0];
					endTime = end.toTimeString().slice(0, 5);
				}
			}
		} catch (err) {
			console.error('Failed to load event:', err);
			error = 'Failed to load event.';
		} finally {
			loading = false;
		}
	});

	function handleCoverImageChange(e: Event & { currentTarget: HTMLInputElement }) {
		const file = e.currentTarget.files?.[0];
		if (file) {
			coverImageFile = file;
			coverImage = URL.createObjectURL(file);
		}
	}

	function removeCoverImage() {
		coverImage = '';
		coverImageFile = null;
	}

	async function handleSubmit() {
		if (!event || !title || !startDate || !startTime) return;

		saving = true;
		try {
			// Upload cover image if changed
			let finalCoverImage = event.cover_image;
			if (coverImageFile) {
				const uploadResult = await uploadFiles([coverImageFile]);
				if (uploadResult.files?.[0]?.url) {
					finalCoverImage = uploadResult.files[0].url;
				}
			} else if (!coverImage && event.cover_image) {
				finalCoverImage = ''; // Clear image
			}

			const startDateTime = new Date(`${startDate}T${startTime}`);
			let endDateTime: Date | undefined;
			if (endDate && endTime) {
				endDateTime = new Date(`${endDate}T${endTime}`);
			}

			const updateData: UpdateEventRequest = {
				title,
				description,
				start_date: startDateTime.toISOString(),
				end_date: endDateTime?.toISOString(),
				location: isOnline ? '' : location,
				is_online: isOnline,
				privacy,
				category,
				cover_image: finalCoverImage
			};

			await updateEvent(event.id, updateData);
			goto(`/events/${event.id}`);
		} catch (err) {
			console.error('Failed to update event:', err);
			alert('Failed to update event');
		} finally {
			saving = false;
		}
	}
</script>

<div class="bg-background text-foreground flex h-[calc(100vh-4rem)] w-full overflow-hidden">
	<EventsSidebar />

	<div class="flex-1 overflow-y-auto">
		<div class="mx-auto max-w-2xl px-4 py-8">
			{#if loading}
				<div class="flex h-[50vh] items-center justify-center">
					<Loader2 class="animate-spin text-white" size={48} />
				</div>
			{:else if error || !event}
				<div class="flex h-[50vh] flex-col items-center justify-center gap-4 text-center">
					<h2 class="text-2xl font-bold">Event not found</h2>
					<p class="text-muted-foreground">{error}</p>
					<Button href="/events" variant="outline">Back to Events</Button>
				</div>
			{:else}
				<!-- Header -->
				<div class="mb-8 flex items-center gap-4">
					<Button variant="ghost" size="icon" href="/events/{event.id}">
						<ArrowLeft size={20} />
					</Button>
					<h1 class="text-2xl font-bold">Edit Event</h1>
				</div>

				<form
					onsubmit={(e) => {
						e.preventDefault();
						handleSubmit();
					}}
					class="space-y-6"
				>
					<!-- Cover Image -->
					<div>
						<Label>Cover Image</Label>
						<div
							class="relative mt-2 aspect-video w-full overflow-hidden rounded-xl border border-white/10 bg-white/5"
						>
							{#if coverImage}
								<img src={coverImage} alt="" class="h-full w-full object-cover" />
								<button
									type="button"
									class="absolute right-2 top-2 rounded-full bg-black/50 p-1 backdrop-blur-sm"
									onclick={removeCoverImage}
								>
									<X size={20} />
								</button>
							{:else}
								<label class="flex h-full cursor-pointer flex-col items-center justify-center">
									<Upload size={32} class="text-muted-foreground mb-2" />
									<span class="text-muted-foreground text-sm">Click to upload</span>
									<input
										type="file"
										accept="image/*"
										class="hidden"
										onchange={handleCoverImageChange}
									/>
								</label>
							{/if}
						</div>
					</div>

					<!-- Title -->
					<div>
						<Label for="title">Event Title</Label>
						<Input id="title" bind:value={title} placeholder="Give your event a name" required />
					</div>

					<!-- Date & Time -->
					<div class="grid gap-4 sm:grid-cols-2">
						<div>
							<Label for="startDate">Start Date</Label>
							<Input id="startDate" type="date" bind:value={startDate} required />
						</div>
						<div>
							<Label for="startTime">Start Time</Label>
							<Input id="startTime" type="time" bind:value={startTime} required />
						</div>
						<div>
							<Label for="endDate">End Date (Optional)</Label>
							<Input id="endDate" type="date" bind:value={endDate} />
						</div>
						<div>
							<Label for="endTime">End Time (Optional)</Label>
							<Input id="endTime" type="time" bind:value={endTime} />
						</div>
					</div>

					<!-- Location -->
					<div class="space-y-4">
						<div class="flex items-center justify-between">
							<Label>Event Type</Label>
							<div class="flex gap-2">
								<button
									type="button"
									class="flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-medium transition-colors
										{!isOnline ? 'bg-primary text-white' : 'bg-white/10 hover:bg-white/20'}"
									onclick={() => (isOnline = false)}
								>
									<MapPin size={16} />
									In Person
								</button>
								<button
									type="button"
									class="flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-medium transition-colors
										{isOnline ? 'bg-primary text-white' : 'bg-white/10 hover:bg-white/20'}"
									onclick={() => (isOnline = true)}
								>
									<Globe size={16} />
									Online
								</button>
							</div>
						</div>

						{#if !isOnline}
							<div>
								<Label for="location">Location</Label>
								<Input id="location" bind:value={location} placeholder="Add a location" />
							</div>
						{/if}
					</div>

					<!-- Category -->
					<div>
						<Label>Category</Label>
						<div class="mt-2 flex flex-wrap gap-2">
							{#each categories as cat}
								<button
									type="button"
									class="rounded-full px-4 py-2 text-sm font-medium transition-colors
										{category === cat ? 'bg-primary text-white' : 'bg-white/10 hover:bg-white/20'}"
									onclick={() => (category = cat)}
								>
									{cat}
								</button>
							{/each}
						</div>
					</div>

					<!-- Privacy -->
					<div>
						<Label>Privacy</Label>
						<div class="mt-2 flex gap-2">
							{#each [{ value: 'public', label: 'Public' }, { value: 'friends', label: 'Friends Only' }, { value: 'private', label: 'Private' }] as opt}
								<button
									type="button"
									class="flex-1 rounded-lg px-4 py-3 text-sm font-medium transition-colors
										{privacy === opt.value ? 'bg-primary text-white' : 'bg-white/10 hover:bg-white/20'}"
									onclick={() => (privacy = opt.value as EventPrivacy)}
								>
									{opt.label}
								</button>
							{/each}
						</div>
					</div>

					<!-- Description -->
					<div>
						<Label for="description">Description</Label>
						<Textarea
							id="description"
							bind:value={description}
							placeholder="What's your event about?"
							rows={5}
						/>
					</div>

					<!-- Submit -->
					<div class="flex justify-end gap-4 pt-4">
						<Button variant="ghost" href="/events/{event.id}">Cancel</Button>
						<Button type="submit" class="gap-2" disabled={saving || !title}>
							{#if saving}
								<Loader2 class="h-4 w-4 animate-spin" />
							{:else}
								<Save size={16} />
							{/if}
							Save Changes
						</Button>
					</div>
				</form>
			{/if}
		</div>
	</div>
</div>
