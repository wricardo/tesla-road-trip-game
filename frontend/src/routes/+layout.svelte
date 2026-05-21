<script lang="ts">
	import favicon from '$lib/assets/favicon.svg';
	import '../app.css';
	import { setContextClient } from '@urql/svelte';
	import { makeClient } from '$lib/graphql';
	import { browser } from '$app/environment';

	let { children } = $props();

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
			<span class="ml-2 inline-flex items-center gap-1 text-xs bg-red-50 text-red-500 border border-red-200 rounded-full px-2 py-0.5">
				<span class="w-1.5 h-1.5 rounded-full bg-red-400 animate-pulse"></span>
				LIVE
			</span>
		</a>
		<nav class="flex items-center gap-6 text-sm text-gray-500">
			<a href="/" class="hover:text-[#393c41] transition-colors">Home</a>
			<a href="/learn" class="hover:text-[#393c41] transition-colors">Learn</a>
			<a href="/lobby" class="hover:text-[#393c41] transition-colors">Live Sessions</a>
			<a href="/multi" class="hover:text-[#393c41] transition-colors">Multi</a>
			<a href="/admin" class="hover:text-[#393c41] transition-colors">Admin</a>
			<a href="/playground" class="hover:text-[#393c41] transition-colors" target="_blank" rel="noreferrer">Playground</a>
			<a href="/llms.txt" class="hover:text-[#393c41] transition-colors" target="_blank" rel="noreferrer">llms.txt</a>
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
					<a href="/lobby" class="hover:text-gray-600 transition-colors">Live sessions</a>
					<a href="/multi" class="hover:text-gray-600 transition-colors">Multi-watch</a>
					<a href="/admin" class="hover:text-gray-600 transition-colors">Admin</a>
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
				</div>
			</div>
		</div>
	</footer>
</div>
