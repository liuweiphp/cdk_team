package service

import "strings"

const maxFileSize = 5 << 20

func normalizeCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

func validCodeChars(code string) bool {
	for _, r := range code {
		if !strings.ContainsRune("ABCDEFGHJKLMNPQRSTUVWXYZ0123456789", r) {
			return false
		}
	}
	return true
}
