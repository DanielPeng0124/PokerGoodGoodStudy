import { useMemo, useState } from 'react';
import type { ClientAction, Game } from '../types/game';

export function ActionPanel({ game, mySeat, onAction, onStart, onSkipTurn }: {
  game?: Game;
  mySeat?: number;
  onAction: (a: ClientAction) => void;
  onStart: () => void;
  onSkipTurn: () => void;
}) {
  const [amount, setAmount] = useState(20);
  const myPlayer = mySeat === undefined ? undefined : game?.players?.[String(mySeat)];
  const toCall = Math.max(0, (game?.currentBet ?? 0) - (myPlayer?.bet ?? 0));
  const minRaiseTo = useMemo(() => Math.max((game?.currentBet ?? 0) + 10, amount), [game?.currentBet, amount]);

  if (!game) {
    return <div className="action-dock"><button className="primary big" onClick={onStart}>Start Game</button></div>;
  }

  const isFinished = game.phase === 'finished';

  const isMyTurn = mySeat !== undefined && game.currentTurn === mySeat;
  const waitingSeat = (game.currentTurn ?? 0) + 1;

  return (
    <div className="action-dock">
      {isFinished ? (
        <button className="primary big" onClick={onStart}>Start Next Hand</button>
      ) : (
        <>
          {isMyTurn ? <div className="your-turn">Your turn</div> : <div className="waiting">Waiting for Seat {waitingSeat}</div>}
          <div className="action-buttons">
            <button disabled={!isMyTurn} onClick={() => onAction({ type: 'fold' })}>Fold</button>
            <button disabled={!isMyTurn || toCall > 0} onClick={() => onAction({ type: 'check' })}>Check</button>
            <button disabled={!isMyTurn || toCall <= 0} onClick={() => onAction({ type: 'call' })}>Call {toCall || ''}</button>
            <button disabled={!isMyTurn} onClick={() => onAction({ type: 'all_in' })}>All-in</button>
          </div>
          <div className="raise-row">
            <input type="range" min={Math.max(10, game.currentBet + 10)} max={Math.max(20, myPlayer?.stack ?? 2000)} value={amount} onChange={(e) => setAmount(Number(e.target.value))} />
            <input type="number" value={amount} min={1} onChange={(e) => setAmount(Number(e.target.value))} />
            <button disabled={!isMyTurn} onClick={() => onAction({ type: game.currentBet > 0 ? 'raise' : 'bet', amount: game.currentBet > 0 ? Math.max(amount, minRaiseTo) : amount })}>{game.currentBet > 0 ? 'Raise' : 'Bet'}</button>
          </div>
          {!isMyTurn && <button className="ghost" onClick={onSkipTurn}>Auto check/fold Seat {waitingSeat}</button>}
        </>
      )}
    </div>
  );
}
