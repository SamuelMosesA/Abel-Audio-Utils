import type { HandleClientError } from '@sveltejs/kit';

export const handleError: HandleClientError = async ({ error, event }) => {
	const message = error instanceof Error ? error.message : String(error);
	const stack = error instanceof Error ? error.stack : '';

	try {
		await fetch('/api/telemetry/errors', {
			method: 'POST',
			headers: {
				'Content-Type': 'application/json',
			},
			body: JSON.stringify({
				message,
				stack,
				url: event.url.toString(),
			}),
		});
	} catch (e) {
		// Log to console locally as a fallback
		console.error('Failed to send error to backend:', e);
	}
};
