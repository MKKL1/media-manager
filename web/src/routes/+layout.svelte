<script lang="ts">
	import './layout.css';
	import favicon from '$lib/assets/favicon.svg';
	import { Button } from '$lib/components/ui/button';
	import * as Avatar from '$lib/components/ui/avatar';
	import { Input } from '$lib/components/ui/input';
	import { Separator } from '$lib/components/ui/separator';
	import {
		Settings,
		User,
		MonitorPlay,
		Film,
		Tv,
		Menu,
		LayoutDashboard,
		Search,
	} from '@lucide/svelte';
	import * as DropdownMenu from "$lib/components/ui/dropdown-menu/index.js";
	import { page } from '$app/state';
	import { goto } from '$app/navigation';

	let { children } = $props();

	let sidebarOpen = $state(!page.url.pathname.startsWith('/search'));
	let searchQuery = $state('');
	let searchFocused = $state(false);
	let searchProvider = $state('all');
	let recentSearches = $state<string[]>([]);
	let activeIndex = $state(-1);

	const providerLabels: Record<string, string> = { all: 'All', tmdb: 'TMDB' };

	$effect(() => {
		if (page.url.pathname.startsWith('/search')) {
			sidebarOpen = false;
		}
	});

	$effect(() => {
		const stored = localStorage.getItem('recentSearches');
		if (stored) {
			try { recentSearches = JSON.parse(stored); } catch {}
		}
	});

	function toggleSidebar() {
		sidebarOpen = !sidebarOpen;
	}

	const navItems = [
		{ href: '/dashboard', label: 'Dashboard', icon: LayoutDashboard },
		{ href: '/movies', label: 'Movies', icon: Film },
		{ href: '/tv', label: 'TV Shows', icon: Tv },
	];

	function isActive(href: string): boolean {
		return page.url.pathname.startsWith(href);
	}

	function handleSearchSubmit(e?: Event) {
		e?.preventDefault();
		if (searchQuery.trim()) {
			const query = searchQuery.trim();
			recentSearches = [query, ...recentSearches.filter(q => q !== query)].slice(0, 5);
			localStorage.setItem('recentSearches', JSON.stringify(recentSearches));

			searchFocused = false;
			(document.activeElement as HTMLElement)?.blur();
			activeIndex = -1;

			const providerParams = searchProvider !== 'all' ? `&provider=${searchProvider}` : '';
			goto(`/search?q=${encodeURIComponent(query)}${providerParams}`);
		}
	}

	function handleKeydown(e: KeyboardEvent) {
		const isTyping = searchQuery.trim().length > 0;
		const itemsCount = isTyping ? 3 : recentSearches.length;
		
		if (itemsCount === 0) return;

		if (e.key === 'ArrowDown') {
			e.preventDefault();
			activeIndex = (activeIndex + 1) % itemsCount;
		} else if (e.key === 'ArrowUp') {
			e.preventDefault();
			activeIndex = (activeIndex - 1 + itemsCount) % itemsCount;
		} else if (e.key === 'Enter') {
			if (!isTyping && activeIndex >= 0 && activeIndex < recentSearches.length) {
				e.preventDefault();
				searchQuery = recentSearches[activeIndex];
				handleSearchSubmit();
			} else if (isTyping && activeIndex > 0) {
				e.preventDefault();
			}
		}
	}
</script>

<svelte:head><link rel="icon" href={favicon} /></svelte:head>

