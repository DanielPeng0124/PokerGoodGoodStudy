import { CardView } from './CardView';
import { PlayerSeat } from './PlayerSeat';
import type { RoomState, Seat } from '../types/game';

const MY_VISUAL_SEAT = 0;
const VISUAL_SEAT_COUNT = 10;

export function PokerTable({ room, myUserId, onSit }: {
  room: RoomState;
  myUserId: string;
  onSit: (seat: number) => void;
}) {
  const maxSeats = room.settings.maxSeats;
  const mySeat = Object.values(room.seats).find((s) => isLiveSeat(room, s) && s.userId === myUserId)?.index;
  const activeGame = room.game?.phase === 'finished' ? undefined : room.game;
  const currentSeat = activeGame?.currentTurn;
  return (
    <section className="table-stage">
      <div className="poker-table">
        <div className="rail rail-outer" />
        <div className="rail rail-inner" />
        <div className="table-center">
          <div className="table-info">
            <div className="room-chip">Room {room.id.slice(0, 8)}</div>
            <div className="pot-pill">Pot <b>{room.game?.pot ?? 0}</b></div>
            {activeGame && <div className="turn-pill">Turn: Seat {(currentSeat ?? 0) + 1}</div>}
          </div>
          <div className="board cards">
            {Array.from({ length: 5 }).map((_, i) => <CardView key={i} card={room.game?.community?.[i]} hidden={!room.game?.community?.[i]} />)}
          </div>
        </div>
        {Array.from({ length: maxSeats }).map((_, i) => {
          const rawSeat = room.seats[String(i)];
          const seat = rawSeat && isLiveSeat(room, rawSeat) ? rawSeat : undefined;
          const player = activeGame?.players?.[String(i)];
          const seatPlayer = player && (!seat || player.userId === seat.userId) ? player : undefined;
          const winAmount = room.game?.phase === 'finished' ? (room.game.winAmounts?.[String(i)] ?? 0) : 0;
          return <PlayerSeat key={i} index={i} visualIndex={visualSeatIndex(i, mySeat, maxSeats)} seat={seat} player={seatPlayer} isMine={seat?.userId === myUserId} isTurn={currentSeat === i} isDealer={activeGame?.dealerSeat === i} isWinner={!!seatPlayer && !!activeGame?.winners?.includes(i)} winAmount={winAmount} sitDisabled={mySeat !== undefined} onSit={() => onSit(i)} />;
        })}
      </div>
      <div className="table-footer">You: {mySeat === undefined ? 'not seated' : `Seat ${mySeat + 1}`} · SB/BB {room.settings.smallBlind}/{room.settings.bigBlind}</div>
    </section>
  );
}

function visualSeatIndex(seatIndex: number, mySeat: number | undefined, maxSeats: number) {
  if (mySeat === undefined) return seatIndex;
  const clockwiseOffset = (seatIndex - mySeat + maxSeats) % maxSeats;
  return (MY_VISUAL_SEAT + clockwiseOffset) % VISUAL_SEAT_COUNT;
}

function isLiveSeat(room: RoomState, seat: Seat) {
  const activeHandPlayer = room.game?.phase !== 'finished' && !!room.game?.players?.[String(seat.index)];
  return seat.stack > 0 || activeHandPlayer;
}
