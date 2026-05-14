package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/pointnoreturn/monitor/libweather"
)

func makeWeatherProvider() libweather.WeatherProvider {
	apiKey := os.Getenv("OWM_KEY")
	if len(apiKey) == 0 {
		fmt.Fprintln(os.Stderr, "WARN: no OWM_KEY, api key for OpenWeatherMap. Weather cannot work")
		return nil
	}

	gps := os.Getenv("GPS_FIX")
	if gps == "" {
		fmt.Fprintln(os.Stderr, "WARN: no GPS_FIX, weather cannot work")
		return nil
	}

	coordsLat, coordsLon, err := parseGPS(gps)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR: Failed to parse GPS_FIX")
		panic(err)
	}

	owm := libweather.NewOpenWeatherMap(apiKey)
	owm.SetCoordinates(coordsLat, coordsLon)

	fmt.Println("Weather provider ready (OpenWeatherMap)")

	return owm
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
