# Transit Server - API Contracts

> Base URL: `http://localhost:8080`
> All protected endpoints require `Authorization: Bearer <access_token>` header.
> Aggregator-scoped endpoints additionally require `X-API-Key: <api_key>` header.

---

## Table of Contents

1. [Health Check](#1-health-check)
2. [Driver Auth](#2-driver-auth)
3. [Aggregator Auth](#3-aggregator-auth)
4. [Shared Auth](#4-shared-auth)
5. [Driver Actions](#5-driver-actions)
6. [Aggregator - Profile](#6-aggregator---profile)
7. [Aggregator - Driver Management](#7-aggregator---driver-management)
8. [Aggregator - Routes & Trips](#8-aggregator---routes--trips)
9. [Aggregator - GTFS-RT Feeds](#9-aggregator---gtfs-rt-feeds)
10. [Aggregator - WebSocket](#10-aggregator---websocket)
11. [Error Format](#11-error-format)
12. [Auth Flow Summary](#12-auth-flow-summary)

---

## 1. Health Check

### `GET /health`

**Auth:** None

**Response** `200 OK`
```json
{
  "status": "ok"
}
```

---

## 2. Driver Auth

### `POST /api/v1/driver/register`

**Auth:** None

**Request Body:**
```json
{
  "email": "driver@example.com",
  "password": "SecurePass1!",
  "first_name": "John",
  "last_name": "Doe",
  "license_number": "DL-12345",
  "phone": "+919876543210",
  "vehicle_number": "KA-01-AB-1234",
  "vehicle_type": "bus"
}
```

| Field            | Type   | Required | Notes                                                     |
|------------------|--------|----------|-----------------------------------------------------------|
| email            | string | yes      | Must be unique, valid email                               |
| password         | string | yes      | Min 8 chars, must have uppercase, lowercase, digit, special char |
| first_name       | string | yes      |                                                           |
| last_name        | string | yes      |                                                           |
| license_number   | string | yes      |                                                           |
| phone            | string | yes      |                                                           |
| vehicle_number   | string | yes      |                                                           |
| vehicle_type     | string | yes      | e.g. "bus", "mini-bus", "van"                             |

**Response** `201 Created`
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
  "token_type": "Bearer",
  "expires_in": 900,
  "user": {
    "id": 1,
    "email": "driver@example.com",
    "first_name": "John",
    "last_name": "Doe",
    "role": "driver",
    "is_active": true,
    "created_at": "2026-03-31T10:00:00Z",
    "updated_at": "2026-03-31T10:00:00Z"
  }
}
```

**Errors:**
- `400` - Validation failed (weak password, missing fields)
- `409` - Email already registered

---

### `POST /api/v1/driver/login`

**Auth:** None

**Request Body:**
```json
{
  "email": "driver@example.com",
  "password": "SecurePass1!"
}
```

| Field    | Type   | Required |
|----------|--------|----------|
| email    | string | yes      |
| password | string | yes      |

**Response** `200 OK`
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
  "token_type": "Bearer",
  "expires_in": 900,
  "user": {
    "id": 1,
    "email": "driver@example.com",
    "first_name": "John",
    "last_name": "Doe",
    "role": "driver",
    "is_active": true,
    "created_at": "2026-03-31T10:00:00Z",
    "updated_at": "2026-03-31T10:00:00Z"
  }
}
```

**Errors:**
- `400` - Missing fields
- `401` - Invalid credentials or account inactive
- `403` - Wrong role (not a driver account)

---

## 3. Aggregator Auth

### `POST /api/v1/aggregator/register`

**Auth:** None

**Request Body:**
```json
{
  "email": "agency@example.com",
  "password": "SecurePass1!",
  "first_name": "Jane",
  "last_name": "Smith",
  "company_name": "City Transit Co.",
  "phone": "+919876543210"
}
```

| Field        | Type   | Required | Notes                                |
|--------------|--------|----------|--------------------------------------|
| email        | string | yes      | Must be unique                       |
| password     | string | yes      | Same strength rules as driver        |
| first_name   | string | yes      |                                      |
| last_name    | string | yes      |                                      |
| company_name | string | yes      |                                      |
| phone        | string | yes      |                                      |

**Response** `201 Created`
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
  "token_type": "Bearer",
  "expires_in": 900,
  "user": {
    "id": 2,
    "email": "agency@example.com",
    "first_name": "Jane",
    "last_name": "Smith",
    "role": "aggregator",
    "is_active": true,
    "created_at": "2026-03-31T10:00:00Z",
    "updated_at": "2026-03-31T10:00:00Z"
  }
}
```

> **Important:** On registration, the server generates:
> - `invite_code` (5-char alphanumeric) — share with drivers to join
> - `api_key` (64-hex string) — use in `X-API-Key` header for all aggregator endpoints
>
> Retrieve these via `GET /api/v1/aggregator/me` after login.

**Errors:**
- `400` - Validation failed
- `409` - Email already registered

---

### `POST /api/v1/aggregator/login`

**Auth:** None

**Request Body:**
```json
{
  "email": "agency@example.com",
  "password": "SecurePass1!"
}
```

**Response** `200 OK` — Same format as driver login, with `role: "aggregator"`

**Errors:**
- `401` - Invalid credentials
- `403` - Wrong role (not aggregator/admin)

---

## 4. Shared Auth

### `POST /api/v1/auth/refresh`

**Auth:** None (uses refresh token in body)

**Request Body:**
```json
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIs..."
}
```

**Response** `200 OK`
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
  "token_type": "Bearer",
  "expires_in": 900,
  "user": {
    "id": 1,
    "email": "driver@example.com",
    "first_name": "John",
    "last_name": "Doe",
    "role": "driver",
    "is_active": true,
    "created_at": "2026-03-31T10:00:00Z",
    "updated_at": "2026-03-31T10:00:00Z"
  }
}
```

**Errors:**
- `400` - Missing refresh token
- `401` - Invalid, expired, or blacklisted refresh token

---

### `POST /api/v1/auth/logout`

**Auth:** `Bearer <access_token>`

**Request Body:** None

**Response** `200 OK`
```json
{
  "message": "logged out successfully"
}
```

> Blacklists the current access token. Client should also discard the refresh token.

**Errors:**
- `401` - Missing or invalid token

---

### `POST /api/v1/auth/forgot-password`

**Auth:** None

**Request Body:**
```json
{
  "email": "driver@example.com"
}
```

**Response** `200 OK`
```json
{
  "message": "if an account exists with this email, a reset link has been sent"
}
```

> Always returns 200 regardless of whether the email exists (prevents enumeration).
> In development, the reset token is logged to stdout.

---

### `POST /api/v1/auth/reset-password`

**Auth:** None

**Request Body:**
```json
{
  "token": "abc123def456...",
  "new_password": "NewSecurePass1!"
}
```

| Field        | Type   | Required | Notes                     |
|--------------|--------|----------|---------------------------|
| token        | string | yes      | From forgot-password flow |
| new_password | string | yes      | Same strength rules apply |

**Response** `200 OK`
```json
{
  "message": "password reset successfully"
}
```

**Errors:**
- `400` - Invalid or expired token, weak password

---

## 5. Driver Actions

> All endpoints require: `Authorization: Bearer <access_token>` with role `driver`

### `GET /api/v1/driver/me`

**Response** `200 OK`
```json
{
  "id": 1,
  "user": {
    "id": 1,
    "email": "driver@example.com",
    "first_name": "John",
    "last_name": "Doe",
    "role": "driver",
    "is_active": true,
    "created_at": "2026-03-31T10:00:00Z",
    "updated_at": "2026-03-31T10:00:00Z"
  },
  "license_number": "DL-12345",
  "phone": "+919876543210",
  "vehicle_number": "KA-01-AB-1234",
  "vehicle_type": "bus",
  "is_available": true,
  "created_at": "2026-03-31T10:00:00Z",
  "updated_at": "2026-03-31T10:00:00Z"
}
```

---

### `POST /api/v1/driver/join`

Join an aggregator using their invite code. This creates the driver-aggregator mapping.

**Request Body:**
```json
{
  "invite_code": "A1B2C"
}
```

| Field       | Type   | Required | Notes                  |
|-------------|--------|----------|------------------------|
| invite_code | string | yes      | 5-char alphanumeric    |

**Response** `200 OK`
```json
{
  "message": "successfully joined aggregator"
}
```

**Errors:**
- `400` - Missing invite code
- `404` - Invalid invite code (no matching aggregator)
- `409` - Already mapped to this aggregator

---

### `PUT /api/v1/driver/location`

Push a GPS location update. This updates cache, database, and broadcasts to aggregator WebSocket subscribers.

**Request Body:**
```json
{
  "lat": 12.9716,
  "lng": 77.5946,
  "heading": 45.5,
  "speed": 25.3
}
```

| Field   | Type   | Required | Notes              |
|---------|--------|----------|--------------------|
| lat     | float  | yes      | Latitude (-90..90) |
| lng     | float  | yes      | Longitude (-180..180) |
| heading | float  | no       | Degrees 0-360      |
| speed   | float  | no       | km/h               |

**Response** `200 OK`
```json
{
  "message": "location updated"
}
```

**Errors:**
- `400` - Missing or invalid coordinates
- `404` - Driver profile not found

---

### `POST /api/v1/driver/locations/batch`

Sync multiple offline-collected locations at once. Server uses the latest timestamp entry.

**Request Body:**
```json
{
  "locations": [
    {
      "lat": 12.9716,
      "lng": 77.5946,
      "heading": 45.5,
      "speed": 25.3,
      "timestamp": "2026-03-31T10:00:00Z"
    },
    {
      "lat": 12.9720,
      "lng": 77.5950,
      "heading": 46.0,
      "speed": 30.1,
      "timestamp": "2026-03-31T10:00:05Z"
    }
  ]
}
```

| Field               | Type   | Required | Notes       |
|---------------------|--------|----------|-------------|
| locations           | array  | yes      | Min 1 entry |
| locations[].lat     | float  | yes      |             |
| locations[].lng     | float  | yes      |             |
| locations[].heading | float  | no       |             |
| locations[].speed   | float  | no       |             |
| locations[].timestamp | string | yes    | ISO 8601    |

**Response** `200 OK`
```json
{
  "message": "batch location update processed",
  "count": 2
}
```

---

### `POST /api/v1/driver/trip/start`

Start driving a trip. Links the driver to a trip as the active vehicle.

**Request Body:**
```json
{
  "trip_id": "TRIP-001",
  "vehicle_id": "BUS-042"
}
```

| Field      | Type   | Required | Notes                     |
|------------|--------|----------|---------------------------|
| trip_id    | string | yes      | GTFS trip ID (must exist) |
| vehicle_id | string | yes      | Vehicle label/identifier  |

**Response** `201 Created`
```json
{
  "id": 1,
  "driver_id": 1,
  "trip": {
    "id": 1,
    "route_id": 1,
    "gtfs_trip_id": "TRIP-001",
    "headsign": "Downtown",
    "direction_id": 0,
    "is_active": true,
    "created_at": "2026-03-31T10:00:00Z",
    "updated_at": "2026-03-31T10:00:00Z"
  },
  "vehicle_id": "BUS-042",
  "started_at": "2026-03-31T10:05:00Z",
  "is_active": true,
  "created_at": "2026-03-31T10:05:00Z",
  "updated_at": "2026-03-31T10:05:00Z"
}
```

**Errors:**
- `400` - Missing fields or driver already has an active trip
- `404` - Trip not found

---

### `POST /api/v1/driver/trip/end`

End the current active trip.

**Request Body:** None

**Response** `200 OK`
```json
{
  "message": "trip ended successfully"
}
```

**Errors:**
- `400` - No active trip found

---

## 6. Aggregator - Profile

> All aggregator endpoints require: `Authorization: Bearer <access_token>` (role: aggregator or admin) **AND** `X-API-Key: <api_key>`

### `GET /api/v1/aggregator/me`

Returns the aggregator profile including the **invite_code** and **api_key**.

**Response** `200 OK`
```json
{
  "id": 1,
  "user": {
    "id": 2,
    "email": "agency@example.com",
    "first_name": "Jane",
    "last_name": "Smith",
    "role": "aggregator",
    "is_active": true,
    "created_at": "2026-03-31T10:00:00Z",
    "updated_at": "2026-03-31T10:00:00Z"
  },
  "company_name": "City Transit Co.",
  "phone": "+919876543210",
  "invite_code": "A1B2C",
  "api_key": "a3f8b2c1d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1",
  "created_at": "2026-03-31T10:00:00Z",
  "updated_at": "2026-03-31T10:00:00Z"
}
```

---

## 7. Aggregator - Driver Management

> All endpoints require: `Bearer <access_token>` + `X-API-Key`
> Only returns drivers **mapped to this aggregator**.

### `GET /api/v1/aggregator/drivers`

List all drivers mapped to this aggregator.

**Response** `200 OK`
```json
[
  {
    "id": 1,
    "user": {
      "id": 1,
      "email": "driver@example.com",
      "first_name": "John",
      "last_name": "Doe",
      "role": "driver",
      "is_active": true,
      "created_at": "2026-03-31T10:00:00Z",
      "updated_at": "2026-03-31T10:00:00Z"
    },
    "license_number": "DL-12345",
    "phone": "+919876543210",
    "vehicle_number": "KA-01-AB-1234",
    "vehicle_type": "bus",
    "is_available": true,
    "created_at": "2026-03-31T10:00:00Z",
    "updated_at": "2026-03-31T10:00:00Z"
  }
]
```

---

### `GET /api/v1/aggregator/drivers/:id`

Get a specific mapped driver by driver ID.

**URL Params:**
| Param | Type | Notes                   |
|-------|------|-------------------------|
| id    | uint | Driver ID (not user ID) |

**Response** `200 OK` — Single driver object (same shape as array item above)

**Errors:**
- `403` - Driver not mapped to this aggregator
- `404` - Driver not found

---

### `GET /api/v1/aggregator/drivers/:id/location`

Get a specific driver's current location.

**URL Params:**
| Param | Type | Notes    |
|-------|------|----------|
| id    | uint | Driver ID |

**Response** `200 OK`
```json
{
  "driver_id": 1,
  "status": "online",
  "lat": 12.9716,
  "lng": 77.5946,
  "heading": 45.5,
  "speed": 25.3,
  "updated_at": "2026-03-31T10:05:00Z"
}
```

| Field      | Notes                                                    |
|------------|----------------------------------------------------------|
| status     | `"online"` if location in cache (< 60s old), else `"offline"` |
| lat/lng    | From cache if online, from DB `last_*` fields if offline |
| updated_at | Timestamp of last location update                        |

**Errors:**
- `403` - Driver not mapped to this aggregator
- `404` - Driver not found or no location data

---

### `GET /api/v1/aggregator/drivers/locations`

Get locations of **all** drivers mapped to this aggregator.

**Response** `200 OK`
```json
[
  {
    "driver_id": 1,
    "status": "online",
    "lat": 12.9716,
    "lng": 77.5946,
    "heading": 45.5,
    "speed": 25.3,
    "updated_at": "2026-03-31T10:05:00Z"
  },
  {
    "driver_id": 2,
    "status": "offline",
    "lat": 12.9800,
    "lng": 77.6000,
    "heading": 0,
    "speed": 0,
    "updated_at": "2026-03-31T09:50:00Z"
  }
]
```

---

## 8. Aggregator - Routes & Trips

> All endpoints require: `Bearer <access_token>` + `X-API-Key`

### `POST /api/v1/aggregator/routes`

Create a transit route (GTFS-compatible).

**Request Body:**
```json
{
  "route_id": "ROUTE-01",
  "short_name": "R1",
  "long_name": "Airport Express",
  "description": "Airport to City Center",
  "route_type": 3,
  "color": "FF0000",
  "text_color": "FFFFFF"
}
```

| Field       | Type   | Required | Notes                                        |
|-------------|--------|----------|----------------------------------------------|
| route_id    | string | yes      | GTFS route_id, must be unique globally       |
| short_name  | string | yes      | Short display name                           |
| long_name   | string | yes      | Full route name                              |
| description | string | no       |                                              |
| route_type  | int    | no       | GTFS route type (default 3 = Bus)            |
| color       | string | no       | Hex color without # (e.g. "FF0000")          |
| text_color  | string | no       | Hex text color                               |

**Response** `201 Created`
```json
{
  "id": 1,
  "agency_id": 1,
  "gtfs_route_id": "ROUTE-01",
  "short_name": "R1",
  "long_name": "Airport Express",
  "description": "Airport to City Center",
  "route_type": 3,
  "color": "FF0000",
  "text_color": "FFFFFF",
  "is_active": true,
  "created_at": "2026-03-31T10:00:00Z",
  "updated_at": "2026-03-31T10:00:00Z"
}
```

**Errors:**
- `400` - Validation failed
- `409` - Route ID already exists

---

### `GET /api/v1/aggregator/routes`

List all routes for this aggregator.

**Response** `200 OK`
```json
[
  {
    "id": 1,
    "agency_id": 1,
    "gtfs_route_id": "ROUTE-01",
    "short_name": "R1",
    "long_name": "Airport Express",
    "description": "Airport to City Center",
    "route_type": 3,
    "color": "FF0000",
    "text_color": "FFFFFF",
    "is_active": true,
    "created_at": "2026-03-31T10:00:00Z",
    "updated_at": "2026-03-31T10:00:00Z"
  }
]
```

---

### `POST /api/v1/aggregator/trips`

Create a trip on an existing route.

**Request Body:**
```json
{
  "route_id": "ROUTE-01",
  "trip_id": "TRIP-001",
  "headsign": "Downtown Terminal",
  "direction_id": 0
}
```

| Field        | Type   | Required | Notes                                   |
|--------------|--------|----------|-----------------------------------------|
| route_id     | string | yes      | GTFS route_id (must exist, owned by you)|
| trip_id      | string | yes      | GTFS trip_id, must be unique globally   |
| headsign     | string | yes      | Destination display text                |
| direction_id | int    | no       | 0 = outbound, 1 = inbound              |

**Response** `201 Created`
```json
{
  "id": 1,
  "route_id": 1,
  "gtfs_trip_id": "TRIP-001",
  "headsign": "Downtown Terminal",
  "direction_id": 0,
  "is_active": true,
  "created_at": "2026-03-31T10:00:00Z",
  "updated_at": "2026-03-31T10:00:00Z"
}
```

**Errors:**
- `400` - Validation failed
- `404` - Route not found or not owned by this aggregator
- `409` - Trip ID already exists

---

### `GET /api/v1/aggregator/trips`

List all trips for routes owned by this aggregator.

**Response** `200 OK`
```json
[
  {
    "id": 1,
    "route_id": 1,
    "gtfs_trip_id": "TRIP-001",
    "headsign": "Downtown Terminal",
    "direction_id": 0,
    "is_active": true,
    "created_at": "2026-03-31T10:00:00Z",
    "updated_at": "2026-03-31T10:00:00Z"
  }
]
```

---

## 9. Aggregator - GTFS-RT Feeds

> All endpoints require: `Bearer <access_token>` + `X-API-Key`
> Returns GTFS-Realtime protobuf data for vehicles with active trips.

### `GET /api/v1/aggregator/feed/vehicle-positions`

Get protobuf feed of all active vehicles for this aggregator.

**Response** `200 OK`
- Content-Type: `application/x-protobuf`
- Body: Binary protobuf `FeedMessage` (GTFS-RT spec)

---

### `GET /api/v1/aggregator/feed/vehicle-positions/debug`

JSON debug version of the above feed.

**Response** `200 OK`
```json
{
  "header": {
    "gtfs_realtime_version": "2.0",
    "incrementality": "FULL_DATASET",
    "timestamp": 1711875900
  },
  "entity": [
    {
      "id": "vehicle-1",
      "vehicle": {
        "trip": {
          "trip_id": "TRIP-001",
          "route_id": "ROUTE-01"
        },
        "position": {
          "latitude": 12.9716,
          "longitude": 77.5946,
          "bearing": 45.5,
          "speed": 7.03
        },
        "vehicle": {
          "id": "1",
          "label": "BUS-042"
        },
        "timestamp": 1711875900
      }
    }
  ]
}
```

> **Note:** `speed` in GTFS-RT is in **meters/second** (converted from km/h internally).

---

### `GET /api/v1/aggregator/feed/vehicle-positions/:driverId`

Protobuf feed for a single vehicle.

**URL Params:**
| Param    | Type | Notes     |
|----------|------|-----------|
| driverId | uint | Driver ID |

**Response** `200 OK` — Binary protobuf (single entity)

**Errors:**
- `403` - Driver not mapped to this aggregator
- `404` - Driver not found or no active trip

---

### `GET /api/v1/aggregator/feed/vehicle-positions/:driverId/debug`

JSON debug version for a single vehicle.

---

## 10. Aggregator - WebSocket

### `GET /api/v1/aggregator/subscribe?token=<jwt>&api_key=<key>`

Upgrade to WebSocket for real-time location streaming.

**Query Params:**
| Param   | Type   | Required | Notes                      |
|---------|--------|----------|----------------------------|
| token   | string | yes      | JWT access token           |
| api_key | string | yes      | Aggregator API key         |

**Connection Flow:**
1. Client connects with query params
2. Server validates JWT + API key + ownership match
3. HTTP upgraded to WebSocket
4. Server pushes events as JSON messages

**Incoming Messages (Server -> Client):**
```json
{
  "event": "location_update",
  "data": {
    "driver_id": 1,
    "vehicle_id": "BUS-042",
    "lat": 12.9716,
    "lng": 77.5946,
    "heading": 45.5,
    "speed": 25.3,
    "timestamp": "2026-03-31T10:05:00Z",
    "is_online": true
  }
}
```

| Field      | Type    | Notes                              |
|------------|---------|------------------------------------|
| event      | string  | Always `"location_update"` for now |
| data.driver_id | uint | Driver ID                        |
| data.vehicle_id | string | From active trip, or empty      |
| data.lat   | float   | Latitude                           |
| data.lng   | float   | Longitude                          |
| data.heading | float | Degrees 0-360                      |
| data.speed | float   | km/h                               |
| data.timestamp | string | ISO 8601                        |
| data.is_online | bool | true if from live update          |

**Keep-Alive:** Server sends ping every 54 seconds. Client must respond with pong within 60 seconds or connection is closed.

**Errors:**
- `401` - Invalid token or API key
- `403` - Token user doesn't match API key owner

---

## 11. Error Format

All errors follow this format:

```json
{
  "error": "human readable error message"
}
```

With optional validation details:
```json
{
  "error": "validation failed",
  "details": {
    "email": "required",
    "password": "must be at least 8 characters"
  }
}
```

**Standard HTTP Status Codes:**
| Code | Meaning                          |
|------|----------------------------------|
| 200  | Success                          |
| 201  | Created                          |
| 400  | Bad request / validation error   |
| 401  | Unauthorized (missing/invalid auth) |
| 403  | Forbidden (wrong role or access) |
| 404  | Not found                        |
| 409  | Conflict (duplicate resource)    |
| 500  | Internal server error            |

---

## 12. Auth Flow Summary

### For Android Driver App:

```
1. Register:    POST /api/v1/driver/register
                -> Store access_token & refresh_token

2. Login:       POST /api/v1/driver/login
                -> Store access_token & refresh_token

3. Join Agency: POST /api/v1/driver/join  { invite_code: "A1B2C" }
                -> Driver is now mapped to that aggregator

4. Start Trip:  POST /api/v1/driver/trip/start  { trip_id, vehicle_id }
                -> Driver is now on an active trip

5. Send GPS:    PUT /api/v1/driver/location  { lat, lng, heading, speed }
                -> Call every 5-10 seconds while trip is active
                -> Use batch endpoint for offline sync

6. End Trip:    POST /api/v1/driver/trip/end

7. Logout:      POST /api/v1/auth/logout

Token Refresh:  POST /api/v1/auth/refresh  { refresh_token }
                -> Call when access_token expires (every 15 min)
```

### For Android Aggregator App:

```
1. Register:    POST /api/v1/aggregator/register
                -> Store access_token & refresh_token

2. Login:       POST /api/v1/aggregator/login
                -> Store access_token & refresh_token

3. Get Profile: GET /api/v1/aggregator/me
                -> Save invite_code (share with drivers)
                -> Save api_key (use in X-API-Key header)

4. Setup:
   - Create routes:  POST /api/v1/aggregator/routes
   - Create trips:   POST /api/v1/aggregator/trips

5. View Drivers:
   - List all:        GET /api/v1/aggregator/drivers
   - All locations:   GET /api/v1/aggregator/drivers/locations

6. Real-time:   Connect WebSocket to /api/v1/aggregator/subscribe?token=X&api_key=Y
                -> Receive live location_update events

7. Logout:      POST /api/v1/auth/logout
```

### Headers Cheat Sheet:

```
# Driver endpoints (after login):
Authorization: Bearer <access_token>

# Aggregator endpoints (after login):
Authorization: Bearer <access_token>
X-API-Key: <api_key>

# Content type for all POST/PUT:
Content-Type: application/json
```
