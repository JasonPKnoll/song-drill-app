<script lang="ts">
	import Furigana from './Furigana.svelte';
	import FlipCard from './FlipCard.svelte';
	import type { VocabItem } from '$lib/api';
	import { cn } from '$lib/utils/cn';

	let { vocab, flipped, onToggle }: { vocab: VocabItem; flipped: boolean; onToggle: () => void } =
		$props();

	const faceClass = cn(
		'flex flex-col items-center justify-center gap-3 p-6',
		'text-center',
		'border border-border bg-surface',
		'rounded-2xl'
	);
</script>

<FlipCard {flipped} {onToggle} frontClass={faceClass} backClass={faceClass}>
	{#snippet front()}
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
	{/snippet}
	{#snippet back()}
		<p class="text-2xl text-ink">
			<Furigana furi={vocab.furi} />
		</p>
		<p class="text-xl font-medium text-good">{vocab.base_meaning}</p>
	{/snippet}
</FlipCard>
