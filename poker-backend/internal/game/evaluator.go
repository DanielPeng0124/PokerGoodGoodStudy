package game

import "sort"

// Lightweight evaluator for MVP. It ranks 7 cards and returns a comparable score.
// Higher is better. Production systems should replace this with a fully audited evaluator.
func EvaluateBest(cards []Card) int64 {
	if len(cards) < 5 { return 0 }
	best := int64(0)
	for a:=0; a<len(cards)-4; a++ { for b:=a+1; b<len(cards)-3; b++ { for c:=b+1; c<len(cards)-2; c++ { for d:=c+1; d<len(cards)-1; d++ { for e:=d+1; e<len(cards); e++ {
		s := eval5([]Card{cards[a],cards[b],cards[c],cards[d],cards[e]})
		if s > best { best = s }
	}}}}}
	return best
}

func eval5(cards []Card) int64 {
	sort.Slice(cards, func(i,j int) bool { return cards[i].Rank > cards[j].Rank })
	counts := map[int]int{}
	suits := map[Suit]int{}
	for _, c := range cards { counts[c.Rank]++; suits[c.Suit]++ }
	flush := false
	for _, n := range suits { if n == 5 { flush = true } }
	ranks := make([]int,0,5)
	seen := map[int]bool{}
	for _, c := range cards { if !seen[c.Rank] { ranks = append(ranks,c.Rank); seen[c.Rank]=true } }
	straightHigh := straightHigh(ranks)
	if flush && straightHigh > 0 { return pack(8, []int{straightHigh}) }
	groups := make([][2]int,0)
	for r,n := range counts { groups = append(groups,[2]int{n,r}) }
	sort.Slice(groups, func(i,j int) bool { if groups[i][0]==groups[j][0] { return groups[i][1]>groups[j][1] }; return groups[i][0]>groups[j][0] })
	if groups[0][0] == 4 { return pack(7, []int{groups[0][1], groups[1][1]}) }
	if groups[0][0] == 3 && groups[1][0] == 2 { return pack(6, []int{groups[0][1], groups[1][1]}) }
	if flush { return pack(5, ranksFromCards(cards)) }
	if straightHigh > 0 { return pack(4, []int{straightHigh}) }
	if groups[0][0] == 3 { return pack(3, append([]int{groups[0][1]}, kickers(counts, groups[0][1], 0)...)) }
	if groups[0][0] == 2 && groups[1][0] == 2 { return pack(2, []int{groups[0][1], groups[1][1], kickers(counts, groups[0][1], groups[1][1])[0]}) }
	if groups[0][0] == 2 { return pack(1, append([]int{groups[0][1]}, kickers(counts, groups[0][1], 0)...)) }
	return pack(0, ranksFromCards(cards))
}

func straightHigh(ranks []int) int {
	m := map[int]bool{}
	for _, r := range ranks { m[r]=true }
	if m[14] { m[1]=true }
	for h:=14; h>=5; h-- { if m[h]&&m[h-1]&&m[h-2]&&m[h-3]&&m[h-4] { return h } }
	return 0
}
func ranksFromCards(cards []Card) []int { out:=make([]int,len(cards)); for i,c:= range cards { out[i]=c.Rank }; sort.Sort(sort.Reverse(sort.IntSlice(out))); return out }
func kickers(counts map[int]int, exclude1, exclude2 int) []int { ks:=[]int{}; for r,n:=range counts { if r!=exclude1 && r!=exclude2 { for i:=0;i<n;i++ { ks=append(ks,r) } } }; sort.Sort(sort.Reverse(sort.IntSlice(ks))); return ks }
func pack(cat int, vals []int) int64 { s:=int64(cat); for i:=0;i<5;i++ { s*=15; if i<len(vals){ s+=int64(vals[i]) } }; return s }
