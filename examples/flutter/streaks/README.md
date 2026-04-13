# Streaks — Habit Tracker with Social Accountability

A Flutter reference implementation for the [Mobpilot](https://github.com/mobpilot/mobpilot) mobile backend platform.

Streaks demonstrates how to build a full-featured mobile app using Mobpilot's core services:
- **Authentication** via Ory Kratos (registration/login) + Ory Hydra (OAuth2 tokens)
- **User Profiles** via the Identity API (`GET/PATCH /v1/identity/me`)
- **Data Storage** via Artifact API (habits, logs, streaks as JSONB key-value pairs)
- **Push Notifications** via Device Token API (FCM/APNs token registration)

## Features

- Create and manage daily habits with icons, colors, and frequency settings
- Track completions with a calendar view
- Automatic streak calculation (current + best streak)
- Daily progress ring showing completion rate
- User profile management (display name, bio, avatar, location)
- Offline-first with optimistic updates
- Full dark mode support (Material Design 3)

## Architecture

```
lib/
  config/          App configuration + CLI args
  models/          Habit, HabitLog, StreakData, UserProfile
  services/        MobpilotClient (REST), AuthService (Kratos/Hydra)
  providers/       ChangeNotifier state management (Provider)
  screens/         Login, Register, Home, HabitDetail, Profile, Settings
  widgets/         HabitCard, ProgressRing, StreakCounter
```

Data model (stored as Mobpilot artifacts):
| Artifact Key | Contents |
|---|---|
| `habits` | List of habit definitions |
| `log/YYYY-MM` | Daily completions, partitioned by month |
| `streaks` | Current/best streak per habit |

## Prerequisites

- Flutter 3.x
- Running Mobpilot instance (see `deploy/docker-compose.yml`)

## Quick Start

```bash
# Start Mobpilot infrastructure + identity service
cd ../../deploy
./scripts/bootstrap-dev.sh
docker compose --profile infra --profile services up -d --build

# Run the Streaks app
cd ../examples/flutter/streaks
flutter pub get
flutter run -d linux
```

## CLI Arguments

The Linux desktop app accepts these arguments for development/testing:

```
--base-url <url>      Mobpilot Identity API base URL (default: http://localhost/v1/identity)
--kratos-url <url>    Ory Kratos public URL (default: http://localhost:4433)
--hydra-url <url>     Ory Hydra public URL (default: http://localhost:4444)
--dev-token <token>   Skip auth, use this bearer token for all API calls
--test-mode           Enable debug info in settings screen
--client-id <id>      OAuth2 client ID (default: mobpilot-dev-app)
--redirect-uri <uri>  OAuth2 redirect URI
```

Example with dev token:
```bash
flutter run -d linux -- --base-url http://localhost:8090/v1/identity --dev-token <token> --test-mode
```

## Testing

Unit tests (models, config):
```bash
flutter test
```

Integration tests (requires running Mobpilot):
```bash
flutter test integration_test/app_test.dart \
  --dart-define=BASE_URL=http://localhost:8090/v1/identity \
  --dart-define=DEV_TOKEN=<token>
```

## Known Issues

- mobpilot/mobpilot#35: Hydra token hook needed for `mobpilot_app_id` claim injection
- mobpilot/mobpilot#34: Docker Compose bootstrap requires several config fixes
