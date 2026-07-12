<script lang="ts">
	import Furigana from './Furigana.svelte';
	import type { VocabItem } from '$lib/api';
	import { cn } from '$lib/utils/cn';

	let {
		vocab,
		flipped,
		onToggle
	}: { vocab: VocabItem; flipped: boolean; onToggle: () => void } = $props();
</script>

<button
	type="button"
	class={cn('flip-card', 'h-64 w-full', 'text-left')}
	onclick={onToggle}
	aria-pressed={flipped}
>
	<div class="flip-card-inner" class:flipped>
		<div
			class={cn(
				'flip-card-face flex flex-col items-center justify-center gap-3 p-6',
				'text-center',
				'border border-border bg-surface',
				'rounded-2xl'
			)}
		>
			<span
				class={cn(
					'px-2 py-0.5',
					'text-xs font-medium',
					'bg-accent/10 text-accent',
					'rounded-full'
				)}
			>
				{vocab.pos}
			</span>
			<p class="text-4xl font-semibold text-ink">{vocab.surface}</p>
			<p class="text-sm text-muted">Tap to reveal</p>
		</div>
		<div
			class={cn(
				'flip-card-face flip-card-back flex flex-col items-center justify-center gap-3 overflow-y-auto p-6',
				'text-center',
				'border border-border bg-surface',
				'rounded-2xl'
			)}
		>
			<p class="text-2xl text-ink">
				<Furigana furi={vocab.furi} />
			</p>
			<p class="text-xl font-medium text-good">{vocab.base_meaning}</p>
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
