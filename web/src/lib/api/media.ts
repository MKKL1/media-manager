import { PUBLIC_API_BASE_URL } from '$env/static/public';

export const MEDIA_PAGE_SIZE = 20;

export type MediaType = 'tv' | 'movie';

export interface MediaIdentity {
	provider: string;
	kind?: string;
	id: string;
}

export interface MediaItem {
	id: string;
	type: MediaType;
	title: string;
	original_title: string;
	original_lang: string;
	monitored: boolean;
	status: string;
	summary: string;
	release_date: string;
	primary_identity: MediaIdentity;
	poster_path: string;
	metadata: unknown;
}

export interface MediaListParams {
	type: MediaType;
	sort?: string;
	limit?: number;
	offset?: number;
}

export interface MediaOverview {
	items: MediaItem[];
	total: number;
	offset: number;
	limit: number;
}

export class ApiError extends Error {
	constructor(
		public readonly status: number,
		message: string
	) {
		super(message);
		this.name = 'ApiError';
	}
}

export async function fetchMediaList(
	params: MediaListParams,
	fetchFn: typeof fetch = fetch
): Promise<MediaOverview> {
	const { type, sort = 'created_at', limit = MEDIA_PAGE_SIZE, offset = 0 } = params;

	const query = new URLSearchParams({
		type,
		sort,
		limit: String(limit),
		offset: String(offset)
	});
	const url = `${PUBLIC_API_BASE_URL}/api/1/media/list?${query}`;

	let response: Response;
	try {
		response = await fetchFn(url);
	} catch (err) {
		const msg = err instanceof Error ? err.message : 'Network error — could not reach server';
		console.error('[media-api]', msg, err);
		throw new ApiError(0, msg);
	}

	if (!response.ok) {
		let detail = '';
		try {
			const body = await response.text();
			detail = body ? `: ${body}` : '';
		} catch {
			// ignore parse errors
		}
		const msg = `API error ${response.status}${detail}`;
		console.error('[media-api]', msg);
		throw new ApiError(response.status, msg);
	}

	const json = await response.json();
	return json.data as MediaOverview;
}

export interface SearchResult {
	identity: MediaIdentity;
	media_type: MediaType;
	title: string;
	year: number;
	overview: string;
	poster: string;
	popularity: number;
}

export interface SearchOptions {
	query: string;
	yearMin?: number;
	yearMax?: number;
	provider?: string;
	mediaType?: string; // 'tv' | 'movie' | 'all'
	seasonCount?: number;
}

export async function fetchSearch(
	options: SearchOptions,
	fetchFn: typeof fetch = fetch
): Promise<SearchResult[]> {
	const params = new URLSearchParams();
	if (options.query) params.set('query', options.query);
	if (options.yearMin) params.set('yearMin', options.yearMin.toString());
	if (options.yearMax) params.set('yearMax', options.yearMax.toString());
	if (options.provider) params.set('provider', options.provider);
	if (options.mediaType && options.mediaType !== 'all') params.set('mediaType', options.mediaType);
	if (options.seasonCount) params.set('seasonCount', options.seasonCount.toString());

	const url = `${PUBLIC_API_BASE_URL}/api/1/media/search?${params.toString()}`;

	let response: Response;
	try {
		response = await fetchFn(url);
	} catch (err) {
		const msg = err instanceof Error ? err.message : 'Network error — could not reach server';
		console.error('[media-api]', msg, err);
		throw new ApiError(0, msg);
	}

	if (!response.ok) {
		let detail = '';
		try {
			const body = await response.text();
			detail = body ? `: ${body}` : '';
		} catch {
			// ignore parse errors
		}
		const msg = `API error ${response.status}${detail}`;
		console.error('[media-api]', msg);
		throw new ApiError(response.status, msg);
	}

	const json = await response.json();
	return json.data as SearchResult[];
}
