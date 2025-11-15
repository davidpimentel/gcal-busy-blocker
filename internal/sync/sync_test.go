package sync

import (
	"testing"
	"time"

	"google.golang.org/api/calendar/v3"
)

type MockCalendarEventsService struct {
	events         []*calendar.Event
	insertedEvents []*calendar.Event
	deletedEvents  []string
	listCalls      []*listCallParams
}

type listCallParams struct {
	calendarId        string
	startTime         string
	endTime           string
	privateProperties map[string]string
}

func NewMockCalendarEventsService(events []*calendar.Event) *MockCalendarEventsService {
	return &MockCalendarEventsService{
		events:         events,
		insertedEvents: []*calendar.Event{},
		deletedEvents:  []string{},
		listCalls:      []*listCallParams{},
	}
}

func (m *MockCalendarEventsService) List(calendarId string, startTime string, endTime string, privateProperties map[string]string) ([]*calendar.Event, error) {
	m.listCalls = append(m.listCalls, &listCallParams{
		calendarId:        calendarId,
		startTime:         startTime,
		endTime:           endTime,
		privateProperties: privateProperties,
	})

	return m.events, nil
}

func (m *MockCalendarEventsService) Insert(calendarId string, event *calendar.Event) (*calendar.Event, error) {
	m.insertedEvents = append(m.insertedEvents, event)
	return event, nil
}

func (m *MockCalendarEventsService) Delete(calendarId string, eventId string) error {
	m.deletedEvents = append(m.deletedEvents, eventId)
	return nil
}

// Helper function to create a test event
func createTestEvent(id string, summary string, startTime, endTime time.Time, privateProps map[string]string) *calendar.Event {
	event := &calendar.Event{
		Id:      id,
		Summary: summary,
		Start: &calendar.EventDateTime{
			DateTime: startTime.Format(time.RFC3339),
		},
		End: &calendar.EventDateTime{
			DateTime: endTime.Format(time.RFC3339),
		},
	}

	if privateProps != nil {
		event.ExtendedProperties = &calendar.EventExtendedProperties{
			Private: privateProps,
		}
	}

	return event
}

func eventTimesEqual(event *calendar.Event, event2 *calendar.Event) bool {
	return event.Start.DateTime != event2.Start.DateTime || event.End.DateTime != event2.End.DateTime
}

func TestRunSync(t *testing.T) {
	mockSourceService := &MockCalendarEventsService{
		events: []*calendar.Event{
			createTestEvent("123", "test summary", time.Now(), time.Now().Add(time.Hour), nil),
			createTestEvent("456", "test summary2", time.Now(), time.Now(), nil),
		}}
	mockDestinationService := &MockCalendarEventsService{}
	syncClient := &SyncClient{
		SourceCalendarService:      mockSourceService,
		DestinationCalendarService: mockDestinationService,
	}

	syncClient.RunSync(30, false)

	if len(mockSourceService.listCalls) != 1 {
		t.Errorf("Sync should only call source List once")
	}
	if len(mockDestinationService.listCalls) != 1 {
		t.Errorf("Sync should only call destination List once")
	}

	for i, event := range mockDestinationService.insertedEvents {
		sourceEvent := mockSourceService.events[i]
		if eventTimesEqual(event, sourceEvent) {
			t.Errorf("start and end times are not the same between source and destination events")
		}

		if event.ExtendedProperties.Private[appName] != propertyAppNameValue {
			t.Errorf("appName private property not set for destination event")
		}

		if event.ExtendedProperties.Private[sourceEventIdPropertyKey] != sourceEvent.Id {
			t.Errorf("Source Event ID private property not set")
		}
	}
}

func TestRunSyncAlreadyAdded(t *testing.T) {
	mockEvents := []*calendar.Event{
		createTestEvent("123", "test summary", time.Now(), time.Now().Add(time.Hour), nil),
		createTestEvent("456", "test summary2", time.Now(), time.Now(), nil),
	}
	mockDestinationEvents := []*calendar.Event{}
	for _, event := range mockEvents {
		mockDestinationEvents = append(
			mockDestinationEvents,
			createTestEvent("abc", "Busy", time.Now(), time.Now(), map[string]string{appName: propertyAppNameValue, sourceEventIdPropertyKey: event.Id}),
		)
	}

	mockSourceService := &MockCalendarEventsService{
		events: mockEvents}
	mockDestinationService := &MockCalendarEventsService{events: mockDestinationEvents}
	syncClient := &SyncClient{
		SourceCalendarService:      mockSourceService,
		DestinationCalendarService: mockDestinationService,
	}

	syncClient.RunSync(30, false)

	if len(mockDestinationService.insertedEvents) != 0 {
		t.Errorf("An event was inserted when it shouldn't be")
	}
}

func TestRunSyncDryRun(t *testing.T) {
	mockSourceService := &MockCalendarEventsService{events: []*calendar.Event{createTestEvent("123", "test summary", time.Now(), time.Now(), nil)}}
	mockDestinationService := &MockCalendarEventsService{}
	syncClient := &SyncClient{
		SourceCalendarService:      mockSourceService,
		DestinationCalendarService: mockDestinationService,
	}

	syncClient.RunSync(30, true)

	if len(mockDestinationService.insertedEvents) != 0 {
		t.Errorf("Expected dry run to insert 0 events, inserted %d", len(mockDestinationService.insertedEvents))
	}
}
