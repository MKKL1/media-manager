<script lang="ts">
	import type { PageData } from './$types';
	import MediaCard from '$lib/components/MediaCard.svelte';
	import MediaCardSkeleton from '$lib/components/MediaCardSkeleton.svelte';
	import type { MediaItem } from '$lib/api/media';
	import { Input } from '$lib/components/ui/input';
	import { Separator } from '$lib/components/ui/separator';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';

    import * as Field from "$lib/components/ui/field/index.js";
    import * as RadioGroup from "$lib/components/ui/radio-group/index.js";
    import { Slider } from "$lib/components/ui/slider/index.js";
    import * as Breadcrumb from '$lib/components/ui/breadcrumb/index.js';

	let { data }: { data: PageData } = $props();

	let mediaType = $state(page.url.searchParams.get('type') || 'all');
	let seasonCount = $state(page.url.searchParams.get('seasonCount') || '');
    let yearMinParam = page.url.searchParams.get('yearMin');
    let yearMaxParam = page.url.searchParams.get('yearMax');
    let provider = $state(page.url.searchParams.get('provider') || 'tmdb'); 

    let yearRange = $state([
        yearMinParam ? parseInt(yearMinParam) : 1900, 
        yearMaxParam ? parseInt(yearMaxParam) : 2026
    ]);

	// Keep state synced if URL changes via back/forward
	$effect(() => {
		const typeStr = page.url.searchParams.get('type') || 'all';
		const yearMinStr = page.url.searchParams.get('yearMin');
		const yearMaxStr = page.url.searchParams.get('yearMax');
		const seasonCountStr = page.url.searchParams.get('seasonCount') || '';
        const providerStr = page.url.searchParams.get('provider') || 'tmdb';
		
		if (mediaType !== typeStr) mediaType = typeStr;
		if (seasonCount !== seasonCountStr) seasonCount = seasonCountStr;
        if (provider !== providerStr) provider = providerStr;

        let newMin = yearMinStr ? parseInt(yearMinStr) : 1900;
        let newMax = yearMaxStr ? parseInt(yearMaxStr) : 2026;
        if (yearRange[0] !== newMin || yearRange[1] !== newMax) {
            yearRange = [newMin, newMax];
        }
	});

    let debounceTimer: ReturnType<typeof setTimeout>;
    function handleSliderChange() {
        clearTimeout(debounceTimer);
        debounceTimer = setTimeout(() => {
            applyFilters();
        }, 400);
    }

	function applyFilters() {
		const params = new URLSearchParams();
		if (data.query) params.set('q', data.query);
		if (mediaType && mediaType !== 'all') params.set('type', mediaType);
        
        if (yearRange[0] > 1900) params.set('yearMin', yearRange[0].toString());
        if (yearRange[1] < 2026) params.set('yearMax', yearRange[1].toString());

		if (provider && provider !== 'all') params.set('provider', provider);
		if (mediaType === 'tv' && seasonCount) params.set('seasonCount', seasonCount);

		goto(`/search?${params.toString()}`, { keepFocus: true, noScroll: true });
	}
</script>

