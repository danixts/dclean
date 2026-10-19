package scanner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"dclean/internal/domain"
)

// browser caches hold session data, wiping them logs the user out
var browserSnaps = map[string]bool{
	"brave":          true,
	"firefox":        true,
	"chromium":       true,
	"opera":          true,
	"vivaldi":        true,
	"microsoft-edge": true,
}

// numbered revision dirs under ~/snap/<app> are full user profiles, not
// cache, so only common/.cache is ever offered
func (ms *MultiScanner) scanSnapCaches() {
	apps, err := os.ReadDir(ms.snapDir)
	if err != nil {
		return
	}

	for _, app := range apps {
		if app.IsDir() {
			ms.collectSnapCache(app.Name(), filepath.Join(ms.snapDir, app.Name()))
		}
	}
}

func (ms *MultiScanner) collectSnapCache(appName, appPath string) {
	if browserSnaps[appName] {
		return
	}

	cachePath := filepath.Join(appPath, "common", ".cache")
	if !isDir(cachePath) {
		return
	}

	cacheSize := calculateDirSize(cachePath)
	if cacheSize > domain.SnapCacheMinSize {
		ms.Result.add(domain.FoundDir{
			Path:     cachePath,
			Size:     cacheSize,
			Category: domain.SnapCacheCategory,
			Target:   appName,
		})
	}
}

func (ms *MultiScanner) scanDisabledSnaps() {
	out, err := exec.Command("snap", "list", "--all").Output()
	if err != nil {
		return
	}

	skippedHeader := false
	for line := range strings.SplitSeq(string(out), "\n") {
		if !skippedHeader {
			skippedHeader = true
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 6 || !isDisabledSnap(fields[len(fields)-1]) {
			continue
		}

		name, revision := fields[0], fields[2]
		// removing a revision also deletes ~/snap/<name>/<rev>, the profile copy
		if browserSnaps[name] {
			continue
		}
		snapFile := filepath.Join("/var/lib/snapd/snaps", fmt.Sprintf("%s_%s.snap", name, revision))
		info, err := os.Stat(snapFile)
		if err != nil {
			continue
		}

		ms.Result.add(domain.FoundDir{
			Path:     snapFile,
			Size:     info.Size(),
			Category: domain.SnapDisabledCategory,
			Target:   fmt.Sprintf("%s (rev %s)", name, revision),
			Cmd:      rootCmd("snap", "remove", "--revision", revision, name),
		})
	}
}

func isDisabledSnap(notes string) bool {
	for note := range strings.SplitSeq(notes, ",") {
		if note == "disabled" || note == "desactivado" {
			return true
		}
	}
	return false
}
