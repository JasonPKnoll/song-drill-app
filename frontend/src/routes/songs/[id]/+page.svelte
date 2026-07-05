<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { getSong, type SongDetail } from '$lib/api';

	let songId = $derived(Number(page.params.id));

	let song = $state<SongDetail | null>(null);
	let loading = $state(true);
	let error = $state<string | null>(null);

	onMount(async () => {
		try {
			song = await getSong(songId);
		} catch (e) {
			error = e instanceof Error ? e.message : String(e);
		} finally {
			loading = false;
		}
	});
</script>

{#if loading}
	<p class="text-muted">Loading…</p>
{:else if error}
	<p class="text-bad">Failed to load song: {error}</p>
{:else if song}
	<div class="mb-6">
		<h1 class="text-2xl font-semibold text-ink">{song.title}</h1>
		<p class="text-muted">{song.artist}</p>
		{#if song.notes}
			<p class="mt-3 rounded-xl border border-border bg-surface p-4 text-sm text-muted italic">
				{song.notes}
			</p>
		{/if}
	</div>

	<div class="grid gap-3 sm:grid-cols-3">
		<a
			href={`/drill/vocab?song_id=${song.id}`}
			class="rounded-2xl border border-border bg-surface p-5 text-center transition hover:border-accent/50"
		>
			<p class="font-medium text-ink">Vocab drill</p>
			<p class="mt-1 text-sm text-muted">{song.vocab.length} words</p>
		</a>
		<a
			href={`/drill/lines?song_id=${song.id}`}
			class="rounded-2xl border border-border bg-surface p-5 text-center transition hover:border-accent/50"
		>
			<p class="font-medium text-ink">Line drill</p>
			<p class="mt-1 text-sm text-muted">{song.lines.length} lines</p>
		</a>
		<a
			href={`/songs/${song.id}/reader`}
			class="rounded-2xl border border-border bg-surface p-5 text-center transition hover:border-accent/50"
		>
			<p class="font-medium text-ink">Song reader</p>
			<p class="mt-1 text-sm text-muted">Read through</p>
		</a>
	</div>
{:else}
	<p class="text-bad">Song not found.</p>
{/if}
