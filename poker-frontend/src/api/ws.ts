import { WS_BASE, authQuery } from './config';
import type { ClientAction, WsMessage } from '../types/game';

export class PokerSocket {
  private ws?: WebSocket;

  connect(
    roomId: string,
    userId: string,
    name: string,
    onMessage: (m: WsMessage) => void,
    onOpen?: () => void,
    onClose?: () => void,
    onError?: () => void,
  ) {
    this.close();
    this.ws = new WebSocket(`${WS_BASE}/rooms/${roomId}/ws?${authQuery(userId, name)}`);
    this.ws.onopen = () => onOpen?.();
    this.ws.onmessage = (ev) => onMessage(JSON.parse(ev.data));
    this.ws.onclose = () => onClose?.();
    this.ws.onerror = () => onError?.();
  }

  send(type: string, payload: Record<string, unknown> = {}) {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) throw new Error('WebSocket 尚未连接');
    this.ws.send(JSON.stringify({ type, ...payload }));
  }

  sitDown(seat: number, buyIn: number, name: string) { this.send('sit_down', { seat, buyIn, name }); }
  startGame() { this.send('start_game'); }
  pauseGame() { this.send('pause_game'); }
  resumeGame() { this.send('resume_game'); }
  endGame(handNumber?: number) { this.send('end_game', handNumber ? { handNumber } : {}); }
  action(action: ClientAction) { this.send('action', { action }); }
  skipTurn() { this.send('skip_turn'); }
  chat(text: string) { this.send('chat', { text }); }
  close() { this.ws?.close(); this.ws = undefined; }
}
