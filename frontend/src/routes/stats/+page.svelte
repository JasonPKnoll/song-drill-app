<script lang="ts">
	import type { PageData } from './$types';
	import { listVocabProgress, burnVocabProgress, resetVocabProgress, type VocabProgressItem } from '$lib/api';
	import Furigana from '$lib/components/Furigana.svelte';
	import BackLink from '$lib/components/BackLink.svelte';
	import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
	import { cn } from '$lib/utils/cn';

	let { data }: { data: PageData } = $props();

	let items = $state<VocabProgressItem[]>([]);
	let query = $state('');
	let bucketFilter = $state<'all' | 'new' | 'progress' | 'done' | 'burned'>('all');
	let busyId = $state<number | null>(null);
	let actionError = $state<string | null>(null);
	let resetTarget = $state<VocabProgressItem | null>(null);
	let markKnownTarget = $state<VocabProgressItem | null>(null);

	$effect(() => {
		items = data.items;
	});

	// Every row shares the same song_title when this page is scoped to one
	// song (data.songId set) — read it off the data instead of a separate
	// fetch just for a heading/back-link label.
	let songTitle = $derived(data.songId !== undefined ? (items[0]?.song_title ?? null) : null);

	// Same three-way grouping as the drill counters (blue new / purple in
	// progress / green done) — learning and relearning both read as
	// "still being worked on" here, there's no reason to split them further
	// on a progress overview. Mastered/burned cards are pulled into their
	// own bucket regardless of stage (in practice always "review"), so
	// "Done" reads as "graduated and still on its own schedule" while
	// "Burned" is specifically the manually-deactivated ones.
	function bucket(it: VocabProgressItem): 'new' | 'progress' | 'done' | 'burned' {
		if (it.mastered) return 'burned';
		if (it.state === 'new') return 'new';
		if (it.state === 'review') return 'done';
		return 'progress';
	}

	let counts = $derived.by(() => {
		const c = { new: 0, progress: 0, done: 0, burned: 0 };
		for (const it of items) c[bucket(it)]++;
		return c;
	});

	let filtered = $derived.by(() => {
		let list = items;
		if (bucketFilter !== 'all') list = list.filter((it) => bucket(it) === bucketFilter);
		const raw = query.trim();
		if (raw) {
			const q = raw.toLowerCase();
			list = list.filter(
				(it) =>
					it.surface.includes(raw) ||
					it.reading.includes(raw) ||
					it.base_meaning.toLowerCase().includes(q) ||
					it.song_title.toLowerCase().includes(q)
			);
		}
		// Known words are done and out of the way — sink them to the bottom
		// (stable partition, so relative order within each group is otherwise
		// untouched) rather than mixing them in with words still being worked on.
		const active = list.filter((it) => !it.mastered);
		const known = list.filter((it) => it.mastered);
		return [...active, ...known];
	});

	function itemKey(it: VocabProgressItem): number {
		// Progress is per (song, vocab), not just vocab — the same word can
		// appear once per song it's in, each with its own SRS track.
		return it.song_id * 1_000_000 + it.vocab_id;
	}

	async function refresh() {
		items = await listVocabProgress(data.songId);
	}

	async function confirmMarkKnown() {
		const it = markKnownTarget;
		markKnownTarget = null;
		if (!it) return;
		busyId = itemKey(it);
		actionError = null;
		try {
			await burnVocabProgress(it.song_id, it.vocab_id);
			await refresh();
		} catch (e) {
			actionError = e instanceof Error ? e.message : String(e);
		} finally {
			busyId = null;
		}
	}

	async function confirmReset() {
		const it = resetTarget;
		resetTarget = null;
		if (!it) return;
		busyId = itemKey(it);
		actionError = null;
		try {
			await resetVocabProgress(it.song_id, it.vocab_id);
			await refresh();
		} catch (e) {
			actionError = e instanceof Error ? e.message : String(e);
		} finally {
			busyId = null;
		}
	}

	const filterChipClass = (active: boolean) =>
		cn(
			'px-3 py-1',
			'text-sm',
			'rounded-full border transition',
			active ? 'border-accent bg-accent/10 text-accent' : 'border-border text-muted hover:border-accent/50'
		);
</script>

<BackLink
	href={data.songId !== undefined ? `/songs/${data.songId}` : '/'}
	label={data.songId !== undefined ? `Back to ${songTitle ?? 'song'}` : 'Back to library'}
/>

<div class="mb-6 flex items-center justify-between">
	<h1 class="text-2xl font-semibold text-ink">
		{songTitle ? `${songTitle} · Progress` : 'Progress'}
	</h1>
</div>

