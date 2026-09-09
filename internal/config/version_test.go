package config

import "testing"

func TestConstraintSelection(t *testing.T) {
	for _, tt := range []struct{ tool, input, yes, no string }{
		{"node", "22", "22.9.1", "23.0.0"},
		{"node", "20.1 - 22.3", "22.3.9", "22.4.0"},
		{"node", "20 - 22.3.1", "22.3.1", "22.3.2"},
		{"node", "20 - 24, <23", "22.9.0", "23.0.0"},
		{"node", ">=20 <23", "22.9.1", "23.0.0"},
		{"node", "^0.2.3 || ^22.1.0", "22.9.1", "0.3.0"},
		{"node", "~22.1", "22.1.9", "22.2.0"},
		{"python", "3.12, >=3.12", "3.12.8", "3.13.0"},
		{"python", ">=3.12, !=3.12.1, <3.13", "3.12.2", "3.12.1"},
		{"python", "~=3.12.1", "3.12.8", "3.13.0"},
		{"python", "<=3.12", "3.12.0", "3.12.1"},
	} {
		t.Run(tt.input, func(t *testing.T) {
			c, err := ParseConstraint(tt.tool, tt.input)
			if err != nil {
				t.Fatal(err)
			}
			if !c.Contains(tt.yes) || c.Contains(tt.no) {
				t.Fatalf("wrong selection for %s", tt.input)
			}
		})
	}
	for _, s := range []string{"", ">=3.13,<3.12", "3.12; echo hello", "3.12\n3.13", "latest", "3.12.1junk", "3.12,", "3.*.1"} {
		if _, err := ParseConstraint("python", s); err == nil {
			t.Errorf("accepted %q", s)
		}
	}
}

func TestMergeConstraintIntersection(t *testing.T) {
	merged, err := MergeConstraints("python", "3.12", ">=3.12, <4")
	if err != nil {
		t.Fatal(err)
	}
	c, err := ParseConstraint("python", merged)
	if err != nil {
		t.Fatal(err)
	}
	if !c.Contains("3.12.9") || c.Contains("3.13.0") {
		t.Fatalf("lost constraint: %s", merged)
	}
	if _, err := MergeConstraints("node", "22", "24"); err == nil {
		t.Fatal("merged incompatible versions")
	}
	merged, err = MergeConstraints("node", "^20 || ^22", ">=22")
	if err != nil {
		t.Fatal(err)
	}
	c, err = ParseConstraint("node", merged)
	if err != nil || !c.Contains("22.2.0") || c.Contains("20.2.0") {
		t.Fatalf("bad union: %s %v", merged, err)
	}
}
