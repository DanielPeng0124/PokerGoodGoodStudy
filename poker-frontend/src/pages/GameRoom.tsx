import { type FormEvent, useEffect, useMemo, useRef, useState } from 'react';
import { PokerSocket } from '../api/ws';
import { getRoom } from '../api/http';
import { usePokerStore } from '../store/usePokerStore';
import { PokerTable } from '../components/PokerTable';
import { ActionPanel } from '../components/ActionPanel';
import { ChatPanel } from '../components/ChatPanel';
import type { ClientAction, RoomState, Seat } from '../types/game';

export function GameRoom({ roomId, onLeave }: { roomId: string; onLeave: () => void }) {
  const { userId, name, room, setUser, setRoom, chats, handleMessage, connected, setConnected, error, clearError } = usePokerStore();
  const ws = useRef(new PokerSocket());
  const [sitForm, setSitForm] = useState<{ seat: number; name: string; buyIn: number }>();
  const [sitError, setSitError] = useState('');
  const [controlError, setControlError] = useState('');
  const [ownerControlPending, setOwnerControlPending] = useState<'pause' | 'resume' | 'end' | ''>('');

  const mySeat = useMemo(() => Object.values(room?.seats ?? {}).find((s) => isLiveSeat(room, s) && s.userId === userId)?.index, [room, userId]);
  const mySeatInfo = mySeat === undefined ? undefined : room?.seats[String(mySeat)];
  const mySeatInActiveHand = mySeat !== undefined && !!room?.game?.players?.[String(mySeat)] && room.game.phase !== 'finished';
  const isOwner = room?.ownerId === userId;
  const duplicateSitName = useMemo(() => {
    if (!room || !sitForm) return false;
    const nextName = normalizeName(sitForm.name);
    if (!nextName) return false;
    return Object.values(room.seats).some((seat) => isLiveSeat(room, seat) && normalizeName(seat.name ?? '') === nextName);
  }, [room, sitForm]);

  useEffect(() => {
    getRoom(roomId, userId, name).then(setRoom).catch(console.error);
    setConnected(false);
    ws.current.connect(
      roomId,
      userId,
      name,
      handleMessage,
      () => setConnected(true),
      () => setConnected(false),
      () => {
        setConnected(false);
        setControlError('WebSocket connection failed');
      },
    );
    return () => ws.current.close();
  }, [roomId, userId, setRoom, handleMessage, setConnected]);

  useEffect(() => {
    if (mySeat === undefined || !room) return;
    const seatedName = room.seats[String(mySeat)]?.name;
    if (seatedName) setUser(seatedName);
    setSitForm(undefined);
    setSitError('');
  }, [mySeat, room, setUser]);

  useEffect(() => {
    if (!room || !ownerControlPending) return;
    if (ownerControlPending === 'pause' && room.paused) setOwnerControlPending('');
    if (ownerControlPending === 'resume' && !room.paused) setOwnerControlPending('');
    if (ownerControlPending === 'end' && (room.endingAfterHand || !room.game)) setOwnerControlPending('');
  }, [room, ownerControlPending]);

  useEffect(() => {
    if (error) setOwnerControlPending('');
  }, [error]);

  function sit(seat: number) {
    if (!room || mySeat !== undefined) return;
    setSitForm({ seat, name, buyIn: room.settings.minBuyIn });
    setSitError('');
  }

  function submitSit(e: FormEvent) {
    e.preventDefault();
    if (!sitForm) return;
    const sitName = sitForm.name.trim();
    if (!sitName) {
      setSitError('Name is required');
      return;
    }
    if (duplicateSitName) {
      setSitError('Name already taken');
      return;
    }
    ws.current.sitDown(sitForm.seat, sitForm.buyIn, sitName);
  }

  function action(a: ClientAction) { ws.current.action(a); }

  function setAway(away: boolean) {
    try {
      ws.current.setAway(away);
    } catch (err) {
      setControlError(err instanceof Error ? err.message : 'Action failed');
    }
  }

  function leaveSeat() {
    try {
      ws.current.leaveSeat();
    } catch (err) {
      setControlError(err instanceof Error ? err.message : 'Action failed');
    }
  }

  function sendOwnerControl(control: 'pause' | 'resume' | 'end') {
    try {
      setControlError('');
      setOwnerControlPending(control);
      if (control === 'pause') ws.current.pauseGame();
      if (control === 'resume') ws.current.resumeGame();
      if (control === 'end') ws.current.endGame(room?.game?.handNumber);
    } catch (err) {
      console.error(err);
      setOwnerControlPending('');
      setControlError(err instanceof Error ? err.message : 'Action failed');
    }
  }

  if (!room) return <main className="loading">加载房间中...</main>;

  return (
    <main className="game-page">
      <header className="topbar">
        <div>
          <h2>Room {room.id.slice(0, 8)}</h2>
          <small>
            {connected ? '已连接' : '未连接'} · {name || 'No name'}
            {ownerControlPending === 'pause' && ' · Pausing...'}
            {ownerControlPending === 'resume' && ' · Resuming...'}
            {ownerControlPending === 'end' && ' · Ending request...'}
            {room.paused && ' · Paused'}
            {room.endingAfterHand && ' · Ending after hand'}
          </small>
        </div>
        <div className="top-actions">
          {mySeatInfo && (
            <>
              {mySeatInfo.away && <span className="top-status away">Away</span>}
              <button
                className={mySeatInfo.away ? 'owner-control active' : 'owner-control'}
                onClick={() => setAway(!mySeatInfo.away)}
              >
                {mySeatInfo.away ? 'Back' : 'Away'}
              </button>
              <button
                className="owner-control danger"
                disabled={mySeatInActiveHand}
                onClick={leaveSeat}
              >
                Quit Seat
              </button>
            </>
          )}
          {isOwner && room.game && (
            <>
              {room.paused && <span className="top-status paused">Paused</span>}
              {room.endingAfterHand && <span className="top-status ending">Ending after hand</span>}
              {ownerControlPending && <span className="top-status pending">Sending...</span>}
              <button
                className={room.paused ? 'owner-control active' : 'owner-control'}
                disabled={!!ownerControlPending}
                onClick={() => sendOwnerControl(room.paused ? 'resume' : 'pause')}
              >
                {room.paused ? 'Resume' : 'Pause'}
              </button>
              <button
                className="owner-control danger"
                disabled={!!ownerControlPending || room.endingAfterHand}
                onClick={() => sendOwnerControl('end')}
              >
                {room.endingAfterHand ? 'Ending...' : 'End After Hand'}
              </button>
            </>
          )}
        </div>
      </header>
      {(error || controlError) && (
        <div
          className="error"
          onClick={() => {
            clearError();
            setControlError('');
          }}
        >
          {error || controlError}
        </div>
      )}
      <div className="layout">
        <div>
          <PokerTable room={room} myUserId={userId} onSit={sit} />
          <ActionPanel
            game={room.game}
            paused={room.paused}
            isOwner={isOwner}
            mySeat={mySeat}
            turnTimer={room.turnTimer}
            onStart={() => ws.current.startGame()}
            onAction={action}
            onSkipTurn={() => ws.current.skipTurn()}
            onAddTime={() => ws.current.addTime()}
          />
        </div>
        <ChatPanel chats={chats} room={room} onSend={(text) => ws.current.chat(text)} />
      </div>
      {sitForm && (
        <div className="sit-modal-backdrop" onClick={() => setSitForm(undefined)}>
          <form className="sit-modal" onSubmit={submitSit} onClick={(e) => e.stopPropagation()}>
            <h3>Seat {sitForm.seat + 1}</h3>
            <label>
              Name
              <input
                autoFocus
                required
                placeholder="Please enter your real name"
                value={sitForm.name}
                onChange={(e) => {
                  setSitError('');
                  setSitForm((current) => current ? { ...current, name: e.target.value } : current);
                }}
              />
            </label>
            <p className="form-hint">Name is required. A real recognizable name is better.</p>
            {(sitError || duplicateSitName) && <div className="sit-modal-error">{sitError || 'Name already taken'}</div>}
            <label>
              Chips
              <input
                type="number"
                required
                min={room.settings.minBuyIn}
                max={room.settings.maxBuyIn}
                value={sitForm.buyIn}
                onChange={(e) => setSitForm((current) => current ? { ...current, buyIn: Number(e.target.value) } : current)}
              />
            </label>
            <div className="sit-modal-actions">
              <button type="button" onClick={() => setSitForm(undefined)}>Cancel</button>
              <button type="submit" disabled={duplicateSitName}>Sit Down</button>
            </div>
          </form>
        </div>
      )}
    </main>
  );
}

function normalizeName(value: string) {
  return value.trim().toLowerCase();
}

function isLiveSeat(room: RoomState | undefined, seat: Seat) {
  const activeHandPlayer = room?.game?.phase !== 'finished' && !!room?.game?.players?.[String(seat.index)];
  return seat.stack > 0 || activeHandPlayer;
}
