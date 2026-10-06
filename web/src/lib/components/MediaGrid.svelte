<script lang="ts">
	import { onMount } from 'svelte';
	import { createMediaStore, type BrowsingMode } from '$lib/stores/mediaStore.svelte';
	import MediaCard from '$lib/components/MediaCard.svelte';
	import MediaCardSkeleton from '$lib/components/MediaCardSkeleton.svelte';
	import { AlertCircle, ChevronLeft, ChevronRight, RefreshCw, Loader2 } from '@lucide/svelte';
	import type { MediaType } from '$lib/api/media';

	let { type, mode }: { type: MediaType; mode: BrowsingMode } = $props();

	// $derived ensures store re-creates if `type` prop changes
	const store = $derived(createMediaStore(type));

	// Only show initial skeletons after a grace period to avoid flash on fast loads
	let showSkeleton = $state(false);
	let skeletonTimer: ReturnType<typeof setTimeout> | null = null;

	$effect(() => {
		if (store.loading && store.items.length === 0) {
			// Start a timer — only show skeletons if loading takes >200ms
			if (!skeletonTimer) {
				skeletonTimer = setTimeout(() => { showSkeleton = true; }, 200);
			}
		} else {
			// Data arrived or not loading anymore — cancel & hide
			if (skeletonTimer) { clearTimeout(skeletonTimer); skeletonTimer = null; }
			showSkeleton = false;
		}
	});

	// Sentinel element for infinite scroll
	let sentinel = $state<HTMLDivElement | null>(null);
	let observer: IntersectionObserver | null = null;

	function setupObserver() {
		observer?.disconnect();
		if (mode !== 'infinite' || !sentinel) return;
		observer = new IntersectionObserver(
			(entries) => {
				if (entries[0].isIntersecting) {
					store.loadMore();
				}
			},
			{ rootMargin: '200px' }
		);
		observer.observe(sentinel);
	}

	onMount(() => {
		store.loadPage(0);
		return () => {
			observer?.disconnect();
			if (skeletonTimer) clearTimeout(skeletonTimer);
		};
	});

	$effect(() => {
		// Re-bind observer when sentinel element or mode changes
		if (sentinel) setupObserver();
	});

	// Pagination helpers
	function goToPage(page: number) {
		const offset = (page - 1) * 20;
		store.loadPage(offset);
	}
</script>

<!-- Error Banner -->
{#if store.error}
	<div
		class="mb-6 flex items-start gap-3 rounded-lg border border-destructive/40 bg-destructive/10
		       px-4 py-3 text-sm text-destructive"
	>
		<AlertCircle class="mt-0.5 h-4 w-4 shrink-0" />
		<div class="flex-1 min-w-0">
			<p class="font-medium">Failed to load media</p>
			<p class="mt-0.5 text-destructive/80 break-all">{store.error}</p>
		</div>
		<button
			onclick={store.retry}
			class="flex items-center gap-1.5 shrink-0 rounded-md px-2 py-1
			       hover:bg-destructive/20 transition-colors font-medium"
		>
			<RefreshCw class="h-3.5 w-3.5" />
			Retry
		</button>
	</div>
{/if}

<!-- Grid -->
{#if store.items.length > 0}
	<div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-6">
		{#each store.items as item (item.id)}
			<MediaCard {item} />
		{/each}

		<!-- Skeleton cards while paginating -->
		{#if store.loading && mode === 'pagination'}
			{#each Array(6) as _, i}
				<MediaCardSkeleton />
			{/each}
		{/if}
	</div>
{:else if store.loading && showSkeleton}
	<!-- Initial skeleton grid — only shown after 200ms grace period -->
	<div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-6">
		{#each Array(12) as _, i}
			<MediaCardSkeleton />
		{/each}
	</div>
{:else if !store.loading && !store.error && store.items.length === 0}
	<div class="flex flex-col items-center justify-center py-24 text-muted-foreground gap-2">
		<p class="text-lg font-medium">No items found</p>
		<p class="text-sm">Your library is empty.</p>
	</div>
{/if}

<!-- Infinite Scroll sentinel + loader -->
{#if mode === 'infinite'}
	<div bind:this={sentinel} class="h-1 w-full" aria-hidden="true"></div>
	{#if store.loading && store.items.length > 0}
		<div class="flex justify-center py-8">
			<Loader2 class="h-6 w-6 animate-spin text-muted-foreground" />
		</div>
	{/if}
{/if}

<!-- Pagination controls -->
{#if mode === 'pagination' && store.totalPages > 1}
	<div class="mt-8 flex items-center justify-center gap-3">
		<button
			onclick={() => goToPage(store.currentPage - 1)}
			disabled={store.currentPage <= 1 || store.loading}
			class="flex items-center gap-1.5 rounded-lg border border-border px-3 py-2 text-sm font-medium
			       hover:bg-muted transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
		>
			<ChevronLeft class="h-4 w-4" />
			Previous
		</button>

		<span class="text-sm text-muted-foreground">
			Page <span class="font-semibold text-foreground">{store.currentPage}</span>
			of
			<span class="font-semibold text-foreground">{store.totalPages}</span>
		</span>

		<button
			onclick={() => goToPage(store.currentPage + 1)}
			disabled={store.currentPage >= store.totalPages || store.loading}
			class="flex items-center gap-1.5 rounded-lg border border-border px-3 py-2 text-sm font-medium
			       hover:bg-muted transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
		>
			Next
			<ChevronRight class="h-4 w-4" />
		</button>
	</div>
{/if}
