package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/pointnoreturn/monitor/libweather"
)

func main() {
	apiKey := os.Getenv("OWM_KEY")
	if len(apiKey) == 0 {
		panic("no API_KEY for OpenWeatherMap")
	}

	gps := os.Getenv("GPS_FIX")
	if gps == "" {
		panic("no GPS_FIX")
	}

	coordsLat, coordsLon, err := parseGPS(gps)
	if err != nil {
		panic(err)
	}

	owm := libweather.NewOpenWeatherMap(apiKey)
	owm.SetCoordinates(coordsLat, coordsLon)

	w, err := owm.GetWeather(context.Background())
	if err != nil {
		panic(err)
	}

	fmt.Println("--- Weather report ---")
	fmt.Printf("GPS: %.6f,%.6f, Name: %s\n", w.Lat, w.Lon, w.Name)

	fmt.Printf("Temp: %.2f°C (feels like %.2f°C)\n",
		w.TempCelsius, w.TempCelsiusFeelsLike)

	fmt.Printf("Condition: %s (%s)\n", w.Main, w.Desc)

	fmt.Printf("Pressure: %d hhMg (%d hPa)\n",
		w.PressureMmhg, w.PressureHpa)

	fmt.Printf("Humidity: %.0f%%\n", w.HumidityPercentage)

	fmt.Printf("Wind: %.1f m/s (gust %.1f m/s)\n",
		w.WindSpeedMs, w.WindGustMs)

	rainText := "no"
	if w.IsRain {
		if w.RainIntensity > 0 {
			rainText = fmt.Sprintf("%.2f mm/h", w.RainIntensity)
		} else {
			rainText = "guessed"
		}
	}

	snowText := "no"
	if w.IsSnow {
		snowText = fmt.Sprintf("%.2f mm/h", w.SnowIntensity)
	}

	fmt.Printf("Rain: %t (%s)\n", w.IsRain, rainText)
	fmt.Printf("Snow: %t (%s)\n", w.IsSnow, snowText)
	fmt.Printf("Cloudiness: %.0f%%\n", w.Cloudiness)
}

func parseGPS(env string) (float32, float32, error) {
	parts := strings.Split(strings.TrimSpace(env), ",")

	if len(parts) < 2 {
		return 0, 0, fmt.Errorf("invalid GPS_FIX, expected lat,lon[,alt]")
	}

	lat64, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 32)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid lat: %w", err)
	}

	lon64, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 32)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid lon: %w", err)
	}

	return float32(lat64), float32(lon64), nil
}
