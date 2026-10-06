import type { Handle } from '@sveltejs/kit';

export const handle: Handle = async ({ event, resolve }) => {
	// Intercept any request starting with /api
	if (event.url.pathname.startsWith('/api')) {
		const targetUrl = 'http://localhost:3000'; // Your real backend API port
		const path = event.url.pathname + event.url.search;

		// Forward the request to your backend using Bun's native fetch
		return fetch(`${targetUrl}${path}`, {
			method: event.request.method,
			headers: event.request.headers,
			body: event.request.body,
			// @ts-ignore - Bun supports this for better performance
			duplex: 'half'
		});
	}

	return resolve(event);
};
