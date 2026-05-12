package libsnake

import (
	"context"
	"fmt"
	"time"

	"github.com/pointnoreturn/snake/libweather"
)

type Telemeter struct {
	conn    *MeshtasticClient
	weather libweather.WeatherProvider
}

func NewTelemeter(conn *MeshtasticClient, weather libweather.WeatherProvider) *Telemeter {
	return &Telemeter{
		conn:    conn,
		weather: weather,
	}
}

func (t *Telemeter) RunLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	fmt.Println("Telemeter loop is running")

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			t.update()
		}
	}
}

func (t *Telemeter) update() {
	// TODO: generate telemetry
}
