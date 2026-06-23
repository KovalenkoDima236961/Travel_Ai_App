export type WeatherDay = {
  date: string;
  condition: string;
  temperatureMinC: number;
  temperatureMaxC: number;
  precipitationChance: number;
  windSpeedKph: number;
  summary: string;
  warnings?: string[];
};

export type WeatherForecast = {
  destination: string;
  provider?: string | null;
  days: WeatherDay[];
};
