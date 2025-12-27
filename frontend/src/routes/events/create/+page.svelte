<script lang="ts">
	import { goto } from '$app/navigation';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import { Textarea } from '$lib/components/ui/textarea';
	import {
		Calendar,
		Clock,
		MapPin,
		Image as ImageIcon,
		Globe,
		Lock,
		Users,
		Loader2
	} from '@lucide/svelte';
	import EventsSidebar from '$lib/components/events/EventsSidebar.svelte';
	import { createEvent, type CreateEventRequest, type EventPrivacy, uploadFiles } from '$lib/api';

	let step = $state(1);
	let submitting = $state(false);

	// Form State
	let form = $state({
		title: '',
		start_date: '',
		start_time: '',
		end_date: '',
		end_time: '',
		location: '',
		is_online: false,
		description: '',
		privacy: 'public' as EventPrivacy,
		cover_image: ''
	});

	let fileInput: HTMLInputElement;

	function nextStep() {
		step++;
	}

	function prevStep() {
		step--;
	}

	async function handleImageUpload(e: Event) {
		const files = (e.target as HTMLInputElement).files;
		if (files && files.length > 0) {
			try {
				const uploaded = await uploadFiles(Array.from(files));
				if (uploaded.length > 0) {
					form.cover_image = uploaded[0].url;
				}
			} catch (err) {
				console.error('Upload failed:', err);
				alert('Failed to upload image');
			}
		}
	}

	async function submitEvent() {
		submitting = true;
		try {
			// Combine Date and Time
			const startDateTime = new Date(`${form.start_date}T${form.start_time || '00:00'}`);
			const endDateTime = form.end_date
				? new Date(`${form.end_date}T${form.end_time || '23:59'}`)
				: undefined;

			const payload: CreateEventRequest = {
				title: form.title,
				description: form.description,
				start_date: startDateTime.toISOString(),
				end_date: endDateTime?.toISOString(),
				location: form.location,
				is_online: form.is_online,
				privacy: form.privacy,
				category: 'General', // Default for now
				cover_image:
					form.cover_image ||
					'https://images.unsplash.com/photo-1492684223066-81342ee5ff30?q=80&w=2940' // Fallback
			};

			const newEvent = await createEvent(payload);
			goto(`/events/${newEvent.id}`);
		} catch (err) {
			console.error('Failed to create event:', err);
			alert('Failed to create event. Please try again.');
		} finally {
			submitting = false;
		}
	}
</script>

