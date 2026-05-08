import type { Card } from '../types/game';

const suitIcon: Record<Card['suit'], string> = { s: '♠', h: '♥', d: '♦', c: '♣' };

function rankLabel(rank: number) {
  if (rank === 14) return 'A';
  if (rank === 13) return 'K';
  if (rank === 12) return 'Q';
  if (rank === 11) return 'J';
  return String(rank);
}

export function CardView({ card, hidden }: { card?: Card; hidden?: boolean }) {
  const red = card?.suit === 'h' || card?.suit === 'd';
  return (
    <div className={`card ${hidden || !card ? 'back' : ''} ${red ? 'red' : 'black'}`}>
      {hidden || !card ? <span /> : <><b>{rankLabel(card.rank)}</b><em>{suitIcon[card.suit]}</em></>}
    </div>
  );
}
