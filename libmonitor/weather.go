package libmonitor

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/pointnoreturn/monitor/libweather"
)

func makeWeatherProvider(log *slog.Logger) libweather.WeatherProvider {
	apiKey := os.Getenv("OWM_KEY")
	if len(apiKey) == 0 {
		log.Warn("[weather] no OWM_KEY, api key for OpenWeatherMap. Not initializing weather")
		return nil
	}

	gps := os.Getenv("GPS_FIX")
	if gps == "" {
		log.Warn("[weather] no GPS_FIX. Not initializing weather")
		return nil
	}

	coordsLat, coordsLon, err := parseGPS(gps)
	if err != nil {
		e := fmt.Sprintf("[weather] Failed to parse GPS_FIX, provided coordinates for weather: '%s'", gps)
		log.Error(e)
		panic(e)
	}

	owm := libweather.NewOpenWeatherMap(apiKey)
	owm.SetCoordinates(coordsLat, coordsLon)

	log.Info("[weather] Weather provider ready (OpenWeatherMap)")

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
