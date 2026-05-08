import React, { useState } from 'react';
import { createRoot } from 'react-dom/client';
import { Home } from './pages/Home';
import { GameRoom } from './pages/GameRoom';
import './styles.css';

function App() {
  const [roomId, setRoomId] = useState(() => new URLSearchParams(location.search).get('room') || '');
  function enter(id: string) {
    setRoomId(id);
    history.replaceState(null, '', `?room=${id}`);
  }
  function leave() {
    setRoomId('');
    history.replaceState(null, '', location.pathname);
  }
  return roomId ? <GameRoom roomId={roomId} onLeave={leave} /> : <Home onEnterRoom={enter} />;
}

createRoot(document.getElementById('root')!).render(<React.StrictMode><App /></React.StrictMode>);
