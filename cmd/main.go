package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

var db *sql.DB

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	return fmt.Sprintf("%d:%02d:%02d", h, m, s)
}

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		fmt.Printf("[%s] Error loading .env file: %v\n", time.Now().Format(time.RFC3339), err)
		return
	}

	// Database connection string
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		fmt.Printf("[%s] DATABASE_URL environment variable is not set\n", time.Now().Format(time.RFC3339))
		return
	}

	// Add sslmode if not already present in the connection string
	if !strings.Contains(dbURL, "sslmode=") {
		if strings.Contains(dbURL, "?") {
			dbURL += "&sslmode=disable"
		} else {
			dbURL += "?sslmode=disable"
		}
	}

	// Initialize database connection
	var err error
	db, err = sql.Open("postgres", dbURL)
	if err != nil {
		fmt.Printf("[%s] Error connecting to database: %v\n", time.Now().Format(time.RFC3339), err)
		return
	}
	defer db.Close()

	// Test database connection
	err = db.Ping()
	if err != nil {
		fmt.Printf("[%s] Error pinging database: %v\n", time.Now().Format(time.RFC3339), err)
		return
	}
	fmt.Printf("[%s] Successfully connected to database\n", time.Now().Format(time.RFC3339))

	// Create uuid-ossp extension if it doesn't exist
	_, err = db.Exec(`CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`)
	if err != nil {
		fmt.Printf("[%s] Error creating uuid-ossp extension: %v\n", time.Now().Format(time.RFC3339), err)
		return
	}

	// Create subscriptions table if it doesn't exist
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS subscriptions (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			subscription JSONB NOT NULL
		)
	`)
	if err != nil {
		fmt.Printf("[%s] Error creating table: %v\n", time.Now().Format(time.RFC3339), err)
		return
	}

	// Parse the initial event time
	initialEvent, err := time.Parse(time.RFC3339, os.Getenv("FIRST_EVENT_TIME"))
	if err != nil {
		fmt.Printf("[%s] Error parsing initial event time: %v\n", time.Now().Format(time.RFC3339), err)
		return
	}

	// Calculate time until next event
	now := time.Now().UTC()
	eventInterval := 3*time.Hour + 30*time.Minute // 3.5 hours
	timeSinceInitial := now.Sub(initialEvent)
	intervalsPassed := timeSinceInitial.Hours() / 3.5
	nextEventTime := initialEvent.Add(time.Duration(int(intervalsPassed)+1) * eventInterval)

	// If we're past the next event, calculate the next one
	if now.After(nextEventTime) {
		nextEventTime = nextEventTime.Add(eventInterval)
	}

	// Calculate delay until 5 minutes before next event
	notificationTime := nextEventTime.Add(-5 * time.Minute)
	delay := notificationTime.Sub(now)

	timezone := time.FixedZone("UTC+2", 2*60*60)
	fmt.Printf("[%s] Next event at: %s\n",
		time.Now().In(timezone).Format(time.RFC3339),
		nextEventTime.In(timezone).Format(time.RFC3339))
	fmt.Printf("[%s] Will send notification at: %s (in %s)\n",
		time.Now().In(timezone).Format(time.RFC3339),
		notificationTime.In(timezone).Format(time.RFC3339),
		formatDuration(delay))

	time.Sleep(delay)

	// Create a ticker that ticks every 3.5 hours
	ticker := time.NewTicker(eventInterval)
	defer ticker.Stop()

	// Send initial notifications
	sendNotifications()

	// Loop forever, sending notifications every 3.5 hours
	for range ticker.C {
		nextEventTime = nextEventTime.Add(eventInterval)
		fmt.Printf("[%s] Next event at: %s\n",
			time.Now().In(timezone).Format(time.RFC3339),
			nextEventTime.In(timezone).Format(time.RFC3339))
		sendNotifications()
	}
}

func sendNotifications() {
	subscriptions, err := getAllSubscriptions()
	if err != nil {
		fmt.Printf("[%s] Error getting subscriptions: %v\n", time.Now().Format(time.RFC3339), err)
		return
	}

	vapidPublicKey := os.Getenv("VAPID_PUBLIC_KEY")
	vapidPrivateKey := os.Getenv("VAPID_PRIVATE_KEY")
	subscriber := os.Getenv("VAPID_SUBSCRIBER")

	// Create notification payload
	payload := struct {
		Title string `json:"title"`
		Body  string `json:"body"`
	}{
		Title: "World Boss Alert!",
		Body:  "A new World Boss event is starting in 5 minutes!",
	}

	// Convert payload to JSON
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("[%s] Error creating notification payload: %v\n", time.Now().In(time.FixedZone("UTC+2", 2*60*60)).Format(time.RFC3339), err)
		return
	}

	for _, subscription := range subscriptions {
		s := &webpush.Subscription{
			Endpoint: subscription.Endpoint,
			Keys: webpush.Keys{
				Auth:   subscription.Keys.Auth,
				P256dh: subscription.Keys.P256dh,
			},
		}

		resp, err := webpush.SendNotification(payloadJSON, s, &webpush.Options{
			Subscriber:      subscriber,
			Urgency:         "high",
			VAPIDPublicKey:  vapidPublicKey,
			VAPIDPrivateKey: vapidPrivateKey,
			TTL:             30,
		})
		if err != nil {
			fmt.Printf("[%s] Error sending notification to %s: %v\n", time.Now().Format(time.RFC3339), subscription.Endpoint, err)
			continue
		}
		resp.Body.Close()
		fmt.Printf("[%s] Successfully sent notification to %s\n", time.Now().Format(time.RFC3339), subscription.Endpoint)
	}
}

func getAllSubscriptions() ([]struct {
	Endpoint       string `json:"endpoint"`
	ExpirationTime *int64 `json:"expirationTime"`
	Keys           struct {
		P256dh string `json:"p256dh"`
		Auth   string `json:"auth"`
	} `json:"keys"`
}, error) {
	rows, err := db.Query("SELECT subscription FROM subscriptions")
	if err != nil {
		return nil, fmt.Errorf("error querying subscriptions: %v", err)
	}
	defer rows.Close()

	var subscriptions []struct {
		Endpoint       string `json:"endpoint"`
		ExpirationTime *int64 `json:"expirationTime"`
		Keys           struct {
			P256dh string `json:"p256dh"`
			Auth   string `json:"auth"`
		} `json:"keys"`
	}

	for rows.Next() {
		var rawSubscription string
		if err := rows.Scan(&rawSubscription); err != nil {
			return nil, fmt.Errorf("error scanning subscription: %v", err)
		}

		// First, unescape the JSON string
		var unescapedJSON string
		if err := json.Unmarshal([]byte(rawSubscription), &unescapedJSON); err != nil {
			return nil, fmt.Errorf("failed to unescape subscription JSON: %v\nRaw JSON: %s", err, rawSubscription)
		}

		var subscription struct {
			Endpoint       string `json:"endpoint"`
			ExpirationTime *int64 `json:"expirationTime"`
			Keys           struct {
				P256dh string `json:"p256dh"`
				Auth   string `json:"auth"`
			} `json:"keys"`
		}

		// Now parse the unescaped JSON
		if err := json.Unmarshal([]byte(unescapedJSON), &subscription); err != nil {
			return nil, fmt.Errorf("failed to parse subscription JSON: %v\nUnescaped JSON: %s", err, unescapedJSON)
		}

		// Validate required fields
		if subscription.Endpoint == "" {
			return nil, fmt.Errorf("subscription missing required endpoint field: %s", unescapedJSON)
		}
		if subscription.Keys.P256dh == "" || subscription.Keys.Auth == "" {
			return nil, fmt.Errorf("subscription missing required keys: %s", unescapedJSON)
		}

		subscriptions = append(subscriptions, subscription)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating subscriptions: %v", err)
	}

	return subscriptions, nil
}
