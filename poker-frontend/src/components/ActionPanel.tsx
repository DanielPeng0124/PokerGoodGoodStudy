import { useEffect, useMemo, useState } from 'react';
import type { ClientAction, Game, TurnTimer } from '../types/game';

export function ActionPanel({ game, paused, isOwner, mySeat, turnTimer, onAction, onStart, onSkipTurn, onAddTime }: {
  game?: Game;
  paused?: boolean;
  isOwner: boolean;
  mySeat?: number;
  turnTimer?: TurnTimer;
  onAction: (a: ClientAction) => void;
  onStart: () => void;
  onSkipTurn: () => void;
  onAddTime: () => void;
}) {
  const [amount, setAmount] = useState(20);
  const [nowMs, setNowMs] = useState(Date.now());
  const myPlayer = mySeat === undefined ? undefined : game?.players?.[String(mySeat)];
  const toCall = Math.max(0, (game?.currentBet ?? 0) - (myPlayer?.bet ?? 0));
  const isMyTurn = !!game && mySeat !== undefined && game.currentTurn === mySeat;
  const needsRaiseAction = !!game && game.currentBet > 0;
  const voluntaryOpenThreshold = game?.phase === 'preflop' ? game.bigBlind : 0;
  const hasVoluntaryOpen = !!game && game.currentBet > voluntaryOpenThreshold;
  const actionLabel = hasVoluntaryOpen ? 'Raise' : 'Bet';
  const maxActionAmount = Math.max(myPlayer ? myPlayer.bet + myPlayer.stack : 0, 0);
  const minActionAmount = game ? (needsRaiseAction ? game.currentBet + game.minRaise : game.bigBlind) : 1;
  const rangeMin = Math.min(minActionAmount, Math.max(maxActionAmount, 1));
  const rangeMax = Math.max(rangeMin, maxActionAmount, 1);
  const quickBets = useMemo(() => {
    const pot = game?.pot ?? 0;
    const raisePot = pot + toCall;
    const baseAmount = hasVoluntaryOpen ? raisePot : pot;
    return [
      { label: '1/3', value: clampAmount(Math.max(game?.bigBlind ?? 1, hasVoluntaryOpen ? ceilRatio(baseAmount, 4, 3) : ceilRatio(baseAmount, 1, 3)), rangeMin, rangeMax) },
      { label: '1/2', value: clampAmount(Math.max(game?.bigBlind ?? 1, hasVoluntaryOpen ? ceilRatio(baseAmount, 3, 2) : ceilRatio(baseAmount, 1, 2)), rangeMin, rangeMax) },
      { label: '2/3', value: clampAmount(Math.max(game?.bigBlind ?? 1, hasVoluntaryOpen ? ceilRatio(baseAmount, 5, 3) : ceilRatio(baseAmount, 2, 3)), rangeMin, rangeMax) },
    ];
  }, [game?.bigBlind, game?.pot, hasVoluntaryOpen, rangeMax, rangeMin, toCall]);
  const defaultAmount = quickBets[0]?.value ?? rangeMin;
  useEffect(() => {
    setAmount(defaultAmount);
  }, [defaultAmount, game?.currentBet, game?.currentTurn, game?.handNumber, game?.phase, myPlayer?.bet, myPlayer?.stack]);
  useEffect(() => {
    if (!turnTimer) return;
    setNowMs(Date.now());
    const id = window.setInterval(() => setNowMs(Date.now()), 250);
    return () => window.clearInterval(id);
  }, [turnTimer?.expiresAt]);
  const actionDisabled = !game || paused || !isMyTurn;
  const normalizedAmount = Math.max(0, Math.ceil(Number.isFinite(amount) ? amount : 0));
  const selectedAmount = clampAmount(normalizedAmount, rangeMin, rangeMax);
  const canBetOrRaise = !actionDisabled && selectedAmount > 0 && (!needsRaiseAction || maxActionAmount > 0);
  const sizingDisabled = actionDisabled || (needsRaiseAction && maxActionAmount <= 0);
  const timerRemaining = turnTimer ? Math.max(0, Math.ceil((new Date(turnTimer.expiresAt).getTime() - nowMs) / 1000)) : 0;
  const extensionsLeft = turnTimer ? Math.max(0, turnTimer.extensionMax - turnTimer.extensionsUsed) : 0;

  function submitBetOrRaise() {
    if (!game) return;
    if (needsRaiseAction) {
      onAction({ type: 'raise', amount: selectedAmount });
      return;
    }
    onAction({ type: 'bet', amount: selectedAmount });
  }

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

  return (
    <div className="action-dock">
      {isFinished ? (
        isOwner ? <button className="primary big" onClick={onStart}>Start Next Hand</button> : <div className="waiting">Waiting for owner to start next hand</div>
      ) : (
        <>
          <div className={paused ? 'waiting' : isMyTurn ? 'your-turn' : 'waiting'}>
            <span>{paused ? 'Game paused' : isMyTurn ? 'Your turn' : `Waiting for Seat ${waitingSeat}`}</span>
            {turnTimer && !paused && <b>{timerRemaining}s</b>}
            {isMyTurn && turnTimer && (
              <button type="button" disabled={extensionsLeft <= 0} onClick={onAddTime}>
                +10s <small>{extensionsLeft}</small>
              </button>
            )}
          </div>
          <div className="action-buttons">
            <button disabled={actionDisabled} onClick={() => onAction({ type: 'fold' })}>Fold</button>
            <button disabled={actionDisabled || toCall > 0} onClick={() => onAction({ type: 'check' })}>Check</button>
            <button disabled={actionDisabled || toCall <= 0} onClick={() => onAction({ type: 'call' })}>Call {toCall || ''}</button>
            <button disabled={actionDisabled} onClick={() => onAction({ type: 'all_in' })}>All-in</button>
          </div>
          <div className="raise-row">
            <input type="range" min={rangeMin} max={rangeMax} value={selectedAmount} onChange={(e) => setAmount(Number(e.target.value))} />
            <div className="quick-bets">
              {quickBets.map((quickBet) => (
                <button key={quickBet.label} type="button" disabled={sizingDisabled} onClick={() => setAmount(quickBet.value)}>
                  {quickBet.label}<span>{quickBet.value}</span>
                </button>
              ))}
            </div>
            <input type="number" step={1} value={selectedAmount} min={rangeMin} max={rangeMax} onChange={(e) => setAmount(Number(e.target.value))} />
            <button disabled={!canBetOrRaise} onClick={submitBetOrRaise}>{actionLabel} {selectedAmount}</button>
          </div>
          {!isMyTurn && !paused && <button className="ghost" onClick={onSkipTurn}>Auto check/fold Seat {waitingSeat}</button>}
        </>
      )}
    </div>
  );
}

function ceilRatio(value: number, numerator: number, denominator: number) {
  return Math.ceil((value * numerator) / denominator);
}

function clampAmount(value: number, min: number, max: number) {
  return Math.min(max, Math.max(min, value));
}
