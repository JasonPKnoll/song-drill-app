import type { PageLoad } from './$types';
import { listVocabProgress, type VocabProgressItem } from '$lib/api';

// Progress is scoped to one song, same as the drill queues — song_id is
// required in the URL (see routes/songs/[id]/+page.svelte's "Progress" link).
export const load: PageLoad = async ({ url, fetch }) => {
	const songIdParam = url.searchParams.get('song_id');
	const songId = songIdParam ? Number(songIdParam) : undefined;
	if (songId === undefined) {
		return { items: [] as VocabProgressItem[], songId, error: 'song_id is required' };
	}
	try {
		const items = await listVocabProgress(songId, fetch);
		return { items, songId, error: null as string | null };
	} catch (e) {
		return { items: [] as VocabProgressItem[], songId, error: e instanceof Error ? e.message : String(e) };
	}
};
