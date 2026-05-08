export const API_BASE = import.meta.env.VITE_API_BASE ?? 'http://localhost:8080';
export const WS_BASE = API_BASE.replace(/^http/, 'ws');

export function authQuery(userId: string, name: string) {
  const p = new URLSearchParams({ userId, name });
  return p.toString();
}
