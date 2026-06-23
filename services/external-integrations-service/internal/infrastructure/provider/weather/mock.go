package weather

import (
	"context"
	"hash/fnv"
	"math"
	"strings"
	"time"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/domain/entity"
)

const providerName = "mock"

// MockWeatherProvider produces deterministic destination-aware daily forecasts.
// It does not call third-party APIs.
type MockWeatherProvider struct{}

func NewMockWeatherProvider() *MockWeatherProvider {
	return &MockWeatherProvider{}
}

// GetForecast returns exactly req.Days daily forecasts starting at req.StartDate.
func (p *MockWeatherProvider) GetForecast(_ context.Context, req entity.WeatherForecastRequest) (*entity.WeatherForecast, error) {
	days := make([]entity.WeatherDay, 0, req.Days)
	for i := 0; i < req.Days; i++ {
		date := req.StartDate.AddDate(0, 0, i)
		day := forecastDay(req.Destination, date, i)
		days = append(days, day)
	}

	return &entity.WeatherForecast{
		Destination: req.Destination,
		Provider:    providerName,
		Days:        days,
	}, nil
}

func forecastDay(destination string, date time.Time, offset int) entity.WeatherDay {
	normalizedDestination := normalizeDestination(destination)
	month := int(date.Month())
	var day entity.WeatherDay

	switch normalizedDestination {
	case "rome":
		day = romeForecast(date, offset)
	case "paris":
		day = parisForecast(date, offset)
	case "vienna":
		day = moderateSeasonalForecast(date, offset, "Vienna", 0)
	case "bratislava":
		day = moderateSeasonalForecast(date, offset, "Bratislava", 1)
	default:
		day = genericForecast(date, offset, destination)
	}

	seasonalAdjustment := seasonalDailyAdjustment(destination, date)
	day.TemperatureMinC = round1(day.TemperatureMinC + seasonalAdjustment)
	day.TemperatureMaxC = round1(day.TemperatureMaxC + seasonalAdjustment)

	if month == 12 || month <= 2 {
		if day.TemperatureMaxC <= 5 {
			day.Condition = "cold"
			day.Summary = "Cold day"
		}
	}

	day.Warnings = weatherWarnings(day)
	return day
}

func romeForecast(date time.Time, offset int) entity.WeatherDay {
	month := int(date.Month())
	var condition, summary string
	var minTemp, maxTemp float64
	var precipitation int
	var wind float64

	if month >= 6 && month <= 8 {
		maxTemp = 33 + float64((stableIndex("rome", date)+offset)%4)
		minTemp = maxTemp - 10
		precipitation = 5 + ((offset + stableIndex("rome-rain", date)) % 12)
		wind = 8 + float64((offset+2)%5)
		if maxTemp >= 35 {
			condition = "hot"
			summary = "Hot and sunny"
		} else {
			condition = "sunny"
			summary = "Sunny and hot"
		}
	} else {
		maxTemp = seasonalBaseMax(date) + 5 + float64(offset%3)
		minTemp = maxTemp - 8
		precipitation = 20 + ((offset + stableIndex("rome-rain", date)) % 25)
		wind = 10 + float64(offset%7)
		condition = "partly_cloudy"
		summary = "Partly cloudy and mild"
	}

	return entity.WeatherDay{
		Date:                date.Format("2006-01-02"),
		Condition:           condition,
		TemperatureMinC:     minTemp,
		TemperatureMaxC:     maxTemp,
		PrecipitationChance: precipitation,
		WindSpeedKph:        wind,
		Summary:             summary,
	}
}

func parisForecast(date time.Time, offset int) entity.WeatherDay {
	maxTemp := seasonalBaseMax(date) + 1 + float64((offset+stableIndex("paris", date))%3)
	minTemp := maxTemp - 7
	precipitation := 35 + ((offset*17 + stableIndex("paris-rain", date)) % 45)
	wind := 12 + float64((offset*3)%12)

	condition := "partly_cloudy"
	summary := "Partly cloudy and mild"
	if precipitation >= 70 {
		condition = "rain"
		summary = "Rain likely"
	} else if precipitation >= 55 {
		condition = "light_rain"
		summary = "Light rain likely"
	} else if precipitation < 45 {
		condition = "cloudy"
		summary = "Cloudy and mild"
	}

	return entity.WeatherDay{
		Date:                date.Format("2006-01-02"),
		Condition:           condition,
		TemperatureMinC:     minTemp,
		TemperatureMaxC:     maxTemp,
		PrecipitationChance: precipitation,
		WindSpeedKph:        wind,
		Summary:             summary,
	}
}

