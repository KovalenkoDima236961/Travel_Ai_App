package weathercontext

// WeatherDay mirrors External Integrations Service weather forecast day output.
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

// WeatherForecast is optional AI context. It is never persisted by Trip Service.
type WeatherForecast struct {
	Destination string       `json:"destination"`
	Provider    string       `json:"provider,omitempty"`
	Days        []WeatherDay `json:"days"`
}
