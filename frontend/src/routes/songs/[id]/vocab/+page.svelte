<script lang="ts">
	import type { PageData } from './$types';
	import BackLink from '$lib/components/BackLink.svelte';
	import VocabFlipCard from '$lib/components/VocabFlipCard.svelte';

	let { data }: { data: PageData } = $props();

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

	let filtered = $derived.by(() => {
		if (!data.song) return [];
		const raw = query.trim();
		if (!raw) return data.song.vocab;
		const q = raw.toLowerCase();
		return data.song.vocab.filter(
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

	<div class="mb-6">
		<h1 class="text-2xl font-semibold text-ink">{song.title}</h1>
		<p class="text-muted">{song.artist} · Browse vocab</p>
	</div>

	<input
		type="text"
		bind:value={query}
		placeholder="Search word, reading, or meaning…"
		class="mb-4 w-full rounded-xl border border-border bg-surface px-4 py-2 text-ink placeholder:text-muted focus:border-accent focus:outline-none"
	/>

	{#if data.song.vocab.length === 0}
		<div class="rounded-2xl border border-border bg-surface p-8 text-center text-muted">
			This song has no vocab yet.
		</div>
	{:else if filtered.length === 0}
		<div class="rounded-2xl border border-border bg-surface p-8 text-center text-muted">
			No vocab matches "{query}".
		</div>
	{:else}
		<p class="mb-3 text-sm text-muted">{filtered.length} of {song.vocab.length} words</p>
		<div class="grid gap-4 sm:grid-cols-2">
			{#each filtered as vocab (vocab.id)}
				<VocabFlipCard {vocab} flipped={flipped.has(vocab.id)} onToggle={() => toggle(vocab.id)} />
			{/each}
		</div>
	{/if}
{:else}
	<p class="text-bad">Song not found.</p>
{/if}
