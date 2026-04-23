# ZedSellAuto Backend

Go Gin backend for the `ZedSellAuto` frontend. This repository intentionally contains only the API layer. The frontend can connect through environment variables later.

## Stack

- Go 1.26
- Gin
- PostgreSQL
- Redis
- pgx

## Features

- Auth: signup, login, profile
- Vehicles: list, filter, categories, detail, related vehicles
- Resources: blog, pricing, how-it-works, homepage snapshot
- User actions: favorites, offers, test-drive bookings, inquiries, finance applications
- Health and readiness endpoints
- PostgreSQL schema migration and seed at startup
- Redis caching for high-read vehicle queries

## Quick start

1. Copy `.env.example` to `.env`
2. Create PostgreSQL and Redis services
3. Run:

```powershell
go mod tidy
go run ./cmd/api
```

## Frontend integration

Suggested frontend env:

```env
NEXT_PUBLIC_API_BASE_URL=http://localhost:8080/api/v1
```

## Key endpoints

- `POST /api/v1/auth/signup`
- `POST /api/v1/auth/login`
- `GET /api/v1/me`
- `GET /api/v1/home`
- `GET /api/v1/vehicles`
- `GET /api/v1/vehicles/categories`
- `GET /api/v1/vehicles/:slug`
- `GET /api/v1/blog/posts`
- `GET /api/v1/blog/posts/:slug`
- `GET /api/v1/resources/pricing`
- `GET /api/v1/resources/how-it-works`
- `GET /api/v1/favorites`
- `POST /api/v1/favorites/:vehicleID`
- `POST /api/v1/leads/offers`
- `POST /api/v1/leads/test-drives`
- `POST /api/v1/leads/inquiries`
- `POST /api/v1/leads/finance`

