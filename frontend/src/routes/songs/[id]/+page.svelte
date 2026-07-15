<script lang="ts">
	import type { PageData } from './$types';
	import BackLink from '$lib/components/BackLink.svelte';
	import ChevronDown from '@lucide/svelte/icons/chevron-down';
	import { cn } from '$lib/utils/cn';

	let { data }: { data: PageData } = $props();

	const studyLinkClass = cn(
		'p-5',
		'text-center',
		'border border-border bg-surface',
		'rounded-2xl transition hover:border-accent/50'
	);

	const progressLinkClass = cn(
		'px-4 py-2',
		'text-sm font-medium',
		'border border-accent/50 bg-accent/10 text-accent',
		'rounded-xl transition hover:bg-accent/20'
	);

	// Collapsed by default — song notes (an AI-generated emotional summary)
	// can run long enough to push the actual study links off screen. Only
	// shows the expand affordance at all if the text actually overflows the
	// collapsed height, measured once on mount while still collapsed.
	let notesEl = $state<HTMLParagraphElement | undefined>();
	let notesExpanded = $state(false);
	let notesOverflows = $state(false);

	$effect(() => {
		if (notesEl) {
			notesOverflows = notesEl.scrollHeight > notesEl.clientHeight + 1;
		}
	});
</script>

{#if data.error}
	<p class="text-bad">Failed to load song: {data.error}</p>
{:else if data.song}
	{@const song = data.song}
	<BackLink href="/" label="Back to library" />

	<div class="mb-6 flex items-start justify-between gap-4">
		<div>
			<h1 class="text-2xl font-semibold text-ink">{song.title}</h1>
			<p class="text-muted">{song.artist}</p>
		</div>
		<a href={`/stats?song_id=${song.id}`} class={cn(progressLinkClass, 'shrink-0')}>Progress</a>
	</div>

	{#if song.notes}
		<div class="relative mb-6">
			<p
				bind:this={notesEl}
				class={cn(
					'p-4 text-sm italic',
					'border border-border bg-surface text-muted',
					'rounded-xl overflow-hidden transition-[max-height] duration-300',
					notesExpanded ? 'max-h-[2000px]' : 'max-h-24',
					notesOverflows && !notesExpanded && 'pb-8'
				)}
			>
				{song.notes}
			</p>
			{#if notesOverflows}
				{#if !notesExpanded}
					<div
						class="pointer-events-none absolute inset-x-0 bottom-0 h-10 rounded-b-xl bg-gradient-to-t from-surface to-transparent"
					></div>
				{/if}
				<button
					type="button"
					onclick={() => (notesExpanded = !notesExpanded)}
					class={cn(
						'absolute bottom-1.5 left-1/2 -translate-x-1/2',
						'flex h-6 w-6 items-center justify-center',
						'text-muted transition hover:text-accent',
						'no-focus-ring'
					)}
					aria-expanded={notesExpanded}
					aria-label={notesExpanded ? 'Collapse summary' : 'Expand summary'}
				>
					<ChevronDown size={16} class={cn('transition-transform', notesExpanded && 'rotate-180')} />
				</button>
			{/if}
		</div>
	{/if}

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
