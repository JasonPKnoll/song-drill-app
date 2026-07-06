<script lang="ts">
	import Furigana from './Furigana.svelte';
	import type { VocabItem } from '$lib/api';

	let {
		vocab,
		flipped,
		onToggle
	}: { vocab: VocabItem; flipped: boolean; onToggle: () => void } = $props();
</script>

<button type="button" class="flip-card h-64 w-full text-left" onclick={onToggle} aria-pressed={flipped}>
	<div class="flip-card-inner" class:flipped>
		<div
			class="flip-card-face flex flex-col items-center justify-center gap-3 rounded-2xl border border-border bg-surface p-6 text-center"
		>
			<span class="rounded-full bg-accent/10 px-2 py-0.5 text-xs font-medium text-accent">{vocab.pos}</span>
			<p class="text-4xl font-semibold text-ink">{vocab.surface}</p>
			<p class="text-sm text-muted">Tap to reveal</p>
		</div>
		<div
			class="flip-card-face flip-card-back flex flex-col items-center justify-center gap-3 overflow-y-auto rounded-2xl border border-border bg-surface p-6 text-center"
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
