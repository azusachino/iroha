package observations

import (
	"testing"
)

func TestNormalizeSport(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"HKWorkoutActivityTypeRunning", "run"},
		{"running", "run"},
		{"run", "run"},
		{"Run", "run"},
		{"HKWorkoutActivityTypeWalking", "walk"},
		{"walking", "walk"},
		{"walk", "walk"},
		{"HKWorkoutActivityTypeCycling", "ride"},
		{"cycling", "ride"},
		{"ride", "ride"},
		{"bike", "ride"},
		{"HKWorkoutActivityTypeHiking", "hike"},
		{"hiking", "hike"},
		{"hike", "hike"},
		{"HKWorkoutActivityTypeSwimming", "swim"},
		{"swimming", "swim"},
		{"swim", "swim"},
		{"HKWorkoutActivityTypeFitnessGaming", "fitness_gaming"},
		{"FitnessGaming", "fitness_gaming"},
		{"Fitness Gaming", "fitness_gaming"},
		{"fitness_gaming", "fitness_gaming"},
		{"Outdoor Run", "run"},
		{"Indoor Run", "run"},
		{"outdoor_run", "run"},
		{"Outdoor Walk", "walk"},
		{"Outdoor Cycling", "ride"},
		{"Pool Swim", "swim"},
		{"HKWorkoutActivityTypeYoga", "yoga"},
		{"Yoga", "yoga"},
		{"TraditionalStrengthTraining", "traditional_strength_training"},
	}

	for _, tt := range tests {
		got := NormalizeSport(tt.input)
		if got != tt.want {
			t.Errorf("NormalizeSport(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestSportAliases(t *testing.T) {
	runAliases := SportAliases("running")
	if len(runAliases) != 2 || runAliases[0] != "run" || runAliases[1] != "running" {
		t.Errorf("SportAliases('running') = %v, want ['run', 'running']", runAliases)
	}

	gamingAliases := SportAliases("FitnessGaming")
	if len(gamingAliases) != 2 || gamingAliases[0] != "fitness_gaming" || gamingAliases[1] != "FitnessGaming" {
		t.Errorf("SportAliases('FitnessGaming') = %v, want ['fitness_gaming', 'FitnessGaming']", gamingAliases)
	}
}
