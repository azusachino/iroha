import { describe, expect, it } from 'vitest';
import { sportColor, sportLabel } from './sport';

describe('sportColor', () => {
	it('maps running sports to the run token', () => {
		expect(sportColor('run')).toBe('var(--sport-run)');
		expect(sportColor('Running')).toBe('var(--sport-run)');
	});

	it('maps walking, hiking, swimming, and riding aliases', () => {
		expect(sportColor('OutdoorWalk')).toBe('var(--sport-walk)');
		expect(sportColor('hiking')).toBe('var(--sport-hike)');
		expect(sportColor('PoolSwim')).toBe('var(--sport-swim)');
		expect(sportColor('cycle')).toBe('var(--sport-ride)');
		expect(sportColor('bike')).toBe('var(--sport-ride)');
		expect(sportColor('ride')).toBe('var(--sport-ride)');
	});

	it('falls back to other for missing or unknown sports', () => {
		expect(sportColor(undefined)).toBe('var(--sport-other)');
		expect(sportColor(null)).toBe('var(--sport-other)');
		expect(sportColor('HighIntensityIntervalTraining')).toBe('var(--sport-other)');
	});
});

describe('sportLabel', () => {
	it('reuses the existing sport formatter', () => {
		expect(sportLabel('HighIntensityIntervalTraining')).toBe('High Intensity Interval Training');
		expect(sportLabel(undefined)).toBe('—');
	});
});
