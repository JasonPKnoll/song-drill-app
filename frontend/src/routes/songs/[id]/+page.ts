import type { PageLoad } from './$types';
import { getSong, type SongDetail } from '$lib/api';

export const load: PageLoad = async ({ params, fetch }) => {
	try {
		const song = await getSong(Number(params.id), fetch);
		return { song, error: null as string | null };
	} catch (e) {
		return { song: null as SongDetail | null, error: e instanceof Error ? e.message : String(e) };
	}
};
