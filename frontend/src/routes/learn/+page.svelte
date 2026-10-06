<script lang="ts">
	const legendTiles = [
		{ label: 'Car', glyph: '🚗', swatch: 'bg-white border border-gray-200', text: '' },
		{ label: 'Home', glyph: '⌂', swatch: 'bg-red-500', text: 'text-white font-bold' },
		{ label: 'Park', glyph: '🌳', swatch: 'bg-emerald-500', text: '' },
		{ label: 'Charger', glyph: '⚡', swatch: 'bg-yellow-400', text: '' },
		{ label: 'Water', glyph: '≈', swatch: 'bg-blue-400', text: 'text-white font-bold' },
		{ label: 'Building', glyph: '', swatch: 'bg-slate-700', text: '' },
		{ label: 'One-way road', glyph: '→', swatch: 'bg-white border border-gray-200', text: 'text-orange-500 font-bold' }
	];

	const rules = [
		{
			icon: '🔋',
			title: 'Every move costs 1 battery',
			body: 'Home (⌂) and chargers (⚡) refill the battery to full. If it reaches 0 anywhere else, the run ends.'
		},
		{
			icon: '💥',
			title: 'Crashes end the run',
			body: 'Driving into water, a building, or off the edge of the map ends the game immediately. The crash itself costs no battery, but there is no undo: only a reset.'
		},
		{
			icon: '↔️',
			title: 'One-way roads reject wrong-way moves',
			body: 'Some roads only allow certain directions, shown as arrows. Moving against them is simply refused: nothing moves, no battery is spent, and the game goes on.'
		},
		{
			icon: '🌳',
			title: 'Visit every park to win',
			body: 'You win the moment the last park is visited. The score is how many parks you have reached so far.'
		}
	];

	const steps = [
		['Read the state', 'Ask for the car’s position, battery, the map around it, and which directions the nearby roads allow.'],
		['Plan a route', 'Pick the nearest park, check that the battery covers the trip, and note where the next charger is. Obey one-way roads.'],
		['Move', 'Send one step at a time (up, down, left or right), or a short batch of steps once a stretch of road is known to be safe.'],
		['Check and repeat', 'Read what came back: did the move succeed, how much battery is left, did the game end? Then plan the next leg.']
	];

	const strategies = [
		['Use clear corridors', 'Find rows and columns with no obstacles and treat them as highways between objectives.'],
		['Charge early', 'Top up before the battery gets critical, and always know the distance to the nearest charger or home.'],
		['Don’t scout by crashing', 'Hitting water or a building ends the run. Read the tile types around the car before every move instead of bumping into things.'],
		['Respect one-way roads', 'A move has to be allowed by both the cell you leave and the cell you enter. Plan around the arrows rather than testing them.'],
		['Clear the map in sections', 'Split the grid into areas and finish one before moving on, so you never backtrack across the whole map.'],
		['Read the grid carefully', 'Roads and buildings can look alike in plain text. Parse the map character by character before deciding a route is blocked.']
	];
</script>

<svelte:head>
	<title>How it works — Tesla Road Trip</title>
</svelte:head>

