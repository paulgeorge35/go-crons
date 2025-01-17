# Diablo Timer Cron

> Mainly used as a learning project for me to learn Go and Web Push Notifications.
> This service only sends the notifications using the existing subscribers.
> It does not handle the subscription process.

A Go-based notification service that sends web push notifications for World Boss events, occurring every 3.5 hours.

## Features

- 🕒 Automated timing system for World Boss events
- 🔔 Web push notifications 5 minutes before each event
- 📊 PostgreSQL database for subscription management
- 🔐 Secure VAPID-based push notifications

## Prerequisites

- Go 1.23.4 or higher
- PostgreSQL database
- VAPID keys for web push notifications

## Installation

1. Clone the repository:
```bash
git clone https://github.com/paulgeorge35/go-crons
cd go-crons
```

2. Install dependencies:
```bash
go mod download
```

3. Create a `.env` file based on `.env.example`:
```bash
cp .env.example .env
```

4. Configure your environment variables in `.env`:
```
DATABASE_URL=your_postgresql_url
FIRST_EVENT_TIME=initial_event_timestamp
VAPID_PUBLIC_KEY=your_vapid_public_key
VAPID_PRIVATE_KEY=your_vapid_private_key
VAPID_SUBSCRIBER=your_vapid_subscriber
```

## Running the Service

```bash
go run cmd/main.go
```

The service will:
- Connect to the PostgreSQL database
- Calculate the next event time
- Send notifications to all subscribers 5 minutes before each event
- Continue running on a 3.5-hour cycle

## Database Schema

The service uses a single table for storing subscriptions:

```sql
CREATE TABLE subscriptions (
    id TEXT PRIMARY KEY,
    subscription JSONB NOT NULL
);
```

## Contributing

Feel free to open issues and pull requests for any improvements.

## License

This project is licensed under the MIT License - see the LICENSE.md file for details.

## Contact

Paul George - contact@paulgeorge.dev

Project Link: https://github.com/paulgeorge35/go-crons
