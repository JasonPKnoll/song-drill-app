// Relative so it works whether the page is loaded from localhost, an ngrok
// tunnel, or the deployed host — the dev server (or nginx in production)
// proxies this path to the Go backend rather than the browser hitting it directly.
const API_BASE = '/api/song-drill';

export interface Song {
	id: number;
	title: string;
	artist: string;
	language: string;
	notes?: string;
	created_at: string;
}

export interface SongSummary extends Song {
	vocab_count: number;
	mastered_count: number;
	line_count: number;
}

export interface Line {
	id: number;
	song_id: number;
	position: number;
	text: string;
	reading: string;
	furi: string;
	literal: string;
	natural: string;
	contextual: string;
	grammar_note?: string;
	section?: string;
}

export interface VocabItem {
	id: number;
	surface: string;
	reading: string;
	furi: string;
	pos: string;
	base_meaning: string;
	context_meaning: string;
	first_line_position: number;
	// Every line this word actually occurs in (from real tokenization at
	// ingest time) — the exact way to answer "is this word in line X,"
	// rather than guessing via substring matching on the line's raw text.
	line_ids: number[];
}

export interface SongDetail extends Song {
	lines: Line[];
	vocab: VocabItem[];
}

export interface VocabCard {
	song_id: number;
	song_title: string;
	vocab_id: number;
	surface: string;
	reading: string;
	furi: string;
	base_meaning: string;
	example_line?: Line;
	state: 'new' | 'learning' | 'review' | 'relearning';
	due: string;
}

// Today's progress against the daily new-word cap (see schema.md's "Daily
// new-word cap" section) — server-computed so the UI's new/in-progress/done
// counts reflect real per-day DB state, not local session bookkeeping that
// would reset on every page load.
export interface VocabSessionSummary {
	new_today: number;
	new_cap: number;
	in_progress_today: number;
	completed_today: number;
}

export interface LineCard {
	line_id: number;
	song_id: number;
	song_title: string;
	text: string;
	furi: string;
	natural: string;
	grammar_note?: string;
	state: 'new' | 'learning' | 'review' | 'relearning';
	due: string;
}

export interface Stats {
	total_songs: number;
	total_vocab: number;
	mastered_vocab: number;
	total_lines: number;
	mastered_lines: number;
	vocab_due_today: number;
	lines_due_today: number;
}

// `fetchFn` defaults to the global fetch but should be given the `fetch`
// passed into a SvelteKit `load` function when called from one — that's the
// fetch SvelteKit tracks for the request lifecycle of the current navigation.
async function request<T>(path: string, init?: RequestInit, fetchFn: typeof fetch = fetch): Promise<T> {
	const res = await fetchFn(`${API_BASE}${path}`, {
		headers: { 'Content-Type': 'application/json' },
		...init
	});
	if (!res.ok) {
		const body = await res.json().catch(() => ({}));
		throw new Error(body.error ?? `${res.status} ${res.statusText}`);
	}
	if (res.status === 204) return undefined as T;
	return res.json();
}

export function listSongs(fetchFn?: typeof fetch): Promise<SongSummary[]> {
	return request('/songs', undefined, fetchFn);
}

export function getSong(id: number, fetchFn?: typeof fetch): Promise<SongDetail> {
	return request(`/songs/${id}`, undefined, fetchFn);
}

export function getSongLines(id: number): Promise<Line[]> {
	return request(`/songs/${id}/lines`);
}

export function ingestSong(payload: unknown): Promise<{ song_id: number }> {
	return request('/songs/ingest', { method: 'POST', body: JSON.stringify(payload) });
}

// A small "handful" rather than the whole due backlog — the drill session
// polls for fresh cards on a timer instead of loading everything up front,
// so this only needs to cover what's visible at once.
export const VOCAB_SESSION_LIMIT = 8;

export function getVocabDrillQueue(
	songId?: number,
	limit = VOCAB_SESSION_LIMIT,
	fetchFn?: typeof fetch
): Promise<{ cards: VocabCard[]; summary: VocabSessionSummary }> {
	const params = new URLSearchParams();
	if (songId !== undefined) params.set('song_id', String(songId));
	params.set('limit', String(limit));
	return request(`/drill/vocab?${params}`, undefined, fetchFn);
}

