package entity

import "time"

// WeatherForecastRequest is the normalized weather forecast input used by the
// provider layer.
type WeatherForecastRequest struct {
	Destination string    `json:"destination"`
	StartDate   time.Time `json:"startDate"`
	Days        int       `json:"days"`
}

// WeatherDay is one daily forecast entry.
type WeatherDay struct {
	Date                string   `json:"date"`
	Condition           string   `json:"condition"`
	TemperatureMinC     float64  `json:"temperatureMinC"`
	TemperatureMaxC     float64  `json:"temperatureMaxC"`
	PrecipitationChance int      `json:"precipitationChance"`
	WindSpeedKph        float64  `json:"windSpeedKph"`
	Summary             string   `json:"summary"`
	Warnings            []string `json:"warnings,omitempty"`
}

// WeatherForecast is the response returned by the weather API.
type WeatherForecast struct {
	Destination string       `json:"destination"`
	Provider    string       `json:"provider"`
	Days        []WeatherDay `json:"days"`
}
