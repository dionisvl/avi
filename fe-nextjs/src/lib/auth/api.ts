/**
 * Typed API functions for authentication endpoints.
 * These wrap the typed `api` client and handle error cases.
 */

import { API_BASE_URL, api } from "@/lib/api/client";

/**
 * Shapes returned by the auth API endpoints.
 */
export interface LoginResponse {
  access_token: string;
  refresh_token: string;
  expires_in: number;
}

export interface UserMe {
  id: string;
  email: string;
  name?: string;
  avatar_url?: string;
  email_verified?: boolean;
  has_profile?: boolean;
  roles?: string[];
  created_at?: string;
  preferences?: Record<string, unknown>;
}

/**
 * POST /api/v1/auth/login - Authenticate with email and password.
 * Returns tokens on success. Throws on 401 (bad credentials) or other errors.
 */
export async function login(email: string, password: string): Promise<LoginResponse> {
  const { data, error } = await api.POST("/api/v1/auth/login", {
    body: { email, password },
  });

  if (error) {
    throw new Error(error.detail || error.title || "Login failed");
  }

  if (!data?.access_token || !data?.refresh_token) {
    throw new Error("Invalid login response");
  }

  return data as LoginResponse;
}

/**
 * POST /api/v1/auth/refresh - Refresh the access token using a refresh token.
 * Returns new tokens on success. Throws on error.
 */
export async function refresh(refreshToken: string): Promise<LoginResponse> {
  const { data, error } = await api.POST("/api/v1/auth/refresh", {
    body: { refresh_token: refreshToken },
  });

  if (error) {
    throw new Error(error.detail || error.title || "Token refresh failed");
  }

  if (!data?.access_token || !data?.refresh_token) {
    throw new Error("Invalid refresh response");
  }

  return data as LoginResponse;
}

/**
 * POST /api/v1/auth/logout - Logout and invalidate the session.
 * Best-effort; does not throw on failure.
 */
export async function logout(accessToken: string): Promise<void> {
  try {
    // Use direct fetch since we need to pass Bearer token
    const response = await fetch(`${API_BASE_URL}/api/v1/auth/logout`, {
      method: "POST",
      headers: {
        Authorization: `Bearer ${accessToken}`,
        "Content-Type": "application/json",
      },
    });
    // Silently ignore errors; we're logging out anyway
    if (!response.ok) {
      console.warn("Logout API call failed, but clearing local tokens");
    }
  } catch (err) {
    console.warn("Logout error:", err);
  }
}

/**
 * GET /api/v1/user/me - Fetch the current authenticated user.
 * Requires a valid access token. Throws on error.
 */
export async function fetchMe(accessToken: string): Promise<UserMe> {
  const { data, error } = await api.GET("/api/v1/user/me", {
    headers: {
      Authorization: `Bearer ${accessToken}`,
    },
  });

  if (error) {
    throw new Error(error.detail || error.title || "Failed to fetch user");
  }

  if (!data) {
    throw new Error("Invalid /me response");
  }

  return data as UserMe;
}