{#if data.error}
	<p class="text-bad">Failed to load progress: {data.error}</p>
{:else}
	<div class="mb-4 flex flex-wrap items-center gap-2">
		<button type="button" class={filterChipClass(bucketFilter === 'all')} onclick={() => (bucketFilter = 'all')}>
			All {items.length}
		</button>
		<button type="button" class={filterChipClass(bucketFilter === 'new')} onclick={() => (bucketFilter = 'new')}>
			<span class="mr-1.5 inline-block h-2 w-2 rounded-full bg-new"></span>New {counts.new}
		</button>
		<button
			type="button"
			class={filterChipClass(bucketFilter === 'progress')}
			onclick={() => (bucketFilter = 'progress')}
		>
			<span class="mr-1.5 inline-block h-2 w-2 rounded-full bg-accent"></span>In progress {counts.progress}
		</button>
		<button type="button" class={filterChipClass(bucketFilter === 'done')} onclick={() => (bucketFilter = 'done')}>
			<span class="mr-1.5 inline-block h-2 w-2 rounded-full bg-good"></span>Done {counts.done}
		</button>
		<button
			type="button"
			class={filterChipClass(bucketFilter === 'burned')}
			onclick={() => (bucketFilter = 'burned')}
		>
			<span class="mr-1.5 inline-block h-2 w-2 rounded-full bg-good grayscale"></span>Burned {counts.burned}
		</button>
	</div>

	<input
		type="text"
		bind:value={query}
		placeholder={data.songId !== undefined
			? 'Search word, reading, or meaning…'
			: 'Search word, reading, meaning, or song…'}
		class={cn(
			'mb-4 w-full px-4 py-2',
			'border border-border bg-surface text-ink placeholder:text-muted',
			'rounded-xl',
			'focus:border-accent focus:outline-none'
		)}
	/>

	{#if actionError}
		<p class="mb-3 text-sm text-bad">{actionError}</p>
	{/if}

	{#if filtered.length === 0}
		<div
			class={cn(
				'p-8',
				'text-center',
				'border border-border bg-surface text-muted',
				'rounded-2xl'
			)}
		>
			{items.length === 0 ? 'No vocab in the library yet.' : `No words match "${query}".`}
		</div>
	{:else}
		<p class="mb-3 text-sm text-muted">{filtered.length} of {items.length} words</p>
		<ul class="flex flex-col gap-2">
			{#each filtered as it (itemKey(it))}
				{@const b = bucket(it)}
				<li
					class={cn(
						'flex flex-wrap items-center gap-3 p-4',
						'border border-border bg-surface',
						'rounded-xl',
						it.mastered && 'grayscale opacity-60'
					)}
				>
					<span
						class={cn(
							'h-2.5 w-2.5 shrink-0 rounded-full',
							b === 'new' ? 'bg-new' : b === 'progress' ? 'bg-accent' : 'bg-good'
						)}
						aria-hidden="true"
					></span>

					<div class="min-w-0 flex-1">
						<div class="flex items-baseline gap-2">
							<p class="text-lg text-ink">
								<Furigana furi={it.furi} />
							</p>
							{#if data.songId === undefined}
								<span class="text-sm text-muted">{it.song_title}</span>
							{/if}
						</div>
						<p class="truncate text-sm text-good">{it.base_meaning}</p>
						{#if it.state !== 'new'}
							<p class="text-xs text-muted">
								{it.mastered ? 'Mastered' : it.state} · seen {it.seen} · correct {it.correct}
							</p>
						{/if}
					</div>

					<div class="flex shrink-0 gap-2">
						{#if it.state !== 'new'}
							<button
								type="button"
								disabled={busyId === itemKey(it)}
								class={cn(
									'px-3 py-1.5 text-sm font-medium',
									'border border-bad bg-bad/10 text-bad',
									'rounded-lg transition hover:bg-bad/20',
									'disabled:opacity-50'
								)}
								onclick={() => (resetTarget = it)}
							>
								Reset
							</button>
						{/if}
						{#if it.mastered}
							<span class="px-3 py-1.5 text-sm font-medium text-good">✓ Known</span>
						{:else}
							<button
								type="button"
								disabled={busyId === itemKey(it)}
								class={cn(
									'px-3 py-1.5 text-sm font-medium',
									'border border-good bg-good/10 text-good',
									'rounded-lg transition hover:bg-good/20',
									'disabled:opacity-50'
								)}
								onclick={() => (markKnownTarget = it)}
							>
								Mark known
							</button>
						{/if}
					</div>
				</li>
			{/each}
		</ul>
	{/if}
{/if}

<ConfirmDialog
	open={resetTarget !== null}
	title="Reset progress?"
	message={resetTarget
		? `Reset progress on "${resetTarget.surface}" (${resetTarget.song_title})? This can't be undone.`
		: ''}
	confirmLabel="Reset"
	danger
	onConfirm={confirmReset}
	onCancel={() => (resetTarget = null)}
/>

<ConfirmDialog
	open={markKnownTarget !== null}
	title="Mark as known?"
	message={markKnownTarget
		? `Mark "${markKnownTarget.surface}" as known? This skips its normal review schedule and marks it mastered — you can always reset it back to new later.`
		: ''}
	confirmLabel="Mark known"
	onConfirm={confirmMarkKnown}
	onCancel={() => (markKnownTarget = null)}
/>
