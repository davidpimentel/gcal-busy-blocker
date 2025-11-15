package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"slices"
	"time"

	"github.com/davidpimentel/gcal-busy-blocker/internal/auth"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

type SyncClient struct {
	SourceCalendarService      CalendarEventsService
	DestinationCalendarService CalendarEventsService
}

const (
	appName                  = "gcal-busy-blocker"
	defaultCalendar          = "primary"
	propertyAppNameValue     = "true"
	sourceEventIdPropertyKey = "gcal-busy-blocker-source-event-id"
)

func NewSyncClient() *SyncClient {

	// Get source client
	sourceClient, err := auth.SourceClient()
	if err != nil {
		log.Fatalf("Unable to get source client: %v", err)
	}

	// Get destination client
	destClient, err := auth.DestinationClient()
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

	return &SyncClient{
		SourceCalendarService:      &calendarEventsService{service: sourceSrv},
		DestinationCalendarService: &calendarEventsService{service: destSrv},
	}
}

func (s *SyncClient) RunSync(daysAhead int, dryRun bool) error {
	if dryRun {
		log.Println("DRY RUN!")
	}
	log.Println("Starting calendar sync...")

	now := time.Now().Format(time.RFC3339)
	endTime := time.Now().AddDate(0, 0, daysAhead).Format(time.RFC3339)

	log.Println("Fetching events from source calendar...")
	log.Printf("Time range: %s to %s\n", now, endTime)

	// List events from source calendar
	sourceEvents := s.fetchSourceEvents(now, endTime)

	if len(sourceEvents) == 0 {
		log.Println("No upcoming events found in source calendar.")
		return nil
	}

	log.Printf("Found %d events in source calendar\n", len(sourceEvents))

	existingDestinationEvents := s.fetchBusyBlockEvents(now, endTime)

	for _, event := range sourceEvents {
		log.Printf("Event: %s (%s)\n", event.Summary, event.Id)

		if eventAlreadyExists(existingDestinationEvents, event.Id) {
			log.Printf("Event already synced: %s, skipping...\n", event.Id)
		} else {
			newEvent := createDestinationEvent(event)

			log.Println("Creating new event")

			if dryRun {
				b, err := json.MarshalIndent(newEvent, "", "  ")
				if err != nil {
					return err
				}
				log.Println(string(b))
			} else {
				_, err := s.DestinationCalendarService.Insert("primary", newEvent)
				if err != nil {
					log.Printf("Error creating event: %v", err)
					return err
				}
			}
		}
	}

	// Remove blocks that don't exist in source calendar anymore
	oldEvents := findOldEvents(sourceEvents, existingDestinationEvents)
	for _, event := range oldEvents {
		err := s.deleteDestinationEvent(event)
		if err != nil {
			return err
		}
	}

	log.Println("Sync completed successfully")
	return nil
}

func (s *SyncClient) fetchBusyBlockEvents(startTime string, endTime string) []*calendar.Event {
	events, err := s.DestinationCalendarService.List(defaultCalendar, startTime, endTime, map[string]string{appName: propertyAppNameValue})
	if err != nil {
		log.Fatalf("Unable to fetch destination calendar events: %v", err)
	}
	return events
}

func (s *SyncClient) fetchSourceEvents(startTime string, endTime string) []*calendar.Event {
	events, err := s.SourceCalendarService.List(defaultCalendar, startTime, endTime, nil)
	if err != nil {
		log.Fatalf("Unable to fetch source calendar events: %v", err)
	}
	return events
}

func findOldEvents(sourceEvents []*calendar.Event, destinationEvents []*calendar.Event) []*calendar.Event {
	sourceEventIds := []string{}
	for _, event := range sourceEvents {
		sourceEventIds = append(sourceEventIds, event.Id)
	}

	oldEvents := []*calendar.Event{}

	for _, destinationEvent := range destinationEvents {
		if !slices.Contains(sourceEventIds, destinationEvent.ExtendedProperties.Private[sourceEventIdPropertyKey]) {
			oldEvents = append(oldEvents, destinationEvent)
		}
	}
	return oldEvents
}

func createDestinationEvent(sourceEvent *calendar.Event) *calendar.Event {
	return &calendar.Event{
		ColorId:     "4",
		Summary:     "Busy",
		Description: "Created with <a href=\"https://github.com/davidpimentel/gcal-busy-blocker\">gcal-busy-blocker</a>. User has a personal commitment and is busy at this time. Please find another time to avoid scheduling conflicts.",
		Start:       sourceEvent.Start,
		End:         sourceEvent.End,
		// Add extended properties to track the source event
		ExtendedProperties: &calendar.EventExtendedProperties{
			Private: map[string]string{
				appName:                  propertyAppNameValue,
				sourceEventIdPropertyKey: sourceEvent.Id,
			},
		},
		Source: &calendar.EventSource{
			Title: "gcal-busy-blocker",
			Url:   "https://github.com/davidpimentel/gcal-busy-blocker",
		},
	}
}

func eventAlreadyExists(destinationEvents []*calendar.Event, sourceEventID string) bool {
	for _, event := range destinationEvents {
		if event.ExtendedProperties != nil && event.ExtendedProperties.Private != nil {
			if event.ExtendedProperties.Private[sourceEventIdPropertyKey] == sourceEventID {
				return true
			}
		}
	}
	return false
}

func (s *SyncClient) Clean(dryRun bool) error {
	events := s.fetchBusyBlockEvents("", "")

	for _, event := range events {

		if dryRun {
			fmt.Printf("DRY RUN - Deleting event at %s - %s\n", event.Start.DateTime, event.End.DateTime)
		} else {
			fmt.Printf("Deleting event at %s - %s\n", event.Start.DateTime, event.End.DateTime)
			err := s.deleteDestinationEvent(event)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *SyncClient) deleteDestinationEvent(event *calendar.Event) error {
	// Sanity check, ensure each event is definitely ours
	if event.ExtendedProperties.Private[appName] != propertyAppNameValue {
		return fmt.Errorf("aborting, almost deleted an event we weren't supposed to! Event ID = %s", event.Id)
	}

	err := s.DestinationCalendarService.Delete(defaultCalendar, event.Id)
	if err != nil {
		return fmt.Errorf("error deleting event %s: %v", event.Id, err)
	}
	return nil
}
