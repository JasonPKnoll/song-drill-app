const API_BASE = 'http://localhost:30001/api/song-drill';

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
	context_meaning: string;
	example_line?: Line;
	streak: number;
	next_review: string;
}

export interface LineCard {
	line_id: number;
	song_id: number;
	song_title: string;
	text: string;
	furi: string;
	natural: string;
	grammar_note?: string;
	streak: number;
	next_review: string;
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

async function request<T>(path: string, init?: RequestInit): Promise<T> {
	const res = await fetch(`${API_BASE}${path}`, {
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

export function listSongs(): Promise<SongSummary[]> {
	return request('/songs');
}

export function getSong(id: number): Promise<SongDetail> {
	return request(`/songs/${id}`);
}

export function getSongLines(id: number): Promise<Line[]> {
	return request(`/songs/${id}/lines`);
}

export function ingestSong(payload: unknown): Promise<{ song_id: number }> {
	return request('/songs/ingest', { method: 'POST', body: JSON.stringify(payload) });
}

export function getVocabDrillQueue(songId?: number, limit = 20): Promise<VocabCard[]> {
	const params = new URLSearchParams();
	if (songId !== undefined) params.set('song_id', String(songId));
	params.set('limit', String(limit));
	return request(`/drill/vocab?${params}`);
}

export function getLineDrillQueue(songId?: number, limit = 20): Promise<LineCard[]> {
	const params = new URLSearchParams();
	if (songId !== undefined) params.set('song_id', String(songId));
	params.set('limit', String(limit));
	return request(`/drill/lines?${params}`);
}

export function recordVocabResult(
	songId: number,
	vocabId: number,
	correct: boolean
): Promise<{ ok: boolean }> {
	return request('/drill/result', {
		method: 'POST',
		body: JSON.stringify({ type: 'vocab', song_id: songId, vocab_id: vocabId, correct })
	});
}

export function recordLineResult(lineId: number, correct: boolean): Promise<{ ok: boolean }> {
	return request('/drill/result', {
		method: 'POST',
		body: JSON.stringify({ type: 'line', line_id: lineId, correct })
	});
}

export function getStats(): Promise<Stats> {
	return request('/stats');
}
