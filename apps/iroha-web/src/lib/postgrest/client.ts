import { POSTGREST_BASE } from '$lib/config';
import type { components, paths } from './types';

export type PostgrestPublicActivity = components['schemas']['public_activities'];
type PublicActivitiesQuery = NonNullable<paths['/public_activities']['get']['parameters']['query']>;

export interface PostgrestSummaryTotals {
	activity_count: number;
	distance_m: number;
	duration_s: number;
	moving_time_s: number;
}

export interface PostgrestSummaryBucket extends PostgrestSummaryTotals {
	key: string;
}

export interface PostgrestSummary {
	totals: PostgrestSummaryTotals;
	by_year: PostgrestSummaryBucket[];
	by_month: PostgrestSummaryBucket[];
	by_sport: PostgrestSummaryBucket[];
}

export interface ListPostgrestPublicActivitiesParams {
	sport_type?: string;
	started_from?: string;
	started_to?: string;
	min_distance_m?: number;
	max_distance_m?: number;
	limit?: number;
	offset?: number;
}

async function postgrestJSON<T>(path: string, init: RequestInit = {}, fetchFn: typeof fetch = fetch): Promise<T> {
	const response = await fetchFn(`${POSTGREST_BASE}${path}`, {
		...init,
		headers: { accept: 'application/json', ...init.headers }
	});
	if (!response.ok) {
		throw new Error(`postgrest request failed: ${response.status} ${response.statusText} (${path})`);
	}
	return (await response.json()) as T;
}

export function listPostgrestPublicActivities(
	params: ListPostgrestPublicActivitiesParams = {},
	fetchFn: typeof fetch = fetch
): Promise<PostgrestPublicActivity[]> {
	const query: PublicActivitiesQuery = {
		order: 'started_at.desc,id.desc',
		limit: String(params.limit ?? 20),
		offset: String(params.offset ?? 0)
	};
	if (params.sport_type) query.sport_type = `eq.${params.sport_type}`;
	if (params.started_from) query.started_at = `gte.${params.started_from}`;
	if (params.started_to) query.started_at = mergePostgrestFilters(query.started_at, `lte.${params.started_to}`);
	if (params.min_distance_m != null) query.distance_m = `gte.${params.min_distance_m}`;
	if (params.max_distance_m != null) {
		query.distance_m = mergePostgrestFilters(query.distance_m, `lte.${params.max_distance_m}`);
	}

	const search = new URLSearchParams();
	for (const [key, value] of Object.entries(query)) {
		if (value == null) continue;
		const parts =
			key === 'started_at' || key === 'distance_m' ? String(value).split(',') : [String(value)];
		for (const part of parts) {
			search.append(key, part);
		}
	}
	return postgrestJSON<PostgrestPublicActivity[]>(`/public_activities?${search}`, {}, fetchFn);
}

export function getPostgrestPublicSummary(fetchFn: typeof fetch = fetch): Promise<PostgrestSummary> {
	return postgrestJSON<PostgrestSummary>(
		'/rpc/public_summary',
		{
			method: 'POST',
			headers: { 'content-type': 'application/json' },
			body: '{}'
		},
		fetchFn
	);
}

function mergePostgrestFilters(existing: string | undefined, next: string): string {
	if (!existing) return next;
	return `${existing},${next}`;
}
