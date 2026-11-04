#!/usr/bin/env node
// Real-browser UI check for the map gallery, the start-session modal and the
// map details page. Drives Chrome over the DevTools Protocol, so it needs no
// npm dependencies and no Playwright install.
//
// Usage (see `make verify-ui`, which builds and starts the server for you):
//   node e2e/ui.mjs                     # headless, against http://127.0.0.1:8000
//   BASE=http://127.0.0.1:9001 node e2e/ui.mjs
//   HEADED=1 node e2e/ui.mjs            # watch the run in a real window
//   REQUIRE=1 node e2e/ui.mjs           # fail (not skip) when Chrome is absent
//
// Why a browser at all: jsdom tests cannot catch layout problems, and these
// checks exist because two real bugs slipped past unit tests — a modal whose
// primary button was unreachable on a short viewport, and a clipped placeholder.
import { spawn } from 'node:child_process';
import { mkdtemp, rm, writeFile } from 'node:fs/promises';
import { existsSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { setTimeout as sleep } from 'node:timers/promises';

const BASE = (process.env.BASE || 'http://127.0.0.1:8000').replace(/\/$/, '');
const HEADED = process.env.HEADED === '1';
const REQUIRE = process.env.REQUIRE === '1';
const SHOTS_DIR = process.env.SHOTS_DIR || '';

const CHROME_CANDIDATES = [
	process.env.CHROME_BIN,
	'/Applications/Google Chrome.app/Contents/MacOS/Google Chrome',
	'/Applications/Chromium.app/Contents/MacOS/Chromium',
	'/usr/bin/google-chrome',
	'/usr/bin/google-chrome-stable',
	'/usr/bin/chromium',
	'/usr/bin/chromium-browser'
].filter(Boolean);

function findChrome() {
	return CHROME_CANDIDATES.find((p) => existsSync(p)) ?? null;
}

const results = [];
function check(name, pass, detail = '') {
	results.push({ name, pass });
	console.log(`${pass ? 'PASS' : 'FAIL'} ${name}${detail ? ` — ${detail}` : ''}`);
}

// ---------------------------------------------------------------- CDP client

const PAGE_HELPERS = `
	const byText = (sel, text) => [...document.querySelectorAll(sel)]
		.find((el) => (el.textContent || '').trim().includes(text));
	const byTextWithin = (root, sel, text) => [...root.querySelectorAll(sel)]
		.find((el) => (el.textContent || '').trim().includes(text));
	const dialog = () => document.querySelector('[role="dialog"]');
	const label = (text) => {
		const l = [...document.querySelectorAll('label')]
			.find((el) => (el.textContent || '').trim() === text);
		return l ? document.getElementById(l.getAttribute('for')) : null;
	};
	const fogBox = () => document.getElementById('start-fog-enabled')
		|| [...document.querySelectorAll('label')]
			 .find((l) => l.textContent.includes('Hide the map beyond'))?.querySelector('input');
	const cardFor = (name) => [...document.querySelectorAll('[data-testid="map-card"]')]
		.find((c) => c.querySelector('h3')?.textContent.includes(name));
`;

function makeClient(ws, sessionId) {
	let id = 0;
	const pending = new Map();
	const errors = [];
	ws.onmessage = (event) => {
		const msg = JSON.parse(event.data);
		if (msg.method === 'Runtime.exceptionThrown') {
			errors.push(
				msg.params.exceptionDetails?.exception?.description || msg.params.exceptionDetails?.text
			);
		}
		if (msg.method === 'Runtime.consoleAPICalled' && msg.params.type === 'error') {
			errors.push(msg.params.args.map((a) => a.value ?? a.description ?? '').join(' '));
		}
		if (msg.id && pending.has(msg.id)) {
			const { resolve, reject } = pending.get(msg.id);
			pending.delete(msg.id);
			msg.error ? reject(new Error(JSON.stringify(msg.error))) : resolve(msg.result);
		}
	};
	const send = (method, params = {}) =>
		new Promise((resolve, reject) => {
			const i = ++id;
			pending.set(i, { resolve, reject });
			ws.send(JSON.stringify({ id: i, method, params, sessionId }));
		});

	const evaluate = async (expression) => {
		const r = await send('Runtime.evaluate', {
			expression: `(async () => { ${PAGE_HELPERS} ${expression} })()`,
			awaitPromise: true,
			returnByValue: true
		});
		if (r.exceptionDetails) {
			throw new Error(r.exceptionDetails.exception?.description || 'page evaluation failed');
		}
		return r.result.value;
	};

	const waitFor = async (expression, label, timeoutMs = 20000) => {
		const start = Date.now();
		while (Date.now() - start < timeoutMs) {
			if (await evaluate(`try { return !!(${expression}); } catch { return false; }`)) return true;
			await sleep(120);
		}
		throw new Error(`timed out waiting for ${label}`);
	};

	const go = async (path) => {
		await send('Page.navigate', { url: BASE + path });
		await waitFor(`document.readyState === 'complete'`, `load ${path}`);
		await sleep(600);
	};

	const screenshot = async (name) => {
		if (!SHOTS_DIR) return;
		const { data } = await send('Page.captureScreenshot', { format: 'png' });
		await writeFile(join(SHOTS_DIR, `${name}.png`), Buffer.from(data, 'base64'));
	};

	return { send, evaluate, waitFor, go, screenshot, errors };
}

// -------------------------------------------------------------------- suites

async function gallerySuite(ui) {
	await ui.go('/maps');
	await ui.waitFor(`cardFor('Classic Layout') !== undefined`, 'Classic card');

	check('gallery: no Edit links', await ui.evaluate(`return byText('a','Edit') === undefined;`));
	check('gallery: no Duplicate links', await ui.evaluate(`return byText('a','Duplicate') === undefined;`));
	check(
		'gallery: map name links to its page',
		(await ui.evaluate(`return byText('a','Classic Layout')?.getAttribute('href');`)) === '/maps/classic'
	);
	await ui.screenshot('gallery');

	await ui.evaluate(`byTextWithin(cardFor('Classic Layout'),'button','Play').click(); return true;`);
	await ui.waitFor(`dialog() !== null`, 'start modal');

	const defaults = await ui.evaluate(`
		const dlg = dialog();
		return {
			title: dlg.querySelector('h2').textContent.trim(),
			fog: fogBox()?.checked,
			radius: label('Radius')?.value,
			delay: label('Move delay (ms)')?.value
		};`);
	check('modal: opens for the chosen map', defaults.title === 'Start Classic Layout', defaults.title);
	check('modal: fog on by default', defaults.fog === true, `checked=${defaults.fog}`);
	check('modal: radius defaults to 1', defaults.radius === '1', `radius=${defaults.radius}`);
	check('modal: move delay defaults to 300', defaults.delay === '300', `delay=${defaults.delay}`);

	// The primary button must stay reachable on a short viewport: this is the
	// regression the browser check originally caught.
	const reachable = await ui.evaluate(`
		const d = dialog();
		const scroller = d.closest('[role="presentation"]');
		scroller.scrollTop = scroller.scrollHeight;
		await new Promise((r) => setTimeout(r, 150));
		const btn = [...d.querySelectorAll('button')].find((b) => b.textContent.trim() === 'Start session');
		const r = btn.getBoundingClientRect();
		return r.bottom <= innerHeight + 1 && r.top >= -1;`);
	check('modal: Start session stays reachable', reachable === true);
	await ui.screenshot('start-modal');

	// Invalid input is reported by the app, not swallowed by native validation.
	await ui.evaluate(`
		const r = label('Radius'); r.value = '0';
		r.dispatchEvent(new Event('input', { bubbles: true }));
		byText('button','Start session').click();
		return true;`);
	await sleep(300);
	const alert = await ui.evaluate(`
		const a = dialog()?.querySelector('[role="alert"]');
		return a ? a.textContent.trim() : null;`);
	check('modal: invalid radius shows a message', !!alert && alert.includes('at least 1'), alert ?? 'none');
	check(
		'modal: invalid submit keeps the modal open',
		await ui.evaluate(`return dialog() !== null && location.pathname === '/maps';`)
	);

	await ui.evaluate(`
		const r = label('Radius'); r.value = '2';
		r.dispatchEvent(new Event('input', { bubbles: true }));
		const m = label('Move delay (ms)'); m.value = '50';
		m.dispatchEvent(new Event('input', { bubbles: true }));
		return true;`);
	await ui.evaluate(`byText('button','Start session').click(); return true;`);
	await ui.waitFor(`location.pathname.startsWith('/watch/')`, 'session created');
	const sessionId = await ui.evaluate(`return location.pathname.split('/').pop();`);
	check('gallery: Play creates a session', !!sessionId, `/watch/${sessionId}`);

	const state = await graphql(
		`query($id: ID!) { gameState(sessionID: $id) { fogEnabled fogRadius moveDelayMs } }`,
		{ id: sessionId }
	);
	const gs = state?.data?.gameState;
	check('gallery: chosen fog radius persisted', gs?.fogEnabled === true && gs?.fogRadius === 2, JSON.stringify(gs));
	check('gallery: chosen move delay persisted', gs?.moveDelayMs === 50, `moveDelayMs=${gs?.moveDelayMs}`);

	const hidden = await graphql(
		`query($id: ID!) { gameState(sessionID: $id) { grid(password: "wrong") { type } } }`,
		{ id: sessionId }
	);
	check(
		'gallery: fog really hides the grid',
		Array.isArray(hidden?.errors) && hidden.errors.length > 0,
		hidden?.errors?.[0]?.message?.slice(0, 60) ?? 'no error'
	);
}

async function homeSuite(ui) {
	await ui.go('/');
	await ui.waitFor(`byText('h2','Pick a map') !== undefined`, 'home map section');

	check('home: no inline Advanced options panel', await ui.evaluate(`return byText('summary','Advanced options') === undefined;`));
	await ui.evaluate(`byText('button','Play').click(); return true;`);
	await ui.waitFor(`dialog() !== null`, 'home start modal');

	const home = await ui.evaluate(`
		return {
			title: dialog().querySelector('h2').textContent.trim(),
			fog: fogBox()?.checked,
			radius: label('Radius')?.value
		};`);
	check('home: Play opens the modal', home.title.startsWith('Start '), home.title);
	check('home: fog on by default', home.fog === true);
	check('home: radius defaults to 1', home.radius === '1', `radius=${home.radius}`);

	await ui.evaluate(`window.dispatchEvent(new KeyboardEvent('keydown',{key:'Escape',bubbles:true})); return true;`);
	await sleep(300);
	check('home: Escape closes the modal', await ui.evaluate(`return dialog() === null;`));
	await ui.screenshot('home');
}

async function detailSuite(ui) {
	await ui.go('/maps');
	await ui.waitFor(`byText('a','Classic Layout') !== undefined`, 'map name link');
	await ui.evaluate(`byText('a','Classic Layout').click(); return true;`);
	await ui.waitFor(`location.pathname === '/maps/classic'`, 'detail route');
	await ui.waitFor(`document.querySelector('h1')?.textContent.includes('Classic Layout')`, 'detail heading');

	check('detail: name click navigates to the page', true, await ui.evaluate(`return location.pathname;`));
	check(
		'detail: layout preview rendered',
		await ui.evaluate(`return !!document.querySelector('[role="img"]');`)
	);
	check(
		'detail: offers Play this map',
		await ui.evaluate(`return byText('button','Play this map') !== undefined;`)
	);
	check(
		'detail: offers Copy link',
		await ui.evaluate(`return byText('button','Copy link') !== undefined;`)
	);
	await ui.screenshot('detail');

	await ui.evaluate(`byText('button','Play this map').click(); return true;`);
	await ui.waitFor(`dialog() !== null`, 'detail start modal');
	check(
		'detail: Play opens the modal for this map',
		(await ui.evaluate(`return dialog().querySelector('h2').textContent.trim();`)) === 'Start Classic Layout'
	);
	await ui.evaluate(`byText('button','Start session').click(); return true;`);
	await ui.waitFor(`location.pathname.startsWith('/watch/')`, 'detail session created');
	check('detail: creates a session and opens it', true, await ui.evaluate(`return location.pathname;`));

	// A shared link must work on a cold load, not just via client navigation.
	await ui.go('/maps/bayou_braids');
	await ui.waitFor(`document.querySelector('h1')?.textContent.includes('Bayou Braids')`, 'deep link heading');
	check('detail: deep link works on a cold load', true, await ui.evaluate(`return document.title;`));
	await ui.screenshot('deep-link');

	await ui.go('/maps/does_not_exist');
	await ui.waitFor(`document.querySelector('h1')?.textContent.includes('Map not found')`, 'not-found heading');
	check('detail: unknown map shows "Map not found"', true);
	check(
		'detail: unknown map has no Play button',
		await ui.evaluate(`return byText('button','Play this map') === undefined;`)
	);
}

// --------------------------------------------------------------------- main

async function graphql(query, variables) {
	const res = await fetch(`${BASE}/graphql`, {
		method: 'POST',
		headers: { 'content-type': 'application/json' },
		body: JSON.stringify({ query, variables })
	});
	return res.json();
}

async function main() {
	try {
		const res = await fetch(`${BASE}/graphql`, {
			method: 'POST',
			headers: { 'content-type': 'application/json' },
			body: JSON.stringify({ query: '{ maps { mapId } }' })
		});
		if (!res.ok) throw new Error(`HTTP ${res.status}`);
	} catch (err) {
		console.error(`Cannot reach ${BASE}/graphql (${err.message}).`);
		console.error('Start the server first, or use `make verify-ui`.');
		return 2;
	}

	const chromePath = findChrome();
	if (!chromePath) {
		const msg = 'No Chrome/Chromium found; set CHROME_BIN to run the browser checks.';
		if (REQUIRE) {
			console.error(msg);
			return 1;
		}
		console.log(`SKIP ${msg}`);
		return 0;
	}

	const profileDir = await mkdtemp(join(tmpdir(), 'trt-ui-'));
	const port = 9500 + Math.floor(Math.random() * 400);
	const args = [
		`--remote-debugging-port=${port}`,
		`--user-data-dir=${profileDir}`,
		'--no-first-run',
		'--no-default-browser-check',
		'--disable-gpu',
		'--window-size=1400,1000',
		'about:blank'
	];
	if (!HEADED) args.unshift('--headless=new');

	const chrome = spawn(chromePath, args, { stdio: 'ignore' });

	try {
		let wsUrl = '';
		for (let i = 0; i < 150 && !wsUrl; i++) {
			try {
				const r = await fetch(`http://127.0.0.1:${port}/json/version`);
				if (r.ok) wsUrl = (await r.json()).webSocketDebuggerUrl;
			} catch {
				// Chrome is still starting.
			}
			if (!wsUrl) await sleep(100);
		}
		if (!wsUrl) throw new Error('Chrome did not expose a debugging endpoint');

		const ws = new WebSocket(wsUrl);
		await new Promise((resolve, reject) => {
			ws.onopen = resolve;
			ws.onerror = () => reject(new Error('could not connect to Chrome'));
		});

		let id = 0;
		const target = await new Promise((resolve, reject) => {
			const i = ++id;
			const onMessage = (event) => {
				const msg = JSON.parse(event.data);
				if (msg.id !== i) return;
				ws.removeEventListener('message', onMessage);
				msg.error ? reject(new Error(JSON.stringify(msg.error))) : resolve(msg.result);
			};
			ws.addEventListener('message', onMessage);
			ws.send(
				JSON.stringify({ id: i, method: 'Target.createTarget', params: { url: 'about:blank' } })
			);
		});

		const attached = await new Promise((resolve, reject) => {
			const i = ++id;
			const onMessage = (event) => {
				const msg = JSON.parse(event.data);
				if (msg.id !== i) return;
				ws.removeEventListener('message', onMessage);
				msg.error ? reject(new Error(JSON.stringify(msg.error))) : resolve(msg.result);
			};
			ws.addEventListener('message', onMessage);
			ws.send(
				JSON.stringify({
					id: i,
					method: 'Target.attachToTarget',
					params: { targetId: target.targetId, flatten: true }
				})
			);
		});

		const ui = makeClient(ws, attached.sessionId);
		await ui.send('Page.enable');
		await ui.send('Runtime.enable');

		await gallerySuite(ui);
		await homeSuite(ui);
		await detailSuite(ui);
		check('no uncaught console errors', ui.errors.length === 0, ui.errors.slice(0, 2).join(' | '));

		try {
			await ui.send('Browser.close');
		} catch {
			// Already closing.
		}
	} finally {
		chrome.kill();
		await rm(profileDir, { recursive: true, force: true }).catch(() => {});
	}

	const failed = results.filter((r) => !r.pass);
	console.log(`\n${results.length - failed.length}/${results.length} browser checks passed`);
	if (failed.length) {
		console.log('failed:\n' + failed.map((f) => `  - ${f.name}`).join('\n'));
	}
	return failed.length ? 1 : 0;
}

process.exit(await main());
