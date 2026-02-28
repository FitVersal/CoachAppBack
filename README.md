# CoachAppBack

Go backend for CoachApp with JWT auth, onboarding/profile APIs for users and coaches, and avatar upload via Supabase Storage.

## Environment

Required:

```env
PORT=8080
DB_URL=postgres://postgres:postgres@localhost:5432/coachapp?sslmode=disable
JWT_SECRET=change-me
```

Optional for avatar upload:

```env
SUPABASE_URL=https://your-project.supabase.co
SUPABASE_BUCKET=avatars
SUPABASE_SERVICE_KEY=your-service-role-key
```

If Supabase storage variables are not set, avatar upload endpoints return `503 Storage service is not configured`.

## Run

Start the API:

```bash
go run ./cmd/server
```

Run migrations:

```bash
go run ./cmd/migrate
```

Roll back migrations:

```bash
go run ./cmd/migrate down
```

## API Docs

OpenAPI spec: [docs/openapi.yaml](docs/openapi.yaml)

Main profile/onboarding endpoints:

- `POST /api/v1/users/onboarding`
- `GET /api/v1/users/profile`
- `PUT /api/v1/users/profile`
- `POST /api/v1/users/profile/avatar`
- `POST /api/v1/coaches/onboarding`
- `GET /api/v1/coaches/profile`
- `PUT /api/v1/coaches/profile`
- `POST /api/v1/coaches/profile/avatar`
- `GET /api/v1/coaches`
- `GET /api/v1/coaches/recommended`
- `GET /api/v1/coaches/:id`
- `GET /api/v1/conversations`
- `POST /api/v1/conversations`
- `GET /api/v1/conversations/:id/messages`
- `WS /api/v1/ws`

## Example Requests

Register a user:

```bash
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "password123",
    "role": "user"
  }'
```

User onboarding:

```bash
curl -X POST http://localhost:8080/api/v1/users/onboarding \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "full_name": "Sam User",
    "age": 29,
    "gender": "male",
    "height_cm": 180,
    "weight_kg": 78,
    "fitness_level": "beginner",
    "goals": ["weight_loss", "mobility"],
    "max_hourly_rate": 60,
    "medical_conditions": "asthma"
  }'
```

Update a user profile:

```bash
curl -X PUT http://localhost:8080/api/v1/users/profile \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "weight_kg": 75,
    "goals": ["weight_loss", "strength"],
    "max_hourly_rate": 70,
    "medical_conditions": "asthma"
  }'
```

Discover coaches:

```bash
curl "http://localhost:8080/api/v1/coaches?specialization=weight_loss&min_rating=4&max_price=80&page=1&limit=10" \
  -H "Authorization: Bearer <TOKEN>"
```

Get recommended coaches:

```bash
curl "http://localhost:8080/api/v1/coaches/recommended?limit=5" \
  -H "Authorization: Bearer <TOKEN>"
```

Get coach detail:

```bash
curl http://localhost:8080/api/v1/coaches/42 \
  -H "Authorization: Bearer <TOKEN>"
```

Upload a user avatar:

```bash
curl -X POST http://localhost:8080/api/v1/users/profile/avatar \
  -H "Authorization: Bearer <TOKEN>" \
  -F "avatar=@/path/to/avatar.png"
```

Coach onboarding:

```bash
curl -X POST http://localhost:8080/api/v1/coaches/onboarding \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "full_name": "Taylor Coach",
    "bio": "Strength and conditioning coach",
    "specializations": ["strength", "fat_loss"],
    "certifications": ["NASM", "ACE"],
    "experience_years": 6,
    "hourly_rate": 75
  }'
```

Update a coach profile:

```bash
curl -X PUT http://localhost:8080/api/v1/coaches/profile \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "bio": "Strength coach for busy professionals",
    "certifications": ["NASM", "Precision Nutrition"],
    "hourly_rate": 85
  }'
```

Upload a coach avatar:

```bash
curl -X POST http://localhost:8080/api/v1/coaches/profile/avatar \
  -H "Authorization: Bearer <TOKEN>" \
  -F "avatar=@/path/to/avatar.jpg"
```

Create or get a conversation with a coach:

```bash
curl -X POST http://localhost:8080/api/v1/conversations \
  -H "Authorization: Bearer <USER_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "coach_id": 42
  }'
```

List conversations for the current user:

```bash
curl http://localhost:8080/api/v1/conversations \
  -H "Authorization: Bearer <TOKEN>"
```

Get paginated messages and mark incoming messages as read:

```bash
curl "http://localhost:8080/api/v1/conversations/7/messages?page=1&limit=20" \
  -H "Authorization: Bearer <TOKEN>"
```

Connect to chat over WebSocket with `wscat`:

```bash
wscat -c "ws://localhost:8080/api/v1/ws?token=<TOKEN>"
```

Send a chat message over the websocket connection:

```json
{"type":"message","conversation_id":"7","content":"Hi coach, can we move tomorrow's session?"}
```

Expected websocket event shape:

```json
{
  "type": "message",
  "conversation_id": "7",
  "sender_id": "12",
  "recipient_id": "42",
  "content": "Hi coach, can we move tomorrow's session?",
  "timestamp": "2026-03-01T09:00:00Z"
}
```

## Notes

- Avatar uploads accept `.jpg`, `.jpeg`, `.png`, and `.webp` files up to 5 MB.
- Chat websocket auth accepts either `?token=<JWT>` or `Authorization: Bearer <JWT>` during upgrade.
- `000002_rename_profile_columns` migrates existing databases from `injuries` to `medical_conditions` and from `credentials` to `certifications[]`.
- The down migration converts coach certifications back to a comma-separated text field because the previous schema stored only a single text value.
- `000003_add_discovery_support` adds persisted user budget preference, coach reviews, and coach availability slots used by discovery endpoints.
- `000004_sync_coach_rating_from_reviews` backfills `coach_profiles.rating` from `coach_reviews` and keeps it synchronized with a database trigger.
- `000006_add_chat_indexes` hardens chat foreign keys and adds indexes for conversation lookup and unread-message scans.
