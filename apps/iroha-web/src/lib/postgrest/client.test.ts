import { describe, expect, it } from 'vitest';
import {
	getPostgrestPublicSummary,
	listPostgrestPublicActivities,
	type PostgrestPublicActivity,
	type PostgrestSummary
} from './client';

function createFakeFetch(responseData: unknown, ok = true, status = 200, statusText = 'OK') {
	let capturedUrl = '';
	let capturedInit: RequestInit | undefined;

	const fakeFetch = (async (input: string | Request | URL, init?: RequestInit) => {
		const url = typeof input === 'string' ? input : input.toString();
		capturedUrl = url;
		capturedInit = init;
		return {
			ok,
			status,
			statusText,
			json: async () => responseData
		};
	}) as typeof fetch;

	return { fakeFetch, getCapturedUrl: () => capturedUrl, getCapturedInit: () => capturedInit };
}

describe('listPostgrestPublicActivities', () => {
	it('builds PostgREST operator filters and offset pagination', async () => {
		const { fakeFetch, getCapturedUrl } = createFakeFetch([]);
		await listPostgrestPublicActivities(
			{
				sport_type: 'run',
				started_from: '2026-01-01T00:00:00Z',
				started_to: '2026-12-31T23:59:59Z',
				min_distance_m: 5000,
				max_distance_m: 20000,
				limit: 10,
				offset: 20
			},
			fakeFetch
		);

		const url = new URL(getCapturedUrl(), 'http://example.test');
		expect(url.pathname).toBe('/_postgrest/public_activities');
		expect(url.searchParams.get('sport_type')).toBe('eq.run');
		expect(url.searchParams.getAll('started_at')).toEqual([
			'gte.2026-01-01T00:00:00Z',
			'lte.2026-12-31T23:59:59Z'
		]);
		expect(url.searchParams.getAll('distance_m')).toEqual(['gte.5000', 'lte.20000']);
		expect(url.searchParams.get('limit')).toBe('10');
		expect(url.searchParams.get('offset')).toBe('20');
		expect(url.searchParams.get('order')).toBe('started_at.desc,id.desc');
	});

	it('returns the typed public activity rows', async () => {
		const rows: PostgrestPublicActivity[] = [
			{
				id: 'act_1',
				sport_type: 'run',
				title: 'Morning Run',
				started_at: '2026-01-01T00:00:00Z'
			}
		];
		const { fakeFetch } = createFakeFetch(rows);
		await expect(listPostgrestPublicActivities({}, fakeFetch)).resolves.toEqual(rows);
	});
});

describe('getPostgrestPublicSummary', () => {
	it('calls the public_summary RPC', async () => {
		const summary: PostgrestSummary = {
			totals: { activity_count: 1, distance_m: 1000, duration_s: 300, moving_time_s: 0 },
			by_year: [],
			by_month: [],
			by_sport: []
		};
		const { fakeFetch, getCapturedUrl, getCapturedInit } = createFakeFetch(summary);

		await expect(getPostgrestPublicSummary(fakeFetch)).resolves.toEqual(summary);

		expect(getCapturedUrl()).toBe('/_postgrest/rpc/public_summary');
		expect(getCapturedInit()?.method).toBe('POST');
		expect(getCapturedInit()?.body).toBe('{}');
	});
});
