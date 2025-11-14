package sync

import (
	"testing"
	"time"

	"google.golang.org/api/calendar/v3"
)

// Mock implementations for testing
type MockCalendarEventsService struct {
	events         []*calendar.Event
	insertedEvents []*calendar.Event
	deletedEvents  []string
	listCalls      []listCallParams
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
		listCalls:      []listCallParams{},
	}
}

func (m *MockCalendarEventsService) List(calendarId string, startTime string, endTime string, privateProperties map[string]string) ([]*calendar.Event, error) {
	m.listCalls = append(m.listCalls, listCallParams{
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
