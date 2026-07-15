import type { PageLoad } from './$types';
import { getVocabDrillQueue, type VocabCard, type VocabSessionSummary } from '$lib/api';

const emptySummary: VocabSessionSummary = {
	new: 0,
	in_progress: 0,
	old: 0,
	introduced_today: 0,
	new_cap: 0
};

export const load: PageLoad = async ({ url, fetch }) => {
	const songIdParam = url.searchParams.get('song_id');
	const songId = songIdParam ? Number(songIdParam) : undefined;
	if (songId === undefined) {
		return {
			queue: [] as VocabCard[],
			summary: emptySummary,
			songId,
			error: 'No song specified — vocab drill is always scoped to a song.'
		};
	}
	try {
		const { cards, summary } = await getVocabDrillQueue(songId, undefined, fetch);
		return { queue: cards, summary, songId, error: null as string | null };
	} catch (e) {
		return {
			queue: [] as VocabCard[],
			summary: emptySummary,
			songId,
			error: e instanceof Error ? e.message : String(e)
		};
	}
};
