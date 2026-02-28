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
    "medical_conditions": "asthma"
  }'
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

## Notes

- Avatar uploads accept `.jpg`, `.jpeg`, `.png`, and `.webp` files up to 5 MB.
- `000002_rename_profile_columns` migrates existing databases from `injuries` to `medical_conditions` and from `credentials` to `certifications[]`.
- The down migration converts coach certifications back to a comma-separated text field because the previous schema stored only a single text value.
