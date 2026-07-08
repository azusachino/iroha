// Presentation helpers. All inputs may be undefined; missing data renders as
// an em dash so the UI degrades gracefully.

const DASH = '—';

export function formatDistance(meters?: number): string {
	if (meters == null) return DASH;
	if (meters < 1000) return `${Math.round(meters)} m`;
	return `${(meters / 1000).toFixed(2)} km`;
}

export function formatDuration(seconds?: number): string {
	if (seconds == null) return DASH;
	const s = Math.round(seconds);
	const h = Math.floor(s / 3600);
	const m = Math.floor((s % 3600) / 60);
	const sec = s % 60;
	const pad = (n: number) => String(n).padStart(2, '0');
	if (h > 0) return `${h}:${pad(m)}:${pad(sec)}`;
	return `${m}:${pad(sec)}`;
}

export function formatPace(secPerKm?: number): string {
	if (secPerKm == null || !isFinite(secPerKm) || secPerKm <= 0) return DASH;
	const m = Math.floor(secPerKm / 60);
	const s = Math.round(secPerKm % 60);
	const pad = (n: number) => String(n).padStart(2, '0');
	return `${m}:${pad(s)} /km`;
}

export function formatElevation(meters?: number): string {
	if (meters == null) return DASH;
	return `${Math.round(meters)} m`;
}

export function formatHr(bpm?: number): string {
	if (bpm == null) return DASH;
	return `${Math.round(bpm)} bpm`;
}

export function formatDate(iso?: string, timezone?: string): string {
	if (!iso) return DASH;
	const d = new Date(iso);
	if (isNaN(d.getTime())) return iso;
	try {
		return new Intl.DateTimeFormat('en-GB', {
			dateStyle: 'medium',
			timeStyle: 'short',
			timeZone: timezone || undefined
		}).format(d);
	} catch {
		return d.toLocaleString();
	}
}

export function formatDateOnly(iso?: string, timezone?: string): string {
	if (!iso) return DASH;
	const d = new Date(iso);
	if (isNaN(d.getTime())) return iso;
	try {
		return new Intl.DateTimeFormat('en-GB', {
			dateStyle: 'medium',
			timeZone: timezone || undefined
		}).format(d);
	} catch {
		return d.toLocaleDateString();
	}
}
