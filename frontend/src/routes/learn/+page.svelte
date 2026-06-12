<script lang="ts">
	let copied = $state('');

	const graphqlCreateSessionCurl = `BASE_URL="http://tesla.wricardo.net" # local backend: http://localhost:9090
curl -s "$BASE_URL/graphql" \
  -H 'Content-Type: application/json' \
  --data '{"query":"mutation { createSession(mapID: \"classic\") { id mapName gameState { battery playerPos { x y } } } }"}'`;

	const graphqlStateCurl = `SESSION_ID="paste-session-id-here"
BASE_URL="http://tesla.wricardo.net"
curl -s "$BASE_URL/graphql" \
  -H 'Content-Type: application/json' \
  --data "$(jq -nc --arg id "$SESSION_ID" '{
    query: "query($id: ID!) { gameState(sessionID: $id) { battery maxBattery score victory gameOver localView3x3 playerPos { x y } grid { type allowedDirections } } }",
    variables: { id: $id }
  }')"`;

	const graphqlMoveCurl = `SESSION_ID="paste-session-id-here"
BASE_URL="http://tesla.wricardo.net"
curl -s "$BASE_URL/graphql" \
  -H 'Content-Type: application/json' \
  --data "$(jq -nc --arg id "$SESSION_ID" '{
    query: "mutation($id: ID!) { move(sessionID: $id, direction: RIGHT) { success message gameState { battery score playerPos { x y } } } }",
    variables: { id: $id }
  }')"`;

	const mcpListToolsCurl = `BASE_URL="http://tesla.wricardo.net"
curl -s "$BASE_URL/mcp" \
  -H 'Content-Type: application/json' \
  --data '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}'`;

	const mcpCreateSessionCurl = `BASE_URL="http://tesla.wricardo.net"
curl -s "$BASE_URL/mcp" \
  -H 'Content-Type: application/json' \
  --data '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"create_session","arguments":{"map_name":"classic"}}}'`;

	const mcpMoveCurl = `SESSION_ID="paste-session-id-here"
BASE_URL="http://tesla.wricardo.net"
curl -s "$BASE_URL/mcp" \
  -H 'Content-Type: application/json' \
  --data "$(jq -nc --arg id "$SESSION_ID" '{
    jsonrpc: "2.0",
    id: 3,
    method: "tools/call",
    params: { name: "move", arguments: { session_id: $id, direction: "right" } }
  }')"`;

	const claudeMcpConfig = `claude mcp add --transport http tesla-game http://tesla.wricardo.net/mcp`;

	const legendTiles = [
		{ label: 'Player', icon: '🚗' },
		{ label: 'Home', swatch: 'bg-red-500' },
		{ label: 'Park', swatch: 'bg-emerald-500' },
		{ label: 'Charger', swatch: 'bg-yellow-400' },
		{ label: 'Water', swatch: 'bg-blue-400' },
		{ label: 'Building', swatch: 'bg-slate-700' },
		{ label: 'One-way', icon: '→' }
	];

	async function copy(text: string, key: string) {
		await navigator.clipboard.writeText(text);
		copied = key;
		setTimeout(() => {
			if (copied === key) copied = '';
		}, 1200);
	}
</script>

