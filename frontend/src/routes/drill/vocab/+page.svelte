<script lang="ts">
	import type { PageData } from './$types';
	import { recordVocabResult, type VocabCard } from '$lib/api';
	import DrillCard from '$lib/components/DrillCard.svelte';
	import Furigana from '$lib/components/Furigana.svelte';
	import BackLink from '$lib/components/BackLink.svelte';
	import { cn } from '$lib/utils/cn';

	let { data }: { data: PageData } = $props();

	let queue = $state<VocabCard[]>([]);
	let done = $state(0);
	let actionError = $state<string | null>(null);

	$effect(() => {
		queue = data.queue;
		done = 0;
		actionError = null;
	});

	let backHref = $derived(data.songId !== undefined ? `/songs/${data.songId}` : '/');
	let backLabel = $derived(data.songId !== undefined ? 'Back to song' : 'Back to library');
	let current = $derived(queue[0] ?? null);

	async function answer(correct: boolean) {
		if (!current) return;
		const card = current;
		queue = queue.slice(1);
		try {
			const result = await recordVocabResult(card.song_id, card.vocab_id, correct);
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
		}
	}
</script>

<BackLink href={backHref} label={backLabel} />

<div class="mb-6 flex items-center justify-between">
	<h1 class="text-2xl font-semibold text-ink">Vocab Drill</h1>
	<span class="text-sm text-muted">{done} done · {queue.length} left</span>
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
	{#key `${current.song_id}-${current.vocab_id}`}
		<DrillCard onGotIt={() => answer(true)} onMissed={() => answer(false)}>
			{#snippet front()}
				<p class="text-sm text-muted">{current.song_title}</p>
				<p class="text-5xl font-semibold text-ink">{current.surface}</p>
			{/snippet}
			{#snippet back()}
				<div class="flex flex-col items-center gap-3 text-center">
					<p class="text-2xl text-ink">
						<Furigana furi={current.furi} />
					</p>
					<p class="text-lg text-good">{current.context_meaning}</p>
					{#if current.example_line}
						<p class="mt-2 text-base text-ink">
							<Furigana furi={current.example_line.furi} />
						</p>
						<p class="text-sm text-muted">{current.example_line.natural}</p>
					{/if}
				</div>
			{/snippet}
		</DrillCard>
	{/key}
{/if}

{#if actionError}
	<p class="mt-3 text-sm text-bad">{actionError}</p>
{/if}
