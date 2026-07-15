<script lang="ts">
	import type { PageData } from './$types';
	import { recordLineResult, getLineDrillQueue, type LineCard, type LineSessionSummary } from '$lib/api';
	import DrillCard from '$lib/components/DrillCard.svelte';
	import Furigana from '$lib/components/Furigana.svelte';
	import BackLink from '$lib/components/BackLink.svelte';
	import { cn } from '$lib/utils/cn';

	let { data }: { data: PageData } = $props();

	let queue = $state<LineCard[]>([]);
	let summary = $state<LineSessionSummary>({ new: 0, in_progress: 0, old: 0 });
	let actionError = $state<string | null>(null);

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
		if (data.songId === undefined) return;
		try {
			const result = await getLineDrillQueue(data.songId, 20);
			queue = result.cards;
			summary = result.summary;
		} catch (e) {
			actionError = e instanceof Error ? e.message : String(e);
		}
	}

	// While there's nothing to show, keep checking — lets an in-progress line
	// (due again in 10s/30s/2m) reappear on its own once its time is near.
	$effect(() => {
		const id = setInterval(() => {
			if (queue.length === 0) refresh();
		}, 2500);
		return () => clearInterval(id);
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
</script>

<BackLink href={backHref} label={backLabel} />

<div class="mb-6 flex items-center justify-between">
	<h1 class="text-2xl font-semibold text-ink">Line Drill</h1>
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
</div>

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
