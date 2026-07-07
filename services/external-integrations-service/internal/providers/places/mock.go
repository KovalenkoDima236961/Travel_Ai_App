package places

import (
	"context"
	"strings"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/domain/entity"
)

const maxMockResults = 10

type mockPlace struct {
	City string
	entity.Place
}

// MockPlaceProvider returns deterministic place data for local development and
// tests. It performs no network calls.
type MockPlaceProvider struct {
	places []mockPlace
	byID   map[string]entity.Place
}

func NewMockPlaceProvider() *MockPlaceProvider {
	data := mockPlaces()
	byID := make(map[string]entity.Place, len(data))
	for _, item := range data {
		byID[item.ProviderPlaceID] = item.Place
	}
	return &MockPlaceProvider{places: data, byID: byID}
}

func (p *MockPlaceProvider) SearchPlaces(_ context.Context, query string, destination string) ([]entity.Place, error) {
	query = normalizeSearchText(query)
	destination = normalizeSearchText(destination)

	candidates := p.filterByDestination(destination)
	matches := make([]entity.Place, 0, maxMockResults)
	for _, candidate := range candidates {
		if mockPlaceMatches(candidate.Place, query) {
			matches = append(matches, candidate.Place)
			if len(matches) == maxMockResults {
				return matches, nil
			}
		}
	}

	if len(matches) > 0 || destination == "" {
		return matches, nil
	}

	for _, candidate := range candidates {
		matches = append(matches, candidate.Place)
		if len(matches) == 3 {
			break
		}
	}
	return matches, nil
}

func (p *MockPlaceProvider) GetPlaceDetails(_ context.Context, providerPlaceID string) (*entity.Place, error) {
	place, ok := p.byID[strings.TrimSpace(providerPlaceID)]
	if !ok {
		return nil, nil
	}
	return &place, nil
}

