<script lang="ts">
	import type { Line } from '$lib/api';
	import Furigana from './Furigana.svelte';
	import LineSearchButton from './LineSearchButton.svelte';

	let {
		line,
		revealed,
		onToggleReveal,
		searchHref,
		showSearchButton
	}: {
		line: Line;
		revealed: boolean;
		onToggleReveal: () => void;
		searchHref: string;
		showSearchButton: boolean;
	} = $props();

	function onKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter' || e.key === ' ') {
			e.preventDefault();
			onToggleReveal();
		}
	}
</script>

{#if line.reading === ''}
	<!-- Junk line (section headers, ad-libs, etc. with no reading annotation) — not interactive. -->
	<div data-line-id={line.id} class="rounded-2xl border border-border bg-surface/50 p-5 text-left text-muted italic">
		{line.text}
	</div>
{:else}
	<div
		data-line-id={line.id}
		role="button"
		tabindex="0"
		class="relative rounded-2xl border border-border bg-surface p-5 text-left transition hover:border-accent/50"
		onclick={onToggleReveal}
		onkeydown={onKeydown}
	>
		{#if revealed}
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

		{#if showSearchButton}
			<LineSearchButton href={searchHref} />
		{/if}
	</div>
{/if}
