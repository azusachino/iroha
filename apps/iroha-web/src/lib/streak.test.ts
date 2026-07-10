import { describe, expect, it } from 'vitest';
import { currentActivityStreak } from './streak';

const NOW = new Date(2026, 6, 10, 12, 0, 0);

function localTimestamp(day: number): string {
	return new Date(2026, 6, day, 9, 0, 0).toISOString();
}

describe('currentActivityStreak', () => {
	it('returns zero without activity today', () => {
		expect(currentActivityStreak([localTimestamp(9)], NOW)).toBe(0);
	});

	it('counts today as a one-day streak', () => {
		expect(currentActivityStreak([localTimestamp(10)], NOW)).toBe(1);
	});

	it('counts one calendar day once even with multiple activities', () => {
		expect(currentActivityStreak([localTimestamp(10), localTimestamp(10)], NOW)).toBe(1);
	});

	it('counts consecutive days ending today', () => {
		expect(currentActivityStreak([localTimestamp(8), localTimestamp(9), localTimestamp(10)], NOW)).toBe(3);
	});

	it('stops at a missing day', () => {
		expect(currentActivityStreak([localTimestamp(7), localTimestamp(8), localTimestamp(10)], NOW)).toBe(1);
	});

	it('ignores invalid timestamps', () => {
		expect(currentActivityStreak(['invalid', localTimestamp(10)], NOW)).toBe(1);
	});
});
