<script lang="ts">
	import { getContextClient, queryStore, gql } from '@urql/svelte';
	import { onMount } from 'svelte';
	import { SESSIONS_QUERY } from '$lib/queries';

	const DELETE_SESSION = `
		mutation DeleteSession($id: ID!) {
			deleteSession(id: $id) { message }
		}
	`;

	const client = getContextClient();
	const sessionsResult = queryStore({ client, query: gql(SESSIONS_QUERY) });

	type Session = { id: string; mapName: string };
	let sessions = $state<Session[]>([]);
	let deleting = $state<Set<string>>(new Set());
	let nukingAll = $state(false);
	let confirmNuke = $state(false);
	let errorMessage = $state('');

	onMount(() => {
		const u = sessionsResult.subscribe((r) => {
			const d = r?.data?.sessions?.sessions;
			if (d) sessions = d;
		});
		return u;
	});

	async function deleteSession(id: string) {
		errorMessage = '';
		deleting.add(id);
		deleting = new Set(deleting);
		const result = await client.mutation(gql(DELETE_SESSION), { id }).toPromise();
		if (result.error) errorMessage = result.error.message;
		deleting.delete(id);
		deleting = new Set(deleting);
		sessionsResult.reexecute?.({ requestPolicy: 'network-only' });
	}

	async function nukeAll() {
		if (!confirmNuke) { confirmNuke = true; return; }
		nukingAll = true;
		confirmNuke = false;
		errorMessage = '';
		const ids = sessions.map(s => s.id);

		for (const id of ids) {
			const result = await client.mutation(gql(DELETE_SESSION), { id }).toPromise();
			if (result.error) {
				errorMessage = `Failed deleting ${id}: ${result.error.message}`;
				break;
			}
			sessions = sessions.filter(s => s.id !== id);
		}

		nukingAll = false;
		sessionsResult.reexecute?.({ requestPolicy: 'network-only' });
	}
</script>

<svelte:head>
	<title>Admin — Tesla Road Trip</title>
</svelte:head>

<div class="max-w-4xl mx-auto px-6 py-10">
	<div class="flex items-center justify-between mb-8">
		<div>
			<h1 class="text-2xl font-light text-[#393c41]">Admin</h1>
			<p class="text-sm text-gray-400 mt-1">{sessions.length} active sessions</p>
		</div>
		<button
			onclick={nukeAll}
			disabled={nukingAll || sessions.length === 0}
			class="px-5 py-2 rounded-full text-sm font-medium transition-colors disabled:opacity-50
				{confirmNuke ? 'bg-red-600 text-white hover:bg-red-700' : 'bg-gray-100 text-gray-700 hover:bg-gray-200'}"
		>
			{#if nukingAll}
				Nuking…
			{:else if confirmNuke}
				⚠ Confirm — delete all {sessions.length} sessions?
			{:else}
				Nuke All Sessions
			{/if}
		</button>
	</div>

	{#if errorMessage}
		<div class="mb-4 rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-600">{errorMessage}</div>
	{/if}

	{#if sessions.length === 0}
		<div class="text-center py-20 text-gray-400">
			<p class="text-lg font-light">No sessions</p>
		</div>
	{:else}
		<div class="bg-white rounded-2xl border border-[#e8e8e8] divide-y divide-gray-100 shadow-sm overflow-hidden">
			{#each sessions as s (s.id)}
				<div class="flex items-center justify-between px-5 py-3">
					<div class="flex items-center gap-3">
						<span class="font-mono text-sm text-[#393c41]">{s.id}</span>
						<span class="text-xs text-gray-400 bg-gray-50 rounded-full px-2 py-0.5">{s.mapName}</span>
					</div>
					<div class="flex items-center gap-3">
						<a href="/watch/{s.id}" class="text-xs text-gray-400 hover:text-gray-600 transition-colors">Watch</a>
						<button
							onclick={() => deleteSession(s.id)}
							disabled={deleting.has(s.id)}
							class="text-xs px-3 py-1 rounded-full border border-red-200 text-red-500 hover:bg-red-50 transition-colors disabled:opacity-40"
						>
							{deleting.has(s.id) ? 'Deleting…' : 'Delete'}
						</button>
					</div>
				</div>
			{/each}
		</div>
	{/if}
</div>
