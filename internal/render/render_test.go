package render

import (
	"reflect"
	"testing"
	"time"
)

func TestBuiltinsCaseVariants(t *testing.T) {
	now := time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC)
	v := Builtins("my-cool_App", "Kaj", "MIT", now)
	want := map[string]string{
		"PROJECT_NAME":        "my-cool_App",
		"PROJECT_NAME_LOWER":  "my-cool_app",
		"PROJECT_NAME_UPPER":  "MY-COOL_APP",
		"PROJECT_NAME_SNAKE":  "my_cool_app",
		"PROJECT_NAME_KEBAB":  "my-cool-app",
		"PROJECT_NAME_PASCAL": "MyCoolApp",
		"PROJECT_NAME_CAMEL":  "myCoolApp",
		"AUTHOR":              "Kaj",
		"LICENSE":             "MIT",
		"YEAR":                "2026",
		"DATE":                "2026-03-09",
	}
	for k, w := range want {
		if v[k] != w {
			t.Errorf("%s = %q, want %q", k, v[k], w)
		}
	}
}

func TestCamelCaseInputIsSplit(t *testing.T) {
	v := Builtins("myApp2Go", "", "", time.Now())
	if v["PROJECT_NAME_SNAKE"] != "my_app2_go" || v["PROJECT_NAME_PASCAL"] != "MyApp2Go" {
		t.Fatalf("unexpected: snake=%q pascal=%q", v["PROJECT_NAME_SNAKE"], v["PROJECT_NAME_PASCAL"])
	}
}

func TestApplyIsSinglePassAndLeavesUnknownTokens(t *testing.T) {
	vars := Vars{"A": "{{B}}", "B": "boom"}
	got := Apply("x {{A}} y {{B}} z {{C}}", vars)
	if got != "x {{B}} y boom z {{C}}" {
		t.Fatalf("got %q", got)
	}
}

func TestApplyToleratesSpaces(t *testing.T) {
	if got := Apply("{{ NAME }}-{{NAME}}", Vars{"NAME": "x"}); got != "x-x" {
		t.Fatalf("got %q", got)
	}
}

func TestCustomOverridesBuiltin(t *testing.T) {
	v := Merge(Vars{"AUTHOR": "a"}, Vars{"AUTHOR": "b", "X": "1"})
	if v["AUTHOR"] != "b" || v["X"] != "1" {
		t.Fatalf("got %+v", v)
	}
}

func TestUsedAndUnresolved(t *testing.T) {
	s := "{{B}} {{A}} {{A}} {{ C }} {{not a token}} {1}"
	if got := Used(s); !reflect.DeepEqual(got, []string{"A", "B", "C"}) {
		t.Fatalf("used = %v", got)
	}
	if got := Unresolved(s, Vars{"A": "", "C": ""}); !reflect.DeepEqual(got, []string{"B"}) {
		t.Fatalf("unresolved = %v", got)
	}
}

func TestValidKey(t *testing.T) {
	for _, ok := range []string{"A", "_x", "PORT_8080"} {
		if !ValidKey(ok) {
			t.Errorf("%q should be valid", ok)
		}
	}
	for _, bad := range []string{"", "1A", "A-B", "A B", "{{A}}"} {
		if ValidKey(bad) {
			t.Errorf("%q should be invalid", bad)
		}
	}
}
