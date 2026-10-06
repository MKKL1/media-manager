<script lang="ts">
	import { Film, Tv, HardDrive, Activity } from '@lucide/svelte';
	import * as Card from '$lib/components/ui/card';
	import * as Breadcrumb from '$lib/components/ui/breadcrumb/index.js';

	const stats = [
		{ label: 'Movies', value: '0', icon: Film },
		{ label: 'TV Shows', value: '0', icon: Tv },
		{ label: 'Storage Used', value: '—', icon: HardDrive },
		{ label: 'Recent Activity', value: '—', icon: Activity },
	];
</script>

<div class="p-8 max-w-6xl">
	<Breadcrumb.Root>
		<Breadcrumb.List>
			<Breadcrumb.Item>
				<Breadcrumb.Link href="/">Home</Breadcrumb.Link>
			</Breadcrumb.Item>
			<Breadcrumb.Separator />
			<Breadcrumb.Item>
				<Breadcrumb.Page>Dashboard</Breadcrumb.Page>
			</Breadcrumb.Item>
		</Breadcrumb.List>
	</Breadcrumb.Root>

	<h1 class="text-2xl font-normal text-foreground mt-4">Dashboard</h1>
	<p class="text-sm text-muted-foreground mt-2">Overview of your media library.</p>

	<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 mt-8">
		{#each stats as { label, value, icon: Icon }}
			<Card.Root class="shadow-md border-none">
				<Card.Header class="pb-2">
					<div class="flex items-center gap-3">
						<Icon class="h-5 w-5 text-muted-foreground" />
						<Card.Title class="text-sm font-medium text-muted-foreground">{label}</Card.Title>
					</div>
				</Card.Header>
				<Card.Content>
					<p class="text-3xl font-normal text-foreground">{value}</p>
				</Card.Content>
			</Card.Root>
		{/each}
	</div>
</div>
