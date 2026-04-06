# 🚍 Transit Server

A real-time transit tracking backend built with **Go**, **Gin**, and **GORM**. It enables transit aggregators to manage drivers, track live vehicle locations via GPS, and serve standardised **GTFS-Realtime** feeds — all secured with JWT authentication and API key authorization.

---

## ✨ Features

| Area | Details |
|------|---------|
| **Authentication** | JWT-based auth with access / refresh tokens, password reset flow, role-based access control (driver · aggregator · admin) |
| **Driver Management** | Driver registration, profile, join aggregator, start/end trips |
| **Live Location** | Single & batch GPS updates, in-memory cache with 30 s eviction, real-time push via WebSockets |
| **GTFS-Realtime** | Protobuf vehicle-position feeds per agency or per vehicle, plus JSON debug endpoints |
| **Aggregator Dashboard** | List drivers, view locations, create routes & trips, manage API keys |
| **WebSocket Subscriptions** | Aggregators subscribe to live location events; auth via query params |

---

## 📁 Project Structure

```
transit-server/
├── main.go              # Application entry point
├── config/              # Environment & app configuration
├── database/            # Database connection (PostgreSQL / SQLite)
├── models/              # GORM models & request/response DTOs
├── handlers/            # HTTP handler functions (auth, driver, aggregator, location, trip)
├── routes/              # Gin route registration
├── middleware/           # Auth, API key, and role middleware
├── cache/               # In-memory location cache
├── gtfsrt/              # GTFS-Realtime feed generator (protobuf)
├── ws/                  # WebSocket hub, client, and handler
├── utils/               # JWT & password helper utilities
└── cmd/                 # CLI commands (if any)
```

---

## 🛠 Prerequisites

- **Go** 1.22+ (module uses Go 1.26)
- **PostgreSQL** (or SQLite for local development)

---

## 🚀 Getting Started

### 1. Clone the repository

```bash
git clone https://github.com/dhanushakkari00/transit-server.git
cd transit-server
```

### 2. Create a `.env` file

Copy the example below and fill in your own values:

```env
JWT_SECRET=your-secret-key
JWT_ACCESS_TOKEN_EXPIRY=15m
JWT_REFRESH_TOKEN_EXPIRY=168h
DATABASE_URL=postgresql://user:password@host:5432/dbname?sslmode=require
DB_PATH=transit.db
PORT=8080
```

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `JWT_SECRET` | ✅ | — | Secret used to sign JWT tokens |
| `JWT_ACCESS_TOKEN_EXPIRY` | | `15m` | Access token lifetime |
| `JWT_REFRESH_TOKEN_EXPIRY` | | `168h` | Refresh token lifetime (7 days) |
| `DATABASE_URL` | | — | PostgreSQL connection string |
| `DB_PATH` | | `transit.db` | SQLite file path (used when `DATABASE_URL` is not set) |
| `PORT` | | `8080` | HTTP server port |

### 3. Install dependencies

```bash
go mod download
```

### 4. Run the server

```bash
go run main.go
```

The server will start at **`http://localhost:8080`**.

You can verify it's running:

```bash
curl http://localhost:8080/health
# → {"status":"ok"}
```

### 5. Build a binary (optional)

```bash
go build -o transit-server .
./transit-server
```

---

## 📡 API Overview

Base URL: `/api/v1`

### Auth
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/auth/forgot-password` | Request password reset |
| POST | `/auth/reset-password` | Reset password with token |
| POST | `/auth/refresh` | Refresh access token |
| POST | `/auth/logout` | Logout (🔒 auth) |

### Driver
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/driver/register` | Register a new driver |
| POST | `/driver/login` | Driver login |
| GET | `/driver/me` | Get current driver profile (🔒) |
| POST | `/driver/join` | Join an aggregator (🔒) |
| PUT | `/driver/location` | Update GPS location (🔒) |
| POST | `/driver/locations/batch` | Batch location update (🔒) |
| POST | `/driver/trip/start` | Start a trip (🔒) |
| POST | `/driver/trip/end` | End a trip (🔒) |

### Aggregator
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/aggregator/register` | Register aggregator |
| POST | `/aggregator/login` | Aggregator login |
| GET | `/aggregator/me` | Profile (🔒 JWT) |
| GET | `/aggregator/api-key` | View API key (🔒 JWT) |
| PUT | `/aggregator/api-key` | Rotate API key (🔒 JWT) |
| GET | `/aggregator/drivers` | List all drivers (🔒 JWT + API Key) |
| GET | `/aggregator/drivers/locations` | All driver locations (🔒 JWT + API Key) |
| GET | `/aggregator/drivers/:id` | Single driver info (🔒 JWT + API Key) |
| GET | `/aggregator/drivers/:id/location` | Single driver location (🔒 JWT + API Key) |
| POST | `/aggregator/routes` | Create a route (🔒 JWT + API Key) |
| GET | `/aggregator/routes` | List routes (🔒 JWT + API Key) |
| POST | `/aggregator/trips` | Create a trip (🔒 JWT + API Key) |
| GET | `/aggregator/trips` | List trips (🔒 JWT + API Key) |

### GTFS-Realtime Feeds
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/aggregator/feed/vehicle-positions` | Protobuf feed — all vehicles (🔒) |
| GET | `/aggregator/feed/vehicle-positions/debug` | JSON debug feed (🔒) |
| GET | `/aggregator/feed/vehicle-positions/:driverId` | Protobuf feed — single vehicle (🔒) |
| GET | `/aggregator/feed/vehicle-positions/:driverId/debug` | JSON debug — single vehicle (🔒) |

### WebSocket
| Endpoint | Description |
|----------|-------------|
| `/api/v1/aggregator/subscribe` | Live location stream (auth via query params) |

> 🔒 = Requires authentication. See `API_CONTRACTS.md` for full request/response schemas.

---

## 🧰 Tech Stack

- **Language:** Go
- **HTTP Framework:** [Gin](https://github.com/gin-gonic/gin)
- **ORM:** [GORM](https://gorm.io) (PostgreSQL + SQLite drivers)
- **Auth:** JWT ([golang-jwt](https://github.com/golang-jwt/jwt)) + bcrypt
- **WebSockets:** [Gorilla WebSocket](https://github.com/gorilla/websocket)
- **GTFS-RT:** [Protocol Buffers](https://github.com/MobilityData/gtfs-realtime-bindings)

---

## 📄 License

This project is unlicensed. Add a `LICENSE` file to specify terms.