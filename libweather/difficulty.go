package libweather

// calculate weather as signle floating point with range from 0 to 3
// representing the RF signal difficulty in current conditions (how weather degrades radio link quality)
func (w Weather) RadioDifficulty() float32 {
	precipationSevereness1h := func(intensity1h, threshold1, threshold2, threshold3 float32) float32 {
		if intensity1h <= threshold1 {
			return 0
		} else if intensity1h <= threshold2 {
			return 1
		} else if intensity1h <= threshold3 {
			return 2
		} else {
			return 3
		}
	}

	switch w.Main {
	case WeatherRain:
		return precipationSevereness1h(w.RainIntensity, 0.8, 5.0, 10.0)
	case WeatherSnow:
		return precipationSevereness1h(w.SnowIntensity, 2.0, 7.0, 15.0)
	case WeatherThunderstorm:
		// Thunderstorms create "Impulsive Noise" (EMI).
		// Even with light rain, the lightning/static should trigger Level 2.
		if w.RainIntensity > 10.0 {
			return 3 // Severe storm + high attenuation
		}
		return 2
	case WeatherSquall:
		return 2
	case WeatherTornado:
		return 3
	case WeatherDrizzle:
	case WeatherAtmosphere:
		return 1
	}

	// rain is not reported directly, but signs of it => should be light difficulty
	if w.Cloudiness > 85 {
		if w.HumidityPercentage > 90 && w.PressureHpa < 1000 {
			return 0.5
		} else if w.HumidityPercentage > 85 && w.PressureHpa < 1000 {
			return 0.3
		}
	}
	return 0
}
