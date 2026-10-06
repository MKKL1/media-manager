<script lang="ts" module>
	import { cn } from "$lib/utils.js";

	const base = "rounded-4xl p-[3px] group-data-horizontal/tabs:h-9 group-data-vertical/tabs:rounded-2xl data-[variant=line]:rounded-none group/tabs-list text-muted-foreground inline-flex w-fit items-center justify-center group-data-[orientation=vertical]/tabs:h-fit group-data-[orientation=vertical]/tabs:flex-col";

	const variantClasses = {
		default: "cn-tabs-list-variant-default bg-muted",
		line: "cn-tabs-list-variant-line gap-1 bg-transparent",
	} as const;

	export function tabsListVariants(opts?: { variant?: TabsListVariant }): string {
		return cn(base, variantClasses[opts?.variant ?? "default"]);
	}

	export type TabsListVariant = keyof typeof variantClasses;
</script>

<script lang="ts">
	import { Tabs as TabsPrimitive } from "bits-ui";

	let {
		ref = $bindable(null),
		variant = "default",
		class: className,
		...restProps
	}: TabsPrimitive.ListProps & {
		variant?: TabsListVariant;
	} = $props();
</script>

<TabsPrimitive.List
	bind:ref
	data-slot="tabs-list"
	data-variant={variant}
	class={cn(tabsListVariants({ variant }), className)}
	{...restProps}
/>
