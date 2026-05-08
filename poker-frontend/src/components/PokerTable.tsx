import { CardView } from './CardView';
import { PlayerSeat } from './PlayerSeat';
import type { RoomState } from '../types/game';

const MY_VISUAL_SEAT = 0;
const VISUAL_SEAT_COUNT = 9;

export function PokerTable({ room, myUserId, onSit }: {
  room: RoomState;
  myUserId: string;
  onSit: (seat: number) => void;
}) {
  const maxSeats = room.settings.maxSeats;
  const mySeat = Object.values(room.seats).find((s) => s.userId === myUserId)?.index;
  const currentSeat = room.game?.currentTurn;
  return (
    <section className="table-stage">
      <div className="poker-table">
        <div className="rail rail-outer" />
        <div className="rail rail-inner" />
        <div className="table-center">
          <div className="table-info">
            <div className="room-chip">Room {room.id.slice(0, 8)}</div>
            <div className="pot-pill">Pot <b>{room.game?.pot ?? 0}</b></div>
            {room.game && <div className="turn-pill">Turn: Seat {(currentSeat ?? 0) + 1}</div>}
          </div>
          <div className="board cards">
            {Array.from({ length: 5 }).map((_, i) => <CardView key={i} card={room.game?.community?.[i]} hidden={!room.game?.community?.[i]} />)}
          </div>
        </div>
        {Array.from({ length: maxSeats }).map((_, i) => {
          const seat = room.seats[String(i)];
          const player = room.game?.players?.[String(i)];
          return <PlayerSeat key={i} index={i} visualIndex={visualSeatIndex(i, mySeat, maxSeats)} seat={seat} player={player} isMine={seat?.userId === myUserId} isTurn={currentSeat === i} isDealer={room.game?.dealerSeat === i} isWinner={!!room.game?.winners?.includes(i)} sitDisabled={mySeat !== undefined} onSit={() => onSit(i)} />;
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
