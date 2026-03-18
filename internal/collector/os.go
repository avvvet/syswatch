package collector

import (
	"bufio"
	"os"
	"strings"
	"syscall"

	"github.com/avvvet/syswatch/pkg/models"
)

// collectOS reads operating system information from /etc/os-release
// and uname syscall.
func collectOS() (models.OS, error) {
	result := models.OS{}

	// Read /etc/os-release for distro info
	osRelease, err := parseOSRelease()
	if err == nil {
		result.Name = osRelease["NAME"]
		result.Version = osRelease["VERSION_ID"]

		// Clean quotes that some distros include
		result.Name = strings.Trim(result.Name, `"`)
		result.Version = strings.Trim(result.Version, `"`)
	}

	// Read kernel version via uname
	var uname syscall.Utsname
	if err := syscall.Uname(&uname); err == nil {
		result.Kernel = int8SliceToString(uname.Release[:])
	}

	// Generate slug for NetBox Platform
	result.Slug = generateSlug(result.Name + "-" + result.Version)

	return result, nil
}

// parseOSRelease reads and parses /etc/os-release into a map.
func parseOSRelease() (map[string]string, error) {
	file, err := os.Open("/etc/os-release")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	result := make(map[string]string)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			result[parts[0]] = parts[1]
		}
	}

	return result, scanner.Err()
}

// int8SliceToString converts a uname field to a Go string.
func int8SliceToString(s []int8) string {
	b := make([]byte, 0, len(s))
	for _, v := range s {
		if v == 0 {
			break
		}
		b = append(b, byte(v))
	}
	return string(b)
}

// generateSlug converts a name to a NetBox compatible slug.
// e.g. "Ubuntu 22.04" → "ubuntu-22-04"
func generateSlug(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, ".", "-")
	s = strings.ReplaceAll(s, "_", "-")

	// Remove any characters that are not alphanumeric or hyphen
	var result strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}

	// Clean up multiple consecutive hyphens
	slug := result.String()
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}

	return strings.Trim(slug, "-")
}
