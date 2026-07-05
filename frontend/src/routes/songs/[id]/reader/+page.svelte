<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { getSong, type SongDetail } from '$lib/api';
	import Furigana from '$lib/components/Furigana.svelte';
	import BackLink from '$lib/components/BackLink.svelte';

	let songId = $derived(Number(page.params.id));

	let song = $state<SongDetail | null>(null);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let revealed = $state<Set<number>>(new Set());

	onMount(async () => {
		try {
			song = await getSong(songId);
		} catch (e) {
			error = e instanceof Error ? e.message : String(e);
		} finally {
			loading = false;
		}
	});

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

{#if loading}
	<p class="text-muted">Loading…</p>
{:else if error}
	<p class="text-bad">Failed to load song: {error}</p>
{:else if song}
	<BackLink href={`/songs/${song.id}`} label="Back to {song.title}" />

	<div class="mb-6">
		<h1 class="text-2xl font-semibold text-ink">{song.title}</h1>
		<p class="text-muted">{song.artist} · Song reader</p>
	</div>

	<div class="flex flex-col gap-3">
		{#each song.lines as line (line.id)}
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
		{/each}
	</div>
{:else}
	<p class="text-bad">Song not found.</p>
{/if}
