import type { PageLoad } from './$types';
import { getLineDrillQueue, type LineCard, type LineSessionSummary } from '$lib/api';

const emptySummary: LineSessionSummary = { new: 0, in_progress: 0, old: 0 };

export const load: PageLoad = async ({ url, fetch }) => {
	const songIdParam = url.searchParams.get('song_id');
	const songId = songIdParam ? Number(songIdParam) : undefined;
	if (songId === undefined) {
		return {
			queue: [] as LineCard[],
			summary: emptySummary,
			songId,
			error: 'No song specified — line drill is always scoped to a song.'
		};
	}
	try {
		const { cards, summary } = await getLineDrillQueue(songId, 20, fetch);
		return { queue: cards, summary, songId, error: null as string | null };
	} catch (e) {
		return {
			queue: [] as LineCard[],
			summary: emptySummary,
			songId,
			error: e instanceof Error ? e.message : String(e)
		};
	}
};
