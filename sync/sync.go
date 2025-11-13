package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/davidpimentel/calendar-sync/auth"
	"github.com/davidpimentel/calendar-sync/config"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

func RunSync() {
	fmt.Println("Starting calendar sync...")

	// Get source client
	sourceClient, err := auth.GetClient(config.SourceTokenFile, config.SourceScope)
	if err != nil {
		log.Fatalf("Unable to get source client: %v", err)
	}

	// Get destination client
	destClient, err := auth.GetClient(config.DestTokenFile, config.DestinationScope)
	if err != nil {
		log.Fatalf("Unable to get destination client: %v", err)
	}

	// Create calendar service for source and destination
	sourceSrv, err := calendar.NewService(context.Background(), option.WithHTTPClient(sourceClient))
	if err != nil {
		log.Fatalf("Unable to retrieve source Calendar client: %v", err)
	}

	destSrv, err := calendar.NewService(context.Background(), option.WithHTTPClient(destClient))
	if err != nil {
		log.Fatalf("Unable to retrieve destination Calendar client: %v", err)
	}

	// Get events from source calendar
	now := time.Now().Format(time.RFC3339)
	// oneMonthFromNow := time.Now().AddDate(0, 1, 0).Format(time.RFC3339)
	oneDayFromNow := time.Now().AddDate(0, 0, 1).Format(time.RFC3339)

	fmt.Println("Fetching events from source calendar...")
	fmt.Printf("Time range: %s to %s\n", now, oneDayFromNow)

	// Get primary calendar for source
	sourceCalendar, err := sourceSrv.Calendars.Get("primary").Do()
	if err != nil {
		log.Fatalf("Unable to retrieve source calendar: %v", err)
	}

	// List events from source calendar
	events, err := sourceSrv.Events.List("primary").
		TimeMin(now).
		TimeMax(oneDayFromNow).
		SingleEvents(true).
		OrderBy("startTime").
		Do()
	if err != nil {
		log.Fatalf("Unable to retrieve events from source calendar: %v", err)
	}

	if len(events.Items) == 0 {
		fmt.Println("No upcoming events found in source calendar.")
		return
	}

	fmt.Printf("Found %d events in source calendar\n", len(events.Items))

	// Get any and all events from the destination calendar that we've created
	destinationEvents, err := destSrv.Events.List("primary").
		TimeMin(now).
		TimeMax(oneDayFromNow).
		PrivateExtendedProperty(fmt.Sprintf("%s=%s", config.AppName, "true")).
		SingleEvents(true).
		OrderBy("startTime").
		Do()

	if err != nil {
		log.Fatalf("Unable to retrieve existing events from destination calendar: %v", err)
	}

	// Process each event
	for _, event := range events.Items {
		fmt.Printf("Event: %s (%s)\n", event.Summary, event.Id)

		if eventAlreadyExists(destinationEvents.Items, event.Id) {
			fmt.Printf("Event already synced: %s\n", event.Id)
		} else {
			// Create a new event in the destination calendar
			newEvent := &calendar.Event{
				ColorId:     "4",
				Summary:     "Busy",
				Description: "Created with calendar-sync",
				Start:       event.Start,
				End:         event.End,
				// Add extended properties to track the source event
				ExtendedProperties: &calendar.EventExtendedProperties{
					Private: map[string]string{
						config.AppName:                   "true",
						config.SourceEventIdPropertyKey:  event.Id,
						config.SourceCalendarPropertyKey: sourceCalendar.Id,
					},
				},
			}

			// Insert the event
			fmt.Printf("Creating new event: %s\n", newEvent.Summary)
			b, err := json.MarshalIndent(newEvent, "", "  ")
			if err != nil {
				fmt.Println(err)
			}
			fmt.Print(string(b))

			_, err = destSrv.Events.Insert("primary", newEvent).Do()
			if err != nil {
				log.Printf("Error creating event: %v", err)
				continue
			}
		}
	}

	fmt.Println("Sync completed successfully")
}

func eventAlreadyExists(destinationEvents []*calendar.Event, sourceEventID string) bool {
	for _, event := range destinationEvents {
		if event.ExtendedProperties != nil && event.ExtendedProperties.Private != nil {
			if event.ExtendedProperties.Private[config.SourceEventIdPropertyKey] == sourceEventID {
				return true
			}
		}
	}
	return false
}
