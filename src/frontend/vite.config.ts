/// <reference types="vitest" />
import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vitest/config';
import path from 'path';

export default defineConfig({
	plugins: [sveltekit(), tailwindcss()],
	resolve: {
		alias: {
			'$lib': path.resolve(__dirname, './src/lib'),
		},
		conditions: ['browser'],
	},
	test: {
		include: ['src/**/*.{test,spec}.{js,ts}'],
		environment: 'jsdom',
		environmentOptions: {
			jsdom: {
				url: 'http://localhost/'
			}
		},
		globals: true,
		setupFiles: ['./vitest-setup.ts'],
	}
});