<div class="flex flex-col md:flex-row gap-6 p-6 max-w-7xl mx-auto items-start animate-in fade-in duration-500">
	<!-- Sidebar Filters -->
	<aside class="w-full md:w-64 shrink-0 bg-card border rounded-xl p-5 shadow-sm overflow-hidden">
        <Field.Set>
            <Field.Legend class="text-xl font-bold tracking-tight mb-4 hidden-visually sr-only">Filters</Field.Legend>
            <h3 class="text-lg font-semibold tracking-tight mb-4">Filters</h3>
            
            <Field.Group class="space-y-6">
                
                <Field.Field>
                    <Field.Label class="text-base font-medium">Provider</Field.Label>
                    <RadioGroup.Root bind:value={provider} onValueChange={applyFilters} class="mt-3 space-y-2">
                        <Field.Field orientation="horizontal">
                            <RadioGroup.Item value="all" id="provider-all" />
                            <Field.Label for="provider-all" class="font-normal cursor-pointer">All</Field.Label>
                        </Field.Field>
                        <Field.Field orientation="horizontal">
                            <RadioGroup.Item value="tmdb" id="provider-tmdb" />
                            <Field.Label for="provider-tmdb" class="font-normal cursor-pointer">TMDB</Field.Label>
                        </Field.Field>
                    </RadioGroup.Root>
                </Field.Field>

                <Separator />

                <Field.Field>
                    <Field.Label class="text-base font-medium">Media Type</Field.Label>
                    <RadioGroup.Root bind:value={mediaType} onValueChange={applyFilters} class="mt-3 space-y-2">
                        <Field.Field orientation="horizontal">
                            <RadioGroup.Item value="all" id="type-all" />
                            <Field.Label for="type-all" class="font-normal cursor-pointer">All</Field.Label>
                        </Field.Field>
                        <Field.Field orientation="horizontal">
                            <RadioGroup.Item value="movie" id="type-movie" />
                            <Field.Label for="type-movie" class="font-normal cursor-pointer">Movies</Field.Label>
                        </Field.Field>
                        <Field.Field orientation="horizontal">
                            <RadioGroup.Item value="tv" id="type-tv" />
                            <Field.Label for="type-tv" class="font-normal cursor-pointer">TV Shows</Field.Label>
                        </Field.Field>
                    </RadioGroup.Root>
                </Field.Field>

                <Separator />

                <Field.Field>
                    <Field.Label class="text-base font-medium flex justify-between items-center w-full">
                        Release Year
                        <span class="text-sm font-normal text-muted-foreground">{yearRange[0]} - {yearRange[1]}</span>
                    </Field.Label>
                    <div class="pt-4 pb-2 px-1">
                        <Slider
                            type="multiple"
                            bind:value={yearRange}
                            max={2026}
                            min={1900}
                            step={1}
                            class="w-full"
                            onValueChange={handleSliderChange}
                        />
                    </div>
                </Field.Field>

                {#if mediaType === 'tv'}
                    <div class="animate-in fade-in slide-in-from-top-2 duration-300">
                        <Separator class="my-4" />
                        <Field.Field>
                            <Field.Label for="season-count" class="text-base font-medium">Min Seasons</Field.Label>
                            <Input id="season-count" type="number" placeholder="e.g. 3" bind:value={seasonCount} class="mt-2" onblur={applyFilters} onkeydown={(e: Event) => (e as KeyboardEvent).key === 'Enter' && applyFilters()} />
                        </Field.Field>
                    </div>
                {/if}

            </Field.Group>
        </Field.Set>
	</aside>

	<!-- Main Results Content -->
	<main class="flex-1 min-w-0 space-y-6 w-full">
		<div>
			<Breadcrumb.Root>
				<Breadcrumb.List>
					<Breadcrumb.Item>
						<Breadcrumb.Link href="/">Home</Breadcrumb.Link>
					</Breadcrumb.Item>
					<Breadcrumb.Separator />
					<Breadcrumb.Item>
						{#if data.query}
							<Breadcrumb.Link href="/search">Search</Breadcrumb.Link>
						{:else}
							<Breadcrumb.Page>Search</Breadcrumb.Page>
						{/if}
					</Breadcrumb.Item>
					{#if data.query}
						<Breadcrumb.Separator />
						<Breadcrumb.Item>
							<Breadcrumb.Page>"{data.query}"</Breadcrumb.Page>
						</Breadcrumb.Item>
					{/if}
				</Breadcrumb.List>
			</Breadcrumb.Root>

			<h1 class="text-3xl font-bold tracking-tight mt-4">Search Results</h1>
			{#if data.query || data.options?.yearMin || data.options?.yearMax}
				<p class="text-muted-foreground mt-2">Showing results for 
					{#if data.query}"<span class="text-foreground font-medium">{data.query}</span>"{/if}
					{#if data.options?.yearMin && data.options?.yearMax}
						from <span class="text-foreground font-medium">{data.options.yearMin} - {data.options.yearMax}</span>
					{/if}
				</p>
			{:else}
				<p class="text-muted-foreground mt-2">Enter a search query or apply filters.</p>
			{/if}
		</div>

		{#if data.error}
			<div class="p-4 bg-destructive/10 text-destructive rounded-md">
				{data.error}
			</div>
		{:else}
			{#await data.streamed?.results}
				<div class="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-4">
					{#each Array(15) as _}
						<MediaCardSkeleton />
					{/each}
				</div>
			{:then results}
				{@const resultsAsMediaItems = results?.map((r) => {
					return {
						id: r.identity.id,
						type: r.media_type,
						title: r.title,
						original_title: r.title,
						original_lang: '',
						monitored: false,
						status: '',
						summary: r.overview || '',
						release_date: r.year ? `${r.year}-01-01` : '',
						primary_identity: r.identity,
						poster_path: r.poster || '',
						metadata: null
					} as MediaItem;
				}) || []}

				{#if (data.query || data.options?.yearMin) && resultsAsMediaItems.length === 0}
					<div class="py-12 text-center text-muted-foreground border rounded-lg border-dashed bg-card/50">
						No results found matching your criteria.
					</div>
				{:else if resultsAsMediaItems.length > 0}
					<div class="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-4">
						{#each resultsAsMediaItems as item}
							<MediaCard {item} />
						{/each}
					</div>
				{/if}
			{:catch error}
				<div class="p-4 bg-destructive/10 text-destructive rounded-md">
					Failed to load results.
				</div>
			{/await}
		{/if}
	</main>
</div>
