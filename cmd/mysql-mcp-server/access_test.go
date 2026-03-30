package main

import (
	"reflect"
	"testing"
)

func TestAllowedDatabasesLower(t *testing.T) {
	t.Cleanup(func() { initAccessControl(nil) })

	initAccessControl([]string{"zebra", "App", "  middle  "})
	got := allowedDatabasesLower()
	want := []string{"app", "middle", "zebra"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("allowedDatabasesLower() = %v, want %v", got, want)
	}

	initAccessControl(nil)
	if got := allowedDatabasesLower(); got != nil {
		t.Fatalf("with nil allowlist expected nil slice, got %#v", got)
	}
}
