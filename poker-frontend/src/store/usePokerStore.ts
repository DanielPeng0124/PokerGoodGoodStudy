import { create } from 'zustand';
import type { RoomState, WsMessage } from '../types/game';

type ChatLine = { name: string; text: string; userId: string };

type PokerStore = {
  userId: string;
  name: string;
  room?: RoomState;
  chats: ChatLine[];
  error?: string;
  connected: boolean;
  setUser: (name: string) => void;
  setRoom: (room: RoomState) => void;
  setConnected: (connected: boolean) => void;
  handleMessage: (message: WsMessage) => void;
  clearError: () => void;
};

function makeUserId() {
  // Use sessionStorage so each browser tab acts as a different player during local testing.
  // localStorage caused multiple seats opened in different tabs to share one userId,
  // which could make the UI wait forever for e.g. Seat 8 while action buttons stayed disabled.
  const key = 'poker-user-id';
  let v = sessionStorage.getItem(key);
  if (!v) {
    v = crypto.randomUUID();
    sessionStorage.setItem(key, v);
  }
  return v;
}

export const usePokerStore = create<PokerStore>((set) => ({
  userId: makeUserId(),
  name: localStorage.getItem('poker-name') || '',
  chats: [],
  connected: false,
  setUser: (name) => {
    localStorage.setItem('poker-name', name);
    set({ name });
  },
  setRoom: (room) => set({ room }),
  setConnected: (connected) => set({ connected }),
  handleMessage: (message) => {
    if (message.type === 'room_state') set({ room: message.payload, error: undefined });
    if (message.type === 'chat') set((s) => ({ chats: [...s.chats.slice(-50), message] }));
    if (message.type === 'error') set({ error: message.error });
  },
  clearError: () => set({ error: undefined }),
}));
