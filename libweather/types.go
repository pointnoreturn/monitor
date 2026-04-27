package libweather

import "context"

type WeatherMain string

const (
	WeatherThunderstorm WeatherMain = "Thunderstorm"
	WeatherDrizzle      WeatherMain = "Drizzle"
	WeatherRain         WeatherMain = "Rain"
	WeatherSnow         WeatherMain = "Snow"
	WeatherAtmosphere   WeatherMain = "Atmosphere" // mist, fog, etc
	WeatherClear        WeatherMain = "Clear"
	WeatherClouds       WeatherMain = "Clouds"
	WeatherTornado      WeatherMain = "Tornado" // rare
	WeatherSquall       WeatherMain = "Squall"  // rare
)

// core weather struct
type Weather struct {
	Lat, Lon float32
	Name     string

	Main WeatherMain
	Desc string // from OWM "description"

	TempCelsius          float32
	TempCelsiusFeelsLike float32

	WindSpeedMs float32
	WindGustMs  float32

	PressureHpa  int32
	PressureMmhg int32

	HumidityPercentage float32
	Cloudiness         float32

	IsSnow        bool
	SnowIntensity float32

	IsRain        bool
	RainIntensity float32
}

// provider interface
type WeatherProvider interface {
	SetCoordinates(lat, lon float32)
	GetWeather(ctx context.Context) (Weather, error)
}
