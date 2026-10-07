# payments-service

`payment-api` takes card charges from checkout and writes them to the `orders-db` ledger.

## API

| Method | Path          | Description                                  |
|--------|---------------|----------------------------------------------|
| POST   | `/v1/charges` | Create a charge: `{"account_id": 7, "items": [1299, 450]}` |
| GET    | `/healthz`    | Liveness/readiness probe                     |

## Running it

Runs in the `payments` namespace next to `orders-db` (Postgres 16).
