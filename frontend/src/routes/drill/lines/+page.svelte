<script lang="ts">
	import type { PageData } from './$types';
	import { recordLineResult, type LineCard } from '$lib/api';
	import DrillCard from '$lib/components/DrillCard.svelte';
	import Furigana from '$lib/components/Furigana.svelte';
	import BackLink from '$lib/components/BackLink.svelte';
	import { cn } from '$lib/utils/cn';

	let { data }: { data: PageData } = $props();

	let queue = $state<LineCard[]>([]);
	let done = $state(0);
	// line_ids that have been answered at least once this session — lets the
	// "left" count split into "new" (never touched yet) vs "in progress" (still
	// in rotation because it hasn't graduated, whether from a miss or just not
	// enough correct reps yet), Anki-style.
	let attemptedIds = $state<Set<number>>(new Set());
	let actionError = $state<string | null>(null);

	$effect(() => {
		queue = data.queue;
		done = 0;
		attemptedIds = new Set();
		actionError = null;
	});

	let backHref = $derived(data.songId !== undefined ? `/songs/${data.songId}` : '/');
	let backLabel = $derived(data.songId !== undefined ? 'Back to song' : 'Back to library');
	let current = $derived(queue[0] ?? null);
	let newCount = $derived(queue.filter((c) => !attemptedIds.has(c.line_id)).length);
	let inProgressCount = $derived(queue.length - newCount);

	async function answer(correct: boolean) {
		if (!current) return;
		const card = current;
		const wasAlreadyAttempted = attemptedIds.has(card.line_id);
		attemptedIds = new Set(attemptedIds).add(card.line_id);
		queue = queue.slice(1);
		try {
			const result = await recordLineResult(card.line_id, correct);
			// Still mid-way through today's learning/relearning steps (e.g. a
			// miss resets it, or a pass hasn't graduated it yet) — keep it in
			// this session's rotation instead of counting it done.
			if (result.state === 'learning' || result.state === 'relearning') {
				queue = [...queue, card];
			} else {
				done += 1;
			}
		} catch (e) {
			actionError = e instanceof Error ? e.message : String(e);
			// The server never recorded this attempt (request failed) — put the
			// card back at the front instead of letting it vanish from the
			// session's counts with no state change actually having happened.
			if (!wasAlreadyAttempted) {
				const reverted = new Set(attemptedIds);
				reverted.delete(card.line_id);
				attemptedIds = reverted;
			}
			queue = [card, ...queue];
		}
	}
</script>

<BackLink href={backHref} label={backLabel} />

<div class="mb-6 flex items-center justify-between">
	<h1 class="text-2xl font-semibold text-ink">Line Drill</h1>
	<div class="flex items-center gap-3" title="{newCount} new · {inProgressCount} in progress · {done} done">
		<span class="flex items-center gap-1.5">
			<span class="h-2 w-2 rounded-full bg-new"></span>
			<span class="text-sm text-muted tabular-nums">{newCount}</span>
		</span>
		<span class="flex items-center gap-1.5">
			<span class="h-2 w-2 rounded-full bg-accent"></span>
			<span class="text-sm text-muted tabular-nums">{inProgressCount}</span>
		</span>
		<span class="flex items-center gap-1.5">
			<span class="h-2 w-2 rounded-full bg-good"></span>
			<span class="text-sm text-muted tabular-nums">{done}</span>
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
		Nothing due right now. Nice work.
	</div>
{:else}
	{#key current.line_id}
		<DrillCard onGotIt={() => answer(true)} onMissed={() => answer(false)}>
			{#snippet front()}
				<p class="text-sm text-muted">{current.song_title}</p>
				<p class="text-3xl leading-relaxed font-semibold text-ink">{current.text}</p>
			{/snippet}
			{#snippet back()}
				<div class="flex flex-col items-center gap-3 text-center">
					<p class="text-xl text-ink">
						<Furigana furi={current.furi} />
					</p>
					<p class="text-lg text-good">{current.natural}</p>
					{#if current.grammar_note}
						<p class="text-sm text-muted">{current.grammar_note}</p>
					{/if}
				</div>
			{/snippet}
		</DrillCard>
	{/key}
{/if}

{#if actionError}
	<p class="mt-3 text-sm text-bad">{actionError}</p>
{/if}
