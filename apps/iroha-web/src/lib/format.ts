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

// Full timestamp as `yyyy-MM-dd HH:mm:ss` in the activity's timezone. The
// sv-SE locale renders exactly this ISO-like shape with a 24-hour clock.
export function formatDate(iso?: string, timezone?: string): string {
	if (!iso) return DASH;
	const d = new Date(iso);
	if (isNaN(d.getTime())) return iso;
	try {
		return new Intl.DateTimeFormat('sv-SE', {
			year: 'numeric',
			month: '2-digit',
			day: '2-digit',
			hour: '2-digit',
			minute: '2-digit',
			second: '2-digit',
			hour12: false,
			timeZone: timezone || undefined
		}).format(d);
	} catch {
		return d.toISOString().slice(0, 19).replace('T', ' ');
	}
}

// Date only as `yyyy-MM-dd` in the activity's timezone.
export function formatDateOnly(iso?: string, timezone?: string): string {
	if (!iso) return DASH;
	const d = new Date(iso);
	if (isNaN(d.getTime())) return iso;
	try {
		return new Intl.DateTimeFormat('sv-SE', {
			year: 'numeric',
			month: '2-digit',
			day: '2-digit',
			timeZone: timezone || undefined
		}).format(d);
	} catch {
		return d.toISOString().slice(0, 10);
	}
}

// Normalize a sport type for display: iroha stores a mix of short lowercase
// codes (run, walk, ride) and raw Apple PascalCase (FitnessGaming,
// HighIntensityIntervalTraining). Render all of them as uniform Title Case.
export function formatSport(sport?: string): string {
	if (!sport) return DASH;
	return sport
		.replace(/([a-z0-9])([A-Z])/g, '$1 $2')
		.replace(/[_-]+/g, ' ')
		.trim()
		.toLowerCase()
		.split(/\s+/)
		.map((w) => w.charAt(0).toUpperCase() + w.slice(1))
		.join(' ');
}
