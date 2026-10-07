/** Short relative time: "just now", "2m ago", "3h ago", "Oct 4". */
export function relativeTime(ts: string | number | Date | null | undefined, now: number = Date.now()): string {
	if (ts === null || ts === undefined || ts === '') return '—';
	const d = new Date(ts);
	if (Number.isNaN(d.getTime())) return '—';
	const secs = Math.floor((now - d.getTime()) / 1000);
	if (secs < 60) return 'just now';
	const mins = Math.floor(secs / 60);
	if (mins < 60) return `${mins}m ago`;
	const hours = Math.floor(mins / 60);
	if (hours < 24) return `${hours}h ago`;
	const days = Math.floor(hours / 24);
	if (days < 7) return `${days}d ago`;
	return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
}

/** Absolute time for title tooltips. */
export function absoluteTime(ts: string | number | Date | null | undefined): string {
	if (ts === null || ts === undefined || ts === '') return '';
	const d = new Date(ts);
	return Number.isNaN(d.getTime()) ? '' : d.toLocaleString();
}
