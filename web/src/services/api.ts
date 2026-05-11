import { seedHouseState } from "../data/seedHouseState";
import type { HouseState } from "../types/house";

export async function loadCurrentHouse(): Promise<{ house: HouseState; source: "api" | "seed" }> {
  try {
    const response = await fetch("/api/house/current", { cache: "no-store", credentials: "include" });
    if (!response.ok) throw new Error(`API returned ${response.status}`);
    return { house: (await response.json()) as HouseState, source: "api" };
  } catch {
    return { house: structuredClone(seedHouseState), source: "seed" };
  }
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
