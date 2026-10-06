<script lang="ts">
	import type { MediaItem } from '$lib/api/media';
	import { Tv, Film, ImageOff } from '@lucide/svelte';

	let { item }: { item: MediaItem } = $props();

	const year = $derived(item.release_date ? new Date(item.release_date).getFullYear() : null);
	let imgError = $state(false);

	const FallbackIcon = $derived(item.type === 'tv' ? Tv : Film);
</script>

<div class="space-y-2 cursor-pointer group">
	<div
		class="aspect-[2/3] rounded-md bg-muted flex items-center justify-center border
		       group-hover:border-primary/50 transition-colors shadow-sm overflow-hidden relative"
	>
		{#if item.poster_path && !imgError}
			<img
				src={item.poster_path}
				alt={item.title}
				class="absolute inset-0 w-full h-full object-cover"
				onerror={() => (imgError = true)}
				loading="lazy"
			/>
		{:else}
			<div class="flex flex-col items-center gap-2 text-muted-foreground/30">
				{#if imgError}
					<ImageOff class="w-8 h-8" />
				{:else}
					<FallbackIcon class="w-8 h-8 group-hover:scale-110 transition-transform duration-300" />
				{/if}
			</div>
		{/if}

		<!-- Hover overlay -->
		<div
			class="absolute inset-0 bg-black/50 opacity-0 group-hover:opacity-100 transition-opacity
			       flex flex-col items-center justify-end p-3 gap-1"
		>
			<span class="text-white text-xs font-semibold uppercase tracking-wide">View</span>
		</div>

		<!-- Status badge -->
		{#if item.status}
			<span
				class="absolute top-2 left-2 text-[10px] font-semibold px-1.5 py-0.5 rounded
				       bg-black/60 text-white backdrop-blur-sm leading-tight"
			>
				{item.status}
			</span>
		{/if}
	</div>

	<div class="space-y-0.5 px-0.5">
		<h4
			class="text-sm font-medium leading-tight group-hover:text-primary transition-colors line-clamp-2"
			title={item.title}
		>
			{item.title}
		</h4>
		{#if year}
			<p class="text-xs text-muted-foreground">{year}</p>
		{/if}
	</div>
</div>
