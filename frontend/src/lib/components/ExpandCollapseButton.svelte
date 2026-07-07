<script lang="ts">
	import { blur } from 'svelte/transition';
	import ChevronDown from '@lucide/svelte/icons/chevron-down';
	import ChevronUp from '@lucide/svelte/icons/chevron-up';

	let { expanded, onToggle }: { expanded: boolean; onToggle: () => void } = $props();
</script>

<button
	type="button"
	class="flex h-12 w-12 shrink-0 items-center justify-center rounded-full bg-accent text-bg shadow-lg shadow-black/30 transition-transform active:scale-95"
	onclick={onToggle}
	aria-label={expanded ? 'Collapse all lines' : 'Expand all lines'}
	in:blur={{ duration: 260, amount: 7 }}
	out:blur={{ duration: 220, amount: 7 }}
>
	{#if expanded}
		<ChevronUp size={24} strokeWidth={2.5} />
	{:else}
		<ChevronDown size={24} strokeWidth={2.5} />
	{/if}
</button>
