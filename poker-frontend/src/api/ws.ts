import { WS_BASE, authQuery } from './config';
import type { ClientAction, WsMessage } from '../types/game';

export class PokerSocket {
  private ws?: WebSocket;

  connect(roomId: string, userId: string, name: string, onMessage: (m: WsMessage) => void, onClose?: () => void) {
    this.close();
    this.ws = new WebSocket(`${WS_BASE}/rooms/${roomId}/ws?${authQuery(userId, name)}`);
    this.ws.onmessage = (ev) => onMessage(JSON.parse(ev.data));
    this.ws.onclose = () => onClose?.();
  }

  send(type: string, payload: Record<string, unknown> = {}) {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) throw new Error('WebSocket 尚未连接');
    this.ws.send(JSON.stringify({ type, ...payload }));
  }

  sitDown(seat: number, buyIn: number, name: string) { this.send('sit_down', { seat, buyIn, name }); }
  startGame() { this.send('start_game'); }
  action(action: ClientAction) { this.send('action', { action }); }
  skipTurn() { this.send('skip_turn'); }
  chat(text: string) { this.send('chat', { text }); }
  close() { this.ws?.close(); this.ws = undefined; }
}
