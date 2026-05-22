import type { HouseState } from "../types/house";

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
