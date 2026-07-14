<script lang="ts">
	import type { PageData } from './$types';
	import SongCard from '$lib/components/SongCard.svelte';
	import ProfileSwitcher from '$lib/components/ProfileSwitcher.svelte';
	import { cn } from '$lib/utils/cn';

	let { data }: { data: PageData } = $props();

	const drillLinkClass = cn(
		'px-4 py-2',
		'text-sm font-medium',
		'border border-accent/50 bg-accent/10 text-accent',
		'rounded-xl transition hover:bg-accent/20'
	);
</script>

<div class="mb-6 flex items-center justify-between">
	<h1 class="text-2xl font-semibold text-ink">Song Library</h1>
	<div class="flex items-center gap-3">
		{#if data.songs.length > 0}
			<div class="flex gap-2">
				<a href="/drill/vocab" class={drillLinkClass}>Drill all vocab</a>
				<a href="/drill/lines" class={drillLinkClass}>Drill all lines</a>
				<a href="/stats" class={drillLinkClass}>Progress</a>
			</div>
		{/if}
		<ProfileSwitcher />
	</div>
</div>

{#if data.error}
	<p class="text-bad">Failed to load songs: {data.error}</p>
{:else if data.songs.length === 0}
	<div
		class={cn(
			'p-8',
			'text-center',
			'border border-border bg-surface text-muted',
			'rounded-2xl'
		)}
	>
		No songs ingested yet. Run <code class="text-accent">lyrics-annotator</code> and POST the
		output to <code class="text-accent">/api/song-drill/songs/ingest</code>.
	</div>
{:else}
	<div class="grid gap-4 sm:grid-cols-2">
		{#each data.songs as song (song.id)}
			<SongCard {song} />
		{/each}
	</div>
{/if}
