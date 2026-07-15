<script lang="ts">
	import type { SongSummary } from '$lib/api';
	import { cn } from '$lib/utils/cn';

	let { song }: { song: SongSummary } = $props();
</script>

<a
	href={`/songs/${song.id}`}
	class={cn(
		'block p-5',
		'border border-border bg-surface',
		'rounded-2xl transition hover:border-accent/50'
	)}
>
	<div class="flex items-start justify-between gap-4">
		<div>
			<h2 class="text-lg font-semibold text-ink">{song.title}</h2>
			<p class="text-sm text-muted">{song.artist}</p>
		</div>
		{#if song.fully_mastered}
			<span
				class={cn(
					'px-2 py-1',
					'text-xs font-medium whitespace-nowrap',
					'bg-mastered/10 text-mastered',
					'rounded-full'
				)}
			>
				Mastered
			</span>
		{/if}
	</div>

	<div class="mt-4 flex gap-4 text-sm text-muted">
		<span><span class="text-ink">{song.mastered_count}</span>/{song.vocab_count} vocab</span>
		<span>{song.line_count} lines</span>
	</div>
</a>
