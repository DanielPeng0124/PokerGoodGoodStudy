export type Phase = 'waiting' | 'preflop' | 'flop' | 'turn' | 'river' | 'showdown' | 'finished';
export type PlayerStatus = 'empty' | 'active' | 'folded' | 'all_in' | 'sitting_out';
export type ActionType = 'fold' | 'check' | 'call' | 'bet' | 'raise' | 'all_in';

export type Card = {
  // Backend sends rank as 2..14 (11=J,12=Q,13=K,14=A)
  rank: number;
  // Backend sends suit as: s,h,d,c
  suit: 's' | 'h' | 'd' | 'c';
};

export type RoomSettings = {
  maxSeats: number;
  smallBlind: number;
  bigBlind: number;
  minBuyIn: number;
  maxBuyIn: number;
};

export type Seat = {
  index: number;
  userId?: string;
  name?: string;
  stack: number;
  buyIn: number;
};

export type Player = {
  userId: string;
  name: string;
  seatIndex: number;
  stack: number;
  bet: number;
  totalBet: number;
  status: PlayerStatus;
  acted: boolean;
  cards?: Card[];
};

export type HandLogEntry = {
  seq: number;
  phase: Phase;
  type: string;
  seatIndex?: number;
  name?: string;
  amount?: number;
  pot: number;
  cards?: Card[];
  message: string;
};

export type Game = {
  handNumber: number;
  startedAt: string;
  phase: Phase;
  dealerSeat: number;
  smallBlind: number;
  bigBlind: number;
  minRaise: number;
  currentTurn: number;
  currentBet: number;
  pot: number;
  community: Card[];
  players: Record<string, Player>;
  winners?: number[];
  log?: HandLogEntry[];
  turnDeadline?: string; // ISO timestamp; present only when TurnTimeoutSecs > 0
};

export type HandPlayerRecord = {
  userId: string;
  name: string;
  seatIndex: number;
  startStack: number;
  endStack: number;
  delta: number;
  totalBet: number;
  status: PlayerStatus;
  cards?: Card[];
};

export type HandRecord = {
  number: number;
  startedAt: string;
  endedAt: string;
  dealerSeat: number;
  smallBlind: number;
  bigBlind: number;
  pot: number;
  community: Card[];
  winners: number[];
  players: HandPlayerRecord[];
  log: HandLogEntry[];
};

export type LedgerEntry = {
  userId: string;
  name: string;
  seatIndex: number;
  buyIn: number;
  currentStack: number;
  net: number;
  lastHandDelta: number;
  handsPlayed: number;
  handsWon: number;
};

export type RoomState = {
  id: string;
  ownerId: string;
  settings: RoomSettings;
  seats: Record<string, Seat>;
  paused: boolean;
  endingAfterHand: boolean;
  game?: Game;
  handHistory: HandRecord[];
  ledger: LedgerEntry[];
  lastHandSummary?: {
    startStack: number;
    endStack: number;
    delta: number;
  };
};

export type WsMessage =
  | { type: 'room_state'; payload: RoomState }
  | { type: 'chat'; userId: string; name: string; text: string }
  | { type: 'error'; error: string };

export type ClientAction = {
  type: ActionType;
  amount?: number;
};
