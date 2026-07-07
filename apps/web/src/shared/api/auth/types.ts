export type AuthUser = {
  id: string;
  email: string;
  createdAt: string;
};

export type AuthResponse = {
  user: AuthUser;
  accessToken: string;
  refreshToken: string;
};

export type TokenResponse = {
  accessToken: string;
  refreshToken: string;
};
