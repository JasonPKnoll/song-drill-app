<script lang="ts">
	import type { SongSummary } from '$lib/api';

	let { song }: { song: SongSummary } = $props();

	let fullyMastered = $derived(song.vocab_count > 0 && song.mastered_count === song.vocab_count);
</script>

<a
	href={`/songs/${song.id}`}
	class="block rounded-2xl border border-border bg-surface p-5 transition hover:border-accent/50"
>
	<div class="flex items-start justify-between gap-4">
		<div>
			<h2 class="text-lg font-semibold text-ink">{song.title}</h2>
			<p class="text-sm text-muted">{song.artist}</p>
		</div>
		{#if fullyMastered}
			<span class="whitespace-nowrap rounded-full bg-mastered/10 px-2 py-1 text-xs font-medium text-mastered">
				Mastered
			</span>
		{/if}
	</div>

	<div class="mt-4 flex gap-4 text-sm text-muted">
		<span><span class="text-ink">{song.mastered_count}</span>/{song.vocab_count} vocab</span>
		<span>{song.line_count} lines</span>
	</div>
</a>