func (p *MockPlaceProvider) filterByDestination(destination string) []mockPlace {
	if destination == "" {
		return p.places
	}

	filtered := make([]mockPlace, 0)
	for _, item := range p.places {
		city := normalizeSearchText(item.City)
		if strings.Contains(city, destination) || strings.Contains(destination, city) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func mockPlaceMatches(place entity.Place, query string) bool {
	if query == "" {
		return false
	}
	searchable := normalizeSearchText(strings.Join([]string{
		place.Name,
		place.Address,
		place.Category,
	}, " "))
	return strings.Contains(searchable, query)
}

func normalizeSearchText(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer(
		"á", "a",
		"à", "a",
		"â", "a",
		"ä", "a",
		"ã", "a",
		"å", "a",
		"č", "c",
		"ć", "c",
		"ď", "d",
		"é", "e",
		"è", "e",
		"ê", "e",
		"ë", "e",
		"í", "i",
		"ì", "i",
		"î", "i",
		"ï", "i",
		"ľ", "l",
		"ĺ", "l",
		"ň", "n",
		"ó", "o",
		"ò", "o",
		"ô", "o",
		"ö", "o",
		"ő", "o",
		"ř", "r",
		"š", "s",
		"ť", "t",
		"ú", "u",
		"ù", "u",
		"û", "u",
		"ü", "u",
		"ý", "y",
		"ž", "z",
	)
	return replacer.Replace(value)
}

func mockPlaces() []mockPlace {
	return []mockPlace{
		mock("Rome", "mock-colosseum-rome", "Colosseum", "Piazza del Colosseo, 1, 00184 Roma RM, Italy", 41.8902, 12.4922, 4.7, 120000, "landmark", "https://example.com/colosseum"),
		mock("Rome", "mock-roman-forum-rome", "Roman Forum", "Via della Salara Vecchia, 5/6, 00186 Roma RM, Italy", 41.8925, 12.4853, 4.7, 67000, "historic site", "https://example.com/roman-forum"),
		mock("Rome", "mock-trevi-fountain-rome", "Trevi Fountain", "Piazza di Trevi, 00187 Roma RM, Italy", 41.9009, 12.4833, 4.8, 185000, "landmark", "https://example.com/trevi-fountain"),
		mock("Rome", "mock-pantheon-rome", "Pantheon", "Piazza della Rotonda, 00186 Roma RM, Italy", 41.8986, 12.4768, 4.8, 150000, "historic site", "https://example.com/pantheon"),
		mock("Rome", "mock-trastevere-trattoria-rome", "Trastevere Local Trattoria", "Via della Scala, 31, 00153 Roma RM, Italy", 41.8894, 12.4708, 4.5, 2800, "restaurant", "https://example.com/trastevere-trattoria"),
		mock("Rome", "mock-villa-borghese-rome", "Villa Borghese", "Piazzale Napoleone I, 00197 Roma RM, Italy", 41.9142, 12.4922, 4.6, 82000, "park", "https://example.com/villa-borghese"),
		mock("Rome", "mock-testaccio-market-rome", "Testaccio Market", "Via Aldo Manuzio, 66b, 00153 Roma RM, Italy", 41.8769, 12.4754, 4.5, 7800, "market", "https://example.com/testaccio-market"),
		mock("Paris", "mock-eiffel-tower-paris", "Eiffel Tower", "Champ de Mars, 5 Av. Anatole France, 75007 Paris, France", 48.8584, 2.2945, 4.7, 390000, "landmark", "https://example.com/eiffel-tower"),
		mock("Paris", "mock-louvre-museum-paris", "Louvre Museum", "Rue de Rivoli, 75001 Paris, France", 48.8606, 2.3376, 4.7, 310000, "museum", "https://example.com/louvre"),
		mock("Paris", "mock-montmartre-paris", "Montmartre", "Montmartre, 75018 Paris, France", 48.8867, 2.3431, 4.7, 90000, "neighborhood", "https://example.com/montmartre"),
		mock("Paris", "mock-luxembourg-gardens-paris", "Luxembourg Gardens", "75006 Paris, France", 48.8462, 2.3372, 4.7, 62000, "park", "https://example.com/luxembourg-gardens"),
		mock("Paris", "mock-le-marais-cafe-paris", "Le Marais Cafe", "Rue Vieille du Temple, 75004 Paris, France", 48.8589, 2.3615, 4.4, 2100, "cafe", "https://example.com/le-marais-cafe"),
		mock("Paris", "mock-musee-dorsay-paris", "Musée d'Orsay", "Esplanade Valéry Giscard d'Estaing, 75007 Paris, France", 48.86, 2.3266, 4.8, 98000, "museum", "https://example.com/musee-dorsay"),
		mock("Vienna", "mock-schonbrunn-palace-vienna", "Schönbrunn Palace", "Schönbrunner Schloßstraße 47, 1130 Wien, Austria", 48.1845, 16.3122, 4.7, 146000, "palace", "https://example.com/schonbrunn-palace"),
		mock("Vienna", "mock-st-stephens-cathedral-vienna", "St. Stephen's Cathedral", "Stephansplatz 3, 1010 Wien, Austria", 48.2084, 16.3731, 4.7, 105000, "cathedral", "https://example.com/st-stephens-cathedral"),
		mock("Vienna", "mock-naschmarkt-vienna", "Naschmarkt", "Naschmarkt, 1060 Wien, Austria", 48.1982, 16.3615, 4.4, 52000, "market", "https://example.com/naschmarkt"),
		mock("Vienna", "mock-belvedere-palace-vienna", "Belvedere Palace", "Prinz-Eugen-Straße 27, 1030 Wien, Austria", 48.1915, 16.3809, 4.7, 71000, "museum", "https://example.com/belvedere-palace"),
		mock("Vienna", "mock-prater-vienna", "Prater", "1020 Wien, Austria", 48.2167, 16.3958, 4.5, 118000, "park", "https://example.com/prater"),
		mock("Vienna", "mock-cafe-central-vienna", "Café Central", "Herrengasse 14, 1010 Wien, Austria", 48.2104, 16.3656, 4.4, 21000, "cafe", "https://example.com/cafe-central"),
		mock("Bratislava", "mock-bratislava-castle-bratislava", "Bratislava Castle", "Hrad, 811 06 Bratislava, Slovakia", 48.1421, 17.1002, 4.5, 56000, "castle", "https://example.com/bratislava-castle"),
		mock("Bratislava", "mock-old-town-bratislava", "Old Town", "Staré Mesto, Bratislava, Slovakia", 48.1486, 17.1077, 4.6, 41000, "neighborhood", "https://example.com/bratislava-old-town"),
		mock("Bratislava", "mock-blue-church-bratislava", "Blue Church", "Bezručova 2, 811 09 Bratislava, Slovakia", 48.1437, 17.1163, 4.6, 18000, "church", "https://example.com/blue-church"),
		mock("Bratislava", "mock-ufo-tower-bratislava", "UFO Tower", "Most SNP 1, 851 01 Bratislava, Slovakia", 48.1367, 17.1048, 4.4, 27000, "viewpoint", "https://example.com/ufo-tower"),
		mock("Bratislava", "mock-slovak-national-theatre-bratislava", "Slovak National Theatre", "Pribinova 17, 819 01 Bratislava, Slovakia", 48.1418, 17.1214, 4.7, 9000, "theatre", "https://example.com/slovak-national-theatre"),
		mock("Bratislava", "mock-urban-house-cafe-bratislava", "Urban House Cafe", "Laurinská 14, 811 01 Bratislava, Slovakia", 48.1447, 17.1106, 4.3, 3800, "cafe", "https://example.com/urban-house-cafe"),
	}
}

func mock(city, id, name, address string, lat, lng, rating float64, ratingCount int, category, website string) mockPlace {
	return mockPlace{
		City: city,
		Place: entity.Place{
			Provider:        "mock",
			ProviderPlaceID: id,
			Name:            name,
			Address:         address,
			Latitude:        floatPtr(lat),
			Longitude:       floatPtr(lng),
			Rating:          floatPtr(rating),
			RatingCount:     intPtr(ratingCount),
			MapURL:          "https://maps.example.com/" + id,
			Category:        category,
			Website:         website,
			OpeningHours:    mockOpeningHours(id, category),
		},
	}
}

func mockOpeningHours(id, category string) []entity.OpeningHoursInterval {
	switch id {
	case "mock-colosseum-rome":
		return everyDay("08:30", "19:15")
	case "mock-louvre-museum-paris":
		return append(hoursForDays([]int{1, 3, 4, 6, 7}, "09:00", "18:00"), hours(5, "09:00", "21:45"))
	case "mock-naschmarkt-vienna":
		return hoursForDays([]int{1, 2, 3, 4, 5, 6}, "06:00", "21:00")
	case "mock-urban-house-cafe-bratislava":
		return everyDay("08:00", "22:00")
	case "mock-testaccio-market-rome":
		return hoursForDays([]int{1, 2, 3, 4, 5, 6}, "08:00", "18:00")
	case "mock-musee-dorsay-paris", "mock-belvedere-palace-vienna":
		return hoursForDays([]int{2, 3, 4, 5, 6, 7}, "09:30", "18:00")
	}

	switch normalizeSearchText(category) {
	case "park", "neighborhood":
		return everyDay("00:00", "23:59")
	case "cafe", "restaurant":
		return everyDay("08:00", "22:00")
	case "market":
		return hoursForDays([]int{1, 2, 3, 4, 5, 6}, "08:00", "18:00")
	case "museum":
		return hoursForDays([]int{2, 3, 4, 5, 6, 7}, "09:00", "18:00")
	default:
		return everyDay("09:00", "18:00")
	}
}

func everyDay(open, close string) []entity.OpeningHoursInterval {
	return hoursForDays([]int{1, 2, 3, 4, 5, 6, 7}, open, close)
}

func hoursForDays(days []int, open, close string) []entity.OpeningHoursInterval {
	intervals := make([]entity.OpeningHoursInterval, 0, len(days))
	for _, day := range days {
		intervals = append(intervals, hours(day, open, close))
	}
	return intervals
}

func hours(day int, open, close string) entity.OpeningHoursInterval {
	return entity.OpeningHoursInterval{DayOfWeek: day, Open: open, Close: close}
}

func floatPtr(value float64) *float64 {
	return &value
}

func intPtr(value int) *int {
	return &value
}
