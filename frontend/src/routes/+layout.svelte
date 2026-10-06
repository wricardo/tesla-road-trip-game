<script lang="ts">
	import favicon from '$lib/assets/favicon.svg';
	import '../app.css';
	import { setContextClient } from '@urql/svelte';
	import { makeClient } from '$lib/graphql';
	import { browser } from '$app/environment';
	import { page } from '$app/stores';

	let { children } = $props();

	const navItems = [
		{ href: '/', label: 'Play' },
		{ href: '/learn', label: 'How it works' },
		{ href: '/lobby', label: 'Live sessions' },
		{ href: '/maps', label: 'Maps' },
		{ href: '/multi', label: 'Watch multiple' }
	];

	function isActive(href: string, pathname: string): boolean {
		return href === '/' ? pathname === '/' || pathname.startsWith('/watch/') : pathname === href || pathname.startsWith(`${href}/`);
	}

	// ssr=false so this always runs in browser — safe to set client synchronously
	if (browser) {
		setContextClient(makeClient());
	}
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
	<link rel="preconnect" href="https://fonts.googleapis.com" />
	<link href="https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600&display=swap" rel="stylesheet" />
</svelte:head>

<div class="min-h-screen flex flex-col bg-[#f7f7f7]">
	<header class="border-b border-[#e8e8e8] bg-white px-6 py-4 flex items-center justify-between">
		<a href="/" class="flex items-center gap-3 no-underline">
			<span class="text-xl font-light tracking-widest text-[#393c41]">TESLA</span>
			<span class="text-xs text-gray-400 font-light">Road Trip</span>
		</a>
		<nav class="flex items-center gap-1 text-sm" aria-label="Main">
			{#each navItems as item}
				{@const active = isActive(item.href, $page.url.pathname)}
				<a
					href={item.href}
					aria-current={active ? 'page' : undefined}
					class="px-3 py-1.5 rounded-full transition-colors {active ? 'bg-[#393c41] text-white' : 'text-gray-500 hover:text-[#393c41] hover:bg-gray-100'}"
				>{item.label}</a>
			{/each}
		</nav>
	</header>

	<main class="flex-1">
		{@render children()}
	</main>

	<footer class="border-t border-[#e8e8e8] bg-white px-6 py-8 mt-auto">
		<div class="max-w-7xl mx-auto grid grid-cols-2 sm:grid-cols-4 gap-6 text-xs text-gray-400">
			<div>
				<p class="font-semibold text-[#393c41] mb-2">Tesla Road Trip</p>
				<p class="leading-relaxed">Teach AI agents to navigate, plan routes, and manage resources through an interactive educational game.</p>
			</div>
			<div>
				<p class="font-semibold text-[#393c41] mb-2">Play</p>
				<div class="flex flex-col gap-1.5">
					<a href="/" class="hover:text-gray-600 transition-colors">Home / Create session</a>
					<a href="/maps" class="hover:text-gray-600 transition-colors">Maps</a>
					<a href="/editor" class="hover:text-gray-600 transition-colors">Map editor</a>
					<a href="/lobby" class="hover:text-gray-600 transition-colors">Live sessions</a>
					<a href="/multi" class="hover:text-gray-600 transition-colors">Multi-watch</a>
				</div>
			</div>
			<div>
				<p class="font-semibold text-[#393c41] mb-2">Docs</p>
				<div class="flex flex-col gap-1.5">
					<a href="/learn" class="hover:text-gray-600 transition-colors">Learn</a>
					<a href="/llms.txt" target="_blank" rel="noreferrer" class="hover:text-gray-600 transition-colors">/llms.txt</a>
					<a href="/graphql" target="_blank" rel="noreferrer" class="hover:text-gray-600 transition-colors">GraphQL endpoint</a>
				</div>
			</div>
			<div>
				<p class="font-semibold text-[#393c41] mb-2">Tools</p>
				<div class="flex flex-col gap-1.5">
					<a href="/playground" target="_blank" rel="noreferrer" class="hover:text-gray-600 transition-colors">GraphQL Playground</a>
					<a href="/admin" class="hover:text-gray-600 transition-colors">Admin</a>
					<a href="https://github.com/wricardo/gqlcli#-quick-start--using-the-cli" target="_blank" rel="noreferrer" class="hover:text-gray-600 transition-colors">gqlcli — GraphQL CLI</a>
				</div>
			</div>
		</div>
	</footer>
</div>
