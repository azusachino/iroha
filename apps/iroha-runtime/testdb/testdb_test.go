package testdb

import "testing"

func TestCheck(t *testing.T) {
	cases := []struct {
		name       string
		dsn        string
		disposable bool
		wantErr    bool
	}{
		{"dev database refused", "postgres://iroha:x@127.0.0.1:5432/iroha?sslmode=disable", false, true},
		{"prefix lookalike refused", "postgres://u@h/my_iroha_test", false, true},
		{"empty name refused", "postgres://u@h:5432", false, true},
		{"test database allowed", "postgres://u@h:5432/iroha_test?sslmode=disable", false, false},
		{"unique test database allowed", "postgres://u@h/iroha_test_1234", false, false},
		{"declared disposable allowed", "postgres://u@h/iroha", true, false},
		{"garbage refused", "://bad", false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := Check(tc.dsn, tc.disposable); (err != nil) != tc.wantErr {
				t.Fatalf("Check(%q, %v) error = %v, wantErr %v", tc.dsn, tc.disposable, err, tc.wantErr)
			}
		})
	}
}
