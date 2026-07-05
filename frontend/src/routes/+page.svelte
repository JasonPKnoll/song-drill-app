<script lang="ts">
	import { onMount } from 'svelte';
	import { listSongs, type SongSummary } from '$lib/api';
	import SongCard from '$lib/components/SongCard.svelte';

	let songs = $state<SongSummary[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);

	onMount(async () => {
		try {
			songs = await listSongs();
		} catch (e) {
			error = e instanceof Error ? e.message : String(e);
		} finally {
			loading = false;
		}
	});
</script>

<div class="mb-6 flex items-center justify-between">
	<h1 class="text-2xl font-semibold text-ink">Song Library</h1>
	{#if songs.length > 0}
		<div class="flex gap-2">
			<a
				href="/drill/vocab"
				class="rounded-xl border border-accent/50 bg-accent/10 px-4 py-2 text-sm font-medium text-accent transition hover:bg-accent/20"
			>
				Drill all vocab
			</a>
			<a
				href="/drill/lines"
				class="rounded-xl border border-accent/50 bg-accent/10 px-4 py-2 text-sm font-medium text-accent transition hover:bg-accent/20"
			>
				Drill all lines
			</a>
		</div>
	{/if}
</div>

{#if loading}
	<p class="text-muted">Loading songs…</p>
{:else if error}
	<p class="text-bad">Failed to load songs: {error}</p>
{:else if songs.length === 0}
	<div class="rounded-2xl border border-border bg-surface p-8 text-center text-muted">
		No songs ingested yet. Run <code class="text-accent">lyrics-annotator</code> and POST the
		output to <code class="text-accent">/api/song-drill/songs/ingest</code>.
	</div>
{:else}
	<div class="grid gap-4 sm:grid-cols-2">
		{#each songs as song (song.id)}
			<SongCard {song} />
		{/each}
	</div>
{/if}
