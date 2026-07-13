<script lang="ts">
	import '../app.css';
	import { onMount } from 'svelte';
	import type { Snippet } from 'svelte';
	import { cn } from '$lib/utils/cn';
	import ProfileSwitcher from '$lib/components/ProfileSwitcher.svelte';

	let { children }: { children: Snippet } = $props();

	// 言 -> Koto (ink), 霊 -> Dama (accent) — capitalizing only the first
	// letter of each half (not full caps) marks the same word+spirit
	// compound boundary the color split marks, instead of shouting it.
	const kanji = [
		{ ch: '言', accent: false },
		{ ch: '霊', accent: true }
	];
	const romaji = [
		{ ch: 'K', accent: false },
		{ ch: 'o', accent: false },
		{ ch: 't', accent: false },
		{ ch: 'o', accent: false },
		{ ch: 'D', accent: true },
		{ ch: 'a', accent: true },
		{ ch: 'm', accent: true },
		{ ch: 'a', accent: true }
	];

	const INITIAL_HOLD = 900; // let 言霊 sit still for a beat before anything moves
	const KANJI_STAGGER = 160;
	const ROMAJI_START = INITIAL_HOLD + 280; // starts while kanji is still mid-dissolve
	const ROMAJI_STAGGER = 85;

	// Replay the roll if the wordmark has been scrolled fully out of view for
	// at least this long, once it's fully back in view — a quick scroll-past
	// shouldn't retrigger it, only actually being away for a while.
	const RESET_AFTER_HIDDEN_MS = 2000;

	let replayKey = $state(0);
	let wordmarkEl: HTMLAnchorElement;

	onMount(() => {
		let hiddenSince: number | null = null;

		const observer = new IntersectionObserver(
			([entry]) => {
				if (!entry) return;
				if (entry.intersectionRatio === 0) {
					hiddenSince = Date.now();
				} else if (entry.intersectionRatio >= 0.99) {
					if (hiddenSince !== null && Date.now() - hiddenSince >= RESET_AFTER_HIDDEN_MS) {
						replayKey += 1;
					}
					hiddenSince = null;
				}
			},
			{ threshold: [0, 0.99, 1] }
		);

		observer.observe(wordmarkEl);
		return () => observer.disconnect();
	});
</script>

<div
	class={cn(
		'min-h-screen',
		'bg-bg'
	)}
>
	<header class="border-b border-border">
		<div class="mx-auto flex max-w-3xl items-center justify-between px-4 py-4">
			<a href="/" class="wordmark" aria-label="Kotodama" bind:this={wordmarkEl}>
				{#key replayKey}
					<span class="wm wm-kanji" aria-hidden="true">
						{#each kanji as { ch, accent }, i (i)}
							<span
								class="letter letter-out"
								class:accent
								style="animation-delay: {INITIAL_HOLD + i * KANJI_STAGGER}ms"
							>
								{ch}
							</span>
						{/each}
					</span>
					<span class="wm wm-romaji" aria-hidden="true">
						{#each romaji as { ch, accent }, i (i)}
							<span
								class="letter letter-in"
								class:accent
								style="animation-delay: {ROMAJI_START + i * ROMAJI_STAGGER}ms"
							>
								{ch}
							</span>
						{/each}
					</span>
				{/key}
			</a>
			<ProfileSwitcher />
		</div>
	</header>

	<main class="mx-auto max-w-3xl px-4 py-6">
		{@render children()}
	</main>
</div>

<style>
	/* Plays once per full page load: 言霊 rolls away one character at a time
	   (each letter dissolving — blur, shrink, drift right — in sequence)
	   while KotoDama rolls in behind it the same way, letter by letter, so
	   the transformation itself travels left to right instead of the whole
	   word wiping as one block. Settles and stays put — this is a root
	   layout, so it only remounts on a hard reload, not client-side nav. */

	.wordmark {
		position: relative;
		display: inline-block;
		height: 1.6rem;
		min-width: 8rem;
	}

	.wm {
		position: absolute;
		inset: 0;
		display: flex;
		align-items: baseline;
		white-space: nowrap;
	}

	.letter {
		display: inline-block;
		color: var(--color-ink);
	}
	.letter.accent {
		color: var(--color-accent);
	}

	.wm-kanji {
		font-family: 'Hiragino Mincho ProN', 'YuMincho', 'Noto Serif JP', 'Songti SC', serif;
		font-size: 1.35rem;
		letter-spacing: 0.04em;
	}

	.wm-romaji {
		font-family: 'Iowan Old Style', 'Palatino Linotype', 'Book Antiqua', Georgia, serif;
		font-size: 1.15rem;
		letter-spacing: 0.02em;
	}

	/* Default (no motion / reduced-motion): settle straight to the final
	   state, no flash of the wrong content. */
	.letter-out {
		opacity: 0;
	}
	.letter-in {
		opacity: 1;
	}

	@media (prefers-reduced-motion: no-preference) {
		.letter-out {
			opacity: 1;
			animation: letterOut 480ms cubic-bezier(0.32, 0, 0.67, 0) both;
		}
		.letter-in {
			opacity: 0;
			animation: letterIn 480ms cubic-bezier(0.33, 1, 0.68, 1) both;
		}
		.letter-in.accent {
			animation-name: letterInGlow;
		}
	}

	@keyframes letterOut {
		0% {
			opacity: 1;
			filter: blur(0);
			transform: translateX(0) scale(1);
		}
		100% {
			opacity: 0;
			filter: blur(3px);
			transform: translateX(5px) scale(0.88);
		}
	}
	@keyframes letterIn {
		0% {
			opacity: 0;
			filter: blur(3px);
			transform: translateX(-5px) scale(0.88);
		}
		100% {
			opacity: 1;
			filter: blur(0);
			transform: translateX(0) scale(1);
		}
	}
	@keyframes letterInGlow {
		0% {
			opacity: 0;
			filter: blur(3px);
			transform: translateX(-5px) scale(0.88);
			text-shadow: none;
		}
		70% {
			text-shadow: 0 0 10px rgba(167, 139, 250, 0.85);
		}
		100% {
			opacity: 1;
			filter: blur(0);
			transform: translateX(0) scale(1);
			text-shadow: none;
		}
	}
</style>
