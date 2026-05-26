import type { HouseState } from "../types/house";

export interface AppEntitlement {
  canAccess: boolean;
  canUseAI: boolean;
}

export interface CurrentUser {
  id: string;
  email: string;
  displayName: string;
  avatarUrl?: string;
}

export interface MeResponse {
  authenticated: boolean;
  user: CurrentUser | null;
  apps: {
    homeplan: AppEntitlement;
  };
}

export async function loadMe(): Promise<MeResponse> {
  const response = await fetch("/api/me", { cache: "no-store", credentials: "include" });
  if (!response.ok) throw new Error(`Me returned ${response.status}`);
  return (await response.json()) as MeResponse;
}

export async function logout(): Promise<void> {
  const response = await fetch("/api/auth/logout", {
    method: "POST",
    credentials: "include"
  });
  if (!response.ok) throw new Error(`Logout failed with ${response.status}`);
}

export async function loadCurrentHouse(): Promise<{ house: HouseState | null; source: "api" | "empty" }> {
  const response = await fetch("/api/house/current", { cache: "no-store", credentials: "include" });
  if (response.status === 404) {
    return { house: null, source: "empty" };
  }
  if (!response.ok) throw new Error(`API returned ${response.status}`);
  return { house: (await response.json()) as HouseState, source: "api" };
}

export async function saveCurrentHouse(house: HouseState): Promise<void> {
  const response = await fetch("/api/house/current", {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    credentials: "include",
    body: JSON.stringify(house)
  });

  if (!response.ok) {
    throw new Error(`Save failed with ${response.status}`);
  }
}

export async function deleteCurrentHouse(): Promise<void> {
  const response = await fetch("/api/house/current", {
    method: "DELETE",
    credentials: "include"
  });

  if (!response.ok) {
    throw new Error(`Delete failed with ${response.status}`);
  }
}