<div class="flex h-screen w-full flex-col bg-background text-foreground">
	<!-- Top Bar -->
	<header class="flex h-16 shrink-0 items-center gap-2 bg-card px-2 shadow-md z-10">
		<Button variant="ghost" size="icon" onclick={toggleSidebar} class="shrink-0 rounded-full h-12 w-12">
			<Menu class="h-6 w-6" />
		</Button>

		<a href="/" class="flex items-center gap-2 shrink-0 px-2 text-foreground no-underline">
			<MonitorPlay class="h-6 w-6 text-primary" />
			<span class="text-lg font-medium tracking-tight">Media Manager</span>
		</a>

		<!-- Search bar -->
		<div class="flex-1 flex justify-center px-4 relative">
			<form
				class="relative w-full max-w-2xl z-50 flex items-center gap-1.5"
				onsubmit={handleSearchSubmit}
			>
				<div class="relative flex-1">
					<Search class="absolute left-3.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground pointer-events-none" />
					<Input
						bind:value={searchQuery}
						onfocus={() => searchFocused = true}
						onblur={() => setTimeout(() => searchFocused = false, 200)}
						onkeydown={handleKeydown}
						type="text"
						placeholder="Search media..."
						class="h-10 w-full pl-10 pr-3"
					/>
				</div>

				<DropdownMenu.Root>
					<DropdownMenu.Trigger>
						{#snippet child({ props })}
							<Button {...props} variant="outline" size="sm" class="h-10 shrink-0 gap-1.5 font-medium">
								{providerLabels[searchProvider]}
								<svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="opacity-50"><path d="m6 9 6 6 6-6"/></svg>
							</Button>
						{/snippet}
					</DropdownMenu.Trigger>
					<DropdownMenu.Content align="end">
						<DropdownMenu.Item onclick={() => searchProvider = 'all'}>All Providers</DropdownMenu.Item>
						<DropdownMenu.Item onclick={() => searchProvider = 'tmdb'}>TMDB</DropdownMenu.Item>
					</DropdownMenu.Content>
				</DropdownMenu.Root>

				{#if searchFocused && (searchQuery.trim() || recentSearches.length > 0)}
					<div class="absolute top-12 left-0 w-full bg-popover border rounded-lg shadow-lg overflow-hidden animate-in fade-in slide-in-from-top-1 duration-150 z-50">
						<div class="p-1.5">
							{#if searchQuery.trim()}
								<button type="submit" class="w-full text-left px-3 py-2 rounded-md text-sm flex items-center gap-3 transition-colors {activeIndex === 0 || activeIndex === -1 ? 'bg-accent text-accent-foreground' : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground'}">
									<Search class="h-4 w-4 shrink-0" />
									Search {providerLabels[searchProvider]} for "<span class="font-medium text-foreground">{searchQuery}</span>"
								</button>
								<button type="button" class="w-full text-left px-3 py-2 rounded-md text-sm flex items-center gap-3 cursor-not-allowed opacity-50 {activeIndex === 1 ? 'bg-accent' : ''}">
									<MonitorPlay class="h-4 w-4 shrink-0" />
									Local Library
									<span class="ml-auto text-[10px] border rounded px-1.5 py-0.5 font-medium">Soon</span>
								</button>
								<button type="button" class="w-full text-left px-3 py-2 rounded-md text-sm flex items-center gap-3 cursor-not-allowed opacity-50 {activeIndex === 2 ? 'bg-accent' : ''}">
									<Settings class="h-4 w-4 shrink-0" />
									Settings
									<span class="ml-auto text-[10px] border rounded px-1.5 py-0.5 font-medium">Soon</span>
								</button>
							{:else}
								<div class="px-3 py-1.5 text-xs font-medium text-muted-foreground">Recent</div>
								{#each recentSearches as recent, i}
									<button type="button" onmousedown={() => { searchQuery = recent; handleSearchSubmit(); }} class="w-full text-left px-3 py-2 rounded-md text-sm flex items-center gap-3 transition-colors {activeIndex === i ? 'bg-accent text-accent-foreground' : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground'}">
										<Search class="h-3.5 w-3.5 shrink-0" />
										{recent}
									</button>
								{/each}
							{/if}
						</div>
					</div>
				{/if}
			</form>
		</div>

		<!-- User avatar -->
		<Button variant="ghost" size="icon" class="shrink-0 rounded-full h-12 w-12">
			<Avatar.Root class="h-8 w-8">
				<Avatar.Image src="https://github.com/shadcn.png" alt="@user" />
				<Avatar.Fallback><User class="h-4 w-4" /></Avatar.Fallback>
			</Avatar.Root>
		</Button>
	</header>

	<div class="flex flex-1 overflow-hidden">
		<!-- Sidebar -->
		<aside
			class="flex shrink-0 flex-col bg-card overflow-x-hidden overflow-y-auto transition-[width] duration-200 ease-in-out"
			style="width: {sidebarOpen ? '256px' : '64px'}"
		>
			<nav class="flex-1 p-2 space-y-1 mt-2">
				{#each navItems as { href, label, icon: Icon }}
					<a
						{href}
						class="flex items-center gap-4 overflow-hidden rounded-full h-12 text-sm font-medium no-underline transition-all duration-200 w-full
						{isActive(href) ? 'bg-accent text-accent-foreground' : 'text-muted-foreground hover:bg-muted hover:text-foreground'}
						{sidebarOpen ? 'px-6' : 'px-[14px]'}"
						title={!sidebarOpen ? label : undefined}
					>
						<Icon class="h-5 w-5 shrink-0" />
						<span class="whitespace-nowrap transition-opacity duration-200 {sidebarOpen ? 'opacity-100' : 'opacity-0'}">
							{label}
						</span>
					</a>
				{/each}
			</nav>

			<div class="p-2 mb-2">
				<Separator class="mb-2 transition-opacity duration-200" />
				<a
					href="/settings"
					class="flex items-center gap-4 overflow-hidden rounded-full h-12 text-sm font-medium text-muted-foreground no-underline transition-all duration-200 hover:bg-muted hover:text-foreground w-full
					{sidebarOpen ? 'px-6' : 'px-[14px]'}"
					title={!sidebarOpen ? 'Settings' : undefined}
				>
					<Settings class="h-5 w-5 shrink-0" />
					<span class="whitespace-nowrap transition-opacity duration-200 {sidebarOpen ? 'opacity-100' : 'opacity-0'}">
						Settings
					</span>
				</a>
			</div>
		</aside>

		<!-- Main Content -->
		<main class="flex-1 overflow-y-auto">
			{@render children()}
		</main>
	</div>
</div>