// Introduces more brand-new words right now, bypassing the daily cap — "add
// more if wanted".
export function addMoreVocab(count?: number, songId?: number): Promise<VocabSessionSummary> {
	return request('/drill/vocab/more', {
		method: 'POST',
		body: JSON.stringify({ count, song_id: songId })
	});
}

export function getLineDrillQueue(
	songId?: number,
	limit = 20,
	fetchFn?: typeof fetch
): Promise<LineCard[]> {
	const params = new URLSearchParams();
	if (songId !== undefined) params.set('song_id', String(songId));
	params.set('limit', String(limit));
	return request(`/drill/lines?${params}`, undefined, fetchFn);
}

export interface DrillResult {
	ok: boolean;
	// Whether this card still needs same-day repetition (learning/relearning)
	// or is done for now (review) — see the drill pages, which re-queue a
	// card in the current session while it's still learning/relearning.
	state: 'new' | 'learning' | 'review' | 'relearning';
}

export function recordVocabResult(songId: number, vocabId: number, correct: boolean): Promise<DrillResult> {
	return request('/drill/result', {
		method: 'POST',
		body: JSON.stringify({ type: 'vocab', song_id: songId, vocab_id: vocabId, correct })
	});
}

export function recordLineResult(lineId: number, correct: boolean): Promise<DrillResult> {
	return request('/drill/result', {
		method: 'POST',
		body: JSON.stringify({ type: 'line', line_id: lineId, correct })
	});
}

export function getStats(): Promise<Stats> {
	return request('/stats');
}

// Profile — no password, just a name/color someone picks. See the Profiles
// section of song_drill_schema.md: the app is Tailscale-only, so this just
// partitions progress/stats between people sharing the install.
export interface Profile {
	id: number;
	display_name: string;
	color: string;
	created_at: string;
}

export function listProfiles(fetchFn?: typeof fetch): Promise<Profile[]> {
	return request('/profiles', undefined, fetchFn);
}

export function getActiveProfile(fetchFn?: typeof fetch): Promise<Profile> {
	return request('/profiles/active', undefined, fetchFn);
}

export function setActiveProfile(id: number): Promise<Profile> {
	return request('/profiles/active', { method: 'POST', body: JSON.stringify({ id }) });
}

export function createProfile(displayName: string, color: string): Promise<Profile> {
	return request('/profiles', {
		method: 'POST',
		body: JSON.stringify({ display_name: displayName, color })
	});
}

export function renameProfile(id: number, displayName: string, color: string): Promise<Profile> {
	return request(`/profiles/${id}`, {
		method: 'PATCH',
		body: JSON.stringify({ display_name: displayName, color })
	});
}

export function deleteProfile(id: number): Promise<void> {
	return request(`/profiles/${id}`, { method: 'DELETE' });
}

// One word's progress for the active profile — the stats sheet's row shape.
// Every word in the library appears here, not just ones actually drilled;
// an untouched word defaults to state "new" with due/last_seen absent.
export interface VocabProgressItem {
	song_id: number;
	song_title: string;
	vocab_id: number;
	surface: string;
	reading: string;
	furi: string;
	base_meaning: string;
	state: 'new' | 'learning' | 'review' | 'relearning';
	interval_days: number;
	lapses: number;
	seen: number;
	correct: number;
	due?: string;
	last_seen?: string;
	mastered: boolean;
}

export function listVocabProgress(songId?: number, fetchFn?: typeof fetch): Promise<VocabProgressItem[]> {
	const params = new URLSearchParams();
	if (songId !== undefined) params.set('song_id', String(songId));
	const qs = params.toString();
	return request(`/progress/vocab${qs ? `?${qs}` : ''}`, undefined, fetchFn);
}

// Manually flags a word as already known, regardless of its real drill
// history — an SRS override, not a substitute for actually drilling it.
export function burnVocabProgress(songId: number, vocabId: number): Promise<{ ok: boolean }> {
	return request('/progress/vocab/burn', {
		method: 'POST',
		body: JSON.stringify({ song_id: songId, vocab_id: vocabId })
	});
}

// Wipes a word's progress for the active profile back to "new".
export function resetVocabProgress(songId: number, vocabId: number): Promise<{ ok: boolean }> {
	return request('/progress/vocab/reset', {
		method: 'POST',
		body: JSON.stringify({ song_id: songId, vocab_id: vocabId })
	});
}