{#snippet Snippet(title: string, code: string, id: string)}
	<div class="bg-white rounded-2xl border border-[#e8e8e8] overflow-hidden">
		<div class="flex items-center justify-between gap-3 px-4 py-3 border-b border-gray-100">
			<p class="text-xs uppercase tracking-widest text-gray-400">{title}</p>
			<button
				type="button"
				onclick={() => copy(code, id)}
				class="text-xs px-3 py-1.5 rounded-full border border-gray-200 text-[#393c41] hover:border-gray-400 transition-colors"
			>
				{copied === id ? 'Copied' : 'Copy'}
			</button>
		</div>
		<pre class="overflow-x-auto bg-gray-50 p-4 text-xs text-gray-700 leading-relaxed"><code>{code}</code></pre>
	</div>
{/snippet}

<div class="max-w-3xl mx-auto px-6 py-12">
	<h1 class="text-4xl font-light text-[#393c41] mb-3">Learn AI with Tesla Road Trip</h1>
	<p class="text-gray-400 font-light text-lg mb-12">A live sandbox for watching and building AI agents</p>

	<!-- game rules -->
	<section class="mb-12">
		<h2 class="text-lg font-medium text-[#393c41] mb-4">The Game</h2>
		<p class="text-sm text-gray-600 leading-relaxed mb-4">
			Tesla Road Trip is a grid-based navigation game. An AI controls a car moving through roads,
			collecting parks, and managing battery. Charging stations restore energy. Obstacles (buildings,
			water) are impassable. Some maps include directional roads: arrows show which way the car may
			enter and leave that road cell. Collect all parks to win.
		</p>
		<div class="grid grid-cols-3 sm:grid-cols-7 gap-3">
			{#each legendTiles as tile}
				<div class="bg-white rounded-xl border border-[#e8e8e8] p-3 text-center">
					{#if tile.icon}
						<div class="text-2xl mb-1">{tile.icon}</div>
					{:else}
						<div class={`h-7 w-7 rounded-md mx-auto mb-2 border border-white/70 ${tile.swatch}`}></div>
					{/if}
					<div class="text-xs text-gray-400">{tile.label}</div>
				</div>
			{/each}
		</div>
	</section>

	<!-- how an AI plays -->
	<section class="mb-12">
		<h2 class="text-lg font-medium text-[#393c41] mb-4">How an AI Plays</h2>
		<div class="flex flex-col gap-3">
			{#each [
				['1','Read state','Query gameState — position, battery, grid, parks, and allowedDirections'],
				['2','Plan','Find nearest park, check battery vs. distance to charger, and obey one-way roads'],
				['3','Move','Send a GraphQL mutation with direction UP, DOWN, LEFT, or RIGHT'],
				['4','Repeat','Loop until victory, dead battery, crash, or a blocked wrong-way move'],
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
		<h2 class="text-lg font-medium text-[#393c41] mb-4">Copy/paste API quickstart</h2>
		<p class="text-sm text-gray-500 mb-4">
			Goal: visit every park without running out of battery, hitting buildings/water, or driving the wrong way on directional roads.
			Use GraphQL for direct HTTP calls, or MCP when connecting an AI client like Claude Code.
		</p>

		<div class="rounded-2xl border border-[#e8e8e8] bg-white p-4 mb-6">
			<p class="text-xs uppercase tracking-widest text-gray-400 mb-2">Endpoints</p>
			<div class="grid gap-2 text-sm text-gray-600">
				<p><span class="font-medium text-[#393c41]">GraphQL:</span> <code class="font-mono">http://tesla.wricardo.net/graphql</code> <span class="text-gray-400">(local backend: <code>http://localhost:9090/graphql</code>)</span></p>
				<p><span class="font-medium text-[#393c41]">MCP:</span> <code class="font-mono">http://tesla.wricardo.net/mcp</code> <span class="text-gray-400">(local backend: <code>http://localhost:9090/mcp</code>)</span></p>
			</div>
		</div>

		<div class="space-y-6">
			<div>
				<h3 class="text-sm font-medium text-[#393c41] mb-2">GraphQL with curl</h3>
				<div class="space-y-3">
					{@render Snippet('Create a session', graphqlCreateSessionCurl, 'gql-create')}
					{@render Snippet('Read game state', graphqlStateCurl, 'gql-state')}
					{@render Snippet('Move right', graphqlMoveCurl, 'gql-move')}
				</div>
			</div>

			<div>
				<h3 class="text-sm font-medium text-[#393c41] mb-2">MCP with curl</h3>
				<p class="text-xs text-gray-500 mb-3">MCP calls use JSON-RPC over the HTTP endpoint. These are useful for debugging what an AI client will call.</p>
				<div class="space-y-3">
					{@render Snippet('List MCP tools', mcpListToolsCurl, 'mcp-tools')}
					{@render Snippet('Create session via MCP', mcpCreateSessionCurl, 'mcp-create')}
					{@render Snippet('Move right via MCP', mcpMoveCurl, 'mcp-move')}
				</div>
			</div>

			<div>
				<h3 class="text-sm font-medium text-[#393c41] mb-2">Add MCP to Claude Code</h3>
				{@render Snippet('Terminal', claudeMcpConfig, 'mcp-config')}
			</div>
		</div>

		<div class="flex flex-wrap gap-3 mt-6">
			<a href="/" class="bg-[#393c41] text-white text-sm px-5 py-2 rounded-full">Create a session</a>
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
				['One-way road planning','For cells with allowedDirections, only move in one of those directions. Wrong-way moves are blocked.'],
				['Section clearing','Divide the grid into sections. Complete each before moving on.'],
				['Character recognition','Parse grid character-by-character. R (road) looks like B (building), and arrows/turns are constrained roads.'],
			] as [title, desc]}
				<div class="bg-white rounded-xl border border-[#e8e8e8] p-4">
					<p class="text-sm font-medium text-[#393c41] mb-1">{title}</p>
					<p class="text-xs text-gray-500 leading-relaxed">{desc}</p>
				</div>
			{/each}
		</div>
	</section>
</div>
