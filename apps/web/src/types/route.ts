export type RouteMode = "walking";

export type RouteStop = {
  name: string;
  latitude: number;
  longitude: number;
};

export type RouteSegment = {
  fromName: string;
  toName: string;
  distanceKm: number;
  durationMinutes: number;
};

export type RouteEstimate = {
  mode: RouteMode;
  provider: string;
  distanceKm: number;
  durationMinutes: number;
  segments: RouteSegment[];
};

export type RouteEstimateRequest = {
  mode: RouteMode;
  stops: RouteStop[];
};
