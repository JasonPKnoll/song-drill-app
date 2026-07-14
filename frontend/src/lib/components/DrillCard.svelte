<script lang="ts">
	import type { Snippet } from 'svelte';
	import FlipCard from './FlipCard.svelte';
	import { cn } from '$lib/utils/cn';

	let {
		front,
		back,
		onGotIt,
		onMissed
	}: {
		front: Snippet;
		back: Snippet;
		onGotIt: () => void;
		onMissed: () => void;
	} = $props();

	let revealed = $state(false);
</script>

<div
	class={cn(
		'p-8',
		'border border-border',
		'bg-surface',
		'rounded-2xl shadow-lg'
	)}
>
	<FlipCard
		flipped={revealed}
		onToggle={() => (revealed = true)}
		disabled={revealed}
		frontClass="flex flex-col items-center justify-center gap-4 text-center"
		backClass="flex flex-col items-center justify-center gap-3 overflow-y-auto text-center"
		{front}
		{back}
	/>

	{#if !revealed}
		<p class="mt-4 text-center text-sm text-muted">Tap to reveal</p>
	{:else}
		<div class="mt-6 flex gap-3">
			<button
				type="button"
				class={cn(
					'flex-1 py-3',
					'font-medium',
					'border border-bad bg-bad/10 text-bad',
					'rounded-xl transition hover:bg-bad/20'
				)}
				onclick={onMissed}
			>
				Missed
			</button>
			<button
				type="button"
				class={cn(
					'flex-1 py-3',
					'font-medium',
					'border border-good bg-good/10 text-good',
					'rounded-xl transition hover:bg-good/20'
				)}
				onclick={onGotIt}
			>
				Got it
			</button>
		</div>
	{/if}
</div>
