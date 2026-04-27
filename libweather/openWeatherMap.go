package libweather

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// to be used to parse weather main
func ParseWeatherMain(s string) WeatherMain {
	switch s {
	case "Thunderstorm":
		return WeatherThunderstorm
	case "Drizzle":
		return WeatherDrizzle
	case "Rain":
		return WeatherRain
	case "Snow":
		return WeatherSnow
	case "Atmosphere":
		return WeatherAtmosphere
	case "Clear":
		return WeatherClear
	case "Clouds":
		return WeatherClouds
	default:
		return WeatherMain(s) // fallback (future-proof)
	}
}

type OpenWeatherMap struct {
	apiKey string
	lat    float32
	lon    float32
}

func NewOpenWeatherMap(apiKey string) *OpenWeatherMap {
	return &OpenWeatherMap{apiKey: apiKey}
}

func (o *OpenWeatherMap) SetCoordinates(lat, lon float32) {
	o.lat = lat
	o.lon = lon
}

// raw response structs (minimal)
type owmResponse struct {
	Name  string `json:"name"`
	Coord struct {
		Lat float32 `json:"lat"`
		Lon float32 `json:"lon"`
	} `json:"coord"`
	Main struct {
		Temp      float32 `json:"temp"`
		FeelsLike float32 `json:"feels_like"`
		Pressure  int32   `json:"pressure"`
		Humidity  float32 `json:"humidity"`
	} `json:"main"`
	Wind struct {
		Speed float32 `json:"speed"`
		Gust  float32 `json:"gust"`
	} `json:"wind"`
	Clouds struct {
		All float32 `json:"all"`
	} `json:"clouds"`
	Weather []struct {
		Main        string `json:"main"`
		Description string `json:"description"`
	} `json:"weather"`
	Rain map[string]float32 `json:"rain"`
	Snow map[string]float32 `json:"snow"`
}

func (o *OpenWeatherMap) GetWeather(ctx context.Context) (Weather, error) {
	if o.lat == 0 || o.lon == 0 {
		return Weather{}, fmt.Errorf("no coordinates set")
	}

	url := fmt.Sprintf(
		"https://api.openweathermap.org/data/2.5/weather?lat=%f&lon=%f&appid=%s&units=metric",
		o.lat, o.lon, o.apiKey,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Weather{}, err
	}

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return Weather{}, err
	}
	defer resp.Body.Close()

	var data owmResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return Weather{}, err
	}

	w := Weather{
		Lat:  data.Coord.Lat,
		Lon:  data.Coord.Lon,
		Name: data.Name,

		Main: ParseWeatherMain(data.Weather[0].Main),
		Desc: data.Weather[0].Description,

		TempCelsius:          data.Main.Temp,
		TempCelsiusFeelsLike: data.Main.FeelsLike,

		WindSpeedMs: data.Wind.Speed,
		WindGustMs:  data.Wind.Gust,

		PressureHpa:  data.Main.Pressure,
		PressureMmhg: int32(float32(data.Main.Pressure) * 0.75006),

		HumidityPercentage: data.Main.Humidity,
		Cloudiness:         data.Clouds.All,
	}

	readIntensity := func(data map[string]float32) float32 {
		if rain1h, ok := data["1h"]; ok {
			w.RainIntensity = rain1h
		} else if rain3h, ok := data["3h"]; ok {
			w.RainIntensity = rain3h / 3
		}
		return 0.1
	}

	// rain/snow
	if data.Rain != nil {
		w.IsRain = true
		w.RainIntensity = readIntensity(data.Rain)
	}
	if data.Snow != nil {
		w.SnowIntensity = readIntensity(data.Snow)
		w.IsSnow = true
	}

	return w, nil
}
