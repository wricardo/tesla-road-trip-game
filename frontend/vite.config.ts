import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vitest/config';

export default defineConfig({
	plugins: [tailwindcss(), sveltekit()],
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
