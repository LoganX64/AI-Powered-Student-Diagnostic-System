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
