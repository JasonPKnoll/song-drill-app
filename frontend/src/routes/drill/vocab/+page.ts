import type { PageLoad } from './$types';
import { getVocabDrillQueue, type VocabCard, type VocabSessionSummary } from '$lib/api';

const emptySummary: VocabSessionSummary = {
	new_today: 0,
	new_cap: 0,
	in_progress_today: 0,
	completed_today: 0
};

export const load: PageLoad = async ({ url, fetch }) => {
	const songIdParam = url.searchParams.get('song_id');
	const songId = songIdParam ? Number(songIdParam) : undefined;
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
