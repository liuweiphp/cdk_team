// Package sanitize 提供 HTML 净化,防止 XSS
package sanitize

import "github.com/microcosm-cc/bluemonday"

var policy = bluemonday.UGCPolicy()

// HTML 使用 bluemonday UGC 策略净化 HTML
func HTML(html string) string {
	return policy.Sanitize(html)
}
