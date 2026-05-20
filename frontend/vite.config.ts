import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [tailwindcss(), sveltekit()],
	server: {
		proxy: {
			'/graphql': { target: 'http://localhost:8000', ws: true },
			'/ws': { target: 'http://localhost:8000', ws: true },
			'/llms.txt': { target: 'http://localhost:8000' }
		}
	}
});
