import { useState } from 'react';
import type { Card, HandLogEntry, HandRecord, RoomState } from '../types/game';

type Tab = 'chat' | 'log' | 'ledger';

export function ChatPanel({ chats, room, onSend }: {
  chats: { name: string; text: string }[];
  room: RoomState;
  onSend: (text: string) => void;
}) {
  const [text, setText] = useState('');
  const [tab, setTab] = useState<Tab>('log');

  return (
    <aside className="side-panel">
      <div className="side-tabs">
        <button className={tab === 'chat' ? 'active' : ''} onClick={() => setTab('chat')}>Chat</button>
        <button className={tab === 'log' ? 'active' : ''} onClick={() => setTab('log')}>Log</button>
        <button className={tab === 'ledger' ? 'active' : ''} onClick={() => setTab('ledger')}>Ledger</button>
      </div>

      {tab === 'chat' && (
        <>
          <div className="chat-lines">
            {chats.map((c, i) => <p key={i}><strong>{c.name}:</strong> {c.text}</p>)}
          </div>
          <form className="chat-form" onSubmit={(e) => { e.preventDefault(); if (text.trim()) { onSend(text.trim()); setText(''); } }}>
            <input value={text} onChange={(e) => setText(e.target.value)} placeholder="输入消息" />
            <button>发送</button>
          </form>
        </>
      )}

      {tab === 'log' && <LogView room={room} />}
      {tab === 'ledger' && <LedgerView room={room} />}
    </aside>
  );
}

function LogView({ room }: { room: RoomState }) {
  const hands = [...(room.handHistory ?? [])].reverse();
  return (
    <div className="log-view">
      {room.game && (
        <section className="log-block">
          <div className="section-title">Current hand #{room.game.handNumber}</div>
          <LogEntries entries={room.game.log ?? []} />
        </section>
      )}
      {hands.length === 0 ? (
        <div className="empty-panel">No completed hands yet.</div>
      ) : hands.map((hand, index) => <HandDetail key={hand.number} hand={hand} open={index === 0} />)}
    </div>
  );
}

function HandDetail({ hand, open }: { hand: HandRecord; open: boolean }) {
  const winners = hand.players
    .filter((player) => hand.winners?.includes(player.seatIndex))
    .map((player) => player.name)
    .join(', ') || 'No winner';

  return (
    <details className="hand-detail" open={open}>
      <summary>
        <span>Hand #{hand.number}</span>
        <b>Pot {hand.pot}</b>
      </summary>
      <div className="hand-meta">
        <span>Pot {hand.pot}</span>
        <span>Winner {winners}</span>
        <span>Board {formatCards(hand.community) || '-'}</span>
      </div>
      <LogEntries entries={hand.log} />
      <div className="hand-player-list">
        {hand.players.map((player) => (
          <div key={player.userId} className="hand-player-row">
            <div>
              <b>{player.name}</b>
              <span>Seat {player.seatIndex + 1} · {formatCards(player.cards ?? []) || '-'}</span>
            </div>
            <strong className={netClass(player.delta)}>{formatSigned(player.delta)}</strong>
          </div>
        ))}
      </div>
    </details>
  );
}

function LogEntries({ entries }: { entries: HandLogEntry[] }) {
  if (!entries.length) return <div className="empty-panel compact">No actions recorded.</div>;

  return (
    <div className="log-list">
      {entries.map((entry) => (
        <div key={entry.seq} className="log-row">
          <span>{entry.seq}</span>
          <p>{entry.message}</p>
          <b>{entry.pot ? `Pot ${entry.pot}` : ''}</b>
        </div>
      ))}
    </div>
  );
}

function LedgerView({ room }: { room: RoomState }) {
  const hands = [...(room.handHistory ?? [])].reverse();

  return (
    <div className="ledger-view">
      <div className="ledger-table">
        <div className="ledger-head">
          <span>Player</span>
          <span>Buy-in</span>
          <span>Stack</span>
          <span>Net</span>
        </div>
        {(room.ledger ?? []).map((row) => (
          <div key={row.userId} className="ledger-row">
            <div>
              <b>{row.name}</b>
              <small>{row.seated ? `Seat ${row.seatIndex + 1}` : 'Away'} · {row.handsWon}/{row.handsPlayed} wins</small>
            </div>
            <span>{row.buyIn}</span>
            <span>{row.currentStack}</span>
            <strong className={netClass(row.net)}>{formatSigned(row.net)}</strong>
          </div>
        ))}
      </div>

      <div className="section-title">Hand ledger</div>
      {hands.length === 0 ? (
        <div className="empty-panel">Completed hands will appear here.</div>
      ) : hands.map((hand) => (
        <section key={hand.number} className="ledger-hand">
          <header>
            <span>Hand #{hand.number}</span>
            <b>Pot {hand.pot}</b>
          </header>
          {hand.players.map((player) => (
            <div key={player.userId} className="ledger-delta-row">
              <span>{player.name}</span>
              <small>{player.startStack} to {player.endStack}</small>
              <strong className={netClass(player.delta)}>{formatSigned(player.delta)}</strong>
            </div>
          ))}
        </section>
      ))}
    </div>
  );
}

function formatCards(cards: Card[]) {
  return cards.map((card) => `${rankLabel(card.rank)}${card.suit.toUpperCase()}`).join(' ');
}

function rankLabel(rank: number) {
  if (rank === 14) return 'A';
  if (rank === 13) return 'K';
  if (rank === 12) return 'Q';
  if (rank === 11) return 'J';
  if (rank === 10) return 'T';
  return String(rank);
}

function formatSigned(value: number) {
  if (value > 0) return `+${value}`;
  return String(value);
}

function netClass(value: number) {
  if (value > 0) return 'net positive';
  if (value < 0) return 'net negative';
  return 'net';
}
