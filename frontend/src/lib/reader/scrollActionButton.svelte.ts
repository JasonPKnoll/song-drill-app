// Drives the reader's single floating action button, which is always
// exactly one of: the expand/collapse-all toggle, or a "look up this line's
// vocab" search button attached to one specific card. Extracted out of
// +page.svelte so the scroll-tracking/state-machine logic can be reasoned
// about (and changed) independently of the page's markup.
//
// Search-button targeting uses one continuous formula, not a set of
// hand-off zones: the point in the *viewport* we aim for slides linearly
// with how far through the whole scrollable page we are. 2% scrolled aims
// near the top of the screen; 25% aims at the screen's first quarter; 50%
// aims at the screen's center; 98% aims near the bottom. Whichever real
// card is nearest that point gets the button.
//
// An earlier version tried two separate metrics — nearest-to-viewport-edge
// for the first/last few cards, nearest-to-viewport-center for everything
// else — handed off at a hard boundary. That kept skipping cards right at
// the boundary: the two metrics advance through the card list at different
// rates (edge-anchoring only lets go of a card once it's fully scrolled
// off-screen, which takes much longer than the moment it stops being the
// best center match), so whichever metric was "behind" at the handoff
// instant would jump straight past cards the other had already moved on
// from. A single sliding target has no seam to desync across, so nothing
// gets skipped — it's just nearest-neighbor lookup against a target that
// happens to move.
//
// "top-cards" / "bottom-cards" below are the *labels* for when the natural
// target has slid near enough to an edge that it's within the first/last N
// cards — they don't change the math, just describe it for readability.
const PUFF_MS = 220;
const TOP_ZONE_COUNT = 4;
const BOTTOM_ZONE_COUNT = 4;
// Entering "at the top/bottom" uses a tight threshold; leaving it again needs
// a much larger one. Without this gap, a few px of scroll jitter right at the
// edge — momentum settling, iOS rubber-band overscroll bouncing back — flips
// between the toggle and the last card's search button over and over, each
// flip re-triggering the full puff-out/puff-in transition. A sticky dead
// zone absorbs that jitter: once we've committed to the edge state, normal
// bounce/settle noise isn't enough to kick us back out of it.
const ENTER_EDGE_EPSILON = 4;
const EXIT_EDGE_EPSILON = 56;

export type Zone = 'top' | 'top-cards' | 'middle' | 'bottom-cards' | 'bottom';
export type ButtonTarget = 'toggle-top' | 'toggle-bottom' | number;

// How far through the whole scrollable page we are, as a 0..1 fraction.
function scrollFraction(): number {
	const maxScrollY = document.documentElement.scrollHeight - window.innerHeight;
	if (maxScrollY <= 0) return 0;
	return Math.min(1, Math.max(0, window.scrollY / maxScrollY));
}

// Whichever real card (rendered with a `data-line-id` + `role="button"`) has
// its vertical center nearest `referenceY` (a point in viewport coordinates).
// Prefers cards fully on screen — this is what guarantees the button (which
// sits in a corner of whichever card gets picked) never gets clipped by the
// viewport edge — and only falls back to a partially-visible one if nothing
// is fully visible (e.g. a single card taller than the viewport itself).
function nearestCard(referenceY: number): number | null {
	const els = document.querySelectorAll<HTMLElement>('[data-line-id][role="button"]');
	for (const requireFullyVisible of [true, false]) {
		let best: number | null = null;
		let bestDistance = Infinity;
		els.forEach((el) => {
			const r = el.getBoundingClientRect();
			const overlapsViewport = r.bottom > 0 && r.top < window.innerHeight;
			if (!overlapsViewport) return;
			if (requireFullyVisible && (r.top < 0 || r.bottom > window.innerHeight)) return;
			const distance = Math.abs((r.top + r.bottom) / 2 - referenceY);
			if (distance < bestDistance) {
				bestDistance = distance;
				best = Number(el.getAttribute('data-line-id'));
			}
		});
		if (best !== null) return best;
	}
	return null;
}

export class ScrollActionButton {
	#lineIds: () => number[];

