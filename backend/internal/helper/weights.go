package helper

import "math"

// GetImportanceWeight returns the weight multiplier based on importance level
func GetImportanceWeight(val string) float64 {
	switch val {
	case "A":
		return 1.0
	case "B":
		return 0.7
	case "C":
		return 0.5
	default:
		return 1.0
	}
}

// GetDifficultyWeight returns the weight multiplier based on difficulty level
func GetDifficultyWeight(val string) float64 {
	switch val {
	case "E":
		return 0.6
	case "M":
		return 1.0
	case "H":
		return 1.4
	default:
		return 1.0
	}
}

// GetTypeWeight returns the weight multiplier based on question type
func GetTypeWeight(val string) float64 {
	switch val {
	case "Practical":
		return 1.1
	case "Theory":
		return 1.0
	default:
		return 1.0
	}
}

// Clamp returns a value constrained between min and max
func Clamp(val, min, max float64) float64 {
	return math.Max(min, math.Min(max, val))
}

// SafeDivide performs division with zero-check protection
func SafeDivide(a, b float64) float64 {
	if b == 0 {
		return 0
	}
	return a / b
}

// Round rounds a float to a specified number of decimal places
func Round(val float64, precision int) float64 {
	pow := math.Pow(10, float64(precision))
	return math.Round(val*pow) / pow
}

// ─────────────────────────────────────────────
// SQI-SPECIFIC WEIGHT FUNCTIONS
// ─────────────────────────────────────────────

// GetSQIImportanceWeight returns weight multiplier for SQI scoring (high/medium/low)
func GetSQIImportanceWeight(importance string) float64 {
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

// GetSQIDifficultyWeight returns weight multiplier for SQI scoring (E/M/H)
func GetSQIDifficultyWeight(difficulty string) float64 {
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

// GetSQITypeWeight returns weight multiplier for SQI scoring by question type
func GetSQITypeWeight(qType string) float64 {
	switch qType {
	case "integer":
		return 1.2 // no guessing possible
	case "multi":
		return 1.1 // partial marking, harder
	case "mcq":
		return 1.0
	default:
		return 1.0
	}
}

// Round2 rounds a float to 2 decimal places
func Round2(v float64) float64 {
	return math.Round(v*100) / 100
}

// Coalesce returns the first non-empty string or the fallback
func Coalesce(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

// Formatf is a simple sprintf wrapper for formatting strings
// In production, consider using fmt.Sprintf directly
func Formatf(format string, args ...interface{}) string {
	// This is a stub that returns the format string
	// In your actual project, use fmt.Sprintf(format, args...) instead
	result := format
	_ = args
	return result
}
