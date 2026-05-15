import { CardView } from './CardView';
import type { Player, Seat } from '../types/game';

export function PlayerSeat({ index, visualIndex, seat, player, isTurn, isDealer, isWinner, isMine, sitDisabled, onSit }: {
  index: number;
  visualIndex?: number;
  seat?: Seat;
  player?: Player;
  isTurn?: boolean;
  isDealer?: boolean;
  isWinner?: boolean;
  isMine?: boolean;
  sitDisabled?: boolean;
  onSit: () => void;
}) {
  const occupied = !!seat;
  const statusLabel = seat?.away ? 'away' : (player?.status ?? 'waiting');
  return (
    <div className={`seat seat-${visualIndex ?? index} ${occupied ? 'occupied' : 'empty'} ${seat?.away ? 'away' : ''} ${isTurn ? 'turn' : ''} ${isWinner ? 'winner' : ''} ${isMine ? 'mine' : ''}`}>
      {occupied ? (
        <>
          {player && (
            <div className="seat-cards">
              <CardView card={player.cards?.[0]} hidden={!player.cards?.[0]} />
              <CardView card={player.cards?.[1]} hidden={!player.cards?.[1]} />
            </div>
          )}
          <div className="avatar">{(seat.name || 'P').slice(0, 1).toUpperCase()}</div>
          <div className="seat-panel">
            <div className="seat-name">{seat.name || `Seat ${index + 1}`}</div>
            <div className="seat-meta">
              <span>Seat {index + 1}</span>
              {isDealer && <b>D</b>}
              <span>{statusLabel}</span>
            </div>
            <div className="stack">{player?.stack ?? seat.stack}</div>
          </div>
          {!!player?.bet && <div className="bet-chip">{player.bet}</div>}
        </>
      ) : (
        <button className="sit-button" disabled={sitDisabled} onClick={onSit}>Seat {index + 1}<span>{sitDisabled ? 'Seated' : 'Sit'}</span></button>
      )}
    </div>
  );
}
