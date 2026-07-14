import type { PageLoad } from './$types';
import { listVocabProgress, type VocabProgressItem } from '$lib/api';

export const load: PageLoad = async ({ fetch }) => {
	try {
		const items = await listVocabProgress(undefined, fetch);
		return { items, error: null as string | null };
	} catch (e) {
		return { items: [] as VocabProgressItem[], error: e instanceof Error ? e.message : String(e) };
	}
};
