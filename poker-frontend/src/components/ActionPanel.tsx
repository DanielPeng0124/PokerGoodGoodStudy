import { useEffect, useMemo, useRef, useState } from 'react';
import type { ClientAction, Game } from '../types/game';

function TurnCountdown({ deadline }: { deadline: string }) {
  const [secsLeft, setSecsLeft] = useState(() => Math.max(0, Math.ceil((new Date(deadline).getTime() - Date.now()) / 1000)));
  const rafRef = useRef<number>(0);

  useEffect(() => {
    function tick() {
      const left = Math.max(0, Math.ceil((new Date(deadline).getTime() - Date.now()) / 1000));
      setSecsLeft(left);
      if (left > 0) rafRef.current = requestAnimationFrame(tick);
    }
    rafRef.current = requestAnimationFrame(tick);
    return () => cancelAnimationFrame(rafRef.current);
  }, [deadline]);

  return <span className={`turn-countdown${secsLeft <= 5 ? ' urgent' : ''}`}>{secsLeft}s</span>;
}

export function ActionPanel({ game, paused, isOwner, mySeat, onAction, onStart, onSkipTurn }: {
  game?: Game;
  paused?: boolean;
  isOwner: boolean;
  mySeat?: number;
  onAction: (a: ClientAction) => void;
  onStart: () => void;
  onSkipTurn: () => void;
}) {
  const [amount, setAmount] = useState(20);
  const myPlayer = mySeat === undefined ? undefined : game?.players?.[String(mySeat)];
  const toCall = Math.max(0, (game?.currentBet ?? 0) - (myPlayer?.bet ?? 0));
  const isMyTurn = !!game && mySeat !== undefined && game.currentTurn === mySeat;
  const maxBet = Math.max(myPlayer ? myPlayer.bet + myPlayer.stack : 0, 1);
  const minBet = game ? (game.currentBet > 0 ? game.currentBet + game.minRaise : game.bigBlind) : 1;
  const rangeMin = Math.min(minBet, maxBet);
  const rangeMax = Math.max(rangeMin, maxBet);
  const minRaiseTo = useMemo(() => Math.max(minBet, amount), [minBet, amount]);
  const actionDisabled = !game || paused || !isMyTurn;

  if (!game) {
    return (
      <div className="action-dock">
        {isOwner ? (
          <button className="primary big" onClick={onStart}>Start Game</button>
        ) : (
          <div className="waiting">Waiting for owner to start</div>
        )}
      </div>
    );
  }

  const isFinished = game.phase === 'finished';

  const waitingSeat = (game.currentTurn ?? 0) + 1;

  function setPotFraction(fraction: number) {
    if (!game) return;
    const nextAmount = Math.ceil(game.pot * fraction);
    setAmount(Math.min(rangeMax, Math.max(rangeMin, nextAmount)));
  }

  return (
    <div className="action-dock">
      {isFinished ? (
        isOwner ? <button className="primary big" onClick={onStart}>Start Next Hand</button> : <div className="waiting">Waiting for owner to start next hand</div>
      ) : (
        <>
          {paused ? (
            <div className="waiting">Game paused</div>
          ) : isMyTurn ? (
            <div className="your-turn">Your turn {game.turnDeadline && <TurnCountdown deadline={game.turnDeadline} />}</div>
          ) : (
            <div className="waiting">Waiting for Seat {waitingSeat} {game.turnDeadline && <TurnCountdown deadline={game.turnDeadline} />}</div>
          )}
          <div className="action-buttons">
            <button disabled={actionDisabled} onClick={() => onAction({ type: 'fold' })}>Fold</button>
            <button disabled={actionDisabled || toCall > 0} onClick={() => onAction({ type: 'check' })}>Check</button>
            <button disabled={actionDisabled || toCall <= 0} onClick={() => onAction({ type: 'call' })}>Call {toCall || ''}</button>
            <button disabled={actionDisabled} onClick={() => onAction({ type: 'all_in' })}>All-in</button>
          </div>
          <div className="raise-row">
            <input type="range" min={rangeMin} max={rangeMax} value={Math.min(rangeMax, Math.max(rangeMin, amount))} onChange={(e) => setAmount(Number(e.target.value))} />
            <div className="quick-bets">
              <button type="button" disabled={actionDisabled} onClick={() => setPotFraction(1 / 3)}>1/3</button>
              <button type="button" disabled={actionDisabled} onClick={() => setPotFraction(1 / 2)}>1/2</button>
              <button type="button" disabled={actionDisabled} onClick={() => setPotFraction(2 / 3)}>2/3</button>
            </div>
            <input type="number" value={amount} min={rangeMin} max={rangeMax} onChange={(e) => setAmount(Number(e.target.value))} />
            <button disabled={actionDisabled} onClick={() => onAction({ type: game.currentBet > 0 ? 'raise' : 'bet', amount: game.currentBet > 0 ? Math.max(amount, minRaiseTo) : Math.max(amount, rangeMin) })}>{game.currentBet > 0 ? 'Raise' : 'Bet'}</button>
          </div>
          {!isMyTurn && !paused && <button className="ghost" onClick={onSkipTurn}>Auto check/fold Seat {waitingSeat}</button>}
        </>
      )}
    </div>
  );
}
