import { fetchSearch, type SearchOptions } from '$lib/api/media';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ url, fetch }) => {
	const query = url.searchParams.get('q') || '';
	const yearMinStr = url.searchParams.get('yearMin');
	const yearMaxStr = url.searchParams.get('yearMax');
	const providerStr = url.searchParams.get('provider') || 'tmdb';
	const typeStr = url.searchParams.get('type') || 'all';
	const seasonCountStr = url.searchParams.get('seasonCount');

	const options: SearchOptions = {
		query,
		yearMin: yearMinStr ? parseInt(yearMinStr, 10) : undefined,
		yearMax: yearMaxStr ? parseInt(yearMaxStr, 10) : undefined,
		provider: providerStr,
		mediaType: typeStr,
		seasonCount: seasonCountStr ? parseInt(seasonCountStr, 10) : undefined
	};
	
	if (!query && !yearMinStr && !yearMaxStr) {
		return { query, options, results: [] };
	}

	try {
		return { 
			query, 
			options, 
			streamed: { results: fetchSearch(options, fetch) } 
		};
	} catch (e) {
		console.error('Search initiation failed:', e);
		return { query, options, streamed: { results: Promise.resolve([]) }, error: 'Failed to fetch search results' };
	}
};
