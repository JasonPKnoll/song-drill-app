<script lang="ts">
    import { onMount, tick } from "svelte";
    import { blur } from "svelte/transition";
    import type { PageData } from "./$types";
    import Furigana from "$lib/components/Furigana.svelte";
    import BackLink from "$lib/components/BackLink.svelte";
    import ChevronDown from "@lucide/svelte/icons/chevron-down";
    import ChevronUp from "@lucide/svelte/icons/chevron-up";
    import Search from "@lucide/svelte/icons/search";

    let { data }: { data: PageData } = $props();

    let revealed = $state<Set<number>>(new Set());

    let realLineIds = $derived(
        data.song
            ? data.song.lines.filter((l) => l.reading !== "").map((l) => l.id)
            : [],
    );
    let allExpanded = $derived(
        realLineIds.length > 0 && realLineIds.every((id) => revealed.has(id)),
    );

    function toggle(lineId: number) {
        const next = new Set(revealed);
        if (next.has(lineId)) {
            next.delete(lineId);
        } else {
            next.add(lineId);
        }
        revealed = next;
    }

    function onCardKeydown(e: KeyboardEvent, lineId: number) {
        if (e.key === "Enter" || e.key === " ") {
            e.preventDefault();
            toggle(lineId);
        }
    }

    // The line currently at (or just above) the top of the viewport — used to
    // keep the view visually anchored when toggling all lines shifts content.
    function findAnchorEl(): HTMLElement | null {
        const els = document.querySelectorAll<HTMLElement>("[data-line-id]");
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
            window.scrollBy(
                0,
                anchorEl.getBoundingClientRect().top - beforeTop,
            );
        }
    }

    // Scrollspy: while scrolling through the middle of the song, the "look up
    // this line's vocab" button lives inside whichever real card is nearest
    // the vertical center of the viewport — not whichever is topmost. A
    // top-anchored band tends to pick a card that's only partially visible
    // (its top edge right at/above the viewport edge), which is exactly what
    // clips the button in its corner; centering the pick keeps the whole
    // card, and its button, comfortably on screen. Checked directly via
    // getBoundingClientRect on each scroll frame (read-only, ~40 elements —
    // cheap) rather than IntersectionObserver: the observer's async event
    // dispatch can coalesce a card's brief window in range during a fast
    // scroll and never report it at all. This is still a discrete decision,
    // not continuous position *animation* — state only changes (and only
    // then triggers a re-render) when the chosen card actually changes.
    //
    // Only one "slot" (a card's search button, or a toggle button) may ever
    // be on screen at once, and each transition is sequenced rather than
    // crossfaded: the outgoing element fully puffs away first, then — after
    // a matching gap — the new one puffs in. `slot` is the raw scrollspy
    // target; `displayed` is what's actually rendered, lagging behind by one
    // puff-out's worth of time whenever the target changes.
    const PUFF_MS = 220;
    const MAX_CENTER_DISTANCE_FRACTION = 0.4; // ignore cards whose center is further than this from viewport center (as a fraction of viewport height) — keeps the toggle buttons showing near the very top/bottom

    let slot = $state<number | "none">("none");
    let displayed = $state<number | "none">("none");
    let inGap = $state(false);
    let gapTimeout: ReturnType<typeof setTimeout> | null = null;

    function applySlot(next: number | "none") {
        if (next === slot) return; // no real change — don't reset an in-flight gap timer
        slot = next;
        if (next === displayed) return;
        if (gapTimeout) clearTimeout(gapTimeout);
        inGap = true;
        gapTimeout = setTimeout(() => {
            displayed = slot; // pick up wherever scrolling ended up, not necessarily `next`
            inGap = false;
            gapTimeout = null;
        }, PUFF_MS);
    }

    function computeCurrentCard(): number | "none" {
        const viewportCenter = window.innerHeight / 2;
        const maxDistance = window.innerHeight * MAX_CENTER_DISTANCE_FRACTION;
        let best: number | null = null;
        let bestDistance = Infinity;
        document
            .querySelectorAll('[data-line-id][role="button"]')
            .forEach((el) => {
                const r = el.getBoundingClientRect();
                if (r.bottom <= 0 || r.top >= window.innerHeight) return; // not visible at all
                const cardCenter = (r.top + r.bottom) / 2;
                const distance = Math.abs(cardCenter - viewportCenter);
                if (distance < bestDistance) {
                    bestDistance = distance;
                    best = Number(el.getAttribute("data-line-id"));
                }
            });
        return best !== null && bestDistance <= maxDistance ? best : "none";
    }

    // Being at the actual scroll ceiling always wins over whatever the band
    // check says: how tall the last card is (a grammar note can add a lot of
    // height) affects exactly how much scroll room it needs to reach the
    // band on its own, and there isn't always enough room left for it to.
    function recomputeSlot() {
        const atBottom =
            window.scrollY + window.innerHeight >=
            document.documentElement.scrollHeight - 2;
        applySlot(atBottom ? "none" : computeCurrentCard());
    }

    let ticking = false;
    function onScrollOrResize() {
        if (ticking) return;
        ticking = true;
        requestAnimationFrame(() => {
            recomputeSlot();
            ticking = false;
        });
    }

    onMount(() => {
        recomputeSlot();
        window.addEventListener("scroll", onScrollOrResize, { passive: true });
        window.addEventListener("resize", onScrollOrResize);
        return () => {
            window.removeEventListener("scroll", onScrollOrResize);
            window.removeEventListener("resize", onScrollOrResize);
            if (gapTimeout) clearTimeout(gapTimeout);
        };
    });
