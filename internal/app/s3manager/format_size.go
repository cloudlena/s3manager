package s3manager

import "fmt"

// formatFileSize converts a size in bytes to a human-readable string using the
// largest appropriate unit.
func formatFileSize(size int64) string {
	const (
		kb = 1024.0
		mb = 1024.0 * kb
		gb = 1024.0 * mb
		tb = 1024.0 * gb
	)

	sizeF := float64(size)
	switch {
	case sizeF >= tb:
		return fmt.Sprintf("%.2f TB", sizeF/tb)
	case sizeF >= gb:
		return fmt.Sprintf("%.2f GB", sizeF/gb)
	case sizeF >= mb:
		return fmt.Sprintf("%.2f MB", sizeF/mb)
	case sizeF >= kb:
		return fmt.Sprintf("%.2f KB", sizeF/kb)
	default:
		return fmt.Sprintf("%d bytes", size)
	}
}
