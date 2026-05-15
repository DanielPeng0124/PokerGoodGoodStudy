import { WS_BASE, authQuery } from './config';
import type { ClientAction, WsMessage } from '../types/game';

const RECONNECT_DELAYS = [500, 1500, 3000, 5000];

export class PokerSocket {
  private ws?: WebSocket;
  private roomId = '';
  private userId = '';
  private name = '';
  private onMessage?: (m: WsMessage) => void;
  private onOpen?: () => void;
  private onClose?: () => void;
  private onError?: () => void;
  private attempt = 0;
  private reconnectTimer?: ReturnType<typeof setTimeout>;
  private stopped = false;

  connect(
    roomId: string,
    userId: string,
    name: string,
    onMessage: (m: WsMessage) => void,
    onOpen?: () => void,
    onClose?: () => void,
    onError?: () => void,
  ) {
    this.stopped = false;
    this.attempt = 0;
    this.roomId = roomId;
    this.userId = userId;
    this.name = name;
    this.onMessage = onMessage;
    this.onOpen = onOpen;
    this.onClose = onClose;
    this.onError = onError;
    this._open();
  }

  private _detach(ws: WebSocket) {
    ws.onopen = null;
    ws.onmessage = null;
    ws.onclose = null;
    ws.onerror = null;
  }

  private _open() {
    if (this.ws) {
      this._detach(this.ws);
      this.ws.close();
      this.ws = undefined;
    }
    const ws = new WebSocket(`${WS_BASE}/rooms/${this.roomId}/ws?${authQuery(this.userId, this.name)}`);
    this.ws = ws;
    ws.onopen = () => {
      this.attempt = 0;
      this.onOpen?.();
    };
    ws.onmessage = (ev) => this.onMessage?.(JSON.parse(ev.data));
    ws.onclose = () => {
      if (this.ws !== ws) return;
      if (this.stopped) { this.onClose?.(); return; }
      this._scheduleReconnect();
    };
    ws.onerror = () => {
      // always followed by onclose — let onclose drive reconnect
    };
  }

  private _scheduleReconnect() {
    if (this.attempt === 0) this.onError?.();
    const delay = RECONNECT_DELAYS[Math.min(this.attempt, RECONNECT_DELAYS.length - 1)];
    this.attempt++;
    this.reconnectTimer = setTimeout(() => {
      if (!this.stopped) this._open();
    }, delay);
  }

  send(type: string, payload: Record<string, unknown> = {}) {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) throw new Error('WebSocket not connected');
    this.ws.send(JSON.stringify({ type, ...payload }));
  }

  sitDown(seat: number, buyIn: number, name: string) { this.send('sit_down', { seat, buyIn, name }); }
  setAway(away: boolean) { this.send('set_away', { away }); }
  leaveSeat() { this.send('leave_seat'); }
  startGame() { this.send('start_game'); }
  pauseGame() { this.send('pause_game'); }
  resumeGame() { this.send('resume_game'); }
  endGame(handNumber?: number) { this.send('end_game', handNumber ? { handNumber } : {}); }
  action(action: ClientAction) { this.send('action', { action }); }
  skipTurn() { this.send('skip_turn'); }
  addTime() { this.send('add_time'); }
  chat(text: string) { this.send('chat', { text }); }

  close() {
    this.stopped = true;
    clearTimeout(this.reconnectTimer);
    if (this.ws) {
      this._detach(this.ws);
      this.ws.close();
      this.ws = undefined;
    }
  }
}
