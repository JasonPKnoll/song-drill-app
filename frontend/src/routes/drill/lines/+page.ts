import type { PageLoad } from './$types';
import { getLineDrillQueue, type LineCard } from '$lib/api';

export const load: PageLoad = async ({ url, fetch }) => {
	const songIdParam = url.searchParams.get('song_id');
	const songId = songIdParam ? Number(songIdParam) : undefined;
	try {
		const queue = await getLineDrillQueue(songId, 20, fetch);
		return { queue, songId, error: null as string | null };
	} catch (e) {
		return { queue: [] as LineCard[], songId, error: e instanceof Error ? e.message : String(e) };
	}
};
