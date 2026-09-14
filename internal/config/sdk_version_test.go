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

func TestJava8UpdateSelector(t *testing.T) {
	for _, row := range []struct {
		selector, release string
		want              bool
	}{
		{"8u422", "jdk8u422-b05", true},
		{"jdk8u422", "jdk8u422-b05", true},
		{"8u422", "jdk8u4220-b05", false},
		{"8u422", "jdk8u422-beta", false},
		{"jdk8u422-b05", "jdk8u422-b06", false},
		{"=8u422", "jdk8u422-b05", false},
		{"=jdk8u422-b05", "jdk8u422-b05", true},
	} {
		c, err := ParseConstraint("java", row.selector)
		if err != nil {
			t.Fatal(err)
		}
		if got := c.Contains(row.release); got != row.want {
			t.Errorf("%q matches %q: got %v, want %v", row.selector, row.release, got, row.want)
		}
	}
}

func TestJavaFamilyAliasesKeepOtherSDKSemantics(t *testing.T) {
	for _, selector := range []string{"1.8", "jdk1.8", "java8", "jdk8", "JAVA8"} {
		c, err := ParseConstraint("java", selector)
		if err != nil || !c.Contains("jdk8u502-b07") || c.Contains("jdk-21.0.8+9") {
			t.Errorf("Java alias %q: %+v %v", selector, c, err)
		}
	}
	for _, tool := range []string{"go", "rust"} {
		c, err := ParseConstraint(tool, "1.8")
		if err != nil || !c.Contains("1.8.4") || c.Contains("8.0.4") {
			t.Errorf("%s 1.8 changed: %+v %v", tool, c, err)
		}
		if _, err = ParseConstraint(tool, "jdk1.8"); err == nil {
			t.Errorf("Java alias accepted for %s", tool)
		}
	}
}
