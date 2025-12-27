<script lang="ts">
	import { clickOutside } from '$lib/actions/clickOutside';
	import { ChevronDown, Check } from '@lucide/svelte';
	import { slide } from 'svelte/transition';

	export let value: string;
	export let options: { value: string; label: string; icon?: any }[] = [];
	export let placeholder: string = 'Select...';
	export let disabled: boolean = false;
	export let style: string = '';
	export let triggerClass: string = '';

	let isOpen = false;

	function toggle() {
		if (!disabled) {
			isOpen = !isOpen;
		}
	}

	function close() {
		isOpen = false;
	}

	function select(optionValue: string) {
		value = optionValue;
		close();
	}

	$: selectedOption = options.find((o) => o.value === value);
</script>

<div class="relative inline-block text-left {style}" use:clickOutside={close}>
	<button
		type="button"
		class="glass-input text-foreground focus:ring-primary inline-flex w-full items-center justify-between rounded-lg border border-white/10 px-3 py-2 text-sm font-medium shadow-sm hover:bg-white/5 focus:outline-none focus:ring-2 focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 {triggerClass}"
		onclick={toggle}
		{disabled}
		aria-haspopup="listbox"
		aria-expanded={isOpen}
	>
		<span class="flex items-center truncate">
			{#if selectedOption?.icon}
				<svelte:component this={selectedOption.icon} size={16} class="text-muted-foreground mr-2" />
			{/if}
			{selectedOption ? selectedOption.label : placeholder}
		</span>
		<ChevronDown
			size={16}
			class="text-muted-foreground -mr-1 ml-2 h-4 w-4 transition-transform duration-200 {isOpen
				? 'rotate-180'
				: ''}"
		/>
	</button>

	{#if isOpen}
		<div
			class="bg-popover absolute right-0 z-[99999999999999] mt-2 w-full origin-top-right overflow-hidden rounded-md border border-white/10 shadow-xl outline-none"
			transition:slide={{ duration: 150 }}
			role="listbox"
			tabindex="-1"
		>
			<div class="py-1">
				{#each options as option (option.value)}
					<button
						type="button"
						class="text-foreground flex w-full items-center px-4 py-2 text-left text-sm transition-colors hover:bg-white/10 {value ===
						option.value
							? 'bg-white/20 font-medium'
							: ''}"
						onclick={() => select(option.value)}
						role="option"
						aria-selected={value === option.value}
					>
						<span class="flex flex-1 items-center truncate">
							{#if option.icon}
								<svelte:component this={option.icon} size={16} class="text-muted-foreground mr-2" />
							{/if}
							{option.label}
						</span>
						{#if value === option.value}
							<Check size={16} class="text-primary" />
						{/if}
					</button>
				{/each}
			</div>
		</div>
	{/if}
</div>
