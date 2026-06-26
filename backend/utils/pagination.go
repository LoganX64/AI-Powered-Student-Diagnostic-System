package utils

import "strconv"

func ParsePagination(limitStr, offsetStr string) (int, int) {
	limit := 50
	offset := 0
	if l := limitStr; l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 10000 {
			limit = v
		}
	}
	if o := offsetStr; o != "" {
		if v, err := strconv.Atoi(o); err == nil && v >= 0 {
			offset = v
		}
	}
	return limit, offset
}
