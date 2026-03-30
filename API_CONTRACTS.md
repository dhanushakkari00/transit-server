# Transit Server - API Contracts for Android Client

> **Target Platform:** Android (Kotlin, Gradle)
> **Server:** Go (Gin framework), SQLite, JWT auth, Gorilla WebSocket
> **Base URL (dev):** `http://10.0.2.2:8080` (Android emulator → localhost)
> **Base URL (prod):** `https://your-domain.com`
> **All request/response bodies:** `Content-Type: application/json` (unless noted otherwise)
> **All timestamps:** ISO 8601 format `"2026-03-31T10:00:00Z"`

---

## Table of Contents

1. [Client Setup & Recommended Libraries](#1-client-setup--recommended-libraries)
2. [Kotlin Data Classes](#2-kotlin-data-classes)
3. [Authentication & Session Management](#3-authentication--session-management)
4. [API Endpoints - Health](#4-api-endpoints---health)
5. [API Endpoints - Driver Auth](#5-api-endpoints---driver-auth)
6. [API Endpoints - Aggregator Auth](#6-api-endpoints---aggregator-auth)
7. [API Endpoints - Shared Auth](#7-api-endpoints---shared-auth)
8. [API Endpoints - Driver Actions](#8-api-endpoints---driver-actions)
9. [API Endpoints - Aggregator Profile](#9-api-endpoints---aggregator-profile)
10. [API Endpoints - Aggregator Driver Management](#10-api-endpoints---aggregator-driver-management)
11. [API Endpoints - Aggregator Routes & Trips](#11-api-endpoints---aggregator-routes--trips)
12. [API Endpoints - Aggregator GTFS-RT Feeds](#12-api-endpoints---aggregator-gtfs-rt-feeds)
13. [WebSocket - Real-Time Location Streaming](#13-websocket---real-time-location-streaming)
14. [Error Handling](#14-error-handling)
15. [Complete Endpoint Reference Table](#15-complete-endpoint-reference-table)
16. [App Flow - Driver](#16-app-flow---driver)
17. [App Flow - Aggregator](#17-app-flow---aggregator)

---

## 1. Client Setup & Recommended Libraries

### Gradle Dependencies (build.gradle.kts)

```kotlin
// Networking
implementation("com.squareup.retrofit2:retrofit:2.9.0")
implementation("com.squareup.retrofit2:converter-gson:2.9.0")
implementation("com.squareup.okhttp3:okhttp:4.12.0")
implementation("com.squareup.okhttp3:logging-interceptor:4.12.0")

// WebSocket (included in OkHttp, no extra dependency needed)

// JSON
implementation("com.google.code.gson:gson:2.10.1")

// Coroutines
implementation("org.jetbrains.kotlinx:kotlinx-coroutines-android:1.7.3")

// Secure token storage
implementation("androidx.security:security-crypto:1.1.0-alpha06")

// Location services
implementation("com.google.android.gms:play-services-location:21.0.1")
```

### Headers

Every HTTP request must include the appropriate headers:

| Scenario | Headers |
|----------|---------|
| Public endpoints (register, login, refresh, forgot/reset-password) | `Content-Type: application/json` |
| Driver endpoints (after login) | `Content-Type: application/json` + `Authorization: Bearer <access_token>` |
| Aggregator profile + key endpoints | `Content-Type: application/json` + `Authorization: Bearer <access_token>` |
| Other aggregator endpoints | `Content-Type: application/json` + `Authorization: Bearer <access_token>` + `X-API-Key: <api_key>` |
| WebSocket connection | Auth passed as **query params**, not headers (see Section 13) |

---

## 2. Kotlin Data Classes

These are the exact JSON shapes returned by the server. Use these as your Retrofit/Gson models.

### Auth Models

```kotlin
// Used by: POST /driver/register, POST /driver/login,
//          POST /aggregator/register, POST /aggregator/login,
//          POST /auth/refresh
data class AuthResponse(
    val access_token: String,
    val refresh_token: String,
    val token_type: String,       // Always "Bearer"
    val expires_in: Int,          // Seconds until access_token expires (default: 900 = 15 min)
    val user: User
)

data class User(
    val id: Int,
    val email: String,
    val first_name: String,
    val last_name: String,
    val role: String,             // "driver" | "aggregator" | "admin"
    val is_active: Boolean,
    val created_at: String,       // ISO 8601
    val updated_at: String        // ISO 8601
)

data class MessageResponse(
    val message: String
)

data class ErrorResponse(
    val error: String,
    val details: Map<String, String>? = null  // Only present for validation errors
)
```

### Driver Models

```kotlin
// Request: POST /driver/register
data class DriverRegisterRequest(
    val email: String,
    val password: String,
    val first_name: String,
    val last_name: String,
    val license_number: String,
    val phone: String,
    val vehicle_number: String,
    val vehicle_type: String       // "bus", "mini-bus", "van"
)

// Request: POST /driver/login, POST /aggregator/login
data class LoginRequest(
    val email: String,
    val password: String
)

// Response: GET /driver/me
data class DriverProfile(
    val id: Int,
    val user: User,
    val license_number: String,
    val phone: String,
    val vehicle_number: String,
    val vehicle_type: String,
    val is_available: Boolean,
    val created_at: String,
    val updated_at: String
)

// Request: POST /driver/join
data class JoinAggregatorRequest(
    val invite_code: String        // 5-char alphanumeric
)

// Request: PUT /driver/location
data class LocationUpdateRequest(
    val lat: Double,
    val lng: Double,
    val heading: Double? = null,   // Degrees 0-360
    val speed: Double? = null      // km/h
)

// Request: POST /driver/locations/batch
data class BatchLocationRequest(
    val locations: List<BatchLocationEntry>
)

data class BatchLocationEntry(
    val lat: Double,
    val lng: Double,
    val heading: Double? = null,
    val speed: Double? = null,
    val timestamp: String          // ISO 8601 - REQUIRED
)

// Response: POST /driver/locations/batch
data class BatchLocationResponse(
    val message: String,
    val count: Int
)

// Request: POST /driver/trip/start
data class StartTripRequest(
    val trip_id: String,           // GTFS trip ID (string, not int)
    val vehicle_id: String         // Vehicle label e.g. "BUS-042"
)

// Response: POST /driver/trip/start
data class ActiveTripResponse(
    val id: Int,
    val driver_id: Int,
    val trip: TripInfo,
    val vehicle_id: String,
    val started_at: String,
    val is_active: Boolean,
    val created_at: String,
    val updated_at: String
)

data class TripInfo(
    val id: Int,
    val route_id: Int,
    val gtfs_trip_id: String,
    val headsign: String,
    val direction_id: Int,
    val is_active: Boolean,
    val created_at: String,
    val updated_at: String
)
```

### Aggregator Models

```kotlin
// Request: POST /aggregator/register
data class AggregatorRegisterRequest(
    val email: String,
    val password: String,
    val first_name: String,
    val last_name: String,
    val company_name: String,
    val phone: String
)

// Response: GET /aggregator/me
data class AggregatorProfile(
    val id: Int,
    val user: User,
    val company_name: String,
    val phone: String,
    val invite_code: String,       // Share with drivers so they can join
    val api_key: String,           // 64-hex-char string, use in X-API-Key header
    val created_at: String,
    val updated_at: String
)

// Response: GET /aggregator/drivers/:id/location, GET /aggregator/drivers/locations
data class DriverLocation(
    val driver_id: Int,
    val status: String,            // "online" | "offline"
    val lat: Double,
    val lng: Double,
    val heading: Double,
    val speed: Double,
    val updated_at: String
)
```

### Route & Trip Models

```kotlin
// Request: POST /aggregator/routes
data class CreateRouteRequest(
    val route_id: String,          // GTFS route_id, globally unique
    val short_name: String,
    val long_name: String,
    val description: String? = null,
    val route_type: Int? = 3,      // GTFS type: 0=Tram, 1=Subway, 2=Rail, 3=Bus
    val color: String? = null,     // Hex without #, e.g. "FF0000"
    val text_color: String? = null
)

// Response: POST /aggregator/routes, GET /aggregator/routes items
data class Route(
    val id: Int,
    val agency_id: Int,
    val gtfs_route_id: String,
    val short_name: String,
    val long_name: String,
    val description: String?,
    val route_type: Int,
    val color: String?,
    val text_color: String?,
    val is_active: Boolean,
    val created_at: String,
    val updated_at: String
)

// Request: POST /aggregator/trips
data class CreateTripRequest(
    val route_id: String,          // GTFS route_id (string, must exist and be owned by you)
    val trip_id: String,           // GTFS trip_id, globally unique
    val headsign: String,
    val direction_id: Int? = 0     // 0 = outbound, 1 = inbound
)

// Response: POST /aggregator/trips, GET /aggregator/trips items
data class Trip(
    val id: Int,
    val route_id: Int,
    val gtfs_trip_id: String,
    val headsign: String,
    val direction_id: Int,
    val is_active: Boolean,
    val created_at: String,
    val updated_at: String
)
```

### WebSocket Models

```kotlin
// Every WebSocket message from server has this envelope
data class WebSocketMessage(
    val event: String,             // Currently only "location_update"
    val data: LocationEventData
)

data class LocationEventData(
    val driver_id: Int,
    val vehicle_id: String,        // From active trip, or "" if none
    val lat: Double,
    val lng: Double,
    val heading: Double,
    val speed: Double,             // km/h
    val timestamp: String,         // ISO 8601
    val is_online: Boolean         // true = live update, false = stale/cached
)
```

### Shared Auth Models

```kotlin
// Request: POST /auth/refresh
data class RefreshTokenRequest(
    val refresh_token: String
)

// Request: POST /auth/forgot-password
data class ForgotPasswordRequest(
    val email: String
)

// Request: POST /auth/reset-password
data class ResetPasswordRequest(
    val token: String,
    val new_password: String
)
```

---

## 3. Authentication & Session Management

### Token Architecture

The server uses **JWT (HS256)** with two token types:

| Token | Lifetime | Purpose | Storage |
|-------|----------|---------|---------|
| `access_token` | **15 minutes** (900s) | Sent in `Authorization: Bearer` header for every protected API call | EncryptedSharedPreferences |
| `refresh_token` | **7 days** (168h) | Used to get a new access_token when it expires | EncryptedSharedPreferences |
| `api_key` | **Permanent** (until regenerated) | Aggregator-only, sent in `X-API-Key` header | EncryptedSharedPreferences |

### Token Refresh Strategy

The `access_token` expires every 15 minutes. You **must** implement an OkHttp Interceptor/Authenticator that:

1. Detects `401 Unauthorized` responses
2. Calls `POST /api/v1/auth/refresh` with the stored `refresh_token`
3. Stores the new `access_token` and `refresh_token` from the response
4. Retries the original request with the new `access_token`
5. If refresh fails (401), redirect user to login screen and clear all stored tokens

**Important:** The refresh endpoint returns a **new** `refresh_token` along with the new `access_token`. You must store **both** — the old refresh token is invalidated.

### Login Flow (Both Roles)

```
1. User enters email + password
2. Call POST /api/v1/{driver|aggregator}/login
3. On 200: Store access_token, refresh_token, user object
4. For aggregator: Also call GET /api/v1/aggregator/me → store api_key and invite_code
5. Navigate to main screen
```

### Logout Flow

```
1. Call POST /api/v1/auth/logout (with current access_token)
   - Server blacklists the access_token
2. Clear ALL stored tokens (access_token, refresh_token, api_key) from device
3. Disconnect WebSocket if connected
4. Navigate to login screen
```

### Password Rules

The server enforces these rules on register and reset-password:
- Minimum 8 characters
- At least 1 uppercase letter
- At least 1 lowercase letter
- At least 1 digit
- At least 1 special character

Validate client-side before sending to avoid unnecessary network calls.

---

## 4. API Endpoints - Health

### `GET /health`

**Auth:** None

**Response** `200 OK`
```json
{
  "status": "ok"
}
```

Use this to check server connectivity on app launch or to implement a retry-on-disconnect mechanism.

---

## 5. API Endpoints - Driver Auth

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

| Field            | Type   | Required | Validation                                                |
|------------------|--------|----------|-----------------------------------------------------------|
| email            | string | yes      | Must be valid email format, must be unique                |
| password         | string | yes      | Min 8 chars, uppercase + lowercase + digit + special char |
| first_name       | string | yes      | Non-empty                                                 |
| last_name        | string | yes      | Non-empty                                                 |
| license_number   | string | yes      | Non-empty                                                 |
| phone            | string | yes      | Non-empty                                                 |
| vehicle_number   | string | yes      | Non-empty                                                 |
| vehicle_type     | string | yes      | e.g. "bus", "mini-bus", "van"                             |

**Response** `201 Created` → `AuthResponse`
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
| Status | Meaning |
|--------|---------|
| `400`  | Validation failed — missing fields, weak password, invalid email |
| `409`  | Email already registered |

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

**Response** `200 OK` → `AuthResponse` (same shape as register response)
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
| Status | Meaning |
|--------|---------|
| `400`  | Missing fields |
| `401`  | Invalid credentials or account deactivated |
| `403`  | Email belongs to a non-driver account (aggregator/admin) |

---

## 6. API Endpoints - Aggregator Auth

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

| Field        | Type   | Required | Validation                         |
|--------------|--------|----------|------------------------------------|
| email        | string | yes      | Valid email, must be unique        |
| password     | string | yes      | Same strength rules as driver      |
| first_name   | string | yes      | Non-empty                          |
| last_name    | string | yes      | Non-empty                          |
| company_name | string | yes      | Non-empty                          |
| phone        | string | yes      | Non-empty                          |

**Response** `201 Created` → `AuthResponse`
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

**Important:** On registration, the server auto-generates:
- `invite_code` (5-char alphanumeric) — share with drivers so they can join your agency
- `api_key` (64-hex string) — required in `X-API-Key` header for all aggregator endpoints

These are **NOT** returned in the register response. You **must** call `GET /api/v1/aggregator/me` after login to retrieve them.

**Errors:**
| Status | Meaning |
|--------|---------|
| `400`  | Validation failed |
| `409`  | Email already registered |

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

| Field    | Type   | Required |
|----------|--------|----------|
| email    | string | yes      |
| password | string | yes      |

**Response** `200 OK` → `AuthResponse` (same shape, `role` will be `"aggregator"`)

**Errors:**
| Status | Meaning |
|--------|---------|
| `400`  | Missing fields |
| `401`  | Invalid credentials |
| `403`  | Email belongs to a non-aggregator account |

**After login:** Immediately call `GET /api/v1/aggregator/me` to retrieve `api_key` and `invite_code`.

---

## 7. API Endpoints - Shared Auth

These endpoints work for **both** driver and aggregator accounts.

### `POST /api/v1/auth/refresh`

**Auth:** None (the refresh token is sent in the body, not as a header)

**Request Body:**
```json
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIs..."
}
```

| Field         | Type   | Required |
|---------------|--------|----------|
| refresh_token | string | yes      |

**Response** `200 OK` → `AuthResponse`
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

**Critical:** Both `access_token` and `refresh_token` are **new**. Store both. The old refresh token is no longer valid.

**Errors:**
| Status | Meaning |
|--------|---------|
| `400`  | Missing refresh_token field |
| `401`  | Refresh token is invalid, expired (>7 days), or was blacklisted by a logout |

---

### `POST /api/v1/auth/logout`

**Auth:** `Authorization: Bearer <access_token>`

**Request Body:** None (empty body)

**Response** `200 OK`
```json
{
  "message": "logged out successfully"
}
```

Server blacklists the current access token so it can't be reused. Client **must** also:
1. Delete stored `access_token`
2. Delete stored `refresh_token`
3. Delete stored `api_key` (if aggregator)
4. Close any open WebSocket connections

**Errors:**
| Status | Meaning |
|--------|---------|
| `401`  | Missing or invalid access token |

---

### `POST /api/v1/auth/forgot-password`

**Auth:** None

**Request Body:**
```json
{
  "email": "driver@example.com"
}
```

| Field | Type   | Required |
|-------|--------|----------|
| email | string | yes      |

**Response** `200 OK` (always, even if email doesn't exist — prevents email enumeration)
```json
{
  "message": "if an account exists with this email, a reset link has been sent"
}
```

In development mode, the reset token is logged to the server's stdout.

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

| Field        | Type   | Required | Notes                                  |
|--------------|--------|----------|----------------------------------------|
| token        | string | yes      | Token received from forgot-password flow |
| new_password | string | yes      | Same strength rules as registration    |

**Response** `200 OK`
```json
{
  "message": "password reset successfully"
}
```

**Errors:**
| Status | Meaning |
|--------|---------|
| `400`  | Token is invalid/expired, or new password doesn't meet strength rules |

---

## 8. API Endpoints - Driver Actions

> **All endpoints in this section require:** `Authorization: Bearer <access_token>` where the user's role is `driver`

### `GET /api/v1/driver/me`

Get the authenticated driver's full profile.

**Request Body:** None

**Response** `200 OK` → `DriverProfile`
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

**Note:** `id` is the **driver** ID (used in aggregator endpoints like `/aggregator/drivers/:id`). `user.id` is the **user** ID (internal, rarely needed client-side).

---

### `POST /api/v1/driver/join`

Join an aggregator agency using their invite code. The aggregator shares this 5-character code out-of-band (e.g., verbally, printed on a card). This creates a driver-aggregator mapping so the aggregator can see this driver's location.

**Request Body:**
```json
{
  "invite_code": "A1B2C"
}
```

| Field       | Type   | Required | Validation              |
|-------------|--------|----------|-------------------------|
| invite_code | string | yes      | 5-char alphanumeric     |

**Response** `200 OK`
```json
{
  "message": "successfully joined aggregator"
}
```

**Errors:**
| Status | Meaning |
|--------|---------|
| `400`  | Missing invite_code |
| `404`  | No aggregator found with this invite code |
| `409`  | Driver is already mapped to this aggregator |

A driver can join **multiple** aggregators using different invite codes.

---

### `PUT /api/v1/driver/location`

Push a single GPS location update. This does three things on the server:
1. Updates in-memory cache (60s TTL) for real-time queries
2. Persists to database (last known location)
3. Broadcasts to all aggregator WebSocket subscribers who own this driver

**Call this every 5-10 seconds** while the driver has an active trip.

**Request Body:**
```json
{
  "lat": 12.9716,
  "lng": 77.5946,
  "heading": 45.5,
  "speed": 25.3
}
```

| Field   | Type   | Required | Validation / Notes         |
|---------|--------|----------|----------------------------|
| lat     | double | yes      | Latitude, range -90 to 90  |
| lng     | double | yes      | Longitude, range -180 to 180 |
| heading | double | no       | Bearing in degrees 0-360, 0 = North |
| speed   | double | no       | Speed in km/h              |

**Response** `200 OK`
```json
{
  "message": "location updated"
}
```

**Errors:**
| Status | Meaning |
|--------|---------|
| `400`  | Missing lat/lng or values out of range |
| `404`  | Driver profile not found (shouldn't happen if token is valid) |

---

### `POST /api/v1/driver/locations/batch`

Sync multiple GPS readings collected while the device was offline or when the network was unavailable. The server processes all entries and uses the one with the **latest timestamp** as the current location.

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

| Field               | Type   | Required | Notes                    |
|---------------------|--------|----------|--------------------------|
| locations           | array  | yes      | Minimum 1 entry          |
| locations[].lat     | double | yes      | Latitude -90..90         |
| locations[].lng     | double | yes      | Longitude -180..180      |
| locations[].heading | double | no       | Degrees 0-360            |
| locations[].speed   | double | no       | km/h                     |
| locations[].timestamp | string | yes   | ISO 8601 (e.g. "2026-03-31T10:00:00Z") |

**Response** `200 OK`
```json
{
  "message": "batch location update processed",
  "count": 2
}
```

**Android implementation note:** Queue location updates in a local Room database or in-memory list when the network is unavailable. On reconnect, send them all via this endpoint, then clear the local queue.

---

### `POST /api/v1/driver/trip/start`

Start an active trip. This links the driver to a specific GTFS trip so their location appears in the aggregator's GTFS-RT feed.

**Preconditions:**
- Driver must NOT already have an active trip (call `/trip/end` first)
- The `trip_id` must be a valid GTFS trip ID that exists in the server

**Request Body:**
```json
{
  "trip_id": "TRIP-001",
  "vehicle_id": "BUS-042"
}
```

| Field      | Type   | Required | Notes                                  |
|------------|--------|----------|----------------------------------------|
| trip_id    | string | yes      | GTFS trip ID (must exist on server)    |
| vehicle_id | string | yes      | Vehicle label/identifier for GTFS-RT   |

**Response** `201 Created` → `ActiveTripResponse`
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
| Status | Meaning |
|--------|---------|
| `400`  | Missing trip_id or vehicle_id, OR driver already has an active trip |
| `404`  | Trip with this GTFS trip ID not found |

---

### `POST /api/v1/driver/trip/end`

End the driver's current active trip.

**Request Body:** None (empty body)

**Response** `200 OK`
```json
{
  "message": "trip ended successfully"
}
```

**Errors:**
| Status | Meaning |
|--------|---------|
| `400`  | Driver has no active trip to end |

---

## 9. API Endpoints - Aggregator Profile

> **Profile and API key management endpoints in this section require only:**
> - `Authorization: Bearer <access_token>` (user role must be `aggregator` or `admin`)
>
> **Aggregator management endpoints in Sections 10-12 require BOTH headers:**
> - `Authorization: Bearer <access_token>`
> - `X-API-Key: <api_key>`
>
> For endpoints that require both, the server validates that the JWT user matches the API key owner. If they don't match → `403`.

### `GET /api/v1/aggregator/me`

Returns the full aggregator profile, including the `invite_code` and `api_key`.

**Request Body:** None

**Response** `200 OK` → `AggregatorProfile`
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

**You must call this after every login** to get the `api_key` (needed for all subsequent aggregator API calls and WebSocket connections) and `invite_code` (to display/share with drivers).

### `GET /api/v1/aggregator/api-key`

Returns only the current aggregator API key payload.

**Request Body:** None

**Response** `200 OK`
```json
{
  "api_key": "a3f8b2c1d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1",
  "invite_code": "A1B2C",
  "updated_at": "2026-03-31T10:00:00Z"
}
```

### `PUT /api/v1/aggregator/api-key`

Rotates the current aggregator API key and returns the new value.

**Request Body:** None

**Response** `200 OK`
```json
{
  "message": "API key rotated successfully",
  "api_key": "b7e2c4a1d9e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a4",
  "invite_code": "A1B2C",
  "updated_at": "2026-03-31T10:05:00Z"
}
```

---

## 10. API Endpoints - Aggregator Driver Management

> **Auth:** `Bearer <access_token>` + `X-API-Key`
> These endpoints only return drivers that are **mapped to this aggregator** (via the invite code join flow).

### `GET /api/v1/aggregator/drivers`

List all drivers mapped to this aggregator.

**Request Body:** None

**Response** `200 OK` → `List<DriverProfile>`
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

Returns an **empty array** `[]` if no drivers have joined yet.

---

### `GET /api/v1/aggregator/drivers/:id`

Get a specific driver by their driver ID.

**URL Path Parameter:**
| Param | Type | Notes                                |
|-------|------|--------------------------------------|
| id    | int  | The `id` field from DriverProfile (NOT `user.id`) |

**Response** `200 OK` → single `DriverProfile` object (same shape as array item above)

**Errors:**
| Status | Meaning |
|--------|---------|
| `403`  | Driver exists but is NOT mapped to this aggregator |
| `404`  | No driver with this ID exists |

---

### `GET /api/v1/aggregator/drivers/:id/location`

Get a single driver's current location.

**URL Path Parameter:**
| Param | Type | Notes     |
|-------|------|-----------|
| id    | int  | Driver ID |

**Response** `200 OK` → `DriverLocation`
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

| Field      | Type   | Notes |
|------------|--------|-------|
| status     | string | `"online"` = driver sent a location update within the last 60 seconds (from cache). `"offline"` = no recent update, location is from last known DB record |
| lat / lng  | double | Current or last known position |
| heading    | double | Bearing in degrees (0 = North). May be 0 if not provided by driver |
| speed      | double | Speed in km/h. May be 0 if not provided or if offline |
| updated_at | string | ISO 8601 timestamp of when this location was recorded |

**Errors:**
| Status | Meaning |
|--------|---------|
| `403`  | Driver not mapped to this aggregator |
| `404`  | Driver not found or has never sent a location |

---

### `GET /api/v1/aggregator/drivers/locations`

Get current locations of **all** drivers mapped to this aggregator in a single call.

**Request Body:** None

**Response** `200 OK` → `List<DriverLocation>`
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

Returns an empty array `[]` if no drivers have location data.

**Use case:** Call this on initial screen load to populate the map, then switch to WebSocket for live updates.

---

## 11. API Endpoints - Aggregator Routes & Trips

> **Auth:** `Bearer <access_token>` + `X-API-Key`

### `POST /api/v1/aggregator/routes`

Create a new transit route (GTFS-compatible). Routes belong to the aggregator who creates them.

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

| Field       | Type   | Required | Validation / Notes                           |
|-------------|--------|----------|----------------------------------------------|
| route_id    | string | yes      | GTFS route_id, must be globally unique       |
| short_name  | string | yes      | Short display name (e.g. "R1", "42")         |
| long_name   | string | yes      | Full route name                              |
| description | string | no       | Optional description                         |
| route_type  | int    | no       | GTFS route type. Default: 3 (Bus). Values: 0=Tram, 1=Subway, 2=Rail, 3=Bus, 4=Ferry, 5=Cable, 6=Gondola, 7=Funicular |
| color       | string | no       | Hex color WITHOUT `#` prefix (e.g. "FF0000") |
| text_color  | string | no       | Hex text color WITHOUT `#` prefix            |

**Response** `201 Created` → `Route`
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
| Status | Meaning |
|--------|---------|
| `400`  | Missing required fields |
| `409`  | A route with this `route_id` already exists |

---

### `GET /api/v1/aggregator/routes`

List all routes belonging to this aggregator.

**Request Body:** None

**Response** `200 OK` → `List<Route>`
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

Returns empty array `[]` if no routes created yet.

---

### `POST /api/v1/aggregator/trips`

Create a trip on an existing route. Trips represent a specific journey along a route (e.g., "the 8:00 AM Airport Express heading downtown").

**Request Body:**
```json
{
  "route_id": "ROUTE-01",
  "trip_id": "TRIP-001",
  "headsign": "Downtown Terminal",
  "direction_id": 0
}
```

| Field        | Type   | Required | Validation / Notes                              |
|--------------|--------|----------|-------------------------------------------------|
| route_id     | string | yes      | GTFS route_id — must exist AND be owned by you  |
| trip_id      | string | yes      | GTFS trip_id — must be globally unique           |
| headsign     | string | yes      | Destination sign text shown to passengers        |
| direction_id | int    | no       | 0 = outbound (default), 1 = inbound             |

**Response** `201 Created` → `Trip`
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
| Status | Meaning |
|--------|---------|
| `400`  | Missing required fields |
| `404`  | Route not found or not owned by this aggregator |
| `409`  | A trip with this `trip_id` already exists |

---

### `GET /api/v1/aggregator/trips`

List all trips for routes owned by this aggregator.

**Request Body:** None

**Response** `200 OK` → `List<Trip>`
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

## 12. API Endpoints - Aggregator GTFS-RT Feeds

> **Auth:** `Bearer <access_token>` + `X-API-Key`
>
> These endpoints return GTFS-Realtime data. Only drivers with **active trips** appear in the feed.

### `GET /api/v1/aggregator/feed/vehicle-positions`

Get a GTFS-RT protobuf feed of all active vehicles for this aggregator.

**Response** `200 OK`
- **Content-Type:** `application/x-protobuf`
- **Body:** Binary protobuf `FeedMessage` per the [GTFS-RT specification](https://gtfs.org/realtime/)

**Android note:** You likely won't need this endpoint for your own app UI. Use the debug (JSON) version below or the WebSocket instead. The protobuf feed is meant for third-party transit apps that consume standard GTFS-RT feeds.

---

### `GET /api/v1/aggregator/feed/vehicle-positions/debug`

JSON version of the GTFS-RT feed, useful for debugging and for displaying in your app.

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

**Important:** In the GTFS-RT feed, `speed` is in **meters/second** (server converts from km/h). `bearing` maps to `heading`. `entity` is an empty array `[]` if no drivers have active trips.

---

### `GET /api/v1/aggregator/feed/vehicle-positions/:driverId`

Protobuf feed for a **single** vehicle/driver.

**URL Path Parameter:**
| Param    | Type | Notes     |
|----------|------|-----------|
| driverId | int  | Driver ID |

**Response** `200 OK` — Binary protobuf (single entity in FeedMessage)

**Errors:**
| Status | Meaning |
|--------|---------|
| `403`  | Driver not mapped to this aggregator |
| `404`  | Driver not found or has no active trip |

---

### `GET /api/v1/aggregator/feed/vehicle-positions/:driverId/debug`

JSON debug version for a single vehicle. Same format as the debug endpoint above but with only one entity.

---

## 13. WebSocket - Real-Time Location Streaming

This is the primary mechanism for the aggregator app to receive **live location updates** from all mapped drivers in real time.

### Connection

**URL:**
```
ws://10.0.2.2:8080/api/v1/aggregator/subscribe?token=<jwt_access_token>&api_key=<api_key>
```

For production:
```
wss://your-domain.com/api/v1/aggregator/subscribe?token=<jwt_access_token>&api_key=<api_key>
```

**Authentication is via query parameters** (not headers), because the WebSocket standard doesn't reliably support custom headers during the upgrade handshake.

| Query Param | Type   | Required | Description                    |
|-------------|--------|----------|--------------------------------|
| token       | string | yes      | JWT access token (same as used in `Authorization: Bearer`) |
| api_key     | string | yes      | Aggregator API key (same as used in `X-API-Key`) |

### Connection Handshake

```
1. Client opens WebSocket with URL including token + api_key query params
2. Server validates:
   a. JWT signature and expiry → extracts userID and role
   b. API key → looks up aggregatorID
   c. JWT user must match the API key's owner → rejects if mismatch
3. On success: HTTP 101 Switching Protocols → WebSocket connection established
4. On failure: HTTP 401 or 403 with JSON error body (connection is NOT upgraded)
```

**Connection Failure Responses (HTTP, before upgrade):**
| Status | Body | Meaning |
|--------|------|---------|
| `401`  | `{"error": "invalid token"}` | JWT is expired, malformed, or blacklisted |
| `401`  | `{"error": "invalid API key"}` | API key not found in database |
| `403`  | `{"error": "token user does not match API key owner"}` | JWT belongs to a different user than the API key |

### Message Format (Server → Client)

Once connected, the server pushes JSON messages whenever any driver mapped to this aggregator sends a location update. **The client does NOT send messages** — this is a server-push-only channel.

Every message has this envelope:

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

| Field              | Type    | Description |
|--------------------|---------|-------------|
| event              | string  | Event type. Currently always `"location_update"`. Parse this field to future-proof your code for new event types. |
| data.driver_id     | int     | The driver's ID (matches `id` from `/aggregator/drivers`) |
| data.vehicle_id    | string  | Vehicle label from the driver's active trip. Empty string `""` if no active trip. |
| data.lat           | double  | Latitude |
| data.lng           | double  | Longitude |
| data.heading       | double  | Bearing in degrees (0-360, 0 = North) |
| data.speed         | double  | Speed in km/h |
| data.timestamp     | string  | ISO 8601 timestamp of when the driver recorded this location |
| data.is_online     | boolean | `true` = this is a fresh live update. `false` = stale/cached data. |

### Keep-Alive / Ping-Pong

| Parameter | Value |
|-----------|-------|
| Server sends ping | Every **54 seconds** |
| Client must respond with pong | Within **60 seconds** of the ping |
| If pong not received | Server closes the connection |
| Max inbound message size | 512 bytes |
| Server send buffer | 256 messages per client |

**OkHttp handles ping/pong automatically** — you do NOT need to manually send pong frames. OkHttp's WebSocket implementation responds to ping frames at the protocol level.

### Kotlin/OkHttp Implementation Guide

```kotlin
// 1. Build the WebSocket URL
val wsUrl = "ws://10.0.2.2:8080/api/v1/aggregator/subscribe" +
    "?token=$accessToken" +
    "&api_key=$apiKey"

// 2. Create the request
val request = Request.Builder()
    .url(wsUrl)
    .build()

// 3. Connect
val webSocket = okHttpClient.newWebSocket(request, object : WebSocketListener() {
    override fun onOpen(webSocket: WebSocket, response: Response) {
        // Connected successfully
    }

    override fun onMessage(webSocket: WebSocket, text: String) {
        // Parse the JSON message
        val message = gson.fromJson(text, WebSocketMessage::class.java)
        when (message.event) {
            "location_update" -> {
                // Update driver marker on map
                val data = message.data
                // data.driver_id, data.lat, data.lng, etc.
            }
        }
    }

    override fun onClosing(webSocket: WebSocket, code: Int, reason: String) {
        webSocket.close(1000, null)
    }

    override fun onFailure(webSocket: WebSocket, t: Throwable, response: Response?) {
        // Connection failed — implement reconnection logic
    }
})
```

### Reconnection Strategy

The WebSocket will disconnect when:
- The access token expires (15 min)
- Network connectivity is lost
- The server restarts
- The client doesn't respond to ping in time

**Recommended reconnection logic:**

```
1. On disconnect/failure:
   a. Wait 1 second
   b. Check if access_token is still valid (check expiry locally)
   c. If expired → call POST /auth/refresh to get new tokens
   d. Reconnect with new token
   e. If reconnect fails, use exponential backoff: 1s, 2s, 4s, 8s, 16s, max 30s
   f. If refresh token is also expired → redirect to login screen
```

### When to Use WebSocket vs REST

| Use case | Approach |
|----------|----------|
| Initial map load (get all driver positions) | `GET /aggregator/drivers/locations` (REST) |
| Live tracking after initial load | WebSocket at `/aggregator/subscribe` |
| Check single driver's status | `GET /aggregator/drivers/:id/location` (REST) |
| Background monitoring (app minimized) | WebSocket with a foreground service |

---

## 14. Error Handling

### Error Response Format

**All** error responses from the server follow this JSON format:

```json
{
  "error": "human readable error message"
}
```

For validation errors, there may be an additional `details` field:

```json
{
  "error": "validation failed",
  "details": {
    "email": "required",
    "password": "must be at least 8 characters"
  }
}
```

### Kotlin Error Model

```kotlin
data class ErrorResponse(
    val error: String,
    val details: Map<String, String>? = null
)

// Parse error from Retrofit Response:
fun <T> Response<T>.parseError(): ErrorResponse {
    val errorBody = errorBody()?.string() ?: return ErrorResponse("Unknown error")
    return gson.fromJson(errorBody, ErrorResponse::class.java)
}
```

### HTTP Status Code Reference

| Code | Meaning | Client Action |
|------|---------|---------------|
| `200` | Success | Process response body |
| `201` | Created (new resource) | Process response body |
| `400` | Bad request / validation error | Show error message to user, check `details` for field-specific messages |
| `401` | Unauthorized — token missing, expired, invalid, or blacklisted | Attempt token refresh. If refresh fails → redirect to login |
| `403` | Forbidden — wrong role, or resource not owned by user | Show "access denied" message |
| `404` | Resource not found | Show appropriate "not found" message |
| `409` | Conflict — duplicate resource (email, route_id, etc.) | Show "already exists" message |
| `500` | Internal server error | Show generic error, retry later |

### Token Expiry Handling

```
On ANY 401 response:
  1. Don't show error to user yet
  2. Try POST /auth/refresh with stored refresh_token
  3. If refresh succeeds:
     - Store new access_token + refresh_token
     - Retry the original request with new access_token
  4. If refresh fails (401):
     - Clear all stored tokens
     - Redirect to login screen
     - Show "Session expired, please log in again"
```

---

## 15. Complete Endpoint Reference Table

| # | Method | Path | Auth | Request Body | Response |
|---|--------|------|------|--------------|----------|
| 1 | GET | `/health` | None | — | `{ "status": "ok" }` |
| 2 | POST | `/api/v1/driver/register` | None | DriverRegisterRequest | AuthResponse (201) |
| 3 | POST | `/api/v1/driver/login` | None | LoginRequest | AuthResponse |
| 4 | POST | `/api/v1/aggregator/register` | None | AggregatorRegisterRequest | AuthResponse (201) |
| 5 | POST | `/api/v1/aggregator/login` | None | LoginRequest | AuthResponse |
| 6 | POST | `/api/v1/auth/refresh` | None | RefreshTokenRequest | AuthResponse |
| 7 | POST | `/api/v1/auth/logout` | Bearer | — | MessageResponse |
| 8 | POST | `/api/v1/auth/forgot-password` | None | ForgotPasswordRequest | MessageResponse |
| 9 | POST | `/api/v1/auth/reset-password` | None | ResetPasswordRequest | MessageResponse |
| 10 | GET | `/api/v1/driver/me` | Bearer (driver) | — | DriverProfile |
| 11 | POST | `/api/v1/driver/join` | Bearer (driver) | JoinAggregatorRequest | MessageResponse |
| 12 | PUT | `/api/v1/driver/location` | Bearer (driver) | LocationUpdateRequest | MessageResponse |
| 13 | POST | `/api/v1/driver/locations/batch` | Bearer (driver) | BatchLocationRequest | BatchLocationResponse |
| 14 | POST | `/api/v1/driver/trip/start` | Bearer (driver) | StartTripRequest | ActiveTripResponse (201) |
| 15 | POST | `/api/v1/driver/trip/end` | Bearer (driver) | — | MessageResponse |
| 16 | GET | `/api/v1/aggregator/me` | Bearer + API Key | — | AggregatorProfile |
| 17 | GET | `/api/v1/aggregator/drivers` | Bearer + API Key | — | List\<DriverProfile\> |
| 18 | GET | `/api/v1/aggregator/drivers/:id` | Bearer + API Key | — | DriverProfile |
| 19 | GET | `/api/v1/aggregator/drivers/:id/location` | Bearer + API Key | — | DriverLocation |
| 20 | GET | `/api/v1/aggregator/drivers/locations` | Bearer + API Key | — | List\<DriverLocation\> |
| 21 | POST | `/api/v1/aggregator/routes` | Bearer + API Key | CreateRouteRequest | Route (201) |
| 22 | GET | `/api/v1/aggregator/routes` | Bearer + API Key | — | List\<Route\> |
| 23 | POST | `/api/v1/aggregator/trips` | Bearer + API Key | CreateTripRequest | Trip (201) |
| 24 | GET | `/api/v1/aggregator/trips` | Bearer + API Key | — | List\<Trip\> |
| 25 | GET | `/api/v1/aggregator/feed/vehicle-positions` | Bearer + API Key | — | Protobuf binary |
| 26 | GET | `/api/v1/aggregator/feed/vehicle-positions/debug` | Bearer + API Key | — | JSON (GTFS-RT) |
| 27 | GET | `/api/v1/aggregator/feed/vehicle-positions/:driverId` | Bearer + API Key | — | Protobuf binary |
| 28 | GET | `/api/v1/aggregator/feed/vehicle-positions/:driverId/debug` | Bearer + API Key | — | JSON (GTFS-RT) |
| 29 | GET (WS) | `/api/v1/aggregator/subscribe?token=X&api_key=Y` | Query params | — | WebSocket stream |

---

## 16. App Flow - Driver

This is the complete sequence of API calls for the **driver Android app**:

### First-Time Setup
```
1. POST /api/v1/driver/register
   → Store access_token + refresh_token in EncryptedSharedPreferences
   → Store user object for profile display

2. POST /api/v1/driver/join  { "invite_code": "A1B2C" }
   → Driver is now linked to an aggregator
   → The invite code comes from the aggregator (shared out-of-band)
```

### Daily Usage (Returning User)
```
1. POST /api/v1/driver/login
   → Store access_token + refresh_token
   → Store user object

2. GET /api/v1/driver/me
   → Display profile, check is_available status

3. POST /api/v1/driver/trip/start  { "trip_id": "TRIP-001", "vehicle_id": "BUS-042" }
   → Driver selects their trip and vehicle before starting
   → Server returns the active trip details

4. [LOOP every 5-10 seconds while trip is active]:
   PUT /api/v1/driver/location  { "lat": ..., "lng": ..., "heading": ..., "speed": ... }
   → Use Android FusedLocationProviderClient for GPS
   → Queue locations locally if network is unavailable
   → On reconnect: POST /api/v1/driver/locations/batch with queued entries

5. POST /api/v1/driver/trip/end
   → Stop sending location updates

6. POST /api/v1/auth/logout
   → Clear all stored tokens
```

### Token Refresh (Background)
```
Access token expires every 15 minutes.
Implement an OkHttp Authenticator that:
  1. Catches 401 responses
  2. Calls POST /api/v1/auth/refresh { "refresh_token": "..." }
  3. Stores new tokens
  4. Retries the failed request
  5. If refresh fails → clear tokens, show login screen
```

### Offline Location Queue
```
When network is unavailable:
  1. Continue collecting GPS readings with timestamps
  2. Store in local DB (Room) or in-memory list
  3. On network restore:
     POST /api/v1/driver/locations/batch  { "locations": [...] }
  4. Clear local queue on success
```

---

## 17. App Flow - Aggregator

This is the complete sequence for the **aggregator Android app**:

### First-Time Setup
```
1. POST /api/v1/aggregator/register
   → Store access_token + refresh_token

2. GET /api/v1/aggregator/me
   → Store api_key (needed for ALL subsequent API calls + WebSocket)
   → Store invite_code (display in app so user can share with drivers)

3. POST /api/v1/aggregator/routes  { "route_id": "ROUTE-01", ... }
   → Create transit routes for your agency

4. POST /api/v1/aggregator/trips  { "route_id": "ROUTE-01", "trip_id": "TRIP-001", ... }
   → Create trips on those routes (drivers will select these when starting trips)
```

### Daily Usage (Returning User)
```
1. POST /api/v1/aggregator/login
   → Store access_token + refresh_token

2. GET /api/v1/aggregator/me
   → Retrieve and store api_key + invite_code

3. GET /api/v1/aggregator/drivers
   → Show list of all drivers who have joined your agency

4. GET /api/v1/aggregator/drivers/locations
   → Initial map load — place all driver markers on map

5. Connect WebSocket:
   ws://host/api/v1/aggregator/subscribe?token=ACCESS_TOKEN&api_key=API_KEY
   → Receive real-time location_update events
   → Update driver markers on map as events arrive

6. [Optional] GET /api/v1/aggregator/feed/vehicle-positions/debug
   → View GTFS-RT feed data for drivers with active trips

7. POST /api/v1/auth/logout
   → Close WebSocket, clear all tokens
```

### Headers Cheat Sheet

```
# No auth (register, login, refresh, forgot/reset password):
Content-Type: application/json

# Driver endpoints (after login):
Content-Type: application/json
Authorization: Bearer <access_token>

# Aggregator endpoints (after login):
Content-Type: application/json
Authorization: Bearer <access_token>
X-API-Key: <api_key>

# WebSocket (auth via query params, not headers):
ws://host/api/v1/aggregator/subscribe?token=<access_token>&api_key=<api_key>
```