</script>

{#snippet toggleButton()}
    <button
        type="button"
        class="flex h-12 w-12 shrink-0 items-center justify-center rounded-full bg-accent text-bg shadow-lg shadow-black/30 transition-transform active:scale-95"
        onclick={toggleAll}
        aria-label={allExpanded ? "Collapse all lines" : "Expand all lines"}
        in:blur={{ duration: 260, amount: 7 }}
        out:blur={{ duration: 220, amount: 7 }}
    >
        {#if allExpanded}
            <ChevronUp size={24} strokeWidth={2.5} />
        {:else}
            <ChevronDown size={24} strokeWidth={2.5} />
        {/if}
    </button>
{/snippet}

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
        {#if displayed === "none" && !inGap}
            {@render toggleButton()}
        {/if}
    </div>

    <div class="flex flex-col gap-3">
        {#each song.lines as line, i (line.id)}
            {@const prevSection = i > 0 ? song.lines[i - 1].section : undefined}
            {#if line.section && line.section !== prevSection}
                <h2
                    class="mt-4 mb-1 text-sm font-semibold tracking-wide text-accent uppercase first:mt-0"
                >
                    {line.section}
                </h2>
            {/if}
            {#if line.reading === ""}
                <div
                    data-line-id={line.id}
                    class="rounded-2xl border border-border bg-surface/50 p-5 text-left text-muted italic"
                >
                    {line.text}
                </div>
            {:else}
                <div
                    data-line-id={line.id}
                    role="button"
                    tabindex="0"
                    class="relative rounded-2xl border border-border bg-surface p-5 text-left transition hover:border-accent/50"
                    onclick={() => toggle(line.id)}
                    onkeydown={(e) => onCardKeydown(e, line.id)}
                >
                    {#if revealed.has(line.id)}
                        <p class="text-xl leading-relaxed text-ink">
                            <Furigana furi={line.furi} />
                        </p>
                        <p class="mt-3 text-good">{line.natural}</p>
                        {#if line.grammar_note}
                            <p class="mt-2 text-sm text-muted">
                                {line.grammar_note}
                            </p>
                        {/if}
                    {:else}
                        <p class="text-xl leading-relaxed text-ink">
                            {line.text}
                        </p>
                    {/if}

                    {#if line.id === displayed && !inGap}
                        <a
                            href={`/songs/${song.id}/vocab?q=${encodeURIComponent(line.text)}`}
                            class="absolute top-3 right-3 flex h-10 w-10 items-center justify-center rounded-full bg-accent text-bg shadow-lg shadow-black/30 transition-transform hover:scale-105 active:scale-95"
                            onclick={(e) => e.stopPropagation()}
                            aria-label="Look up this line's vocab"
                            in:blur={{ duration: 260, amount: 7 }}
                            out:blur={{ duration: 220, amount: 7 }}
                        >
                            <Search size={18} strokeWidth={2.5} />
                        </a>
                    {/if}
                </div>
            {/if}
        {/each}
    </div>

    <div class="mt-6 flex justify-end">
        {#if displayed === "none" && !inGap}
            {@render toggleButton()}
        {/if}
    </div>
{:else}
    <p class="text-bad">Song not found.</p>
{/if}
