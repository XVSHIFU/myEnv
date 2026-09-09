package config

import "testing"

func TestProviderAndPreviewConstraints(t *testing.T) {
	if c, e := ParseConstraint("java", "jdk26u-2026-09-05-13-43-beta"); e != nil || !c.Contains("jdk26u-2026-09-05-13-43-beta") {
		t.Fatal("dated Java EA", e)
	}
	for _, r := range [][3]string{{"python", "python.org/3.14", "3.14.7"}, {"python", "astral/3.12", "3.12.13"}, {"python", "python.org/3.15.0rc2", "3.15.0rc2"}, {"go", "1.27rc1", "1.27rc1"}, {"java", "jdk-27+14-ea-beta", "jdk-27+14-ea-beta"}, {"rust", "nightly", "nightly-2026-09-08"}, {"node", "27.0.0-nightly20260908abcdef", "27.0.0-nightly20260908abcdef"}} {
		c, e := ParseConstraint(r[0], r[1])
		if e != nil || !c.Contains(r[2]) {
			t.Fatalf("%v %v", r, e)
		}
	}
	for _, r := range [][2]string{{"python", "evil/3.14"}, {"rust", "nightly/../../x"}, {"node", "27.0.0-nightly../../x"}} {
		if _, e := ParseConstraint(r[0], r[1]); e == nil {
			t.Fatalf("accepted %v", r)
		}
	}
	stable, _ := ParseConstraint("python", "3.15")
	if stable.Contains("3.15.0rc2") {
		t.Fatal("implicit preview")
	}
}
