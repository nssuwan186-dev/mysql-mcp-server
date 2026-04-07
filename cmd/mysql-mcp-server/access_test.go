package main

import (
	"reflect"
	"testing"
)

func TestRequireReferencedSchemasBlocksShowDatabases(t *testing.T) {
	t.Cleanup(func() { initAccessControl(nil) })
	initAccessControl([]string{"app"})
	if err := requireReferencedSchemasInQuery("SHOW DATABASES"); err == nil {
		t.Fatal("expected SHOW DATABASES to be rejected with allowlist")
	}
	if err := requireReferencedSchemasInQuery("SHOW DATABASES LIKE 'x%'"); err == nil {
		t.Fatal("expected SHOW DATABASES LIKE to be rejected with allowlist")
	}
	if err := requireReferencedSchemasInQuery("SELECT 1 FROM app.t"); err != nil {
		t.Fatalf("allowed schema should pass: %v", err)
	}
}

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
