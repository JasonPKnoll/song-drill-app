import type { PageLoad } from './$types';
import { listSongs, type SongSummary } from '$lib/api';

export const load: PageLoad = async ({ fetch }) => {
	try {
		const songs = await listSongs(fetch);
		return { songs, error: null as string | null };
	} catch (e) {
		return { songs: [] as SongSummary[], error: e instanceof Error ? e.message : String(e) };
	}
};
