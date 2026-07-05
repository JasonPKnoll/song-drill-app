<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { getVocabDrillQueue, recordVocabResult, type VocabCard } from '$lib/api';
	import DrillCard from '$lib/components/DrillCard.svelte';
	import Furigana from '$lib/components/Furigana.svelte';
	import BackLink from '$lib/components/BackLink.svelte';

	let songIdParam = $derived(page.url.searchParams.get('song_id'));
	let songIdNum = $derived(songIdParam ? Number(songIdParam) : undefined);
	let backHref = $derived(songIdNum !== undefined ? `/songs/${songIdNum}` : '/');
	let backLabel = $derived(songIdNum !== undefined ? 'Back to song' : 'Back to library');

	let queue = $state<VocabCard[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let done = $state(0);

	onMount(load);

	async function load() {
		loading = true;
		error = null;
		try {
			queue = await getVocabDrillQueue(songIdNum);
		} catch (e) {
			error = e instanceof Error ? e.message : String(e);
		} finally {
			loading = false;
		}
	}

	let current = $derived(queue[0] ?? null);

	async function answer(correct: boolean) {
		if (!current) return;
		const card = current;
		queue = queue.slice(1);
		done += 1;
		try {
			await recordVocabResult(card.song_id, card.vocab_id, correct);
		} catch (e) {
			error = e instanceof Error ? e.message : String(e);
		}
	}
</script>

<BackLink href={backHref} label={backLabel} />

<div class="mb-6 flex items-center justify-between">
	<h1 class="text-2xl font-semibold text-ink">Vocab Drill</h1>
	<span class="text-sm text-muted">{done} done · {queue.length} left</span>
</div>

{#if loading}
	<p class="text-muted">Loading queue…</p>
{:else if error}
	<p class="text-bad">{error}</p>
{:else if !current}
	<div class="rounded-2xl border border-border bg-surface p-8 text-center text-muted">
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
