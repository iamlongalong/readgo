package readgo

// Common constants for file operations
const (
	maxFileSize = 10 * 1024 * 1024 // 10MB
)

// isAllowedExtension checks if the file extension is allowed
func isAllowedExtension(ext string) bool {
	allowedExts := map[string]bool{
		".go":  true,
		".mod": true,
		".sum": true,
	}
	return allowedExts[ext]
}
