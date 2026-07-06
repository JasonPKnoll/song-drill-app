<script lang="ts">
	import { tick } from 'svelte';
	import type { PageData } from './$types';
	import Furigana from '$lib/components/Furigana.svelte';
	import BackLink from '$lib/components/BackLink.svelte';
	import ChevronDown from '@lucide/svelte/icons/chevron-down';
	import ChevronUp from '@lucide/svelte/icons/chevron-up';

	let { data }: { data: PageData } = $props();

	let revealed = $state<Set<number>>(new Set());

	let realLineIds = $derived(data.song ? data.song.lines.filter((l) => l.reading !== '').map((l) => l.id) : []);
	let allExpanded = $derived(realLineIds.length > 0 && realLineIds.every((id) => revealed.has(id)));

	function toggle(lineId: number) {
		const next = new Set(revealed);
		if (next.has(lineId)) {
			next.delete(lineId);
		} else {
			next.add(lineId);
		}
		revealed = next;
	}

	// The line currently at (or just above) the top of the viewport — used to
	// keep the view visually anchored when toggling all lines shifts content.
	function findAnchorEl(): HTMLElement | null {
		const els = document.querySelectorAll<HTMLElement>('[data-line-id]');
		for (const el of els) {
			if (el.getBoundingClientRect().bottom > 0) return el;
		}
		return null;
	}

	async function toggleAll() {
		const anchorEl = findAnchorEl();
		const beforeTop = anchorEl?.getBoundingClientRect().top ?? null;

		revealed = allExpanded ? new Set() : new Set(realLineIds);

		await tick();

		if (anchorEl && beforeTop !== null) {
			window.scrollBy(0, anchorEl.getBoundingClientRect().top - beforeTop);
		}
	}
</script>

{#if data.error}
	<p class="text-bad">Failed to load song: {data.error}</p>
{:else if data.song}
	{@const song = data.song}
	<BackLink href={`/songs/${song.id}`} label="Back to {song.title}" />

	<div class="mb-6">
		<h1 class="text-2xl font-semibold text-ink">{song.title}</h1>
		<p class="text-muted">{song.artist} · Song reader</p>
	</div>

	<div class="flex flex-col gap-3">
		{#each song.lines as line, i (line.id)}
			{@const prevSection = i > 0 ? song.lines[i - 1].section : undefined}
			{#if line.section && line.section !== prevSection}
				<h2 class="mt-4 mb-1 text-sm font-semibold tracking-wide text-accent uppercase first:mt-0">
					{line.section}
				</h2>
			{/if}
			{#if line.reading === ''}
				<div data-line-id={line.id} class="rounded-2xl border border-border bg-surface/50 p-5 text-left text-muted italic">
					{line.text}
				</div>
			{:else}
				<button
					data-line-id={line.id}
					type="button"
					class="rounded-2xl border border-border bg-surface p-5 text-left transition hover:border-accent/50"
					onclick={() => toggle(line.id)}
				>
					{#if revealed.has(line.id)}
						<p class="text-xl leading-relaxed text-ink">
							<Furigana furi={line.furi} />
						</p>
						<p class="mt-3 text-good">{line.natural}</p>
						{#if line.grammar_note}
							<p class="mt-2 text-sm text-muted">{line.grammar_note}</p>
						{/if}
					{:else}
						<p class="text-xl leading-relaxed text-ink">{line.text}</p>
					{/if}
				</button>
			{/if}
		{/each}
	</div>

	<button
		type="button"
		class="fixed right-4 bottom-[calc(1rem+env(safe-area-inset-bottom))] z-20 flex h-12 w-12 items-center justify-center rounded-full bg-accent text-bg shadow-lg shadow-black/40 transition active:scale-95"
		onclick={toggleAll}
		aria-label={allExpanded ? 'Collapse all lines' : 'Expand all lines'}
	>
		{#if allExpanded}
			<ChevronUp size={24} strokeWidth={2.5} />
		{:else}
			<ChevronDown size={24} strokeWidth={2.5} />
		{/if}
	</button>
{:else}
	<p class="text-bad">Song not found.</p>
{/if}
