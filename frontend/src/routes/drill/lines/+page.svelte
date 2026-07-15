<script lang="ts">
	import type { PageData } from './$types';
	import { recordLineResult, getLineDrillQueue, addMoreLines, type LineCard, type LineSessionSummary } from '$lib/api';
	import DrillCard from '$lib/components/DrillCard.svelte';
	import Furigana from '$lib/components/Furigana.svelte';
	import BackLink from '$lib/components/BackLink.svelte';
	import { cn } from '$lib/utils/cn';

	let { data }: { data: PageData } = $props();

	let queue = $state<LineCard[]>([]);
	let summary = $state<LineSessionSummary>({
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
	// Bumped on a failed refresh, cleared on a successful one — see the
	// vocab drill page's identical comment. Without this, a single dropped
	// request would leave `queue`/`summary` untouched forever, and the
	// scheduling effect below (which only reacts to those two changing)
	// would never fire again to retry.
	let failedAt = $state<number | null>(null);
	const RETRY_DELAY_MS = 5000;

	$effect(() => {
		queue = data.queue;
		summary = data.summary;
		actionError = null;
	});

	// Re-fetches the small due batch + live summary from the server — the
	// source of truth for "what's due right now," same pattern as the vocab
	// drill page. Called right after every answer, so a newly-due card can
	// surface as soon as the very next fetch reflects it.
	async function refresh() {
		if (data.songId === undefined || refreshing) return;
		refreshing = true;
		try {
			const result = await getLineDrillQueue(data.songId, 20);
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

	// The backend already knows the exact moment the next line becomes due
	// (summary.next_due_at, set whenever the batch comes back empty) — arm a
	// single precise timer for that instant instead of polling on a fixed
	// interval. Capped so a far-future due date can't overflow setTimeout's
	// ~24.8-day limit or fire immediately in a loop; falls back to a slow
	// re-check in that case instead. A pending failure always wins and gets
	// a fixed short retry instead — see failedAt above.
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
	let backLabel = 'Back to song';
	let current = $derived(queue[0] ?? null);

	async function answer(correct: boolean) {
		if (!current) return;
		const card = current;
		queue = queue.slice(1);
		try {
			await recordLineResult(card.line_id, correct);
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
			await addMoreLines(data.songId);
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
	<h1 class="text-2xl font-semibold text-ink">Line Drill</h1>
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
			Nothing due right now — in-progress lines will come back around shortly.
		{:else if summary.at_cap}
			All caught up — today's {summary.new_cap} new lines are underway. Nice work.
		{:else}
			Nothing due right now. Nice work.
		{/if}
	</div>
{:else}
	{#key current.line_id}
		<DrillCard onGotIt={() => answer(true)} onMissed={() => answer(false)}>
			{#snippet front()}
				<p class="text-3xl leading-relaxed font-semibold text-ink">{current.text}</p>
			{/snippet}
			{#snippet back()}
				<p class="text-xl text-ink">
					<Furigana furi={current.furi} />
				</p>
				<p class="text-lg text-good">{current.natural}</p>
				{#if current.grammar_note}
					<p class="text-sm text-muted">{current.grammar_note}</p>
				{/if}
			{/snippet}
		</DrillCard>
	{/key}
{/if}

{#if actionError}
	<p class="mt-3 text-sm text-bad">{actionError}</p>
{/if}
