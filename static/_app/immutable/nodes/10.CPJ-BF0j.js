import{B as e,D as t,E as n,F as r,G as i,H as a,I as o,J as s,K as c,N as l,S as u,T as d,U as f,V as p,W as m,X as h,Y as g,b as _,c as v,d as ee,et as y,h as b,i as x,it as S,k as C,m as w,o as T,p as te,rt as E,s as D,tt as ne,v as O,w as k,x as A,y as j,z as re}from"../chunks/BZbOjNpy.js";import"../chunks/I8rQr1kY.js";import{n as ie,s as ae,t as oe}from"../chunks/Bq6-sIW7.js";import{t as se}from"../chunks/DuFLOf5Q.js";import{t as ce}from"../chunks/DlMPkuA-.js";import{i as le,t as ue}from"../chunks/CTe5u0vL.js";var de=k(`<div class="flex items-center gap-2 flex-1"><span class="text-xs text-gray-400">Radius</span> <input type="range" min="1" max="10" class="flex-1 h-1 accent-[#393c41]"/> <span class="text-xs text-gray-500 w-4 text-right"> </span></div> <p class="hidden text-xs text-gray-400 italic">Fog is your view only — not the AI's</p>`,1),fe=k(`<div class="mt-4 bg-white rounded-xl border border-[#e8e8e8] px-4 py-3 flex items-center gap-4"><label class="flex items-center gap-2 cursor-pointer select-none"><input type="checkbox" class="sr-only peer"/> <div class="w-8 h-4 bg-gray-200 peer-checked:bg-[#393c41] rounded-full relative transition-colors"><span></span></div> <span class="text-xs text-gray-500">Fog Mode</span></label> <!></div>`);function pe(e,t){ne(t,!0);let n=x(t,`enabled`,15,!1),i=x(t,`radius`,15,3);o(()=>{let e=localStorage.getItem(`caveMode`);if(e){let t=JSON.parse(e);n(t.enabled??!0),i(t.radius??3)}}),o(()=>{localStorage.setItem(`caveMode`,JSON.stringify({enabled:n(),radius:i()}))});var s=fe(),c=p(s),l=p(c);v(l);var d=f(l,2),m=p(d);S(d),E(2),S(c);var h=f(c,2),g=e=>{var t=de(),n=a(t),o=f(p(n),2);v(o);var s=f(o,2),c=p(s,!0);S(s),S(n),E(2),r(()=>A(c,i())),D(o,i),u(e,t)};_(h,e=>{n()&&e(g)}),S(s),r(()=>w(m,1,`absolute top-0.5 left-0.5 w-3 h-3 bg-white rounded-full shadow transition-transform peer-checked:translate-x-4 ${n()?`translate-x-4`:``}`)),T(l,n),u(e,s),y()}var me=k(`<span>·</span> <span class="normal-case tracking-normal font-mono"> </span>`,1),he=k(`<span class="text-lg leading-none text-gray-800 font-medium"> </span> <button class="font-mono text-sm leading-none text-gray-400 hover:text-blue-600 transition-colors" title="Copy session ID"> </button>`,1),ge=k(`<button class="font-mono text-lg leading-none text-gray-800 hover:text-blue-600 transition-colors" title="Copy session ID"> </button>`),_e=k(`<div class="flex-1 min-w-[16rem] max-w-xl"><div class="flex justify-between text-xs text-gray-400 mb-1"><span>Battery</span><span> </span></div> <div class="h-2 bg-gray-100 rounded-full overflow-hidden"><div></div></div></div> <div class="grid grid-cols-3 gap-2 text-center"><div class="bg-gray-50 rounded-xl px-4 py-2"><div class="text-xl font-light leading-tight"> </div> <div class="text-[11px] text-gray-400">Parks</div></div> <div class="bg-gray-50 rounded-xl px-4 py-2"><div class="text-xl font-light leading-tight"> </div> <div class="text-[11px] text-gray-400">Moves</div></div> <div class="bg-gray-50 rounded-xl px-4 py-2"><div> </div> <div class="text-[11px] text-gray-400"> </div></div></div>`,1),ve=k(`<p class="text-sm text-gray-400 font-light">Loading…</p>`),ye=k(`<span class="text-sky-500 leading-none">•</span>`),be=k(`<span> </span>`),xe=k(`<td><!></td>`),Se=k(`<tr></tr>`),Ce=k(`<table class="game-board border-collapse svelte-1oiicp0"><tbody></tbody></table>`),we=k(`<div class="flex items-center justify-center h-64 text-gray-400"><div class="text-center"><span class="text-4xl block mb-3">🚗</span> <p class="text-sm font-light">Loading <code class="font-mono"> </code>…</p></div></div>`),Te=k(`<div class="flex items-center justify-center h-64 text-red-400"><p class="text-sm"> </p></div>`),Ee=k(`<div class="flex items-center justify-center h-64 text-gray-400"><div class="text-center"><span class="text-4xl block mb-3">🚗</span> <p class="text-sm font-light">Waiting for moves on <code class="font-mono"> </code>…</p> <p class="text-xs mt-2">Point an AI at this session to see it play</p></div></div>`),De=k(`<p class="text-xs text-gray-600 font-light leading-snug"> </p>`),Oe=k(`<p class="text-xs text-gray-400 italic">Waiting for moves…</p>`),ke=k(`<div class="grid grid-cols-1 xl:grid-cols-[minmax(0,1fr)_minmax(18rem,24rem)] gap-4 border-t border-gray-100 bg-gray-50/40 p-3 sm:p-4"><div class="rounded-xl bg-white border border-gray-100 px-4 py-3"><div class="flex items-center justify-between gap-3 mb-2"><span class="text-[11px] uppercase tracking-widest text-gray-400">Event log</span> <span class="text-[11px] text-gray-300">latest first</span></div> <div class="space-y-1 max-h-28 overflow-y-auto"></div></div> <div><span class="text-[11px] uppercase tracking-widest text-gray-400 px-1">View options</span> <!> <p class="text-xs text-gray-400 mt-2 px-1">Fog Mode is viewer-only and does not affect the AI.</p></div></div>`),Ae=k(`<div class="max-w-[1900px] mx-auto px-3 sm:px-4 py-4 lg:py-6"><div><section class="min-w-0 bg-white rounded-2xl border border-[#e8e8e8] shadow-sm overflow-visible"><div class="p-3 sm:p-4 border-b border-gray-100"><div class="flex flex-wrap items-start justify-between gap-3"><div class="min-w-0"><div class="flex flex-wrap items-center gap-2 text-xs uppercase tracking-widest text-gray-400"><span>Session</span> <!></div> <div class="mt-1 flex items-center gap-2"><!></div></div> <!></div></div> <div class="p-3 sm:p-4 flex items-start justify-start board-pane svelte-1oiicp0"><!></div> <div class="flex flex-wrap items-center justify-between gap-x-4 gap-y-2 text-xs text-gray-400 px-4 pb-3"><div class="flex flex-wrap items-center gap-x-4 gap-y-1"><span><span class="text-sky-400">•</span> </span> <span class="flex items-center gap-x-3 gap-y-1 flex-wrap"><span class="flex items-center gap-1"><span class="inline-block w-3 h-3 rounded-sm bg-red-500"></span> Home</span> <span class="flex items-center gap-1"><span class="inline-block w-3 h-3 rounded-sm bg-emerald-500"></span> Park</span> <span class="flex items-center gap-1"><span class="inline-block w-3 h-3 rounded-sm bg-yellow-400"></span> Charger</span> <span class="flex items-center gap-1"><span class="inline-block w-3 h-3 rounded-sm bg-slate-700"></span> Blocked</span> <span class="flex items-center gap-1"><span class="inline-block w-3 h-3 rounded-sm bg-blue-400"></span> Water</span></span></div> <a href="/lobby" class="hover:text-gray-600 transition-colors">← Back to sessions</a></div> <!></section> <aside><div class="flex items-start justify-between gap-3 mb-3"><div class="min-w-0"><span class="text-xs uppercase tracking-widest text-gray-400">Prompt for LLM</span> <div class="mt-2 rounded-xl border border-blue-100 bg-blue-50 px-3 py-2 text-sm text-blue-900"><strong class="font-medium">Copy this into an AI chat</strong> to control session <code class="font-mono font-semibold"> </code>.</div></div> <button> </button></div> <textarea readonly="" class="w-full text-sm font-mono text-gray-700 bg-gray-50 rounded-xl p-4 resize-none prompt-pane focus:outline-none leading-relaxed border border-gray-100 svelte-1oiicp0"></textarea></aside></div></div>`);function M(n,v){ne(v,!0);let x=()=>h(ce,`$page`,k),T=()=>h(je,`$sessionQuery`,k),D=()=>h(Me,`$initialQuery`,k),[k,de]=g(),fe=oe(),M=x().params.id??``,je=ie({client:fe,query:ae(`
		query Session($id: ID!) {
			session(id: $id) { id displayName mapName }
		}
	`),variables:{id:M}}),N=s(()=>T().data?.session?.displayName??null),Me=ie({client:fe,query:ae(`
		query GameState($sessionID: ID!) {
			gameState(sessionID: $sessionID) {
				battery maxBattery score victory gameOver totalMoves message mapName
				playerPos { x y }
				grid { type visited id allowedDirections }
				currentMoves { fromPosition { x y } toPosition { x y } success }
			}
		}
	`),variables:{sessionID:M}}),Ne=c(null),P=c(m([])),F=c(null),I=c(m(new Set)),L=c(!1),R=``,z=s(()=>C(Ne)??D().data?.gameState??null),B=s(()=>C(F)??C(z)?.playerPos??null),V=s(()=>(C(z)?.grid.length??0)>=30);o(()=>{let e=se({url:typeof window<`u`?`${window.location.protocol===`https:`?`wss`:`ws`}://${window.location.host}/graphql`:`ws://localhost:8080/graphql`}).subscribe({query:`
		subscription SessionUpdated($sessionID: ID!) {
			sessionUpdated(sessionID: $sessionID) {
				battery maxBattery score victory gameOver totalMoves message mapName
				playerPos { x y }
				grid { type visited id allowedDirections }
				currentMoves { fromPosition { x y } toPosition { x y } success }
			}
		}
	`,variables:{sessionID:M}},{next(e){let t=e.data?.sessionUpdated;t&&(i(Ne,t,!0),t.message&&i(P,[t.message,...C(P)].slice(0,10),!0))},error(e){console.error(`WS error`,e)},complete(){}});return()=>e()});let H=c(!1),U=c(3),W=c(!1),Pe=s(()=>`Use this GraphQL API to control an existing Tesla Road Trip game session.

Goal: visit all parks without getting stranded or hitting a building.

Session ID: ${M}
GraphQL endpoint: ${typeof window<`u`?window.location.origin:``}/graphql
Playground: ${typeof window<`u`?window.location.origin:``}/playground
MCP endpoint: http://tesla.wricardo.net/mcp (Streamable HTTP transport)

To use MCP in Claude Code, run:
claude mcp add --transport http tesla-game http://tesla.wricardo.net/mcp

GraphQL introspection is enabled. Use the Playground Docs panel or query __schema/__type to discover fields before constructing operations.

## Inspect the API
query {
  __type(name: "GameState") {
    fields { name type { kind name ofType { kind name } } }
  }
}

## Read current session state
query {
  gameState(sessionID: "${M}") {
    mapName
    playerPos { x y }
    battery
    maxBattery
    score
    victory
    gameOver
    message
    localView3x3
    grid { type visited id allowedDirections }
    visitedParks { id visited }
  }
}

## Send one move
mutation {
  move(sessionID: "${M}", direction: RIGHT) {
    success
    message
    attemptedTo { x y tileChar tileType passable }
    gameState { playerPos { x y } battery score victory gameOver }
  }
}

## Send a move sequence
mutation {
  bulkMove(sessionID: "${M}", moves: [UP, RIGHT, DOWN]) {
    success
    movesExecuted
    requestedMoves
    stoppedReason
    stopReasonCode
    truncated
    limit
    gameState { playerPos { x y } battery score victory gameOver }
  }
}

bulkMove accepts at most 50 moves per call. Check success, stoppedReason, stopReasonCode, truncated, gameOver, and victory before sending another operation.

## Manage this session
mutation { reset(sessionID: "${M}") { playerPos { x y } battery score victory gameOver } }
query { history(sessionID: "${M}", page: 1, limit: 20, order: DESC) { totalMoves moves { moveNumber action success battery } } }
mutation { deleteSession(id: "${M}") { message } }

Directions: UP DOWN LEFT RIGHT. Grid coordinates are grid[y][x].`);function Fe(){navigator.clipboard.writeText(C(Pe)),i(W,!0),setTimeout(()=>i(W,!1),2e3)}function Ie(e){return e.type===`road`&&le(e)?`text-orange-500 font-bold`:``}function Le(e){switch(e){case`home`:return`bg-red-500 border-red-200`;case`park`:return`bg-emerald-500 border-emerald-200`;case`supercharger`:return`bg-yellow-400 border-yellow-200`;case`water`:return`bg-blue-400 border-blue-200`;case`building`:return`bg-slate-700 border-slate-600`;default:return`bg-white border-gray-50`}}o(()=>{let e=C(z)?.currentMoves?.filter(e=>e.success)??[];if(!C(z)||e.length===0){i(F,null),i(I,new Set,!0),i(L,!1),R=``;return}let t=e.map(e=>`${e.fromPosition.x},${e.fromPosition.y}>${e.toPosition.x},${e.toPosition.y}`).join(`|`);if(t===R)return;R=t;let n=!1,r=0,a=new Set;i(L,!0),i(F,e[0].fromPosition,!0),a.add(`${e[0].fromPosition.x},${e[0].fromPosition.y}`),i(I,new Set(a),!0);let o=()=>{if(n)return;let t=e[r];if(!t){i(F,C(z).playerPos,!0),i(L,!1);return}a.add(`${t.fromPosition.x},${t.fromPosition.y}`),a.add(`${t.toPosition.x},${t.toPosition.y}`),i(I,new Set(a),!0),i(F,t.toPosition,!0),r+=1,setTimeout(o,140)},s=setTimeout(o,180);return()=>{n=!0,clearTimeout(s)}});let Re=s(()=>{let e=new Set;if(!C(z))return e;if(C(L))return C(I);for(let t of C(z).currentMoves??[])t.success&&(e.add(`${t.fromPosition.x},${t.fromPosition.y}`),e.add(`${t.toPosition.x},${t.toPosition.y}`));return e});function ze(e,t){return!C(H)||!C(B)?!0:Math.max(Math.abs(e-C(B).x),Math.abs(t-C(B).y))<=C(U)}var G=Ae();b(`1oiicp0`,t=>{l(()=>{e.title=`${C(N)??M??``} — Tesla Road Trip`})});var Be=p(G),K=p(Be),q=p(K),Ve=p(q),J=p(Ve),Y=p(J),He=f(p(Y),2),Ue=e=>{var t=me(),n=f(a(t),2),i=p(n,!0);S(n),r(()=>A(i,C(z).mapName)),u(e,t)};_(He,e=>{C(z)&&e(Ue)}),S(Y);var We=f(Y,2),Ge=p(We),Ke=e=>{var n=he(),i=a(n),o=p(i,!0);S(i);var s=f(i,2),c=p(s);S(s),r(()=>{A(o,C(N)),A(c,`(${M??``})`)}),t(`click`,s,()=>navigator.clipboard.writeText(M)),u(e,n)},qe=e=>{var n=ge(),i=p(n,!0);S(n),r(()=>A(i,M)),t(`click`,n,()=>navigator.clipboard.writeText(M)),u(e,n)};_(Ge,e=>{C(N)?e(Ke):e(qe,-1)}),S(We),S(J);var Je=f(J,2),Ye=e=>{var t=_e(),n=a(t),i=p(n),o=f(p(i)),s=p(o);S(o),S(i);var c=f(i,2),l=p(c);S(c),S(n);var d=f(n,2),m=p(d),h=p(m),g=p(h,!0);S(h),E(2),S(m);var _=f(m,2),v=p(_),ee=p(v,!0);S(v),E(2),S(_);var y=f(_,2),b=p(y),x=p(b,!0);S(b);var T=f(b,2),D=p(T,!0);S(T),S(y),S(d),r(e=>{A(s,`${C(z).battery??``}/${C(z).maxBattery??``}`),w(l,1,`h-full rounded-full transition-all duration-300 ${C(z).battery/C(z).maxBattery>.5?`bg-green-400`:C(z).battery/C(z).maxBattery>.25?`bg-orange-400`:`bg-red-400`}`),te(l,`width: ${e??``}%`),A(g,C(z).score),A(ee,C(z).totalMoves),w(b,1,`text-lg leading-tight ${C(z).victory?`text-green-500`:C(z).gameOver?`text-red-500`:`text-gray-300`}`),A(x,C(z).victory?`🏆`:C(z).gameOver?`💥`:`🟢`),A(D,C(z).victory?`Won`:C(z).gameOver?`Crashed`:`Active`)},[()=>Math.max(0,C(z).battery/C(z).maxBattery*100)]),u(e,t)},Xe=e=>{u(e,ve())};_(Je,e=>{C(z)?e(Ye):e(Xe,-1)}),S(Ve),S(q);var X=f(q,2),Ze=p(X),Qe=e=>{var t=Ce(),n=p(t);O(n,21,()=>C(z).grid,j,(e,t,n)=>{var i=Se();O(i,21,()=>C(t),j,(e,t,i)=>{let a=s(()=>ze(i,n)),o=s(()=>C(B)&&i===C(B).x&&n===C(B).y),c=s(()=>C(a)&&C(Re).has(`${i},${n}`));var l=xe(),f=p(l),m=e=>{var t=d();r(()=>A(t,C(z).victory?`🚗`:C(z).gameOver?`💥`:`🚗`)),u(e,t)},h=e=>{u(e,ye())},g=e=>{var n=be(),i=p(n,!0);S(n),r((e,t)=>{w(n,1,e,`svelte-1oiicp0`),A(i,t)},[()=>`leading-none ${Ie(C(t))}`,()=>ue(C(t).allowedDirections)]),u(e,n)},v=s(()=>C(a)&&le(C(t)));_(f,e=>{C(o)?e(m):C(c)?e(h,1):C(v)&&e(g,2)}),S(l),r(e=>w(l,1,`game-cell text-center border transition-colors
										${e??``}
										${C(a)&&C(c)&&!C(o)?`ring-2 ring-inset ring-sky-300`:``}
										${C(a)&&C(t).visited&&!C(o)?`opacity-60`:``}`,`svelte-1oiicp0`),[()=>C(a)?Le(C(t).type):`bg-slate-800 border-slate-700`]),u(e,l)}),S(i),u(e,i)}),S(n),S(t),r(()=>te(t,`--grid-size: ${C(z).grid.length}; --board-width: ${C(V)?`92vw`:`60vw`}`)),u(e,t)},$e=e=>{var t=we(),n=p(t),i=f(p(n),2),a=f(p(i)),o=p(a,!0);S(a),E(),S(i),S(n),S(t),r(()=>A(o,M)),u(e,t)},et=e=>{var t=Te(),n=p(t),i=p(n);S(n),S(t),r(()=>A(i,`Session not found: ${D().error.message??``}`)),u(e,t)},tt=e=>{var t=Ee(),n=p(t),i=f(p(n),2),a=f(p(i)),o=p(a,!0);S(a),E(),S(i),E(2),S(n),S(t),r(()=>A(o,M)),u(e,t)};_(Ze,e=>{C(z)?.grid?e(Qe):D().fetching?e($e,1):D().error?e(et,2):e(tt,-1)}),S(X);var Z=f(X,2),nt=p(Z),rt=p(nt),it=f(p(rt));S(rt),E(2),S(nt),E(2),S(Z);var at=f(Z,2),ot=e=>{var t=ke(),n=p(t),a=f(p(n),2);O(a,21,()=>C(P),j,(e,t)=>{var n=De(),i=p(n,!0);S(n),r(()=>A(i,C(t))),u(e,n)},e=>{u(e,Oe())}),S(a),S(n);var o=f(n,2);pe(f(p(o),2),{get enabled(){return C(H)},set enabled(e){i(H,e,!0)},get radius(){return C(U)},set radius(e){i(U,e,!0)}}),E(2),S(o),S(t),u(e,t)};_(at,e=>{C(z)&&e(ot)}),S(K);var Q=f(K,2),st=p(Q),ct=p(st),lt=f(p(ct),2),ut=f(p(lt),2),dt=p(ut,!0);S(ut),E(),S(lt),S(ct);var $=f(ct,2),ft=p($,!0);S($),S(st);var pt=f(st,2);re(pt),S(Q),S(Be),S(G),r(()=>{w(Be,1,`grid grid-cols-1 gap-4 xl:gap-6 items-start ${C(V)?``:`lg:grid-cols-[minmax(0,3fr)_minmax(24rem,2fr)]`}`,`svelte-1oiicp0`),A(it,` movement trail${C(L)?` · animating route…`:``}`),w(Q,1,`min-w-0 bg-white rounded-2xl border border-[#e8e8e8] p-4 shadow-sm ${C(V)?``:`lg:sticky lg:top-4`}`,`svelte-1oiicp0`),A(dt,M),w($,1,`text-sm px-4 py-2 rounded-full border transition-colors shrink-0 ${C(W)?`bg-green-50 border-green-200 text-green-600`:`border-blue-300 text-blue-700 hover:bg-blue-50`}`),A(ft,C(W)?`Copied!`:`Copy`),ee(pt,C(Pe))}),t(`click`,$,Fe),t(`click`,pt,e=>e.target.select()),u(n,G),y(),de()}n([`click`]);export{M as component};