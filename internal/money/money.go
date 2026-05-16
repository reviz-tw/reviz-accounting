package money

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseCents converts a user-entered decimal string ("1234.5", "-30000") to
// signed int64 cents. Whitespace, commas, and a trailing/leading currency
// symbol are tolerated. Empty string is treated as 0.
func ParseCents(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	s = strings.ReplaceAll(s, ",", "")
	s = strings.TrimPrefix(s, "NT$")
	s = strings.TrimPrefix(s, "$")
	neg := false
	if strings.HasPrefix(s, "-") {
		neg = true
		s = s[1:]
	} else if strings.HasPrefix(s, "+") {
		s = s[1:]
	}
	parts := strings.SplitN(s, ".", 2)
	whole := parts[0]
	frac := "00"
	if len(parts) == 2 {
		frac = parts[1]
		if len(frac) > 2 {
			frac = frac[:2]
		}
		for len(frac) < 2 {
			frac += "0"
		}
	}
	if whole == "" {
		whole = "0"
	}
	w, err := strconv.ParseInt(whole, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid amount %q: %w", s, err)
	}
	f, err := strconv.ParseInt(frac, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid amount %q: %w", s, err)
	}
	c := w*100 + f
	if neg {
		c = -c
	}
	return c, nil
}

// FormatCents renders int64 cents as a 2-decimal string, e.g. "1234.50".
func FormatCents(c int64) string {
	neg := c < 0
	if neg {
		c = -c
	}
	s := fmt.Sprintf("%d.%02d", c/100, c%100)
	if neg {
		s = "-" + s
	}
	return s
}

// FormatCentsThousands renders cents with thousand separators, e.g. "1,234.50".
func FormatCentsThousands(c int64) string {
	s := FormatCents(c)
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}
	dot := strings.Index(s, ".")
	whole, frac := s[:dot], s[dot:]
	var b strings.Builder
	n := len(whole)
	for i, r := range whole {
		if i > 0 && (n-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	out := b.String() + frac
	if neg {
		out = "-" + out
	}
	return out
}
