<script lang="ts">
	import type { Snippet } from 'svelte';
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
	<button
		type="button"
		class={cn('flip-card', 'h-64 w-full', 'text-center')}
		onclick={() => (revealed = true)}
		disabled={revealed}
		aria-pressed={revealed}
	>
		<div class="flip-card-inner" class:flipped={revealed}>
			<div
				class={cn(
					'flip-card-face flex flex-col items-center justify-center gap-4',
					'text-center'
				)}
			>
				{@render front()}
			</div>
			<div
				class={cn(
					'flip-card-face flip-card-back flex flex-col items-center justify-center gap-3 overflow-y-auto',
					'text-center'
				)}
			>
				{@render back()}
			</div>
		</div>
	</button>

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

<style>
	.flip-card {
		perspective: 1200px;
		display: block;
	}
	.flip-card-inner {
		position: relative;
		height: 100%;
		width: 100%;
		transition: transform 0.5s;
		transform-style: preserve-3d;
	}
	.flip-card-inner.flipped {
		transform: rotateY(180deg);
	}
	.flip-card-face {
		position: absolute;
		inset: 0;
		backface-visibility: hidden;
	}
	.flip-card-back {
		transform: rotateY(180deg);
	}
	@media (prefers-reduced-motion: reduce) {
		.flip-card-inner {
			transition: none;
		}
	}
</style>
