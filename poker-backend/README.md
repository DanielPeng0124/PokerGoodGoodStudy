# Poker Backend MVP

Go + WebSocket 的在线德州扑克后端 MVP。

## Run

```bash
go mod tidy
go run ./cmd/server
```

## HTTP

Create room:

```bash
curl -X POST http://localhost:8080/rooms \
  -H 'Content-Type: application/json' \
  -H 'X-User-ID: u1' -H 'X-User-Name: Daniel' \
  -d '{"settings":{"maxSeats":9,"smallBlind":5,"bigBlind":10,"minBuyIn":200,"maxBuyIn":2000}}'
```

Get room:

```bash
curl http://localhost:8080/rooms/{roomId} -H 'X-User-ID: u1'
```

WebSocket:

```text
ws://localhost:8080/rooms/{roomId}/ws
```

Client messages:

```json
{"type":"sit_down","seat":0,"buyIn":1000}
{"type":"start_game"}
{"type":"action","action":{"type":"call"}}
{"type":"action","action":{"type":"raise","amount":40}}
{"type":"chat","text":"gg"}
```

## Notes

- This is play-money / chip accounting only.
- No rake, wallet, fiat, crypto deposit, withdrawal, KYC, anti-fraud, or compliance layer is included.
- Side-pot logic is simplified; production poker needs full all-in side-pot settlement.
- The evaluator is functional for MVP but should be replaced by a well-tested audited evaluator before production.
