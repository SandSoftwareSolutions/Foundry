package detect

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/SandSoftwareSolutions/Foundry/internal/config"
)

func fake(installed map[string]bool, files map[string]bool, goos string, env map[string]string) Finder {
	return Finder{
		LookPath: func(n string) (string, error) {
			if installed[n] {
				return "/usr/bin/" + n, nil
			}
			return "", errors.New("not found")
		},
		Stat: func(n string) (os.FileInfo, error) {
			if files[n] {
				return nil, nil
			}
			return nil, os.ErrNotExist
		},
		Getenv: func(k string) string { return env[k] },
		GOOS:   goos,
	}
}

func TestScanFindsOnlyInstalledToolsSorted(t *testing.T) {
	f := fake(map[string]bool{"go": true, "node": true, "npm": true, "git": true, "python3": true}, nil, "linux", nil)
	r := f.Scan()
	if !reflect.DeepEqual(r.Languages, []string{"Go", "Node.js", "Python"}) {
		t.Fatalf("languages = %v", r.Languages)
	}
	if !reflect.DeepEqual(r.PackageManagers, []string{"npm"}) || !reflect.DeepEqual(r.DevTools, []string{"git"}) {
		t.Fatalf("pm=%v tools=%v", r.PackageManagers, r.DevTools)
	}
	if r.VSCodePath != "" {
		t.Fatalf("vscode = %q", r.VSCodePath)
	}
}

func TestEmptyScanGivesEmptySlicesNotNil(t *testing.T) {
	r := fake(nil, nil, "linux", nil).Scan()
	if r.Languages == nil || r.PackageManagers == nil || r.DevTools == nil {
		t.Fatal("slices must be non-nil so JSON shows [] and not null")
	}
}

func TestVSCodeFromPathThenKnownLocations(t *testing.T) {
	if got := fake(map[string]bool{"code": true}, nil, "linux", nil).VSCode(); got != "/usr/bin/code" {
		t.Fatalf("PATH lookup: %q", got)
	}
	if got := fake(nil, map[string]bool{"/snap/bin/code": true}, "linux", nil).VSCode(); got != "/snap/bin/code" {
		t.Fatalf("linux fallback: %q", got)
	}
	local := filepath.Join(`C:\Users\x\AppData\Local`, "Programs", "Microsoft VS Code", "bin", "code.cmd")
	env := map[string]string{"LOCALAPPDATA": `C:\Users\x\AppData\Local`}
	if got := fake(nil, map[string]bool{local: true}, "windows", env).VSCode(); got != local {
		t.Fatalf("windows fallback: %q", got)
	}
	if got := fake(nil, nil, "darwin", nil).VSCode(); got != "" {
		t.Fatalf("nothing installed: %q", got)
	}
}

func TestApplyKeepsExistingVSCodePathWhenNoneFound(t *testing.T) {
	cfg := config.Default()
	cfg.VSCodePath = "/old/code"
	Apply(cfg, Result{Languages: []string{"Go"}})
	if cfg.VSCodePath != "/old/code" || !reflect.DeepEqual(cfg.InstalledLanguages, []string{"Go"}) {
		t.Fatalf("unexpected: %+v", cfg)
	}
}
