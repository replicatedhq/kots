package util

import (
	"bytes"
)

// ExtractReadableText takes potentially binary data and extracts human-readable
// portions to help with debugging. It tries to preserve meaningful text sections
// while replacing binary sections with "..." placeholders.
func ExtractReadableText(data []byte) string {
	if len(data) == 0 {
		return ""
	}

	var result bytes.Buffer
	var currentText bytes.Buffer
	inReadableSection := false

	for _, b := range data {
		// Check if byte is a readable ASCII character
		if (b >= 32 && b <= 126) || b == '\n' || b == '\t' || b == '\r' {
			currentText.WriteByte(b)
			inReadableSection = true
		} else if inReadableSection {
			// End of a readable section
			if currentText.Len() >= 5 { // Only keep text sections with at least 5 characters
				if result.Len() > 0 {
					result.WriteString(" ... ")
				}
				result.Write(currentText.Bytes())
			}
			currentText.Reset()
			inReadableSection = false
		}
	}

	// Add the last text section if it's long enough
	if currentText.Len() >= 5 {
		if result.Len() > 0 {
			result.WriteString(" ... ")
		}
		result.Write(currentText.Bytes())
	}

	return result.String()
}
