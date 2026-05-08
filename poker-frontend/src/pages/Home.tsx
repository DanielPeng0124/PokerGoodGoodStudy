import { useState } from 'react';
import { createRoom } from '../api/http';
import { usePokerStore } from '../store/usePokerStore';

export function Home({ onEnterRoom }: { onEnterRoom: (roomId: string) => void }) {
  const { userId, name, setUser } = usePokerStore();
  const [roomId, setRoomId] = useState('');
  const [displayName, setDisplayName] = useState(name);
  const [busy, setBusy] = useState(false);
  const validName = displayName.trim().length > 0;

  async function create() {
    if (!validName) return;
    setBusy(true);
    try {
      const cleanName = displayName.trim();
      setUser(cleanName);
      const room = await createRoom(userId, cleanName);
      onEnterRoom(room.id);
    } finally { setBusy(false); }
  }

  function enterExistingRoom() {
    if (!validName || !roomId.trim()) return;
    setUser(displayName.trim());
    onEnterRoom(roomId.trim());
  }

  return (
    <main className="home">
      <div className="panel hero">
        <h1>React Poker Room</h1>
        <p>对接 Go 后端的 Texas Hold'em 私人房间 MVP。</p>
        <label>你的名字<input value={displayName} onChange={(e) => setDisplayName(e.target.value)} placeholder="Please enter your real name" /></label>
        <p className="form-hint">Name is required. A real recognizable name is better for the table.</p>
        <button className="primary" disabled={busy || !validName} onClick={create}>创建房间</button>
        <div className="divider" />
        <label>房间 ID<input value={roomId} onChange={(e) => setRoomId(e.target.value)} placeholder="粘贴 room id" /></label>
        <button disabled={!roomId.trim() || !validName} onClick={enterExistingRoom}>进入房间</button>
      </div>
    </main>
  );
}
