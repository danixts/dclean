package scanner

import (
	"encoding/json"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"dclean/internal/domain"
)

var anonymousVolumeName = regexp.MustCompile(`^[0-9a-f]{64}$`)

type dockerVolume struct {
	Name  string
	Links string
	Size  string
}

func (ms *MultiScanner) scanDocker() {
	if _, err := exec.LookPath("docker"); err != nil {
		return
	}
	ms.scanDockerVolumes()
	ms.scanDockerPrune()
}

func (ms *MultiScanner) scanDockerVolumes() {
	out, err := exec.Command("docker", "system", "df", "-v", "--format", "{{json .Volumes}}").Output()
	if err != nil {
		return
	}

	var volumes []dockerVolume
	if json.Unmarshal(out, &volumes) != nil {
		return
	}

	for _, v := range volumes {
		if v.Links != "0" || !anonymousVolumeName.MatchString(v.Name) {
			continue
		}
		ms.Result.add(domain.FoundDir{
			Path:     "docker volume: " + v.Name,
			Size:     parseHumanSize(v.Size),
			Category: domain.DockerOrphanVolumeCategory,
			Target:   v.Name,
			Cmd:      []string{"docker", "volume", "rm", v.Name},
		})
	}
}

func (ms *MultiScanner) scanDockerPrune() {
	out, err := exec.Command("docker", "system", "df", "--format", "{{.Type}}|{{.Reclaimable}}").Output()
	if err != nil {
		return
	}

	var reclaimable int64
	for line := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
		parts := strings.SplitN(line, "|", 2)
		if len(parts) != 2 {
			continue
		}
		if parts[0] == "Images" || parts[0] == "Build Cache" {
			reclaimable += parseHumanSize(parts[1])
		}
	}

	if reclaimable <= 0 {
		return
	}

	ms.Result.add(domain.FoundDir{
		Path:     "docker system prune -a",
		Size:     reclaimable,
		Category: domain.DockerSystemPruneCategory,
		Target:   "unused images + build cache + stopped containers + networks",
		Cmd:      []string{"docker", "system", "prune", "-af"},
	})
}

var sizeUnits = map[string]float64{
	"B":   1,
	"KB":  1e3,
	"KIB": 1024,
	"MB":  1e6,
	"MIB": 1 << 20,
	"GB":  1e9,
	"GIB": 1 << 30,
	"TB":  1e12,
	"TIB": 1 << 40,
}

func parseHumanSize(s string) int64 {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, ' '); i != -1 {
		s = s[:i]
	}
	i := 0
	for i < len(s) && (s[i] == '.' || (s[i] >= '0' && s[i] <= '9')) {
		i++
	}
	num, err := strconv.ParseFloat(s[:i], 64)
	if err != nil {
		return 0
	}
	mult, ok := sizeUnits[strings.ToUpper(s[i:])]
	if !ok {
		return 0
	}
	return int64(num * mult)
}
