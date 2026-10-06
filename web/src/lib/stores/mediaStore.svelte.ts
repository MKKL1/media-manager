import { fetchMediaList, MEDIA_PAGE_SIZE, type MediaItem, type MediaType } from '$lib/api/media';

// Simple in-memory cache keyed by request fingerprint
const cache = new Map<string, { items: MediaItem[]; total: number }>();

function cacheKey(type: MediaType, offset: number, limit: number): string {
	return `${type}:${offset}:${limit}`;
}

export type BrowsingMode = 'infinite' | 'pagination';

export function createMediaStore(type: MediaType, limit = MEDIA_PAGE_SIZE) {
	let items = $state<MediaItem[]>([]);
	let total = $state(0);
	let offset = $state(0);
	let loading = $state(false);
	let error = $state<string | null>(null);

	async function loadPage(pageOffset: number) {
		const key = cacheKey(type, pageOffset, limit);
		const cached = cache.get(key);
		if (cached) {
			items = cached.items;
			total = cached.total;
			offset = pageOffset;
			error = null;
			return;
		}

		loading = true;
		error = null;
		try {
			const data = await fetchMediaList({ type, limit, offset: pageOffset });
			cache.set(key, { items: data.items, total: data.total });
			items = data.items;
			total = data.total;
			offset = pageOffset;
		} catch (err) {
			error = err instanceof Error ? err.message : 'Unknown error';
		} finally {
			loading = false;
		}
	}

	async function loadMore() {
		if (loading || items.length >= total) return;
		const nextOffset = offset + limit;
		const key = cacheKey(type, nextOffset, limit);
		const cached = cache.get(key);

		loading = true;
		error = null;
		try {
			let next: MediaItem[];
			let newTotal: number;
			if (cached) {
				next = cached.items;
				newTotal = cached.total;
			} else {
				const data = await fetchMediaList({ type, limit, offset: nextOffset });
				cache.set(key, { items: data.items, total: data.total });
				next = data.items;
				newTotal = data.total;
			}
			items = [...items, ...next];
			total = newTotal;
			offset = nextOffset;
		} catch (err) {
			error = err instanceof Error ? err.message : 'Unknown error';
		} finally {
			loading = false;
		}
	}

	function retry() {
		loadPage(offset);
	}

	// Page number helpers (1-based, for pagination UI)
	const currentPage = $derived(Math.floor(offset / limit) + 1);
	const totalPages = $derived(Math.ceil(total / limit));
	const hasMore = $derived(items.length < total);

	return {
		get items() { return items; },
		get total() { return total; },
		get loading() { return loading; },
		get error() { return error; },
		get currentPage() { return currentPage; },
		get totalPages() { return totalPages; },
		get hasMore() { return hasMore; },
		loadPage,
		loadMore,
		retry
	};
}
