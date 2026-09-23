// Copyright (c) Tailcat-QUIC contributors
// SPDX-License-Identifier: BSD-3-Clause

// Command install builds the verified fork source, including its public module
// replacements. Remote `go install ...@version` rejects replacement directives;
// this small stdlib-only bootstrap installs the real binary, not a launcher.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const version = "v0.7.0-quic.2"
const sourceSum = "h1:torKf2HIsQ3If1/ii+U2hzVhHS0/x2UQ9Tm/+4uP0hg="

func goOutput(args ...string) (string, error) {
	cmd := exec.Command("go", args...)
	cmd.Stderr = os.Stderr
	b, err := cmd.Output()
	return strings.TrimSpace(string(b)), err
}

func run() error {
	dirFlag := flag.String("bin-dir", "", "absolute installation directory; defaults to GOBIN or GOPATH/bin")
	flag.Parse()
	if flag.NArg() != 0 {
		return errors.New("unexpected arguments")
	}
	binDir := *dirFlag
	if binDir == "" {
		var err error
		binDir, err = goOutput("env", "GOBIN")
		if err != nil {
			return err
		}
		if binDir == "" {
			gopath, err := goOutput("env", "GOPATH")
			if err != nil {
				return err
			}
			paths := filepath.SplitList(gopath)
			if len(paths) == 0 {
				return errors.New("GOPATH is empty; specify -bin-dir")
			}
			binDir = filepath.Join(paths[0], "bin")
		}
	}
	if !filepath.IsAbs(binDir) {
		return errors.New("-bin-dir/GOBIN must be absolute")
	}
	fmt.Fprintln(os.Stderr, "Building verified Tailcat-QUIC", version)
	text, err := goOutput("mod", "download", "-json", "github.com/LiuTangLei/tailcat-quic@"+version)
	if err != nil {
		return err
	}
	var module struct {
		Dir, Sum, Error string
	}
	if err := json.Unmarshal([]byte(text), &module); err != nil {
		return err
	}
	if module.Error != "" || module.Dir == "" || module.Sum != sourceSum {
		return errors.New("source module integrity mismatch; nothing installed")
	}
	work, err := os.MkdirTemp("", "tailcat-source-install-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(work)
	// Build in a private writable copy: never modify the module cache.
	var copied int64
	err = filepath.WalkDir(module.Dir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(module.Dir, path)
		if err != nil {
			return err
		}
		dest := filepath.Join(work, rel)
		if d.IsDir() {
			return os.MkdirAll(dest, 0755)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("non-regular source entry: %s", rel)
		}
		copied += info.Size()
		if info.Size() > 32<<20 || copied > 256<<20 {
			return errors.New("source size exceeds installer bounds")
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(dest, data, info.Mode().Perm()|0600)
	})
	if err != nil {
		return err
	}
	tags, err := os.ReadFile(filepath.Join(work, "build-tags.txt"))
	if err != nil {
		return err
	}
	name := "tailcat"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	built := filepath.Join(work, name)
	cmd := exec.Command("go", "build", "-mod=readonly", "-trimpath", "-tags="+strings.TrimSpace(string(tags)),
		"-ldflags=-s -w -X main.version="+version, "-o", built, "./cmd/tailcat")
	cmd.Dir = work
	cmd.Env = append(os.Environ(), "GOWORK=off")
	cmd.Stdout, cmd.Stderr = os.Stderr, os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	check := exec.Command(built, "version")
	out, err := check.Output()
	if err != nil || strings.TrimSpace(string(out)) != version {
		return errors.New("built binary failed native version verification")
	}
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return err
	}
	stage, err := os.CreateTemp(binDir, ".tailcat-install-*")
	if err != nil {
		return err
	}
	defer os.Remove(stage.Name())
	data, err := os.ReadFile(built)
	if err == nil {
		_, err = stage.Write(data)
	}
	if err == nil {
		err = stage.Chmod(0755)
	}
	closeErr := stage.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	destination := filepath.Join(binDir, name)
	if err := os.Rename(stage.Name(), destination); err != nil {
		return fmt.Errorf("install without changing existing configuration: %w", err)
	}
	fmt.Printf("Installed %s at %s\n", version, destination)
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "tailcat-quic install:", err)
		os.Exit(1)
	}
}
