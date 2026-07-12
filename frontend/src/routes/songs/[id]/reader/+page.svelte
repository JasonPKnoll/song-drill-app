<script lang="ts">
	import { onMount, tick } from 'svelte';
	import type { PageData, Snapshot } from './$types';
	import BackLink from '$lib/components/BackLink.svelte';
	import ExpandCollapseButton from '$lib/components/ExpandCollapseButton.svelte';
	import ReaderLineCard from '$lib/components/ReaderLineCard.svelte';
	import { createScrollActionButton } from '$lib/reader/scrollActionButton.svelte';
	import { cn } from '$lib/utils/cn';

	let { data }: { data: PageData } = $props();

	let revealed = $state<Set<number>>(new Set());

	// Persists which lines are expanded across navigating away and back (e.g.
	// out to the vocab page's search result and back via its "back to
	// reader" button) — restored per SvelteKit's normal snapshot rules,
	// which key off the actual history entry: real back/forward navigation
	// restores it, a fresh link to this URL does not. Scroll position and
	// which card has the search icon aren't captured here — SvelteKit
	// restores scroll natively on back-navigation, and the scroll listener
	// in scrollActionButton.svelte.ts re-derives the icon's card from
	// wherever scroll ends up, so both just fall out of restoring scroll.
	export const snapshot: Snapshot<number[]> = {
		capture: () => [...revealed],
		restore: (value) => {
			revealed = new Set(value);
		}
	};

	let realLineIds = $derived(
		data.song ? data.song.lines.filter((l) => l.reading !== '').map((l) => l.id) : []
	);
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

	// Drives the single floating action button (expand/collapse-all at the
	// top/bottom, or a per-line "look up this line's vocab" search button
	// while scrolling through the middle) — see scrollActionButton.svelte.ts
	// for the five-zone state machine.
	const actionButton = createScrollActionButton(() => realLineIds);

	onMount(() => actionButton.mount());

	function hasSearchIcon(lineId: number): boolean {
		return actionButton.displayed === lineId && !actionButton.inGap;
	}

	// Centers the clicked line in the viewport, as far as there's scroll room
	// to do so — a card near the very top/bottom of the page just goes as far
	// as `scrollBy` can take it, which is already the closest achievable
	// approximation of centered. Plain (non-smooth) scroll, and deliberately
	// run *before* `pinToCard` below: pinning records the scroll position at
	// that moment as the pin's baseline, so centering first means the small
	// jitter-tolerance threshold is measured from the settled, centered
	// position — not from wherever the page happened to be pre-click, which
	// would otherwise make this very scroll immediately look like the user
	// scrolling away and release the pin it just set.
	function centerOnLine(lineId: number) {
		const el = document.querySelector<HTMLElement>(`[data-line-id="${lineId}"]`);
		if (!el) return;
		const rect = el.getBoundingClientRect();
		window.scrollBy(0, rect.top + rect.height / 2 - window.innerHeight / 2);
	}

	// Clicking an already-expanded card that doesn't currently have the
	// search icon just claims the icon for it — it doesn't collapse the
	// card. Otherwise a click always toggles reveal as normal. This means
	// bringing the icon over to a card you're already reading takes one
	// click, not one click-that-also-collapses-it followed by a second to
	// reopen it.
	async function handleCardClick(lineId: number) {
		const claimOnly = revealed.has(lineId) && !hasSearchIcon(lineId);
		if (!claimOnly) toggle(lineId);

		await tick(); // let a reveal/collapse's height change land before measuring
		centerOnLine(lineId);
		actionButton.pinToCard(lineId);
	}
</script>

{#if data.error}
	<p class="text-bad">Failed to load song: {data.error}</p>
{:else if data.song}
	{@const song = data.song}
	<BackLink href={`/songs/${song.id}`} label="Back to {song.title}" />

	<div class="mb-6 flex items-start justify-between gap-4">
		<div>
			<h1 class="text-2xl font-semibold text-ink">{song.title}</h1>
			<p class="text-muted">{song.artist} · Song reader</p>
		</div>
		{#if actionButton.displayed === 'toggle-top' && !actionButton.inGap}
			<ExpandCollapseButton expanded={allExpanded} onToggle={toggleAll} />
		{/if}
	</div>

	<div class="flex flex-col gap-3">
		{#each song.lines as line, i (line.id)}
			{@const prevSection = i > 0 ? song.lines[i - 1].section : undefined}
			{#if line.section && line.section !== prevSection}
				<h2
					class={cn(
						'mt-4 mb-1',
						'text-sm font-semibold tracking-wide uppercase',
						'text-accent',
						'first:mt-0'
					)}
				>
					{line.section}
				</h2>
			{/if}
			<ReaderLineCard
				{line}
				revealed={revealed.has(line.id)}
				onCardClick={() => handleCardClick(line.id)}
				onHoverStart={() => actionButton.hoverCard(line.id)}
				onHoverEnd={() => actionButton.unhoverCard(line.id)}
				searchHref={`/songs/${song.id}/vocab?q=${encodeURIComponent(line.text)}`}
				showSearchButton={hasSearchIcon(line.id)}
			/>
		{/each}
	</div>

	<div class="mt-6 flex justify-end">
		{#if actionButton.displayed === 'toggle-bottom' && !actionButton.inGap}
			<ExpandCollapseButton expanded={allExpanded} onToggle={toggleAll} />
		{/if}
	</div>
{:else}
	<p class="text-bad">Song not found.</p>
{/if}
