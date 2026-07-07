// Drives the reader's single floating action button, which is always
// exactly one of: the expand/collapse-all toggle, or a "look up this line's
// vocab" search button attached to one specific card. Extracted out of
// +page.svelte so the scroll-tracking/state-machine logic can be reasoned
// about (and changed) independently of the page's markup.
//
// Five behavior zones, driven purely by scroll position:
//   1. top           — scrolled to the very top            -> toggle button, in the header
//   2. top-cards     — scrolling through the first N cards -> search button, pinned to nearest of those N
//   3. middle        — everything else                      -> search button, pinned to whichever card is nearest viewport center
//   4. bottom-cards  — scrolling through the last N cards   -> search button, pinned to nearest of those N
//   5. bottom        — scrolled to the true bottom          -> toggle button, below the last card
//
// Zones 2-4 all render the same way (search button on the nearest visible
// card) — they're kept as named zones rather than collapsed into one branch
// because "nearest card" alone breaks down at the very first/last cards:
// there's no scroll room left to bring their centers to the viewport's
// center, so a naive distance-to-center rule would either never pick them or
// need an awkward cutoff that excludes them. Zones 2 and 4 remove that
// cutoff for exactly the first/last N cards, guaranteeing every real card
// gets the button at some point while scrolling past it.
const PUFF_MS = 220;
const TOP_ZONE_COUNT = 4;
const BOTTOM_ZONE_COUNT = 4;
const EDGE_EPSILON = 2; // px slack for "at true top/bottom of the document" checks

export type Zone = 'top' | 'top-cards' | 'middle' | 'bottom-cards' | 'bottom';
export type ButtonTarget = 'toggle-top' | 'toggle-bottom' | number;

function isAtTop(): boolean {
	return window.scrollY <= EDGE_EPSILON;
}

function isAtBottom(): boolean {
	return window.scrollY + window.innerHeight >= document.documentElement.scrollHeight - EDGE_EPSILON;
}

// Whichever real card (rendered with a `data-line-id` + `role="button"`) has
// its vertical center nearest the viewport's vertical center, among cards
// that are at least partially visible. No max-distance cutoff: zones 2-4
// need a card picked even when it's nowhere near centered.
function nearestVisibleCardId(): number | null {
	const viewportCenter = window.innerHeight / 2;
	let best: number | null = null;
	let bestDistance = Infinity;
	document.querySelectorAll<HTMLElement>('[data-line-id][role="button"]').forEach((el) => {
		const r = el.getBoundingClientRect();
		if (r.bottom <= 0 || r.top >= window.innerHeight) return; // not visible at all
		const cardCenter = (r.top + r.bottom) / 2;
		const distance = Math.abs(cardCenter - viewportCenter);
		if (distance < bestDistance) {
			bestDistance = distance;
			best = Number(el.getAttribute('data-line-id'));
		}
	});
	return best;
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

	#classify(): { zone: Zone; target: ButtonTarget } {
		if (isAtTop()) return { zone: 'top', target: 'toggle-top' };
		if (isAtBottom()) return { zone: 'bottom', target: 'toggle-bottom' };

		const nearest = nearestVisibleCardId();
		if (nearest === null) return { zone: 'top', target: 'toggle-top' };

		const ids = this.#lineIds();
		const idx = ids.indexOf(nearest);
		if (idx !== -1 && idx < TOP_ZONE_COUNT) return { zone: 'top-cards', target: nearest };
		if (idx !== -1 && idx >= ids.length - BOTTOM_ZONE_COUNT) return { zone: 'bottom-cards', target: nearest };
		return { zone: 'middle', target: nearest };
	}

	// Being at the actual scroll ceiling/floor always wins over the nearest-card
	// check: how tall the first/last card is (a grammar note can add a lot of
	// height) affects exactly how much scroll room it needs, and there isn't
	// always enough for the classification to land on 'top'/'bottom' otherwise.
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
