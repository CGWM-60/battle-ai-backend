package weather

// ApplyWeatherModifiers is the transverse resolver used by all engines.
func ApplyWeatherModifiers(current map[string]float64, eventType string) map[string]float64 {
	switch eventType {
	case "drought":
		current["food"] *= 0.4
		current["water"] *= 0.3
	case "storm":
		// construction and army movement blocked at higher level
	case "heatwave":
		current["energy"] *= 1.3
	case "flood":
		current["food"] *= 0.0
		current["materials"] *= 0.5
	default:
		// clear or unknown = no change
	}
	return current
}