<div class="max-w-3xl mx-auto px-6 py-8">
	<h1 class="sr-only">Learn AI with Tesla Road Trip</h1>

	<!-- the game -->
	<section id="game" class="mb-12 scroll-mt-6">
		<h2 class="text-lg font-medium text-[#393c41] mb-4">The game</h2>
		<p class="text-sm text-gray-600 leading-relaxed mb-5">
			Tesla Road Trip is a grid-based navigation puzzle. A car starts at home on a map made of roads, parks,
			chargers, water and buildings. The goal is simple: visit every park without running out of battery or
			driving into something. People play it with the arrow keys; AI agents play it through an API. Either way
			you can watch a run live, which makes it a handy way to see how an AI plans, recovers from mistakes and
			manages a limited resource.
		</p>
		<div class="grid grid-cols-3 sm:grid-cols-7 gap-3">
			{#each legendTiles as tile}
				<div class="bg-white rounded-xl border border-[#e8e8e8] p-3 text-center">
					<div class="h-8 w-8 rounded-md mx-auto mb-2 flex items-center justify-center text-base leading-none {tile.swatch} {tile.text}" aria-hidden="true">{tile.glyph}</div>
					<div class="text-xs text-gray-600">{tile.label}</div>
				</div>
			{/each}
		</div>
	</section>

	<!-- rules -->
	<section id="rules" class="mb-12 scroll-mt-6">
		<h2 class="text-lg font-medium text-[#393c41] mb-4">Rules at a glance</h2>
		<div class="grid gap-3 sm:grid-cols-2">
			{#each rules as rule}
				<div class="bg-white rounded-xl border border-[#e8e8e8] p-4">
					<p class="text-sm font-medium text-[#393c41] mb-1"><span aria-hidden="true">{rule.icon}</span> {rule.title}</p>
					<p class="text-sm text-gray-600 leading-relaxed">{rule.body}</p>
				</div>
			{/each}
		</div>
	</section>

	<!-- how an AI plays -->
	<section id="ai" class="mb-12 scroll-mt-6">
		<h2 class="text-lg font-medium text-[#393c41] mb-4">How an AI plays</h2>
		<ol class="flex flex-col gap-3">
			{#each steps as [title, desc], i}
				<li class="flex items-start gap-4 bg-white rounded-xl border border-[#e8e8e8] p-4">
					<span class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-[#393c41] text-sm font-medium text-white" aria-hidden="true">{i + 1}</span>
					<div>
						<p class="text-sm font-medium text-[#393c41]">{title}</p>
						<p class="text-sm text-gray-600 mt-0.5 leading-relaxed">{desc}</p>
					</div>
				</li>
			{/each}
		</ol>
		<p class="text-sm text-gray-600 leading-relaxed mt-4">
			The loop ends when the car wins, crashes, or runs out of battery. After that the agent can reset the session and try again, and the
			reset count shows up in the live view.
		</p>
	</section>

	<!-- connect -->
	<section id="connect" class="mb-12 scroll-mt-6">
		<h2 class="text-lg font-medium text-[#393c41] mb-4">Let an AI play</h2>
		<div class="bg-white rounded-xl border border-[#e8e8e8] p-5">
			<p class="text-sm text-gray-600 leading-relaxed mb-3">
				Everything an AI needs to play is in one page: <a href="/llms.txt" target="_blank" rel="noreferrer" class="font-medium underline hover:text-[#393c41]">llms.txt</a>.
				It explains the rules, how to connect, and how to create a session, read the state and move the car, so you don’t have to.
			</p>
			<p class="text-sm text-gray-600 leading-relaxed mb-4">
				Point your AI at that page (or paste the session prompt from the play screen’s “Play with an AI” card, which links to it) and let it take it from there.
			</p>
			<a href="/llms.txt" target="_blank" rel="noreferrer" class="inline-block bg-[#393c41] text-white text-sm px-5 py-2.5 rounded-full hover:bg-black transition-colors">Open llms.txt</a>
		</div>
	</section>

	<!-- fog -->
	<section id="fog" class="mb-12 scroll-mt-6">
		<h2 class="text-lg font-medium text-[#393c41] mb-4">Fog mode</h2>
		<p class="text-sm text-gray-600 leading-relaxed mb-3">
			Fog turns the puzzle into an exploration problem. In a fog session the agent only sees the tiles within a few cells of the car
			(you choose the radius), so it has to discover the map as it drives instead of planning the whole route up front.
		</p>
		<p class="text-sm text-gray-600 leading-relaxed">
			The full map is locked behind a password, for people who want to see what the agent is missing. Leave the password blank when
			creating a session and one is generated for you: the web app shows it on the play screen, and the API returns it once at
			creation. Agents should not try to guess it.
		</p>
	</section>

	<!-- strategies -->
	<section id="strategies" class="scroll-mt-6">
		<h2 class="text-lg font-medium text-[#393c41] mb-4">Strategies that work</h2>
		<div class="grid gap-3 sm:grid-cols-2">
			{#each strategies as [title, desc]}
				<div class="bg-white rounded-xl border border-[#e8e8e8] p-4">
					<p class="text-sm font-medium text-[#393c41] mb-1">{title}</p>
					<p class="text-sm text-gray-600 leading-relaxed">{desc}</p>
				</div>
			{/each}
		</div>
	</section>
</div>
