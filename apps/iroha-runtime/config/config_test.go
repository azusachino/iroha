package config

import "testing"

func TestDefaultTimezone(t *testing.T) {
	if got := Default().Server.Timezone; got != "Asia/Tokyo" {
		t.Fatalf("default timezone = %q, want Asia/Tokyo", got)
	}
}

func TestLoadTimezoneFromEnvironment(t *testing.T) {
	t.Setenv(EnvTimezone, "UTC")
	got, err := Load("does-not-exist.toml")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if got.Server.Timezone != "UTC" {
		t.Fatalf("timezone = %q, want UTC", got.Server.Timezone)
	}
}

func TestLoadRejectsInvalidTimezone(t *testing.T) {
	t.Setenv(EnvTimezone, "Not/A/Timezone")
	if _, err := Load("does-not-exist.toml"); err == nil {
		t.Fatal("invalid timezone loaded successfully")
	}
}
