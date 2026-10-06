// Change this line
import adapter from 'svelte-adapter-bun';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	compilerOptions: {
		// Simpler approach: enable runes globally.
		// If a library in node_modules fails, you can revisit the filter.
		runes: true
	},
	kit: {
		// The Bun adapter is now active
		adapter: adapter()
	}
};

export default config;
