package service

func CalculateSQI(score int) float64 {

	if score >= 90 {
		return 4.0
	} else if score >= 75 {
		return 3.0
	} else if score >= 60 {
		return 2.0
	}
	return 1.0
}
