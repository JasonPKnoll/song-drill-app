<script lang="ts">
	import type { PageData } from './$types';
	import { page } from '$app/state';
	import BackLink from '$lib/components/BackLink.svelte';
	import VocabFlipCard from '$lib/components/VocabFlipCard.svelte';
	import Furigana from '$lib/components/Furigana.svelte';
	import ArrowLeft from '@lucide/svelte/icons/arrow-left';
	import { cn } from '$lib/utils/cn';

	let { data }: { data: PageData } = $props();

	const emptyStateClass = cn(
		'p-8',
		'text-center',
		'border border-border bg-surface text-muted',
		'rounded-2xl'
	);

	// Set when arriving from the song reader's "look up this line's vocab"
	// link (?line=<line id>). That same link is also what the top-right
	// "back to reader" button requires: without a specific line to return
	// to, there's nothing to go back to, so it only makes sense to show up
	// in that referred-from flow.
	let sourceLineId = $derived(
		page.url.searchParams.has('line') ? Number(page.url.searchParams.get('line')) : null
	);
	let cameFromReader = $derived(sourceLineId !== null);
	let sourceLine = $derived(data.song?.lines.find((l) => l.id === sourceLineId));

	let query = $state('');
	let flipped = $state<Set<number>>(new Set());

	function toggle(id: number) {
		const next = new Set(flipped);
		if (next.has(id)) {
			next.delete(id);
		} else {
			next.add(id);
		}
		flipped = next;
	}

	// Scoped to the exact words tagged as occurring in that line (via
	// line_words, real tokenization from ingest) when arriving from the
	// reader — not a guess from the line's raw text, which is what used to
	// false-positive on short words that are substrings of longer ones
	// actually present (e.g. 人 inside 二人).
	let baseVocab = $derived.by(() => {
		if (!data.song) return [];
		if (sourceLineId === null) return data.song.vocab;
		return data.song.vocab.filter((v) => v.line_ids.includes(sourceLineId));
	});

	let filtered = $derived.by(() => {
		const raw = query.trim();
		if (!raw) return baseVocab;
		const q = raw.toLowerCase();
		return baseVocab.filter(
			(v) =>
				v.surface.includes(raw) ||
				v.reading.includes(raw) ||
				v.base_meaning.toLowerCase().includes(q) ||
				v.context_meaning.toLowerCase().includes(q)
		);
	});
</script>

{#if data.error}
	<p class="text-bad">Failed to load song: {data.error}</p>
{:else if data.song}
	{@const song = data.song}
	<BackLink href={`/songs/${song.id}`} label="Back to {song.title}" />

	<div class="mb-6 flex items-start justify-between gap-4">
		<div>
			<h1 class="text-2xl font-semibold text-ink">{song.title}</h1>
			<p class="text-muted">{song.artist} · Browse vocab</p>
		</div>
		{#if cameFromReader}
			<button
				type="button"
				onclick={() => history.back()}
				class={cn(
					'flex h-12 w-12 shrink-0 items-center justify-center',
					'bg-accent text-bg shadow-black/30',
					'rounded-full shadow-lg',
					'transition-transform active:scale-95',
					'no-focus-ring'
				)}
				aria-label="Back to reader"
			>
				<ArrowLeft size={24} strokeWidth={2.5} />
			</button>
		{/if}
	</div>

	{#if sourceLine}
		<div class="mb-4">
			<p class="text-lg text-ink">
				<Furigana furi={sourceLine.furi} />
			</p>
			<p class="mt-1 text-good">{sourceLine.natural}</p>
		</div>
	{/if}

	<input
		type="text"
		bind:value={query}
		placeholder={sourceLine ? 'Filter these words…' : 'Search word, reading, or meaning…'}
		class={cn(
			'mb-4 w-full px-4 py-2',
			'border border-border bg-surface text-ink placeholder:text-muted',
			'rounded-xl',
			'focus:border-accent focus:outline-none'
		)}
	/>

	{#if data.song.vocab.length === 0}
		<div class={emptyStateClass}>This song has no vocab yet.</div>
	{:else if baseVocab.length === 0}
		<div class={emptyStateClass}>No vocab tagged for this line.</div>
	{:else if filtered.length === 0}
		<div class={emptyStateClass}>No vocab matches "{query}".</div>
	{:else}
		<p class="mb-3 text-sm text-muted">{filtered.length} of {baseVocab.length} words</p>
		<div class="grid gap-4 sm:grid-cols-2">
			{#each filtered as vocab (vocab.id)}
				<VocabFlipCard {vocab} flipped={flipped.has(vocab.id)} onToggle={() => toggle(vocab.id)} />
			{/each}
		</div>
	{/if}
{:else}
	<p class="text-bad">Song not found.</p>
{/if}
