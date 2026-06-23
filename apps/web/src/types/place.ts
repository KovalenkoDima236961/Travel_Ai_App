export type Place = {
  provider: string;
  providerPlaceId: string;
  name: string;
  address: string;
  latitude?: number | null;
  longitude?: number | null;
  rating?: number | null;
  ratingCount?: number | null;
  mapUrl?: string | null;
  category?: string | null;
  website?: string | null;
};

export type SearchPlacesResponse = {
  items: Place[];
};
