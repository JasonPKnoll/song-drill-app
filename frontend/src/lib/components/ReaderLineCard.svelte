<script lang="ts">
	import type { Line } from '$lib/api';
	import Furigana from './Furigana.svelte';
	import LineSearchButton from './LineSearchButton.svelte';
	import { cn } from '$lib/utils/cn';

	let {
		line,
		revealed,
		onCardClick,
		onHoverStart,
		onHoverEnd,
		searchHref,
		showSearchButton
	}: {
		line: Line;
		revealed: boolean;
		onCardClick: () => void;
		onHoverStart: () => void;
		onHoverEnd: () => void;
		searchHref: string;
		showSearchButton: boolean;
	} = $props();

	function onKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter' || e.key === ' ') {
			e.preventDefault();
			onCardClick();
		}
	}

	// Hover-priority is a mouse-only affordance. Touch has no real "leave" —
	// tapping a card fires a synthetic pointerenter with nothing to ever fire
	// a matching pointerleave, which would otherwise glue the search button
	// to that card permanently (the click-pin has a scroll-release; hover
	// deliberately doesn't, since a real mouse leaving is a reliable signal).
	// Filtering to pointerType === 'mouse' keeps touch taps on the click-pin
	// path only.
	function onPointerEnter(e: PointerEvent) {
		if (e.pointerType === 'mouse') onHoverStart();
	}
	function onPointerLeave(e: PointerEvent) {
		if (e.pointerType === 'mouse') onHoverEnd();
	}
</script>

{#if line.reading === ''}
	<!-- Junk line (section headers, ad-libs, etc. with no reading annotation) — not interactive. -->
	<div
		data-line-id={line.id}
		class={cn(
			'p-5',
			'text-left italic',
			'border border-border bg-surface/50 text-muted',
			'rounded-2xl'
		)}
	>
		{line.text}
	</div>
{:else}
	<div
		data-line-id={line.id}
		role="button"
		tabindex="0"
		class={cn(
			'relative p-5',
			'text-left',
			'border border-border bg-surface',
			'rounded-2xl transition hover:border-accent/50'
		)}
		onclick={onCardClick}
		onkeydown={onKeydown}
		onpointerenter={onPointerEnter}
		onpointerleave={onPointerLeave}
	>
		{#if revealed}
			<p class={cn('text-xl leading-relaxed', 'text-ink', { 'pr-14': showSearchButton })}>
				<Furigana furi={line.furi} />
			</p>
			<p class="mt-3 text-good">{line.natural}</p>
			{#if line.grammar_note}
				<p class="mt-2 text-sm text-muted">{line.grammar_note}</p>
			{/if}
		{:else}
			<p class={cn('text-xl leading-relaxed', 'text-ink', { 'pr-14': showSearchButton })}>{line.text}</p>
		{/if}

		{#if showSearchButton}
			<LineSearchButton href={searchHref} />
		{/if}
	</div>
{/if}
