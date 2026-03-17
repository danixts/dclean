package domain

import "fmt"

const (
	bytesPerKB = 1024
	bytesPerMB = bytesPerKB * 1024
	bytesPerGB = bytesPerMB * 1024
)

func FormatSize(bytes int64) string {
	units := []struct {
		threshold int64
		suffix    string
	}{
		{bytesPerGB, "GB"},
		{bytesPerMB, "MB"},
		{bytesPerKB, "KB"},
	}

	for _, unit := range units {
		if bytes >= unit.threshold {
			return fmt.Sprintf("%.1f %s", float64(bytes)/float64(unit.threshold), unit.suffix)
		}
	}
	return fmt.Sprintf("%d B", bytes)
}
