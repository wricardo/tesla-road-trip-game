<script lang="ts">
	// TODO: optional globalMoves live ticker via subscription (ticket ji8.2)
	const curlSnippet = `curl -X POST /graphql \\
  -H 'Content-Type: application/json' \\
  -d '{"query":"mutation{createSession(mapID:\\"classic\\"){id}}"}'`;

	const moveSnippet = `mutation {
  move(sessionID: "<sessionId>", direction: RIGHT) {
    success
    message
    gameState { battery score playerPos { x y } }
  }
}`;
</script>

<div class="max-w-3xl mx-auto px-6 py-12">
	<h1 class="text-4xl font-light text-[#393c41] mb-3">Learn AI with Tesla Road Trip</h1>
	<p class="text-gray-400 font-light text-lg mb-12">A live sandbox for watching and building AI agents</p>

	<!-- game rules -->
	<section class="mb-12">
		<h2 class="text-lg font-medium text-[#393c41] mb-4">The Game</h2>
		<p class="text-sm text-gray-600 leading-relaxed mb-4">
			Tesla Road Trip is a grid-based navigation game. An AI controls a car moving through roads,
			collecting parks, and managing battery. Charging stations restore energy. Obstacles (buildings,
			water) are impassable. Collect all parks to win.
		</p>
		<div class="grid grid-cols-3 sm:grid-cols-6 gap-3">
			{#each [['🚗','Player'],['🏠','Home'],['🌳','Park'],['⚡','Charger'],['💧','Water'],['🏢','Building']] as [icon, label]}
				<div class="bg-white rounded-xl border border-[#e8e8e8] p-3 text-center">
					<div class="text-2xl mb-1">{icon}</div>
					<div class="text-xs text-gray-400">{label}</div>
				</div>
			{/each}
		</div>
	</section>

	<!-- how an AI plays -->
	<section class="mb-12">
		<h2 class="text-lg font-medium text-[#393c41] mb-4">How an AI Plays</h2>
		<div class="flex flex-col gap-3">
			{#each [
				['1','Read state','Query gameState — position, battery, grid, parks'],
				['2','Plan','Find nearest park, check battery vs. distance to charger'],
				['3','Move','Send a GraphQL mutation with direction UP, DOWN, LEFT, or RIGHT'],
				['4','Repeat','Loop until victory or crash'],
			] as [n, title, desc]}
				<div class="flex items-start gap-4 bg-white rounded-xl border border-[#e8e8e8] p-4">
					<span class="text-2xl font-light text-gray-200 w-8 shrink-0">{n}</span>
					<div>
						<p class="text-sm font-medium text-[#393c41]">{title}</p>
						<p class="text-xs text-gray-400 mt-0.5">{desc}</p>
					</div>
				</div>
			{/each}
		</div>
	</section>

	<!-- try it yourself -->
	<section class="mb-12">
		<h2 class="text-lg font-medium text-[#393c41] mb-4">Try It Yourself</h2>
		<p class="text-sm text-gray-500 mb-4">Create a session, then point any AI at these endpoints:</p>

		<div class="space-y-4">
			<div>
				<p class="text-xs uppercase tracking-widest text-gray-400 mb-2">GraphQL endpoint</p>
				<code class="block bg-gray-50 rounded-lg p-3 text-xs font-mono text-gray-700">/graphql</code>
			</div>
			<div>
				<p class="text-xs uppercase tracking-widest text-gray-400 mb-2">Create session (curl)</p>
				<code class="block bg-gray-50 rounded-lg p-3 text-xs font-mono text-gray-700 whitespace-pre">{curlSnippet}</code>
			</div>
			<div>
				<p class="text-xs uppercase tracking-widest text-gray-400 mb-2">Move mutation</p>
				<code class="block bg-gray-50 rounded-lg p-3 text-xs font-mono text-gray-700 whitespace-pre">{moveSnippet}</code>
			</div>
		</div>

		<div class="flex gap-3 mt-6">
			<a href="/" class="bg-[#393c41] text-white text-sm px-5 py-2 rounded-full">Go to Lobby</a>
			<a href="/playground" target="_blank" rel="noreferrer" class="border border-gray-200 text-sm px-5 py-2 rounded-full hover:bg-gray-50">GraphQL Playground</a>
			<a href="/llms.txt" target="_blank" rel="noreferrer" class="border border-gray-200 text-sm px-5 py-2 rounded-full hover:bg-gray-50">llms.txt</a>
		</div>
	</section>

	<!-- strategies -->
	<section>
		<h2 class="text-lg font-medium text-[#393c41] mb-4">Strategies AIs Use</h2>
		<div class="space-y-3">
			{#each [
				['Corridor navigation','Find obstacle-free rows/columns. Use them as highways between objectives.'],
				['Proactive charging','Recharge before battery is critical. Always know distance to nearest charger.'],
				['Section clearing','Divide the grid into sections. Complete each before moving on.'],
				['Character recognition','Parse grid character-by-character. R (road) looks like B (building) — never assume a row is fully blocked.'],
			] as [title, desc]}
				<div class="bg-white rounded-xl border border-[#e8e8e8] p-4">
					<p class="text-sm font-medium text-[#393c41] mb-1">{title}</p>
					<p class="text-xs text-gray-500 leading-relaxed">{desc}</p>
				</div>
			{/each}
		</div>
	</section>
</div>
