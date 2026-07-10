import { describe, expect, it } from 'vitest';
import { load } from './+page';

const routePages = import.meta.glob('./**/+page.svelte', { eager: true });

describe('root route', () => {
	it('redirects to the dashboard cockpit', () => {
		try {
			load();
			throw new Error('expected redirect');
		} catch (error: any) {
			expect(error.status).toBe(307);
			expect(error.location).toBe('/dashboard');
		}
	});
});

describe('cockpit route layout', () => {
	it('has concrete route files for the dashboard, activities domain, and share page', () => {
		expect(routePages['./dashboard/+page.svelte']).toBeDefined();
		expect(routePages['./activities/+page.svelte']).toBeDefined();
		expect(routePages['./share/+page.svelte']).toBeDefined();
	});

	it('does not keep the old /u page route', () => {
		expect(routePages['./u/+page.svelte']).toBeUndefined();
	});
});
