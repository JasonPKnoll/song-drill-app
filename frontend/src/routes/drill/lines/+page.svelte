<script lang="ts">
	import type { PageData } from './$types';
	import { recordLineResult, type LineCard } from '$lib/api';
	import DrillCard from '$lib/components/DrillCard.svelte';
	import Furigana from '$lib/components/Furigana.svelte';
	import BackLink from '$lib/components/BackLink.svelte';

	let { data }: { data: PageData } = $props();

	let queue = $state<LineCard[]>([]);
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
		done += 1;
		try {
			await recordLineResult(card.line_id, correct);
		} catch (e) {
			actionError = e instanceof Error ? e.message : String(e);
		}
	}
</script>

<BackLink href={backHref} label={backLabel} />

<div class="mb-6 flex items-center justify-between">
	<h1 class="text-2xl font-semibold text-ink">Line Drill</h1>
	<span class="text-sm text-muted">{done} done · {queue.length} left</span>
</div>

{#if data.error}
	<p class="text-bad">{data.error}</p>
{:else if !current}
	<div class="rounded-2xl border border-border bg-surface p-8 text-center text-muted">
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
