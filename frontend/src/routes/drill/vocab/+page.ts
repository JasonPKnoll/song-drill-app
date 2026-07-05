import type { PageLoad } from './$types';
import { getVocabDrillQueue, type VocabCard } from '$lib/api';

export const load: PageLoad = async ({ url, fetch }) => {
	const songIdParam = url.searchParams.get('song_id');
	const songId = songIdParam ? Number(songIdParam) : undefined;
	try {
		const queue = await getVocabDrillQueue(songId, 20, fetch);
		return { queue, songId, error: null as string | null };
	} catch (e) {
		return { queue: [] as VocabCard[], songId, error: e instanceof Error ? e.message : String(e) };
	}
};