<div class="bg-background text-foreground flex h-[calc(100vh-4rem)] w-full overflow-hidden">
	<EventsSidebar />

	<div class="flex-1 overflow-y-auto p-4 md:p-8">
		<div class="mx-auto max-w-2xl pb-20">
			<div class="bg-card w-full rounded-2xl border border-white/5 p-6 shadow-xl md:p-10">
				<!-- Progress -->
				<div class="mb-8 flex items-center justify-between">
					<h1 class="text-2xl font-bold">Create Event</h1>
					<div class="text-muted-foreground text-sm">Step {step} of 3</div>
				</div>

				<!-- Step 1: Basic Info -->
				{#if step === 1}
					<div class="animate-in fade-in slide-in-from-right-4 space-y-6 duration-300">
						<div class="space-y-2">
							<Label>Event Name</Label>
							<Input
								placeholder="E.g., Summer Rooftop Party"
								bind:value={form.title}
								class="bg-secondary/30"
							/>
						</div>

						<div class="grid gap-4 md:grid-cols-2">
							<div class="space-y-2">
								<Label>Start Date</Label>
								<div class="relative">
									<Calendar
										class="text-muted-foreground absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2"
									/>
									<Input type="date" class="bg-secondary/30 pl-9" bind:value={form.start_date} />
								</div>
							</div>
							<div class="space-y-2">
								<Label>Start Time</Label>
								<div class="relative">
									<Clock
										class="text-muted-foreground absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2"
									/>
									<Input type="time" class="bg-secondary/30 pl-9" bind:value={form.start_time} />
								</div>
							</div>
						</div>

						<div class="grid gap-4 md:grid-cols-2">
							<div class="space-y-2">
								<Label>End Date</Label>
								<div class="relative">
									<Calendar
										class="text-muted-foreground absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2"
									/>
									<Input type="date" class="bg-secondary/30 pl-9" bind:value={form.end_date} />
								</div>
							</div>
							<div class="space-y-2">
								<Label>End Time</Label>
								<div class="relative">
									<Clock
										class="text-muted-foreground absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2"
									/>
									<Input type="time" class="bg-secondary/30 pl-9" bind:value={form.end_time} />
								</div>
							</div>
						</div>

						<div class="space-y-2">
							<Label>Privacy</Label>
							<div class="grid grid-cols-3 gap-2">
								<button
									class="hover:bg-secondary/50 flex flex-col items-center gap-2 rounded-lg border p-4 text-sm transition-all {form.privacy ===
									'public'
										? 'border-primary bg-secondary/50'
										: 'border-white/5'}"
									onclick={() => (form.privacy = 'public')}
								>
									<Globe size={20} />
									<span>Public</span>
								</button>
								<button
									class="hover:bg-secondary/50 flex flex-col items-center gap-2 rounded-lg border p-4 text-sm transition-all {form.privacy ===
									'friends'
										? 'border-primary bg-secondary/50'
										: 'border-white/5'}"
									onclick={() => (form.privacy = 'friends')}
								>
									<Users size={20} />
									<span>Friends</span>
								</button>
								<button
									class="hover:bg-secondary/50 flex flex-col items-center gap-2 rounded-lg border p-4 text-sm transition-all {form.privacy ===
									'private'
										? 'border-primary bg-secondary/50'
										: 'border-white/5'}"
									onclick={() => (form.privacy = 'private')}
								>
									<Lock size={20} />
									<span>Private</span>
								</button>
							</div>
						</div>
					</div>
				{/if}

				<!-- Step 2: Location & Details -->
				{#if step === 2}
					<div class="animate-in fade-in slide-in-from-right-4 space-y-6 duration-300">
						<div class="space-y-2">
							<Label>Location</Label>
							<div class="relative">
								<MapPin
									class="text-muted-foreground absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2"
								/>
								<Input
									placeholder="Search for a location or address"
									bind:value={form.location}
									class="bg-secondary/30 pl-9"
								/>
							</div>
						</div>

						<div class="space-y-2">
							<Label>Description</Label>
							<Textarea
								placeholder="What are the details? (Agenda, Dress code, etc.)"
								bind:value={form.description}
								class="bg-secondary/30 min-h-[150px]"
							/>
						</div>
					</div>
				{/if}

				<!-- Step 3: Media & Review -->
				{#if step === 3}
					<div class="animate-in fade-in slide-in-from-right-4 space-y-6 duration-300">
						<div class="space-y-2">
							<Label>Cover Photo</Label>
							<input
								type="file"
								accept="image/*"
								class="hidden"
								bind:this={fileInput}
								onchange={handleImageUpload}
							/>
							<!-- svelte-ignore a11y_click_events_have_key_events -->
							<!-- svelte-ignore a11y_no_static_element_interactions -->
							<div
								class="bg-secondary/10 hover:bg-secondary/20 relative flex aspect-[21/9] w-full cursor-pointer flex-col items-center justify-center gap-4 overflow-hidden rounded-xl border-2 border-dashed border-white/10 transition-all hover:border-white/20"
								onclick={() => fileInput.click()}
							>
								{#if form.cover_image}
									<img
										src={form.cover_image}
										alt="Cover"
										class="absolute inset-0 h-full w-full object-cover"
									/>
									<div
										class="absolute inset-0 flex items-center justify-center bg-black/40 opacity-0 transition-opacity hover:opacity-100"
									>
										<span class="font-semibold text-white">Change Photo</span>
									</div>
								{:else}
									<div
										class="bg-secondary/50 flex h-12 w-12 items-center justify-center rounded-full"
									>
										<ImageIcon class="text-muted-foreground" />
									</div>
									<div class="text-center">
										<div class="font-semibold">Click to upload</div>
										<div class="text-muted-foreground text-xs">Recommended: 1920 x 1080</div>
									</div>
								{/if}
							</div>
						</div>

						<div class="bg-secondary/20 rounded-xl p-4">
							<h3 class="mb-2 font-bold">Summary</h3>
							<div class="space-y-2 text-sm">
								<div class="flex justify-between">
									<span class="text-muted-foreground">Event:</span>
									<span>{form.title || 'Untitled Event'}</span>
								</div>
								<div class="flex justify-between">
									<span class="text-muted-foreground">Date:</span>
									<span>{form.start_date || 'TBD'}</span>
								</div>
								<div class="flex justify-between">
									<span class="text-muted-foreground">Location:</span>
									<span>{form.location || 'TBD'}</span>
								</div>
								<div class="flex justify-between">
									<span class="text-muted-foreground">Privacy:</span>
									<span class="capitalize">{form.privacy}</span>
								</div>
							</div>
						</div>
					</div>
				{/if}

				<!-- Actions -->
				<div class="mt-8 flex justify-between border-t border-white/5 pt-6">
					{#if step > 1}
						<Button variant="outline" onclick={prevStep}>Back</Button>
					{:else}
						<div></div>
						<!-- Spacer -->
					{/if}

					{#if step < 3}
						<Button onclick={nextStep} class="w-32">Next</Button>
					{:else}
						<Button
							class="bg-primary text-primary-foreground hover:bg-primary/90 w-32"
							onclick={submitEvent}
							disabled={submitting}
						>
							{#if submitting}
								<Loader2 class="mr-2 h-4 w-4 animate-spin" />
							{/if}
							Create Event
						</Button>
						>
					{/if}
				</div>
			</div>
		</div>
	</div>
</div>
