package dynamicmodule

import "testing"

func TestGeneratedModuleRelativePathRejectsTraversal(t *testing.T) {
	cases := [][]string{
		{"schema", "generated", "business", "../secret.json"},
		{"schema", "generated", "business", "cmdb/../secret.json"},
		{"schema", "generated", "business", `C:\windows\system.ini`},
	}
	for _, parts := range cases {
		if _, ok := generatedModuleRelativePath(parts...); ok {
			t.Fatalf("expected traversal to be rejected for %v", parts)
		}
	}
}

func TestGeneratedModuleRelativePathAllowsNestedModuleSchema(t *testing.T) {
	target, ok := generatedModuleRelativePath("schema", "generated", "business", "cmdb/host.json")
	if !ok {
		t.Fatal("expected nested generated schema path to be valid")
	}
	if target != "schema/generated/business/cmdb/host.json" {
		t.Fatalf("unexpected relative path: %s", target)
	}
}
