<script lang="ts">
	import MediaGrid from '$lib/components/MediaGrid.svelte';
	import type { BrowsingMode } from '$lib/stores/mediaStore.svelte';
	import { browser } from '$app/environment';
	import * as Breadcrumb from '$lib/components/ui/breadcrumb/index.js';

	const mode: BrowsingMode =
		browser && localStorage.getItem('mm:browsing-mode') === 'pagination'
			? 'pagination'
			: 'infinite';
</script>

<div class="p-8 max-w-6xl mx-auto space-y-8 animate-in fade-in duration-500">
	<div>
		<Breadcrumb.Root>
			<Breadcrumb.List>
				<Breadcrumb.Item>
					<Breadcrumb.Link href="/">Home</Breadcrumb.Link>
				</Breadcrumb.Item>
				<Breadcrumb.Separator />
				<Breadcrumb.Item>
					<Breadcrumb.Link href="/tv">Collections</Breadcrumb.Link>
				</Breadcrumb.Item>
				<Breadcrumb.Separator />
				<Breadcrumb.Item>
					<Breadcrumb.Page>TV Shows</Breadcrumb.Page>
				</Breadcrumb.Item>
			</Breadcrumb.List>
		</Breadcrumb.Root>

		<h1 class="text-3xl font-bold tracking-tight mt-4">TV Shows</h1>
		<p class="text-muted-foreground mt-2">Your TV show library collection.</p>
	</div>

	<MediaGrid type="tv" {mode} />
</div>
