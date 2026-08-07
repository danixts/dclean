package scanner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"dclean/internal/domain"
)

func (ms *MultiScanner) scanSystem() {
	ms.scanDisabledSnaps()
	ms.scanAptCache()
	ms.scanAptAutoremove()
	ms.scanJournal()
	ms.scanStaleTemp()
	ms.scanCrashReports()
}

// snapd keeps the previous revisions of every app after an update
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

func (ms *MultiScanner) scanAptCache() {
	if _, err := exec.LookPath("apt-get"); err != nil {
		return
	}

	size := calculateDirSize("/var/cache/apt/archives")
	if size < domain.SystemMinSize {
		return
	}

	ms.Result.add(domain.FoundDir{
		Path:     "/var/cache/apt/archives",
		Size:     size,
		Category: domain.AptCacheCategory,
		Target:   "downloaded .deb packages",
		Cmd:      rootCmd("apt-get", "clean"),
	})
}

// packages kept only as dependencies of something already removed, old kernels included
func (ms *MultiScanner) scanAptAutoremove() {
	if _, err := exec.LookPath("apt-get"); err != nil {
		return
	}

	out, err := exec.Command("apt-get", "-s", "autoremove").Output()
	if err != nil {
		return
	}

	var packages []string
	for line := range strings.SplitSeq(string(out), "\n") {
		if name, found := strings.CutPrefix(line, "Remv "); found {
			if fields := strings.Fields(name); len(fields) > 0 {
				packages = append(packages, fields[0])
			}
		}
	}

	if len(packages) == 0 {
		return
	}

	ms.Result.add(domain.FoundDir{
		Path:     "apt-get autoremove",
		Size:     installedSize(packages),
		Category: domain.AptAutoremoveCategory,
		Target:   fmt.Sprintf("%d orphaned packages", len(packages)),
		Cmd:      rootCmd("apt-get", "-y", "autoremove", "--purge"),
	})
}

func (ms *MultiScanner) scanJournal() {
	out, err := exec.Command("journalctl", "--disk-usage").Output()
	if err != nil {
		return
	}

	used := parseJournalUsage(string(out))
	if used <= domain.JournalKeepSize+domain.SystemMinSize {
		return
	}

	ms.Result.add(domain.FoundDir{
		Path:     "systemd journal",
		Size:     used - domain.JournalKeepSize,
		Category: domain.JournalCategory,
		Target:   "logs beyond the last 100M",
		Cmd:      rootCmd("journalctl", "--vacuum-size=100M"),
	})
}

// leftovers from crashed or long-gone processes; anything recent stays
func (ms *MultiScanner) scanStaleTemp() {
	cutoff := time.Now().Add(-domain.TempMaxAge)

	for _, root := range []string{"/tmp", "/var/tmp"} {
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if isProtectedTemp(entry.Name()) {
				continue
			}

			info, err := entry.Info()
			if err != nil || info.ModTime().After(cutoff) || !isOwnedByCurrentUser(info) {
				continue
			}

			path := filepath.Join(root, entry.Name())
			size := entrySize(path, info)
			if size < domain.SystemMinSize {
				continue
			}

			ms.Result.add(domain.FoundDir{
				Path:     path,
				Size:     size,
				Category: domain.StaleTempCategory,
				Target:   fmt.Sprintf("untouched for %d days", int(time.Since(info.ModTime()).Hours()/24)),
			})
		}
	}
}

func (ms *MultiScanner) scanCrashReports() {
	for _, root := range []string{"/var/crash", "/var/lib/systemd/coredump"} {
		size := calculateDirSize(root)
		if size < domain.SystemMinSize {
			continue
		}

		ms.Result.add(domain.FoundDir{
			Path:     root,
			Size:     size,
			Category: domain.CrashReportCategory,
			Target:   "crash dumps",
			Cmd:      rootCmd("find", root, "-mindepth", "1", "-delete"),
		})
	}
}

func rootCmd(name string, args ...string) []string {
	cmd := append([]string{name}, args...)
	if os.Geteuid() == 0 {
		return cmd
	}
	// -n keeps the TUI from blocking on a password prompt it cannot render
	return append([]string{"sudo", "-n"}, cmd...)
}

// notes is a comma-separated list like "core,desactivado" or "desactivado,classic"
func isDisabledSnap(notes string) bool {
	for note := range strings.SplitSeq(notes, ",") {
		if note == "disabled" || note == "desactivado" {
			return true
		}
	}
	return false
}

var protectedTemp = []string{".X11-unix", ".XIM-unix", ".ICE-unix", ".font-unix", ".Test-unix", "systemd-private-", "snap-private-tmp", "snap.", ".org.chromium", "dclean"}

func isProtectedTemp(name string) bool {
	for _, prefix := range protectedTemp {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

func entrySize(path string, info os.FileInfo) int64 {
	if info.IsDir() {
		return calculateDirSize(path)
	}
	return info.Size()
}

func installedSize(packages []string) int64 {
	args := append([]string{"-W", "-f", "${Installed-Size}\n"}, packages...)
	out, err := exec.Command("dpkg-query", args...).Output()
	if err != nil {
		return 0
	}

	var total int64
	for field := range strings.FieldsSeq(string(out)) {
		if kb, err := strconv.ParseInt(field, 10, 64); err == nil {
			total += kb * 1024
		}
	}
	return total
}

var journalUnits = map[byte]int64{'K': 1 << 10, 'M': 1 << 20, 'G': 1 << 30, 'T': 1 << 40}

// journalctl reports like "take up 91.8M in the file system"
func parseJournalUsage(out string) int64 {
	for field := range strings.FieldsSeq(out) {
		if len(field) < 2 {
			continue
		}
		unit, known := journalUnits[field[len(field)-1]]
		if !known {
			continue
		}
		num, err := strconv.ParseFloat(field[:len(field)-1], 64)
		if err != nil {
			continue
		}
		return int64(num * float64(unit))
	}
	return 0
}

func isOwnedByCurrentUser(info os.FileInfo) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && int(stat.Uid) == os.Geteuid()
}
