<script lang="ts">
	import type { PageData } from './$types';
	import Furigana from '$lib/components/Furigana.svelte';
	import BackLink from '$lib/components/BackLink.svelte';

	let { data }: { data: PageData } = $props();

	let revealed = $state<Set<number>>(new Set());

	function toggle(lineId: number) {
		const next = new Set(revealed);
		if (next.has(lineId)) {
			next.delete(lineId);
		} else {
			next.add(lineId);
		}
		revealed = next;
	}
</script>

{#if data.error}
	<p class="text-bad">Failed to load song: {data.error}</p>
{:else if data.song}
	{@const song = data.song}
	<BackLink href={`/songs/${song.id}`} label="Back to {song.title}" />

	<div class="mb-6">
		<h1 class="text-2xl font-semibold text-ink">{song.title}</h1>
		<p class="text-muted">{song.artist} · Song reader</p>
	</div>

	<div class="flex flex-col gap-3">
		{#each song.lines as line, i (line.id)}
			{@const prevSection = i > 0 ? song.lines[i - 1].section : undefined}
			{#if line.section && line.section !== prevSection}
				<h2 class="mt-4 mb-1 text-sm font-semibold tracking-wide text-accent uppercase first:mt-0">
					{line.section}
				</h2>
			{/if}
			{#if line.reading === ''}
				<div class="rounded-2xl border border-border bg-surface/50 p-5 text-left text-muted italic">
					{line.text}
				</div>
			{:else}
				<button
					type="button"
					class="rounded-2xl border border-border bg-surface p-5 text-left transition hover:border-accent/50"
					onclick={() => toggle(line.id)}
				>
					{#if revealed.has(line.id)}
						<p class="text-xl leading-relaxed text-ink">
							<Furigana furi={line.furi} />
						</p>
						<p class="mt-3 text-good">{line.natural}</p>
						{#if line.grammar_note}
							<p class="mt-2 text-sm text-muted">{line.grammar_note}</p>
						{/if}
					{:else}
						<p class="text-xl leading-relaxed text-ink">{line.text}</p>
					{/if}
				</button>
			{/if}
		{/each}
	</div>
{:else}
	<p class="text-bad">Song not found.</p>
{/if}
