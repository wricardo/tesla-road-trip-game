import{B as e,D as t,E as n,F as r,G as i,H as a,I as o,J as s,K as c,N as l,S as u,T as d,U as f,V as p,W as m,X as h,Y as g,b as _,c as ee,d as te,et as v,h as y,it as b,k as x,m as S,n as ne,p as re,rt as C,s as ie,tt as w,v as ae,w as T,x as E,y as oe,z as se}from"../chunks/DzEw6oMf.js";import"../chunks/CP97kCR3.js";import{n as ce,s as D,t as le}from"../chunks/DwquRVKQ.js";import{t as ue}from"../chunks/C9h3bAgZ.js";import{t as de}from"../chunks/TtMn8oWR.js";import{a as fe,i as pe}from"../chunks/DfNbAqqf.js";import{i as me,t as he}from"../chunks/sWqJ-lm0.js";var ge=T(`<span class="normal-case tracking-normal inline-flex items-center rounded-full bg-blue-100 text-blue-700 px-2 py-0.5 text-[10px]"> </span>`),_e=T(`<span>·</span> <span class="normal-case tracking-normal font-mono"> </span> <!>`,1),ve=T(`<span class="text-lg leading-none text-gray-800 font-medium"> </span> <button class="font-mono text-sm leading-none text-gray-400 hover:text-blue-600 transition-colors" title="Copy session ID"> </button>`,1),ye=T(`<button class="font-mono text-lg leading-none text-gray-800 hover:text-blue-600 transition-colors" title="Copy session ID"> </button>`),be=T(`<div class="flex-1 min-w-[16rem] max-w-xl"><div class="flex justify-between text-xs text-gray-400 mb-1"><span>Battery</span><span> </span></div> <div class="h-2 bg-gray-100 rounded-full overflow-hidden"><div></div></div></div> <div class="grid grid-cols-4 gap-2 text-center"><div class="bg-gray-50 rounded-xl px-4 py-2"><div class="text-xl font-light leading-tight"> </div> <div class="text-[11px] text-gray-400">Parks</div></div> <div class="bg-gray-50 rounded-xl px-4 py-2"><div class="text-xl font-light leading-tight"> </div> <div class="text-[11px] text-gray-400">Moves</div></div> <div class="bg-gray-50 rounded-xl px-4 py-2"><div class="text-xl font-light leading-tight"> </div> <div class="text-[11px] text-gray-400">Resets</div></div> <div class="bg-gray-50 rounded-xl px-4 py-2"><div> </div> <div class="text-[11px] text-gray-400"> </div></div></div>`,1),xe=T(`<p class="text-sm text-gray-400 font-light">Loading…</p>`),Se=T(`<span class="text-sky-500 leading-none">•</span>`),Ce=T(`<span> </span>`),we=T(`<td><!></td>`),Te=T(`<tr></tr>`),Ee=T(`<table class="game-board border-collapse svelte-1oiicp0"><tbody></tbody></table>`),De=T(`<div class="flex items-center justify-center h-64 text-gray-400"><div class="text-center"><span class="text-4xl block mb-3">🚗</span> <p class="text-sm font-light">Loading <code class="font-mono"> </code>…</p></div></div>`),Oe=T(`<div class="flex items-center justify-center h-64 text-red-400"><p class="text-sm"> </p></div>`),ke=T(`<div class="flex items-center justify-center h-64 text-gray-400"><div class="text-center"><span class="text-4xl block mb-3">🚗</span> <p class="text-sm font-light">Waiting for moves on <code class="font-mono"> </code>…</p> <p class="text-xs mt-2">Point an AI at this session to see it play</p></div></div>`),Ae=T(`<p class="text-xs text-gray-400 mt-2"> </p>`),je=T(`<p class="text-xs text-red-500 mt-2"> </p>`),Me=T(`<button type="button" class="text-xs px-3 py-1.5 rounded-full border border-gray-200 text-gray-600 hover:bg-gray-50 transition-colors">Use nearby grid</button>`),Ne=T(`<p class="text-[11px] text-green-600 mt-2">Full grid unlocked for this session.</p>`),Pe=T(`<p class="text-[11px] text-red-500 mt-2"> </p>`),Fe=T(`<div class="rounded-xl bg-white border border-gray-100 px-4 py-3"><span class="text-[11px] uppercase tracking-widest text-gray-400">Fog mode</span> <p class="text-xs text-gray-500 mt-1"> </p> <div class="mt-3 flex flex-wrap items-center gap-2"><input type="text" placeholder="Grid password" class="min-w-[13rem] flex-1 border border-gray-200 rounded-lg px-3 py-1.5 text-xs bg-white focus:outline-none focus:border-gray-400"/> <button type="button" class="text-xs px-3 py-1.5 rounded-full border border-blue-200 text-blue-600 hover:bg-blue-50 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"> </button> <!></div> <!> <!></div>`),Ie=T(`<div class="border-t border-gray-100 bg-gray-50/40 p-3 sm:p-4"><div class="max-w-md space-y-3"><div class="rounded-xl bg-white border border-gray-100 px-4 py-3"><div class="flex items-center justify-between gap-3 mb-2"><span class="text-[11px] uppercase tracking-widest text-gray-400">Keyboard controls</span> <span> </span></div> <p class="text-xs text-gray-500">Use <kbd class="kbd svelte-1oiicp0">↑</kbd><kbd class="kbd svelte-1oiicp0">↓</kbd><kbd class="kbd svelte-1oiicp0">←</kbd><kbd class="kbd svelte-1oiicp0">→</kbd> or <kbd class="kbd svelte-1oiicp0">W</kbd><kbd class="kbd svelte-1oiicp0">A</kbd><kbd class="kbd svelte-1oiicp0">S</kbd><kbd class="kbd svelte-1oiicp0">D</kbd> to drive. Press <kbd class="kbd svelte-1oiicp0">R</kbd> to reset.</p> <div class="mt-3 flex items-center gap-2"><button type="button" class="text-xs px-3 py-1.5 rounded-full border border-red-200 text-red-500 hover:bg-red-50 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"> </button> <span class="text-[11px] text-gray-400">confirmation required</span></div> <!> <!> <p class="text-[11px] text-gray-400 mt-2">Ignored while typing in the prompt or any form field.</p></div> <!></div></div>`),Le=T(`<div class="max-w-[1900px] mx-auto px-3 sm:px-4 py-4 lg:py-6"><div><section class="min-w-0 bg-white rounded-2xl border border-[#e8e8e8] shadow-sm overflow-visible"><div class="p-3 sm:p-4 border-b border-gray-100"><div class="flex flex-wrap items-start justify-between gap-3"><div class="min-w-0"><div class="flex flex-wrap items-center gap-2 text-xs uppercase tracking-widest text-gray-400"><span>Session</span> <!></div> <div class="mt-1 flex items-center gap-2"><!></div></div> <!></div></div> <div class="p-3 sm:p-4 flex items-start justify-start board-pane svelte-1oiicp0"><!></div> <div class="flex flex-wrap items-center justify-between gap-x-4 gap-y-2 text-xs text-gray-400 px-4 pb-3"><div class="flex flex-wrap items-center gap-x-4 gap-y-1"><span><span class="text-sky-400">•</span> movement trail</span> <span class="flex items-center gap-x-3 gap-y-1 flex-wrap"><span class="flex items-center gap-1"><span class="inline-block w-3 h-3 rounded-sm bg-red-500"></span> Home</span> <span class="flex items-center gap-1"><span class="inline-block w-3 h-3 rounded-sm bg-emerald-500"></span> Park</span> <span class="flex items-center gap-1"><span class="inline-block w-3 h-3 rounded-sm bg-yellow-400"></span> Charger</span> <span class="flex items-center gap-1"><span class="inline-block w-3 h-3 rounded-sm bg-slate-700"></span> Blocked</span> <span class="flex items-center gap-1"><span class="inline-block w-3 h-3 rounded-sm bg-blue-400"></span> Water</span></span></div> <a href="/lobby" class="hover:text-gray-600 transition-colors">← Back to sessions</a></div> <!></section> <aside><div class="flex items-start justify-between gap-3 mb-3"><div class="min-w-0"><span class="text-xs uppercase tracking-widest text-gray-400">Prompt for LLM</span> <div class="mt-2 rounded-xl border border-blue-100 bg-blue-50 px-3 py-2 text-sm text-blue-900"><strong class="font-medium">Copy this into an AI chat</strong> to control session <code class="font-mono font-semibold"> </code>.</div></div> <button> </button></div> <textarea readonly="" class="w-full text-sm font-mono text-gray-700 bg-gray-50 rounded-xl p-4 resize-none prompt-pane focus:outline-none leading-relaxed border border-gray-100 svelte-1oiicp0"></textarea></aside></div></div>`);function Re(n,T){w(T,!0);let Re=()=>h(de,`$page`,Be),ze=()=>h(He,`$sessionQuery`,Be),[Be,Ve]=g(),O=le(),k=Re().params.id??``,He=ce({client:O,query:D(`
		query Session($id: ID!) {
			session(id: $id) {
				id
				displayName
				mapName
				gameMap { gridSize }
			}
		}
	`),variables:{id:k}}),A=s(()=>ze().data?.session?.displayName??null),Ue=s(()=>ze().data?.session?.gameMap?.gridSize??null);function j(e,t){return!t||!t.fogEnabled?e:{...e,fogEnabled:!0,fogRadius:e.fogRadius>0?e.fogRadius:t.fogRadius}}let M=c(null),We=c(!0),Ge=c(null),N=c(null),P=c(m(new Set)),F=c(!1),I=``,L=0,R=c(!1),z=c(!1),B=c(!1),V=c(null),Ke=c(null),H=null,U=s(()=>x(M)),W=c(null),G=c(``),K=c(``),q=c(null),J=c(!1),qe=c(!1),Y=c(`none`),X=s(()=>!!x(W)),Je=s(()=>x(U)?.fogEnabled&&!x(X)?x(U).playerPos??null:x(N)??x(U)?.playerPos??null),Z=s(()=>x(W)??x(U)?.nearbyGrid??[]),Q=s(()=>!!x(U)?.fogEnabled&&!x(X)&&!!x(Ue)),Ye=s(()=>x(Q)?x(Ue)??0:x(Z).length),Xe=s(()=>Array.from({length:x(Ye)},(e,t)=>t)),Ze=s(()=>(x(Ye)??0)>=30);o(()=>{let e=ue({url:typeof window<`u`?`${window.location.protocol===`https:`?`wss`:`ws`}://${window.location.host}/graphql`:`ws://localhost:8080/graphql`}).subscribe({query:`
		subscription SessionUpdated($sessionID: ID!) {
			sessionUpdated(sessionID: $sessionID) {
				battery maxBattery score victory gameOver totalMoves resetCount mapName
				fogEnabled fogRadius
				playerPos { x y }
				nearbyGrid { type visited id allowedDirections }
				currentMoves { fromPosition { x y } toPosition { x y } success }
			}
		}
	`,variables:{sessionID:k}},{next(e){let t=e.data?.sessionUpdated;t&&i(M,j(t,x(M)),!0)},error(e){console.error(`WS error`,e)},complete(){}});return()=>e()}),ne(()=>{let e=!1;return(async()=>{i(We,!0),i(Ge,null);try{let t=await O.query(D(`
		query GameState($sessionID: ID!) {
			gameState(sessionID: $sessionID) {
				battery maxBattery score victory gameOver totalMoves resetCount mapName
				fogEnabled fogRadius
				playerPos { x y }
				nearbyGrid { type visited id allowedDirections }
				currentMoves { fromPosition { x y } toPosition { x y } success }
			}
		}
	`),{sessionID:k},{requestPolicy:`network-only`}).toPromise();if(e)return;if(t.error)throw t.error;let n=t.data?.gameState;if(!n)throw Error(`Session state is unavailable`);i(M,j(n,x(M)),!0)}catch(t){if(e)return;i(Ge,t instanceof Error?t.message:`Failed to load session state`,!0)}finally{e||i(We,!1)}})(),()=>{e=!0}});let Qe=c(!1),$e=s(()=>`Use this GraphQL API to control an existing Tesla Road Trip game session.

Goal: visit every park. Each move costs 1 battery; home (H) and superchargers (S) refill it. Reaching 0 battery away from a charger ends the game. Buildings, water and wrong-way moves on one-way roads are rejected (hitting a building/water ends the game on maps with wallCrashEndsGame). Full rules and every field: ${typeof window<`u`?window.location.origin:``}/llms.txt

Session ID: ${k}
GraphQL endpoint: ${typeof window<`u`?window.location.origin:``}/graphql
Playground: ${typeof window<`u`?window.location.origin:``}/playground
MCP endpoint: ${typeof window<`u`?window.location.origin:``}/mcp (Streamable HTTP transport)

To use MCP in Claude Code, run:
claude mcp add --transport http tesla-game ${typeof window<`u`?window.location.origin:``}/mcp

GraphQL introspection is enabled. Use the Playground Docs panel or query __schema/__type to discover fields before constructing operations.

## Inspect the API
query {
  __type(name: "GameState") {
    fields { name type { kind name ofType { kind name } } }
  }
}

## Read current session state (fog-safe)
query {
  gameState(sessionID: "${k}") {
    mapName
    fogEnabled
    fogRadius
    playerPos { x y }
    battery
    maxBattery
    score
    victory
    gameOver
    message
    nearbyGrid { type visited id allowedDirections }
    visitedParks { id visited }
  }
}

## Read full grid
# If fogEnabled=true, pass the correct password.
# If fogEnabled=false, password is optional.
query {
  gameState(sessionID: "${k}") {
    grid(password: "YOUR_GRID_PASSWORD") { type visited id allowedDirections }
  }
}

## Send one move
mutation {
  move(sessionID: "${k}", direction: RIGHT) {
    success
    message
    attemptedTo { x y tileChar tileType passable }
    gameState {
      playerPos { x y }
      battery
      score
      victory
      gameOver
      nearbyGrid { type visited id allowedDirections }
    }
  }
}

## Send a move sequence
mutation {
  bulkMove(sessionID: "${k}", moves: [UP, RIGHT, DOWN]) {
    success
    movesExecuted
    requestedMoves
    stoppedReason
    stopReasonCode
    truncated
    limit
    gameState {
      playerPos { x y }
      battery
      score
      victory
      gameOver
      nearbyGrid { type visited id allowedDirections }
    }
  }
}

bulkMove accepts at most 50 moves per call. Check success, stoppedReason, stopReasonCode, truncated, gameOver, and victory before sending another operation.
stopReasonCode "already_over" (or a move message starting "Game is already over") means the game had already ended; nothing moved. It is not a blocked path: call reset.

## Manage this session
mutation { reset(sessionID: "${k}") { playerPos { x y } battery score victory gameOver nearbyGrid { type visited id allowedDirections } } }
query { history(sessionID: "${k}", page: 1, limit: 20, order: DESC) { totalMoves moves { moveNumber action success battery } } }
mutation { deleteSession(id: "${k}") { message } }

Directions: UP DOWN LEFT RIGHT. RIGHT = x+1, DOWN = y+1. Full grid coordinates are grid[y][x].
One-way roads: a move must be listed (north/south/east/west) in allowedDirections of both the cell you leave and the cell you enter, when those lists are non-empty.
${x(U)?.fogEnabled?`This session uses FOG (radius ${x(U).fogRadius}). nearbyGrid is the (2r+1)x(2r+1) window around the car: nearbyGrid[j][i] is cell (x - r + i, y - r + j); off-map cells read as building. grid(password: ...) needs the password chosen at creation.`:`Fog is off: grid needs no password (the password argument is ignored). nearbyGrid is the 3x3 window around the car.`}`);function et(){navigator.clipboard.writeText(x($e)),i(Qe,!0),setTimeout(()=>i(Qe,!1),2e3)}async function tt(e){let t=await O.query(D(`
		query FullGrid($sessionID: ID!, $password: String!) {
			gameState(sessionID: $sessionID) {
				grid(password: $password) { type visited id allowedDirections }
			}
		}
	`),{sessionID:k,password:e},{requestPolicy:`network-only`}).toPromise();if(t.error)throw t.error;let n=t.data?.gameState?.grid;if(!n||!n.length)throw Error(`No grid returned`);return n}async function nt(){if(!(!x(G).trim()||x(J))){i(J,!0),i(q,null);try{i(W,await tt(x(G)),!0),i(K,x(G),!0),i(Y,`password`)}catch(e){i(W,null),i(Y,`none`),i(q,e instanceof Error?e.message:`Failed to unlock full grid`,!0)}finally{i(J,!1)}}}function rt(){i(W,null),i(K,``),i(Y,`none`),i(q,null)}async function it(){if(x(Y)===`password`&&x(K))try{i(W,await tt(x(K)),!0)}catch{i(W,null),i(Y,`none`)}}o(()=>{let e=x(U);if(e){if(e.fogEnabled===!0){i(qe,!1),x(Y)===`auto`&&(i(W,null),i(Y,`none`));return}e.fogEnabled===!1&&(x(W)||x(J)||x(qe)||(i(qe,!0),(async()=>{try{i(W,await tt(``),!0),i(Y,`auto`)}catch{}})()))}});function at(e){switch(e.toLowerCase()){case`arrowup`:case`w`:return`UP`;case`arrowdown`:case`s`:return`DOWN`;case`arrowleft`:case`a`:return`LEFT`;case`arrowright`:case`d`:return`RIGHT`;default:return null}}function ot(e){if(!(e instanceof HTMLElement))return!1;let t=e.tagName.toLowerCase();return t===`input`||t===`textarea`||t===`select`||e.isContentEditable}function st(e,t){return!!e&&!!t&&e.x===t.x&&e.y===t.y}function ct(e,t,n){H&&clearTimeout(H),i(B,!0),i(F,!0),i(N,e,!0),i(P,new Set([`${e.x},${e.y}`]),!0),H=setTimeout(()=>{i(N,t,!0),i(P,new Set([`${e.x},${e.y}`,`${t.x},${t.y}`]),!0),H=setTimeout(()=>{i(M,n,!0),I=(n.currentMoves??[]).filter(e=>e.success).map(e=>`${e.fromPosition.x},${e.fromPosition.y}>${e.toPosition.x},${e.toPosition.y}`).join(`|`),L=(n.currentMoves??[]).filter(e=>e.success).length,i(N,null),i(F,!1),i(B,!1),H=null},180)},60)}async function lt(e){if(x(R)||x(z)||x(B)||x(U)?.gameOver||x(U)?.victory)return;let t=x(Je)??x(U)?.playerPos??null;i(R,!0),i(V,null),i(Ke,e,!0);try{let n=await O.mutation(D(pe),{sessionID:k,direction:e}).toPromise();if(n.error)throw n.error;let r=n.data?.move;if(!r)throw Error(`Move did not return a response`);let a=r.gameState,o=a?j(a,x(M)):null,s=!!a&&!(x(U)?.fogEnabled&&!x(X));r.success&&o&&t&&!st(t,o.playerPos)&&s?ct(t,o.playerPos,o):o&&i(M,o,!0),x(U)?.fogEnabled===!1&&await it(),r.success||i(V,r.message||`Could not move ${e.toLowerCase()}`,!0)}catch(e){i(V,e instanceof Error?e.message:`Move failed`,!0)}finally{i(R,!1)}}async function ut(){if(!(x(R)||x(z))&&confirm(`Reset session ${x(A)??k}? This clears progress for this run.`)){i(z,!0),i(V,null);try{let e=await O.mutation(D(fe),{sessionID:k}).toPromise();if(e.error)throw e.error;let t=e.data?.reset;if(!t)throw Error(`Reset did not return a game state`);i(M,j(t,x(M)),!0),i(Ke,null),i(N,null),i(P,new Set,!0),i(B,!1),H&&clearTimeout(H),H=null,I=``,L=0,x(U)?.fogEnabled===!1&&await it()}catch(e){i(V,e instanceof Error?e.message:`Reset failed`,!0)}finally{i(z,!1)}}}ne(()=>{function e(e){if(ot(e.target))return;if(e.key.toLowerCase()===`r`){e.preventDefault(),ut();return}let t=at(e.key);t&&(x(U)?.gameOver||x(U)?.victory||(e.preventDefault(),lt(t)))}return window.addEventListener(`keydown`,e),()=>{window.removeEventListener(`keydown`,e),H&&clearTimeout(H)}});function dt(e){return e.type===`road`&&me(e)?`text-orange-500 font-bold`:``}function ft(e){switch(e){case`home`:return`bg-red-500 border-red-200`;case`park`:return`bg-emerald-500 border-emerald-200`;case`supercharger`:return`bg-yellow-400 border-yellow-200`;case`water`:return`bg-blue-400 border-blue-200`;case`building`:return`bg-slate-700 border-slate-600`;default:return`bg-white border-gray-50`}}o(()=>{if(x(B))return;if(x(Q)){i(N,null),i(P,new Set,!0),i(F,!1),I=``,L=0;return}let e=x(U)?.currentMoves?.filter(e=>e.success)??[];if(!x(U)||e.length===0){i(N,null),i(P,new Set,!0),i(F,!1),I=``,L=0;return}let t=e.map(e=>`${e.fromPosition.x},${e.fromPosition.y}>${e.toPosition.x},${e.toPosition.y}`).join(`|`);if(t===I)return;let n=I,r=L;if(I=t,L=e.length,!(n!==``&&t.startsWith(`${n}|`))){i(N,null),i(P,new Set,!0),i(F,!1);return}let a=e.slice(r);if(a.length===0)return;let o=e.slice(0,r),s=!1,c=0,l=new Set;for(let e of o)l.add(`${e.fromPosition.x},${e.fromPosition.y}`),l.add(`${e.toPosition.x},${e.toPosition.y}`);i(F,!0),i(N,a[0].fromPosition,!0),l.add(`${a[0].fromPosition.x},${a[0].fromPosition.y}`),i(P,new Set(l),!0);let u=()=>{if(s)return;let e=a[c];if(!e){i(N,x(U).playerPos,!0),i(F,!1);return}l.add(`${e.fromPosition.x},${e.fromPosition.y}`),l.add(`${e.toPosition.x},${e.toPosition.y}`),i(P,new Set(l),!0),i(N,e.toPosition,!0),c+=1,setTimeout(u,140)},d=setTimeout(u,180);return()=>{s=!0,clearTimeout(d)}});let pt=s(()=>{let e=new Set;if(!x(U))return e;if(x(F))return x(P);for(let t of x(U).currentMoves??[])t.success&&(e.add(`${t.fromPosition.x},${t.fromPosition.y}`),e.add(`${t.toPosition.x},${t.toPosition.y}`));return e});function mt(e,t){if(!x(U))return null;let n=x(U).playerPos,r=x(U).fogRadius>0?x(U).fogRadius:1,i=n.x-r,a=n.y-r,o=e-i,s=t-a;return s<0||s>=x(U).nearbyGrid.length||o<0||o>=(x(U).nearbyGrid[s]?.length??0)?null:{ix:o,iy:s}}function ht(e,t){if(x(X))return x(W)?.[t]?.[e]??null;if(x(Q)){let n=mt(e,t);return!n||!x(U)?null:x(U).nearbyGrid[n.iy]?.[n.ix]??null}return x(Z)[t]?.[e]??null}function gt(e,t){if(!x(Je))return!1;if(x(X))return e===x(Je).x&&t===x(Je).y;if(x(Q)&&x(U)?.playerPos)return e===x(U).playerPos.x&&t===x(U).playerPos.y;let n=Math.floor((x(Z).length-1)/2);return e===(x(Z)[n]?Math.floor((x(Z)[n].length-1)/2):0)&&t===n}function _t(e,t){return x(X)||x(Q)?x(pt).has(`${e},${t}`):!1}var vt=Le();y(`1oiicp0`,t=>{l(()=>{e.title=`${x(A)??k??``} — Tesla Road Trip`})});var yt=p(vt),bt=p(yt),xt=p(bt),St=p(xt),Ct=p(St),wt=p(Ct),Tt=f(p(wt),2),Et=e=>{var t=_e(),n=f(a(t),2),i=p(n,!0);b(n);var o=f(n,2),s=e=>{var t=ge(),n=p(t);b(t),r(()=>E(n,`🌫 Fog r${x(U).fogRadius??``}`)),u(e,t)};_(o,e=>{x(U).fogEnabled&&e(s)}),r(()=>E(i,x(U).mapName)),u(e,t)};_(Tt,e=>{x(U)&&e(Et)}),b(wt);var Dt=f(wt,2),Ot=p(Dt),kt=e=>{var n=ve(),i=a(n),o=p(i,!0);b(i);var s=f(i,2),c=p(s);b(s),r(()=>{E(o,x(A)),E(c,`(${k??``})`)}),t(`click`,s,()=>navigator.clipboard.writeText(k)),u(e,n)},At=e=>{var n=ye(),i=p(n,!0);b(n),r(()=>E(i,k)),t(`click`,n,()=>navigator.clipboard.writeText(k)),u(e,n)};_(Ot,e=>{x(A)?e(kt):e(At,-1)}),b(Dt),b(Ct);var jt=f(Ct,2),Mt=e=>{var t=be(),n=a(t),i=p(n),o=f(p(i)),s=p(o);b(o),b(i);var c=f(i,2),l=p(c);b(c),b(n);var d=f(n,2),m=p(d),h=p(m),g=p(h,!0);b(h),C(2),b(m);var _=f(m,2),ee=p(_),te=p(ee,!0);b(ee),C(2),b(_);var v=f(_,2),y=p(v),ne=p(y,!0);b(y),C(2),b(v);var ie=f(v,2),w=p(ie),ae=p(w,!0);b(w);var T=f(w,2),oe=p(T,!0);b(T),b(ie),b(d),r(e=>{E(s,`${x(U).battery??``}/${x(U).maxBattery??``}`),S(l,1,`h-full rounded-full transition-all duration-300 ${x(U).battery/x(U).maxBattery>.5?`bg-green-400`:x(U).battery/x(U).maxBattery>.25?`bg-orange-400`:`bg-red-400`}`),re(l,`width: ${e??``}%`),E(g,x(U).score),E(te,x(U).totalMoves),E(ne,x(U).resetCount),S(w,1,`text-lg leading-tight ${x(U).victory?`text-green-500`:x(U).gameOver?`text-red-500`:`text-gray-300`}`),E(ae,x(U).victory?`🏆`:x(U).gameOver?`💥`:`🟢`),E(oe,x(U).victory?`Won`:x(U).gameOver?`Crashed`:`Active`)},[()=>Math.max(0,x(U).battery/x(U).maxBattery*100)]),u(e,t)},Nt=e=>{u(e,xe())};_(jt,e=>{x(U)?e(Mt):e(Nt,-1)}),b(St),b(xt);var Pt=f(xt,2),Ft=p(Pt),It=e=>{var t=Ee(),n=p(t);ae(n,21,()=>x(Xe),oe,(e,t)=>{var n=Te();ae(n,21,()=>x(Xe),oe,(e,n)=>{let i=s(()=>ht(x(n),x(t))),a=s(()=>gt(x(n),x(t))),o=s(()=>_t(x(n),x(t)));var c=we(),l=p(c),f=e=>{var t=d();r(()=>E(t,x(U)?.victory?`🚗`:x(U)?.gameOver?`💥`:`🚗`)),u(e,t)},m=e=>{u(e,Se())},h=e=>{var t=Ce(),n=p(t,!0);b(t),r((e,r)=>{S(t,1,e,`svelte-1oiicp0`),E(n,r)},[()=>`leading-none ${dt(x(i))}`,()=>he(x(i).allowedDirections)]),u(e,t)},g=s(()=>x(i)&&me(x(i)));_(l,e=>{x(a)?e(f):x(o)?e(m,1):x(g)&&e(h,2)}),b(c),r(e=>S(c,1,`game-cell text-center border transition-colors
										${e??``}
										${x(o)&&!x(a)?`ring-2 ring-inset ring-sky-300`:``}
										${x(i)?.visited&&!x(a)?`opacity-60`:``}`,`svelte-1oiicp0`),[()=>x(i)?ft(x(i).type):`bg-slate-300 border-slate-300`]),u(e,c)}),b(n),u(e,n)}),b(n),b(t),r(()=>re(t,`--grid-size: ${x(Ye)}; --board-width: ${x(Ze)?`92vw`:`60vw`}`)),u(e,t)},Lt=e=>{var t=De(),n=p(t),i=f(p(n),2),a=f(p(i)),o=p(a,!0);b(a),C(),b(i),b(n),b(t),r(()=>E(o,k)),u(e,t)},Rt=e=>{var t=Oe(),n=p(t),i=p(n);b(n),b(t),r(()=>E(i,`Session not found: ${x(Ge)??``}`)),u(e,t)},zt=e=>{var t=ke(),n=p(t),i=f(p(n),2),a=f(p(i)),o=p(a,!0);b(a),C(),b(i),C(2),b(n),b(t),r(()=>E(o,k)),u(e,t)};_(Ft,e=>{x(Ye)>0?e(It):x(We)?e(Lt,1):x(Ge)?e(Rt,2):e(zt,-1)}),b(Pt);var Bt=f(Pt,4),Vt=e=>{var n=Ie(),a=p(n),o=p(a),s=p(o),c=f(p(s),2),l=p(c,!0);b(c),b(s);var d=f(s,4),m=p(d),h=p(m,!0);b(m),C(2),b(d);var g=f(d,2),te=e=>{var t=Ae(),n=p(t);b(t),r(e=>E(n,`Last manual move: ${e??``}`),[()=>x(Ke).toLowerCase()]),u(e,t)};_(g,e=>{x(Ke)&&!x(V)&&e(te)});var v=f(g,2),y=e=>{var t=je(),n=p(t,!0);b(t),r(()=>E(n,x(V))),u(e,t)};_(v,e=>{x(V)&&e(y)}),C(2),b(o);var ne=f(o,2),re=e=>{var n=Fe(),a=f(p(n),2),o=p(a);b(a);var s=f(a,2),c=p(s);ee(c);var l=f(c,2),d=p(l,!0);b(l);var m=f(l,2),h=e=>{var n=Me();t(`click`,n,rt),u(e,n)};_(m,e=>{x(X)&&e(h)}),b(s);var g=f(s,2),te=e=>{u(e,Ne())};_(g,e=>{x(K)&&x(X)&&e(te)});var v=f(g,2),y=e=>{var t=Pe(),n=p(t,!0);b(t),r(()=>E(n,x(q))),u(e,t)};_(v,e=>{x(q)&&e(y)}),b(n),r(e=>{E(o,`This session hides the map beyond ${x(U).fogRadius??``} cell${x(U).fogRadius===1?``:`s`} of the car. Enter the password chosen at creation to reveal the full map.`),l.disabled=e,E(d,x(J)?`Unlocking…`:`Unlock full grid`)},[()=>x(J)||!x(G).trim()]),ie(c,()=>x(G),e=>i(G,e)),t(`click`,l,nt),u(e,n)};_(ne,e=>{x(U)?.fogEnabled&&e(re)}),b(a),b(n),r(()=>{S(c,1,`text-[11px] ${x(z)?`text-orange-500`:x(U).gameOver||x(U).victory?`text-gray-300`:x(R)?`text-blue-500`:`text-green-500`}`),E(l,x(z)?`resetting…`:x(U).gameOver||x(U).victory?`disabled`:x(R)?`moving…`:`ready`),m.disabled=x(R)||x(z),E(h,x(z)?`Resetting…`:`Reset session`)}),t(`click`,m,ut),u(e,n)};_(Bt,e=>{x(U)&&e(Vt)}),b(bt);var Ht=f(bt,2),Ut=p(Ht),Wt=p(Ut),Gt=f(p(Wt),2),Kt=f(p(Gt),2),qt=p(Kt,!0);b(Kt),C(),b(Gt),b(Wt);var $=f(Wt,2),Jt=p($,!0);b($),b(Ut);var Yt=f(Ut,2);se(Yt),b(Ht),b(yt),b(vt),r(()=>{S(yt,1,`grid grid-cols-1 gap-4 xl:gap-6 items-start ${x(Ze)?``:`lg:grid-cols-[minmax(0,3fr)_minmax(24rem,2fr)]`}`,`svelte-1oiicp0`),S(Ht,1,`min-w-0 bg-white rounded-2xl border border-[#e8e8e8] p-4 shadow-sm ${x(Ze)?``:`lg:sticky lg:top-4`}`,`svelte-1oiicp0`),E(qt,k),S($,1,`text-sm px-4 py-2 rounded-full border transition-colors shrink-0 ${x(Qe)?`bg-green-50 border-green-200 text-green-600`:`border-blue-300 text-blue-700 hover:bg-blue-50`}`),E(Jt,x(Qe)?`Copied!`:`Copy`),te(Yt,x($e))}),t(`click`,$,et),t(`click`,Yt,e=>e.target.select()),u(n,vt),v(),Ve()}n([`click`]);export{Re as component};