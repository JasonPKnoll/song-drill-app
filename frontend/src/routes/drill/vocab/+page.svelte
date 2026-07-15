<script lang="ts">
	import type { PageData } from './$types';
	import {
		recordVocabResult,
		getVocabDrillQueue,
		addMoreVocab,
		type VocabCard,
		type VocabSessionSummary
	} from '$lib/api';
	import DrillCard from '$lib/components/DrillCard.svelte';
	import Furigana from '$lib/components/Furigana.svelte';
	import BackLink from '$lib/components/BackLink.svelte';
	import { cn } from '$lib/utils/cn';

	let { data }: { data: PageData } = $props();

	let queue = $state<VocabCard[]>([]);
	let summary = $state<VocabSessionSummary>({
		new: 0,
		in_progress: 0,
		old: 0,
		introduced_today: 0,
		new_cap: 0
	});
	let actionError = $state<string | null>(null);
	let addingMore = $state(false);

	$effect(() => {
		queue = data.queue;
		summary = data.summary;
		actionError = null;
	});

	// Re-fetches the small due batch + today's cap summary from the server —
	// the source of truth for "what's due right now," replacing the old
	// approach of manually re-inserting an answered card into a local array
	// (which ignored its real `due` timestamp entirely). Called right after
	// every answer, so a newly-due card can surface as soon as the very next
	// fetch reflects it, not just once the whole local batch is drained.
	async function refresh() {
		if (data.songId === undefined) return;
		try {
			const result = await getVocabDrillQueue(data.songId);
			queue = result.cards;
			summary = result.summary;
		} catch (e) {
			actionError = e instanceof Error ? e.message : String(e);
		}
	}

	// While there's nothing to show, keep checking — this is what makes an
	// in-progress card (due again in 10s/30s/2m) reappear on its own once its
	// time is near, instead of requiring the user to reload the page.
	$effect(() => {
		const id = setInterval(() => {
			if (queue.length === 0) refresh();
		}, 2500);
		return () => clearInterval(id);
	});

	let backHref = $derived(data.songId !== undefined ? `/songs/${data.songId}` : '/');
	let backLabel = $derived(data.songId !== undefined ? 'Back to song' : 'Back to library');
	let current = $derived(queue[0] ?? null);
	let atCap = $derived(summary.introduced_today >= summary.new_cap);

	async function answer(correct: boolean) {
		if (!current) return;
		const card = current;
		queue = queue.slice(1);
		try {
			await recordVocabResult(card.song_id, card.vocab_id, correct);
			actionError = null;
		} catch (e) {
			actionError = e instanceof Error ? e.message : String(e);
			// The server never recorded this attempt (request failed) — put the
			// card back at the front instead of letting it silently vanish from
			// the session with no state change actually having happened.
			queue = [card, ...queue];
			return;
		}
		await refresh();
	}

	async function handleAddMore() {
		if (data.songId === undefined) return;
		addingMore = true;
		try {
			await addMoreVocab(data.songId);
			actionError = null;
		} catch (e) {
			actionError = e instanceof Error ? e.message : String(e);
		}
		await refresh();
		addingMore = false;
	}
</script>

<BackLink href={backHref} label={backLabel} />

<h1 class="mb-2 text-2xl font-semibold text-ink">Vocab Drill</h1>

{#if !data.error}
	<div
		class="mb-2 flex items-center justify-between"
		title="{summary.new} new · {summary.in_progress} in progress · {summary.old} from previous days"
	>
		<div class="flex items-center gap-3">
			<span class="flex items-center gap-1.5">
				<span class="h-2 w-2 rounded-full bg-new"></span>
				<span class="text-sm text-muted tabular-nums">{summary.new}</span>
			</span>
			<span class="flex items-center gap-1.5">
				<span class="h-2 w-2 rounded-full bg-accent"></span>
				<span class="text-sm text-muted tabular-nums">{summary.in_progress}</span>
			</span>
			<span class="flex items-center gap-1.5">
				<span class="h-2 w-2 rounded-full bg-good"></span>
				<span class="text-sm text-muted tabular-nums">{summary.old}</span>
			</span>
		</div>
	</div>

	<div class="mb-6 flex items-center justify-end">
		<button
			type="button"
			disabled={addingMore}
			onclick={handleAddMore}
			class={cn(
				'text-sm font-medium text-accent',
				'transition hover:opacity-80 disabled:opacity-50',
				'no-focus-ring'
			)}
		>
			{addingMore ? 'Adding…' : '+ Add 5 more'}
		</button>
	</div>
{/if}

{#if data.error}
	<p class="text-bad">{data.error}</p>
{:else if !current}
	<div
		class={cn(
			'p-8',
			'text-center',
			'border border-border bg-surface text-muted',
			'rounded-2xl'
		)}
	>
		{#if summary.in_progress > 0}
			Nothing due right now — in-progress words will come back around shortly.
		{:else if atCap}
			All caught up — today's {summary.new_cap} new words are underway. Nice work.
		{:else}
			Nothing due right now. Nice work.
		{/if}
	</div>
{:else}
	{#key `${current.song_id}-${current.vocab_id}`}
		<DrillCard onGotIt={() => answer(true)} onMissed={() => answer(false)}>
			{#snippet front()}
				<p class="text-5xl font-semibold text-ink">{current.surface}</p>
				{#if current.example_line}
					<p class="text-lg text-ink">{current.example_line.text}</p>
				{/if}
			{/snippet}
			{#snippet back()}
				<p class="text-2xl text-ink">
					<Furigana furi={current.furi} />
				</p>
				<p class="text-lg text-good">{current.base_meaning}</p>
				{#if current.example_line}
					<p class="mt-2 text-base text-ink">
						<Furigana furi={current.example_line.furi} />
					</p>
					<p class="text-sm text-muted">{current.example_line.natural}</p>
				{/if}
			{/snippet}
		</DrillCard>
	{/key}
{/if}

{#if actionError}
	<p class="mt-3 text-sm text-bad">{actionError}</p>
{/if}
