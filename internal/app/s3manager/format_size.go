package s3manager

import "fmt"

// FormatFileSize converts a size in bytes to a human-readable string using the largest appropriate unit.
func FormatFileSize(size int64) string {
	const (
		KB = 1024.0
		MB = 1024.0 * KB
		GB = 1024.0 * MB
		TB = 1024.0 * GB
	)

	sizeF := float64(size)
	switch {
	case sizeF >= TB:
		return fmt.Sprintf("%.2f TB", sizeF/TB)
	case sizeF >= GB:
		return fmt.Sprintf("%.2f GB", sizeF/GB)
	case sizeF >= MB:
		return fmt.Sprintf("%.2f MB", sizeF/MB)
	case sizeF >= KB:
		return fmt.Sprintf("%.2f KB", sizeF/KB)
	default:
		return fmt.Sprintf("%d bytes", size)
	}
}
