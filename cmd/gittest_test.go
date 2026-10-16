package cmd

import (
	"os/exec"
)

func gitAvailable() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

func gitOutput(dir string, args ...string) (string, error) {
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	return string(out), err
}

func gitInitCommit(dir string) (string, error) {
	for _, args := range [][]string{
		{"init", "--quiet"},
		{"add", "--all"},
		{"-c", "user.name=T", "-c", "user.email=t@example.com", "commit", "--quiet", "-m", "template"},
	} {
		if out, err := gitOutput(dir, args...); err != nil {
			return out, err
		}
	}
	return "", nil
}
