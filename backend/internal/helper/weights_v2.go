package helper

import (
	"fmt"
	"math"
)

// GetSQIV2ImportanceWeight returns the v2 SQI weight multiplier for importance.
func GetSQIV2ImportanceWeight(importance string) float64 {
	switch importance {
	case "high":
		return 1.5
	case "medium":
		return 1.0
	case "low":
		return 0.6
	default:
		return 1.0
	}
}

// GetSQIV2DifficultyWeight returns the v2 SQI weight multiplier for difficulty.
func GetSQIV2DifficultyWeight(difficulty string) float64 {
	switch difficulty {
	case "E":
		return 0.8
	case "M":
		return 1.0
	case "H":
		return 1.3
	default:
		return 1.0
	}
}

// GetSQIV2TypeWeight returns the v2 SQI weight multiplier for question type.
func GetSQIV2TypeWeight(qType string) float64 {
	switch qType {
	case "integer":
		return 1.2
	case "multi":
		return 1.1
	case "mcq":
		return 1.0
	default:
		return 1.0
	}
}

// BuildAnswerMapV2 builds an int-keyed lookup map without tying helper to services types.
func BuildAnswerMapV2[T any](items []T, keyFn func(T) int) map[int]T {
	m := make(map[int]T, len(items))
	for _, item := range items {
		m[keyFn(item)] = item
	}
	return m
}

// SafeDivideV2 performs division with zero-check protection.
func SafeDivideV2(a, b float64) float64 {
	if b == 0 {
		return 0
	}
	return a / b
}

// ClampV2 returns v constrained between min and max.
func ClampV2(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// Round2V2 rounds a float to 2 decimal places.
func Round2V2(v float64) float64 {
	return math.Round(v*100) / 100
}

// CoalesceV2 returns s unless it is empty, then returns fallback.
func CoalesceV2(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

// FormatfV2 formats a string using fmt.Sprintf.
func FormatfV2(format string, args ...interface{}) string {
	return fmt.Sprintf(format, args...)
}
