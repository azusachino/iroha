import { API_BASE } from './config';

// Types mirror the iroha-server read API JSON contract (snake_case).
// Optional fields use `?` because the server omits them when absent.

export interface Activity {
	id: string;
	sport_type: string;
	title: string;
	started_at: string;
	ended_at?: string;
	timezone: string;
	distance_m?: number;
	duration_s?: number;
	moving_time_s?: number;
	elevation_gain_m?: number;
	avg_hr?: number;
	max_hr?: number;
	avg_pace_s_per_km?: number;
	source_kind: string;
	source_activity_id?: string;
	first_raw_file_id: string;
	created_at: string;
	updated_at: string;
}

export interface RoutePoint {
	seq: number;
	ts?: string;
	lat: number;
	lon: number;
	elevation_m?: number;
	distance_m?: number;
	speed_mps?: number;
	heart_rate?: number;
}

export interface SamplingPoint {
	id: string;
	sampling_type: string;
	ts: string;
	value: number;
	unit: string;
}

export interface Lap {
	id: string;
	lap_no: number;
	start_ts?: string;
	end_ts?: string;
	distance_m?: number;
	duration_s?: number;
	avg_hr?: number;
	avg_pace_s_per_km?: number;
}

export interface ListActivitiesParams {
	sport_type?: string;
	limit?: number;
}

async function getJSON<T>(path: string, fetchFn: typeof fetch = fetch): Promise<T> {
	const res = await fetchFn(`${API_BASE}${path}`, {
		headers: { accept: 'application/json' }
	});
	if (!res.ok) {
		throw new Error(`request failed: ${res.status} ${res.statusText} (${path})`);
	}
	return (await res.json()) as T;
}

export function listActivities(
	params: ListActivitiesParams = {},
	fetchFn: typeof fetch = fetch
): Promise<Activity[]> {
	const query = new URLSearchParams();
	if (params.sport_type) query.set('sport_type', params.sport_type);
	if (params.limit != null) query.set('limit', String(params.limit));
	const suffix = query.toString() ? `?${query.toString()}` : '';
	return getJSON<Activity[]>(`/api/v1/activities${suffix}`, fetchFn);
}

export function getActivity(id: string, fetchFn: typeof fetch = fetch): Promise<Activity> {
	return getJSON<Activity>(`/api/v1/activities/${encodeURIComponent(id)}`, fetchFn);
}

export function getActivityRoute(id: string, fetchFn: typeof fetch = fetch): Promise<RoutePoint[]> {
	return getJSON<RoutePoint[]>(`/api/v1/activities/${encodeURIComponent(id)}/route`, fetchFn);
}

export function getActivitySamplings(
	id: string,
	fetchFn: typeof fetch = fetch
): Promise<SamplingPoint[]> {
	return getJSON<SamplingPoint[]>(`/api/v1/activities/${encodeURIComponent(id)}/samplings`, fetchFn);
}

export function getActivityLaps(id: string, fetchFn: typeof fetch = fetch): Promise<Lap[]> {
	return getJSON<Lap[]>(`/api/v1/activities/${encodeURIComponent(id)}/laps`, fetchFn);
}
