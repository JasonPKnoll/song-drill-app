<script lang="ts">
	import type { Snippet } from 'svelte';
	import { cn } from '$lib/utils/cn';

	// Shared 3D flip mechanics for both the drill cards (DrillCard.svelte,
	// self-managed reveal state, buttons below) and the vocab browser
	// (VocabFlipCard.svelte, flip state lifted to the parent so it can track
	// many cards at once). Deliberately controlled, not self-managing state
	// here — callers differ on who owns `flipped` and whether it can flip
	// back, so this only renders the mechanics for whatever state it's given.
	let {
		flipped,
		onToggle,
		front,
		back,
		disabled = false,
		frontClass = '',
		backClass = ''
	}: {
		flipped: boolean;
		onToggle: () => void;
		front: Snippet;
		back: Snippet;
		disabled?: boolean;
		frontClass?: string;
		backClass?: string;
	} = $props();
</script>

<button
	type="button"
	class={cn('flip-card', 'h-64 w-full')}
	onclick={onToggle}
	{disabled}
	aria-pressed={flipped}
>
	<div class="flip-card-inner" class:flipped>
		<div class={cn('flip-card-face', frontClass)}>
			{@render front()}
		</div>
		<div class={cn('flip-card-face flip-card-back', backClass)}>
			{@render back()}
		</div>
	</div>
</button>

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
