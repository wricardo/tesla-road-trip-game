import{$ as e,A as t,C as n,D as r,E as i,G as a,I as o,K as s,M as c,Q as l,R as u,S as d,T as f,U as p,X as m,Y as h,at as g,ct as _,et as v,it as y,k as b,q as x,st as S,u as C,v as w,w as T,x as ee,z as te}from"../chunks/D1OJptGk.js";import{t as E}from"../chunks/B-KqxgKf.js";import"../chunks/CP97kCR3.js";import{n as D,s as ne,t as O}from"../chunks/Du21IcFO.js";import{t as re}from"../chunks/C8IhN0cf.js";import"../chunks/Cjb4Od71.js";import{n as k}from"../chunks/DqFkiEfT.js";import{i as A,t as j}from"../chunks/C-hNJNUn.js";import{n as ie,r as M,t as ae}from"../chunks/B8fn68uT.js";import{t as N}from"../chunks/D05INHKf.js";import{t as oe}from"../chunks/DcSGOyYD.js";var se=class{#e;#t;#n=m(null);get map(){return c(this.#n)}set map(e){h(this.#n,e,!0)}#r=m(!1);get starting(){return c(this.#r)}set starting(e){h(this.#r,e,!0)}#i=m(``);get error(){return c(this.#i)}set error(e){h(this.#i,e,!0)}constructor(e,t=e=>void E(`/watch/${e}`)){this.#e=e,this.#t=t}open(e){this.starting||(this.error=``,this.map=e)}cancel(){this.starting||(this.map=null,this.error=``)}async begin(e,t){let n=this.map;if(!(!n||this.starting)){this.starting=!0,this.error=``;try{let r=await N(this.#e,{mapID:n.mapId,name:t,fogEnabled:e.fogEnabled,fogRadius:e.fogRadius,gridPassword:e.gridPassword,moveDelayMs:e.moveDelayMs});this.#t(r)}catch(e){this.error=e instanceof Error?e.message:`Could not create the session`}finally{this.starting=!1}}}};function ce(){return typeof window<`u`?`${window.location.origin}/graphql`:`/graphql`}function P({map:e,editKey:t,apiUrl:n}){let r=e.name||e.mapId,i=t?`The map's edit key (password) is:

    ${t}`:`I do not have the edit key for this map, so you will need to ask me for it before you can save.`,a=t?"Save the result with updateMap, passing the edit key above as `editKey`.":"Once I give you the edit key, save with updateMap passing it as `editKey`.";return`I want to change a map in the Tesla Road Trip game. Please help me edit it with GraphQL.

GRAPHQL ENDPOINT
    ${n}

MAP
    id:     ${e.mapId}
    name:   ${r}
    grid:   ${e.gridSize}x${e.gridSize}
    battery: max ${e.maxBattery}

${i}

WHAT I WANT CHANGED
    <describe your change here — e.g. "move the park in the top-left to the middle",
     "add two superchargers near the home tile", "make the outer walls solid">

HOW TO DO IT
1. Read the current map first (this requires no credentials):

    query GetMap {
      map(name: "${e.mapId}", password: "tesla-ui-map") {
        name description gridSize maxBattery startingBattery
        layout
        legend { key value }
        cellConfigs { key type allowedDirections }
      }
    }

   Send it as POST ${n} with header "content-type: application/json" and a
   body of {"query": "...", "variables": {}}.

2. Change the layout. Rules the server enforces:
   - layout must have exactly ${e.gridSize} rows of ${e.gridSize} characters.
   - Characters: R road, H home, P park, S supercharger, W water, B building.
   - At least one H and at least one P are required.
   - \`legend\` and \`cellConfigs\` are LISTS of {key, value} / {key, type,
     allowedDirections} objects, both when read and when written. The query
     above returns them in exactly the shape updateMap accepts, so pass the
     read values straight back rather than rebuilding them.
   - Every park must be reachable from a home without running out of battery.
     Homes and superchargers refill the battery.
   - Keep \`legend\` with all six standard characters.

3. ${a}

    mutation UpdateMap($name: String!, $patch: GameMapPatchInput!, $editKey: String) {
      updateMap(name: $name, patch: $patch, editKey: $editKey) {
        name gridSize maxBattery
      }
    }

    variables:
    {
      "name": "${e.mapId}",
      "editKey": ${t?`"${t}"`:`"<ASK_ME>"`},
      "patch": {
        "description": "...",
        "layout": ["...", "..."],
        "legend": [{"key":"R","value":"road"}, ...],
        "cellConfigs": [],
        "startingBattery": ${e.maxBattery}
      }
    }

4. To check a candidate map before saving, call validateMap with the full map
   (same fields plus "name"). It needs no credentials and reports whether the
   map parses and is solvable.

Only the fields you put in "patch" change; omitted fields keep their current
values. Tell me what you are about to change before you save.`}var F=r(`<meta name="description"/>`),le=r(`<h1 class="text-3xl font-light text-[#171a20] mt-4">Map not found</h1> <p class="text-gray-500 mt-2">There is no map called <code class="text-[#393c41]"> </code>. It may have been renamed or deleted.</p> <a href="/maps" class="inline-block mt-6 bg-[#393c41] text-white text-sm px-5 py-2.5 rounded-full hover:bg-black transition-colors">Browse all maps</a>`,1),I=r(`<p class="text-lg text-gray-500 font-light max-w-2xl mt-3"> </p>`),L=r(`<div class="mt-4 flex flex-col lg:flex-row lg:items-end lg:justify-between gap-6"><div class="min-w-0"><p class="text-xs font-bold uppercase tracking-widest text-red-600 mb-3">🗺️ Map</p> <h1 class="text-3xl lg:text-5xl font-light text-[#171a20] tracking-tight"> </h1> <!></div> <div class="flex flex-wrap gap-3"><button type="button" class="bg-[#393c41] text-white text-sm px-6 py-3 rounded-full hover:bg-black transition-colors disabled:opacity-50"> </button> <button type="button" class="border border-gray-200 bg-white text-[#393c41] text-sm px-6 py-3 rounded-full hover:border-gray-400 transition-colors"> </button></div></div>`),ue=r(`<div class="grid gap-6 lg:grid-cols-[minmax(0,1fr)_20rem]"><div class="aspect-square w-full rounded-2xl bg-white border border-[#e8e8e8] animate-pulse"></div> <div class="h-64 rounded-2xl bg-white border border-[#e8e8e8] animate-pulse"></div></div>`),de=r(`<div class="aspect-square w-full rounded-xl bg-red-50 flex items-center justify-center text-sm text-red-700 p-6 text-center">The layout for this map could not be loaded.</div>`),fe=r(`<div class="rounded-xl bg-[#f7f7f7] px-3 py-2"><p class="text-[11px] text-gray-500"> </p> <p class="text-base font-medium text-[#171a20] tabular-nums"> </p></div>`),pe=r(`<div class="mt-4 grid grid-cols-2 gap-2"></div>`),me=r(`<p class="text-xs text-gray-500"> </p> <div class="mt-3 flex items-start gap-2"><code class="flex-1 min-w-0 break-all rounded-xl bg-[#f7f7f7] px-3 py-2 text-xs text-[#393c41] font-mono" data-testid="edit-key-value"> </code> <button type="button" class="shrink-0 border border-gray-200 bg-white text-[#393c41] text-xs px-3 py-2 rounded-full hover:border-gray-400 transition-colors"> </button></div>`,1),he=r(`<p class="text-xs text-gray-500"> </p>`),ge=r(`<div class="grid gap-6 lg:grid-cols-[minmax(0,1fr)_20rem]"><section class="bg-white rounded-2xl border border-[#e8e8e8] shadow-sm p-4 sm:p-6"><!></section> <aside class="space-y-6"><section class="bg-white rounded-2xl border border-[#e8e8e8] shadow-sm p-6"><h2 class="text-lg font-light text-[#393c41] mb-4">Details</h2> <dl class="space-y-2 text-sm"><div class="flex items-baseline justify-between gap-3"><dt class="text-gray-500">Grid</dt> <dd class="text-[#393c41] tabular-nums"> </dd></div> <div class="flex items-baseline justify-between gap-3"><dt class="text-gray-500">Battery</dt> <dd class="text-[#393c41] tabular-nums"> </dd></div> <div class="flex items-baseline justify-between gap-3"><dt class="text-gray-500">Map ID</dt> <dd class="text-[#393c41] font-mono text-xs"> </dd></div></dl> <!></section> <section class="bg-white rounded-2xl border border-[#e8e8e8] shadow-sm p-6"><h2 class="text-lg font-light text-[#393c41] mb-4">Legend</h2> <ul class="space-y-1.5 text-xs text-gray-600"><li class="flex items-center gap-2"><span class="h-4 w-4 rounded-sm bg-red-500"></span>Home — charge to full</li> <li class="flex items-center gap-2"><span class="h-4 w-4 rounded-sm bg-emerald-500"></span>Park — visit them all to win</li> <li class="flex items-center gap-2"><span class="h-4 w-4 rounded-sm bg-yellow-400"></span>Supercharger — partial charge</li> <li class="flex items-center gap-2"><span class="h-4 w-4 rounded-sm bg-blue-400"></span>Water — impassable</li> <li class="flex items-center gap-2"><span class="h-4 w-4 rounded-sm bg-slate-700"></span>Blocked — impassable</li></ul></section> <section class="bg-white rounded-2xl border border-[#e8e8e8] shadow-sm p-6"><h2 class="text-lg font-light text-[#393c41] mb-2">Share this map</h2> <p class="text-xs text-gray-500"> </p> <button type="button" class="mt-4 w-full border border-gray-200 bg-white text-[#393c41] text-sm px-4 py-2.5 rounded-full hover:border-gray-400 transition-colors"> </button></section> <section class="bg-white rounded-2xl border border-[#e8e8e8] shadow-sm p-6" data-testid="edit-key-section"><h2 class="text-lg font-light text-[#393c41] mb-2">Edit key</h2> <!></section> <section class="bg-white rounded-2xl border border-[#e8e8e8] shadow-sm p-6" data-testid="edit-with-ai-section"><h2 class="text-lg font-light text-[#393c41] mb-2">Edit with AI</h2> <p class="text-xs text-gray-500">Copy this prompt into an AI agent. It includes the GraphQL endpoint, this map's id and its edit key,
							so the agent can read the map, change it and save it back.</p> <button type="button" class="mt-4 w-full bg-[#393c41] text-white text-sm px-4 py-2.5 rounded-full hover:bg-black transition-colors"> </button> <details class="mt-3"><summary class="cursor-pointer text-xs text-gray-500 hover:text-[#393c41]">Preview prompt</summary> <pre class="mt-2 max-h-64 overflow-auto rounded-xl bg-[#f7f7f7] p-3 text-[11px] leading-relaxed text-[#393c41] whitespace-pre-wrap" data-testid="ai-prompt-preview"> </pre></details></section></aside></div>`),_e=r(`<div class="max-w-7xl mx-auto px-6 py-10"><!></div>`),ve=r(`<div class="bg-white border-b border-[#e8e8e8]"><div class="max-w-7xl mx-auto px-6 py-8"><a href="/maps" class="text-sm text-gray-500 hover:text-[#393c41]">← All maps</a> <!></div></div> <!> <!>`,1);function R(r,b){g(b,!0);let E=()=>v(re,`$page`,R),N=()=>v(be,`$mapsResult`,R),[R,ye]=e(),z=O(),be=D({client:z,query:ne(k)}),B=l(()=>decodeURIComponent(E().params.id??``)),V=l(()=>(N()?.data?.maps??[]).find(e=>e.mapId===c(B))??null),H=new se(z),U=m(void 0);te(()=>{let e=c(B);e&&(h(U,void 0),M(z,e).then(e=>h(U,e,!0)))});let W=l(()=>A(c(V)?.name||c(U)?.name||c(B))),G=l(()=>c(V)?.gridSize??c(U)?.gridSize??0),K=l(()=>c(V)?.maxBattery??c(U)?.maxBattery??0),xe=l(()=>c(U)?.startingBattery??0),q=l(()=>c(V)?.description||c(U)?.description||``),Se=l(()=>c(U)?[{label:`Parks`,value:j(c(U),`park`),icon:`🌳`},{label:`Chargers`,value:j(c(U),`supercharger`),icon:`⚡`},{label:`Homes`,value:j(c(U),`home`),icon:`⌂`},{label:`Obstacles`,value:j(c(U),`water`)+j(c(U),`building`),icon:`▨`}]:[]),J=l(()=>!N().fetching&&N().error===void 0&&c(U)===null&&c(V)===null),Y=l(()=>c(B)?oe(c(B)):null),Ce=l(()=>P({map:{mapId:c(B),name:c(W),gridSize:c(G),maxBattery:c(K)},editKey:c(Y),apiUrl:ce()})),X=m(`idle`);async function Z(e,t){try{await navigator.clipboard.writeText(e),h(X,t,!0)}catch{h(X,`failed`)}setTimeout(()=>h(X,`idle`),2500)}async function we(){await Z(E().url.href,`link`)}var Te=ve();w(`1izqlkc`,e=>{var t=i(),r=s(t),a=e=>{var t=F();u(()=>C(t,`content`,c(q))),f(e,t)};n(r,e=>{!c(J)&&c(q)&&e(a)}),o(()=>{p.title=c(J)?`Map not found`:`${c(W)} — Tesla Road Trip`}),f(e,t)});var Q=s(Te),Ee=a(Q),De=x(a(Ee),2),Oe=e=>{var t=le(),n=x(s(t),2),r=x(a(n)),i=a(r,!0);_(r),S(),_(n),S(2),u(()=>T(i,c(B))),f(e,t)},ke=e=>{var r=L(),i=a(r),o=x(a(i),2),s=a(o,!0);_(o);var l=x(o,2),d=e=>{var t=I(),n=a(t,!0);_(t),u(()=>T(n,c(q))),f(e,t)};n(l,e=>{c(q)&&e(d)}),_(i);var p=x(i,2),m=a(p),h=a(m,!0);_(m);var g=x(m,2),v=a(g,!0);_(g),_(p),_(r),u(()=>{T(s,c(W)),m.disabled=H.starting||c(U)===void 0,T(h,H.starting?`Starting…`:`Play this map`),T(v,c(X)===`link`?`Link copied ✓`:c(X)===`failed`?`Copy failed`:`Copy link`)}),t(`click`,m,()=>H.open({mapId:c(B),name:c(W),description:c(q),gridSize:c(G),maxBattery:c(K)})),t(`click`,g,we),f(e,r)};n(De,e=>{c(J)?e(Oe):e(ke,-1)}),_(Ee),_(Q);var $=x(Q,2),Ae=e=>{var r=_e(),i=a(r),o=e=>{f(e,ue())},p=e=>{var r=ge(),i=a(r),o=a(i),p=e=>{{let t=l(()=>`Layout of ${c(W)}`);ie(e,{get map(){return c(U)},get label(){return c(t)}})}},m=e=>{f(e,de())};n(o,e=>{c(U)?e(p):e(m,-1)}),_(i);var h=x(i,2),g=a(h),v=x(a(g),2),y=a(v),b=x(a(y),2),S=a(b,!0);_(b),_(y);var C=x(y,2),w=x(a(C),2),te=a(w);_(w),_(C);var E=x(C,2),D=x(a(E),2),ne=a(D,!0);_(D),_(E),_(v);var O=x(v,2),re=e=>{var t=pe();ee(t,21,()=>c(Se),d,(e,t)=>{var n=fe(),r=a(n),i=a(r);_(r);var o=x(r,2),s=a(o,!0);_(o),_(n),u(()=>{T(i,`${c(t).icon??``} ${c(t).label??``}`),T(s,c(t).value)}),f(e,n)}),_(t),f(e,t)};n(O,e=>{c(Se).length&&e(re)}),_(g);var k=x(g,4),A=x(a(k),2),j=a(A);_(A);var M=x(A,2),ae=a(M,!0);_(M),_(k);var N=x(k,2),oe=x(a(N),2),se=e=>{var n=me(),r=s(n),i=a(r);_(r);var o=x(r,2),l=a(o),d=a(l,!0);_(l);var p=x(l,2),m=a(p,!0);_(p),_(o),u(()=>{T(i,`Anyone with this key can change ${c(W)??``}. It was issued once, when the map was created,
								and is stored in this browser — keep it if you want to edit the map later.`),T(d,c(Y)),T(m,c(X)===`key`?`Copied ✓`:`Copy`)}),t(`click`,p,()=>Z(c(Y),`key`)),f(e,n)},ce=e=>{var t=he(),n=a(t);_(t),u(()=>T(n,`This browser has no edit key for ${c(W)??``}, so it can only be changed by an administrator
								or with a key you saved when you created the map.`)),f(e,t)};n(oe,e=>{c(Y)?e(se):e(ce,-1)}),_(N);var P=x(N,2),F=x(a(P),4),le=a(F,!0);_(F);var I=x(F,2),L=x(a(I),2),ue=a(L,!0);_(L),_(I),_(P),_(h),_(r),u(()=>{T(S,c(G)?`${c(G)}×${c(G)}`:`—`),T(te,`${(c(xe)||`—`)??``} / ${(c(K)||`—`)??``}`),T(ne,c(B)),T(j,`Send this page's link and your friend can start a session on ${c(W)??``} straight away.`),T(ae,c(X)===`link`?`Link copied ✓`:c(X)===`failed`?`Copy failed`:`Copy link`),T(le,c(X)===`prompt`?`Prompt copied ✓`:`Copy AI prompt`),T(ue,c(Ce))}),t(`click`,M,we),t(`click`,F,()=>Z(c(Ce),`prompt`)),f(e,r)};n(i,e=>{N().fetching&&c(U)===void 0?e(o):e(p,-1)}),_(r),f(e,r)};n($,e=>{c(J)||e(Ae)});var je=x($,2),Me=e=>{ae(e,{get map(){return H.map},get starting(){return H.starting},get error(){return H.error},onstart:e=>H.begin(e),onclose:()=>H.cancel()})};n(je,e=>{H.map&&e(Me)}),f(r,Te),y(),ye()}b([`click`]);export{R as component};