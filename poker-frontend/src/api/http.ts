import { API_BASE, authQuery } from './config';
import type { RoomSettings, RoomState } from '../types/game';

const defaultSettings: RoomSettings = {
  maxSeats: 10,
  smallBlind: 1,
  bigBlind: 2,
  minBuyIn: 200,
  maxBuyIn: 2000,
};

export async function createRoom(userId: string, name: string, settings: Partial<RoomSettings> = {}) {
  const res = await fetch(`${API_BASE}/rooms?${authQuery(userId, name)}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ settings: { ...defaultSettings, ...settings } }),
  });
  if (!res.ok) throw new Error(await res.text());
  return (await res.json()) as RoomState;
}

export async function getRoom(roomId: string, userId: string, name: string) {
  const res = await fetch(`${API_BASE}/rooms/${roomId}?${authQuery(userId, name)}`);
  if (!res.ok) throw new Error(await res.text());
  return (await res.json()) as RoomState;
}
