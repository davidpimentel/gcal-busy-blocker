# Gcal Busy Blocker

Keep your meeting availability up to date on your work Google Calendar by creating "busy" blocks based on your personal calendar

## Setup Instructions

### Configure a GCP project

- Create a new GCP project
- Enable the Google Calendar api
- Create Oauth credentials (desktop type). Save the json file
- Important! - Complete the Oauth consent screen, and add both your personal and work emails as test users under the audience section

### Installation

- Run `go install github.com/davidpimentel/gcal-busy-blocker@latest`
- Run `gcal-busy-blocker set-oauth-credentials -p /path/to/credentials`

### Autenticate your accounts

Run `gcal-busy-blocker login source` for your personal account and `gcal-busy-blocker login destination` for your work account to authenticate each

### Sync your calendars

Run `gcal-busy-blocker sync` to sync events from the source calendar to the destination calendar