func moderateSeasonalForecast(date time.Time, offset int, city string, shift int) entity.WeatherDay {
	maxTemp := seasonalBaseMax(date) + float64(shift) + float64((offset+stableIndex(city, date))%4)
	minTemp := maxTemp - 8
	precipitation := 20 + ((offset*13 + stableIndex(city+"-rain", date)) % 45)
	wind := 10 + float64((offset*5+shift)%22)
	condition := "partly_cloudy"
	summary := "Partly cloudy and mild"

	if precipitation >= 60 {
		condition = "light_rain"
		summary = "Light rain likely"
	} else if wind >= 28 {
		condition = "windy"
		summary = "Cool and windy"
	} else if precipitation >= 45 {
		condition = "cloudy"
		summary = "Cloudy and moderate"
	}

	return entity.WeatherDay{
		Date:                date.Format("2006-01-02"),
		Condition:           condition,
		TemperatureMinC:     minTemp,
		TemperatureMaxC:     maxTemp,
		PrecipitationChance: precipitation,
		WindSpeedKph:        wind,
		Summary:             summary,
	}
}

func genericForecast(date time.Time, offset int, destination string) entity.WeatherDay {
	maxTemp := seasonalBaseMax(date) + float64((offset+stableIndex(destination, date))%5-2)
	minTemp := maxTemp - 7
	precipitation := 25 + ((offset*11 + stableIndex(destination+"-rain", date)) % 35)
	wind := 9 + float64((offset+stableIndex(destination+"-wind", date))%18)
	condition := "partly_cloudy"
	summary := "Partly cloudy and mild"

	if precipitation >= 60 {
		condition = "light_rain"
		summary = "Light rain likely"
	} else if wind >= 30 {
		condition = "windy"
		summary = "Cool and windy"
	}

	return entity.WeatherDay{
		Date:                date.Format("2006-01-02"),
		Condition:           condition,
		TemperatureMinC:     minTemp,
		TemperatureMaxC:     maxTemp,
		PrecipitationChance: precipitation,
		WindSpeedKph:        wind,
		Summary:             summary,
	}
}

func seasonalBaseMax(date time.Time) float64 {
	switch date.Month() {
	case time.December, time.January, time.February:
		return 4
	case time.March, time.April, time.November:
		return 14
	case time.May, time.September, time.October:
		return 20
	default:
		return 25
	}
}

func seasonalDailyAdjustment(destination string, date time.Time) float64 {
	return float64(stableIndex(destination+"-"+date.Format("2006-01-02"), date)%3) - 1
}

func weatherWarnings(day entity.WeatherDay) []string {
	warnings := make([]string, 0, 4)
	if day.TemperatureMaxC >= 32 {
		warnings = append(warnings, "High heat: avoid long outdoor walks at midday")
	}
	if day.PrecipitationChance >= 60 {
		warnings = append(warnings, "Rain likely: consider indoor alternatives")
	}
	if day.WindSpeedKph >= 35 {
		warnings = append(warnings, "Windy: viewpoints and exposed areas may be uncomfortable")
	}
	if day.TemperatureMaxC <= 5 {
		warnings = append(warnings, "Cold day: plan warm indoor breaks")
	}
	return warnings
}

func normalizeDestination(destination string) string {
	return strings.ToLower(strings.TrimSpace(destination))
}

func stableIndex(seed string, date time.Time) int {
	h := fnv.New32a()
	_, _ = h.Write([]byte(strings.ToLower(strings.TrimSpace(seed))))
	_, _ = h.Write([]byte(date.Format("2006-01-02")))
	return int(h.Sum32() % 32767)
}

func round1(value float64) float64 {
	return math.Round(value*10) / 10
}