	// The raw, up-to-the-frame scroll target.
	slot = $state<ButtonTarget>('toggle-top');
	zone = $state<Zone>('top');
	// What's actually rendered right now — lags behind `slot` by one puff
	// transition whenever the target changes, so the outgoing element can
	// fully dissolve before the new one appears (see `#applySlot`).
	displayed = $state<ButtonTarget>('toggle-top');
	inGap = $state(false);

	#gapTimeout: ReturnType<typeof setTimeout> | null = null;
	#ticking = false;
	#stickyAtTop = false;
	#stickyAtBottom = false;
	#onScrollOrResize = () => {
		if (this.#ticking) return;
		this.#ticking = true;
		requestAnimationFrame(() => {
			this.recompute();
			this.#ticking = false;
		});
	};

	constructor(lineIds: () => number[]) {
		this.#lineIds = lineIds;
	}

	#applySlot(next: ButtonTarget) {
		if (next === this.slot) return; // no real change — don't reset an in-flight gap timer
		this.slot = next;
		if (next === this.displayed) return;
		if (this.#gapTimeout) clearTimeout(this.#gapTimeout);
		this.inGap = true;
		this.#gapTimeout = setTimeout(() => {
			this.displayed = this.slot; // pick up wherever scrolling ended up, not necessarily `next`
			this.inGap = false;
			this.#gapTimeout = null;
		}, PUFF_MS);
	}

	// Hysteresis wrapper around "how close to the top/bottom is scrollY":
	// entering the edge state uses the tight threshold, leaving it again
	// needs the much larger one, with which threshold currently applies
	// depending on the sticky flag from the previous call.
	#isAtTop(): boolean {
		const threshold = this.#stickyAtTop ? EXIT_EDGE_EPSILON : ENTER_EDGE_EPSILON;
		this.#stickyAtTop = window.scrollY <= threshold;
		return this.#stickyAtTop;
	}

	#isAtBottom(): boolean {
		const maxScrollY = document.documentElement.scrollHeight - window.innerHeight;
		const threshold = this.#stickyAtBottom ? EXIT_EDGE_EPSILON : ENTER_EDGE_EPSILON;
		this.#stickyAtBottom = window.scrollY >= maxScrollY - threshold;
		return this.#stickyAtBottom;
	}

	#classify(): { zone: Zone; target: ButtonTarget } {
		if (this.#isAtTop()) return { zone: 'top', target: 'toggle-top' };
		if (this.#isAtBottom()) return { zone: 'bottom', target: 'toggle-bottom' };

		const referenceY = scrollFraction() * window.innerHeight;
		const pick = nearestCard(referenceY);
		if (pick === null) return { zone: 'top', target: 'toggle-top' }; // safety net: nothing visible at all

		const ids = this.#lineIds();
		const idx = ids.indexOf(pick);
		let zone: Zone = 'middle';
		if (idx !== -1 && idx < TOP_ZONE_COUNT) zone = 'top-cards';
		else if (idx !== -1 && idx >= ids.length - BOTTOM_ZONE_COUNT) zone = 'bottom-cards';

		return { zone, target: pick };
	}

	// Being at the actual scroll ceiling/floor always wins over the sliding
	// target check: how tall the first/last card is (a grammar note can add a
	// lot of height) affects exactly how much scroll room it needs, and there
	// isn't always enough for the classification to land on 'top'/'bottom'
	// otherwise.
	recompute() {
		const { zone, target } = this.#classify();
		this.zone = zone;
		this.#applySlot(target);
	}

	mount(): () => void {
		this.recompute();
		window.addEventListener('scroll', this.#onScrollOrResize, { passive: true });
		window.addEventListener('resize', this.#onScrollOrResize);
		return () => {
			window.removeEventListener('scroll', this.#onScrollOrResize);
			window.removeEventListener('resize', this.#onScrollOrResize);
			if (this.#gapTimeout) clearTimeout(this.#gapTimeout);
		};
	}
}

export function createScrollActionButton(lineIds: () => number[]): ScrollActionButton {
	return new ScrollActionButton(lineIds);
}
