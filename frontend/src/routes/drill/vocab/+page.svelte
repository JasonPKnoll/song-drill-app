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
		new_cap: 0,
		at_cap: false
	});
	let actionError = $state<string | null>(null);
	let addingMore = $state(false);
	// Guards against the empty-queue timer below piling up overlapping
	// requests — if one refresh is slow (or hangs) and fires again before it
	// resolves, the second call is skipped rather than stacking another
	// in-flight fetch on top of it.
	let refreshing = false;
	// Bumped on a failed refresh, cleared on a successful one. A failure
	// leaves `queue`/`summary` untouched, so without this the scheduling
	// effect below — which only reacts to those two changing — would never
	// fire again after a single dropped request (a Tailscale hiccup, a slow
	// moment on the Pi): the page would sit there looking stuck until
	// manually reloaded. Tracking failure as its own piece of state gives
	// the effect something to react to so it can retry instead of going
	// silent.
	let failedAt = $state<number | null>(null);
	const RETRY_DELAY_MS = 5000;

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
		if (data.songId === undefined || refreshing) return;
		refreshing = true;
		try {
			const result = await getVocabDrillQueue(data.songId);
			queue = result.cards;
			summary = result.summary;
			failedAt = null;
		} catch (e) {
			actionError = e instanceof Error ? e.message : String(e);
			failedAt = Date.now();
		} finally {
			refreshing = false;
		}
	}

	// The backend already knows the exact moment the next card becomes due
	// (summary.next_due_at, set whenever the batch comes back empty) — arm a
	// single precise timer for that instant instead of polling on a fixed
	// interval. Capped so a far-future due date (e.g. a mature review card,
	// days out) can't overflow setTimeout's ~24.8-day limit or produce a
	// zero/negative delay that fires immediately in a loop; in that case it
	// just falls back to a slow re-check rather than auto-refreshing right
	// up to the moment. A pending failure always wins and gets a fixed
	// short retry instead — see failedAt above.
	const MAX_TIMER_DELAY_MS = 60 * 60 * 1000;
	$effect(() => {
		if (failedAt !== null) {
			const id = setTimeout(refresh, RETRY_DELAY_MS);
			return () => clearTimeout(id);
		}
		if (queue.length > 0 || !summary.next_due_at) return;
		const delayMs = Math.min(
			MAX_TIMER_DELAY_MS,
			Math.max(250, new Date(summary.next_due_at).getTime() - Date.now())
		);
		const id = setTimeout(refresh, delayMs);
		return () => clearTimeout(id);
	});

	let backHref = $derived(data.songId !== undefined ? `/songs/${data.songId}` : '/');
	let backLabel = $derived(data.songId !== undefined ? 'Back to song' : 'Back to library');
	let current = $derived(queue[0] ?? null);

	async function answer(correct: boolean) {
		if (!current) return;
		const card = current;
		queue = queue.slice(1);
		try {
			await recordVocabResult(card.vocab_id, correct);
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

<div class="mb-1 flex items-center justify-between">
	<h1 class="text-2xl font-semibold text-ink">Vocab Drill</h1>
	{#if !data.error}
		<div
			class="flex items-center gap-3"
			title="{summary.new} new · {summary.in_progress} in progress · {summary.old} from previous days"
		>
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
	{/if}
</div>

{#if !data.error}
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
		{:else if summary.at_cap}
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
