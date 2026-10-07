import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig, type UserConfig } from 'vitest/config';

// vitest bundles its own (older) vite, so its Plugin type differs from vite 8's; the plugins work fine at runtime.
const plugins = [tailwindcss(), sveltekit()] as unknown as NonNullable<UserConfig['plugins']>;

export default defineConfig({
	plugins,
	resolve: {
		conditions: ['browser']
	},
	server: {
		proxy: {
			'/graphql': { target: 'http://localhost:9090', ws: true },
			'/ws': { target: 'http://localhost:9090', ws: true },
			'/llms.txt': { target: 'http://localhost:9090' }
		}
	},
	test: {
		environment: 'jsdom',
		setupFiles: ['./src/test/setup.ts'],
		include: ['src/**/*.{test,spec}.{js,ts}']
	}
});
