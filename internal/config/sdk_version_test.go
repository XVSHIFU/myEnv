package config

import "testing"

func TestSDKFullVersions(t *testing.T) {
	exact, _ := ParseConstraint("go", "=1.2")
	if !exact.Contains("1.2") || exact.Contains("1.2.1") {
		t.Fatal("historical exact version treated as a family")
	}
	if merged, err := MergeConstraints("java", "21", "21.0.12.1+1"); err != nil || merged != "21.0.12.1+1" {
		t.Fatalf("SDK merge: %s %v", merged, err)
	}
	for _, row := range [][3]string{{"java", "21", "jdk-21.0.12.1+1"}, {"java", "jdk-21.0.12.1+1", "jdk-21.0.12.1+1"}, {"java", "8", "jdk8u462-b08"}, {"go", "1.26", "1.26.6"}, {"rust", "1.9", "1.9.0"}} {
		c, e := ParseConstraint(row[0], row[1])
		if e != nil {
			t.Fatal(e)
		}
		if !c.Contains(row[2]) {
			t.Fatalf("%v mismatch", row)
		}
	}
	c, _ := ParseConstraint("rust", "1.9")
	if c.Contains("1.90.0") {
		t.Fatal("prefix collision")
	}
}
