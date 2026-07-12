<script lang="ts">
	import type { PageData } from './$types';
	import BackLink from '$lib/components/BackLink.svelte';
	import { cn } from '$lib/utils/cn';

	let { data }: { data: PageData } = $props();

	const studyLinkClass = cn(
		'p-5',
		'text-center',
		'border border-border bg-surface',
		'rounded-2xl transition hover:border-accent/50'
	);
</script>

{#if data.error}
	<p class="text-bad">Failed to load song: {data.error}</p>
{:else if data.song}
	{@const song = data.song}
	<BackLink href="/" label="Back to library" />

	<div class="mb-6">
		<h1 class="text-2xl font-semibold text-ink">{song.title}</h1>
		<p class="text-muted">{song.artist}</p>
		{#if song.notes}
			<p
				class={cn(
					'mt-3 p-4',
					'text-sm italic',
					'border border-border bg-surface text-muted',
					'rounded-xl'
				)}
			>
				{song.notes}
			</p>
		{/if}
	</div>

	<div class="grid gap-3 sm:grid-cols-2">
		<a href={`/drill/vocab?song_id=${song.id}`} class={studyLinkClass}>
			<p class="font-medium text-ink">Vocab drill</p>
			<p class="mt-1 text-sm text-muted">{song.vocab.length} words</p>
		</a>
		<a href={`/drill/lines?song_id=${song.id}`} class={studyLinkClass}>
			<p class="font-medium text-ink">Line drill</p>
			<p class="mt-1 text-sm text-muted">{song.lines.length} lines</p>
		</a>
		<a href={`/songs/${song.id}/vocab`} class={studyLinkClass}>
			<p class="font-medium text-ink">Browse vocab</p>
			<p class="mt-1 text-sm text-muted">Search & flip through</p>
		</a>
		<a href={`/songs/${song.id}/reader`} class={studyLinkClass}>
			<p class="font-medium text-ink">Song reader</p>
			<p class="mt-1 text-sm text-muted">Read through</p>
		</a>
	</div>
{:else}
	<p class="text-bad">Song not found.</p>
{/if}
