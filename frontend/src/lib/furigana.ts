export type FuriSegment = { type: 'ruby'; base: string; reading: string } | { type: 'text'; value: string };

const KANJI_RE = /[一-鿿㐀-䶿]/;

// A 漢字[よみ] bracket's reading only applies to the contiguous kanji run
// immediately before the bracket — any kana earlier in the same chunk
// (e.g. a preceding particle) renders as plain text.
function splitKanjiTail(chunk: string): { prefix: string; kanji: string } {
	let i = chunk.length;
	while (i > 0 && KANJI_RE.test(chunk[i - 1])) i--;
	return { prefix: chunk.slice(0, i), kanji: chunk.slice(i) };
}

export function parseFurigana(furi: string): FuriSegment[] {
	const segments: FuriSegment[] = [];
	const re = /([^[\]]*)\[([^[\]]+)\]/g;
	let lastIndex = 0;
	let match: RegExpExecArray | null;

	while ((match = re.exec(furi)) !== null) {
		const [, preChunk, reading] = match;
		const { prefix, kanji } = splitKanjiTail(preChunk);
		if (prefix) segments.push({ type: 'text', value: prefix });
		if (kanji) segments.push({ type: 'ruby', base: kanji, reading });
		lastIndex = re.lastIndex;
	}

	const tail = furi.slice(lastIndex);
	if (tail) segments.push({ type: 'text', value: tail });

	return segments;
}
