package utilities

import (
	"amlwwalker/gmail-backend/backend/pkg/database"
)
type Report struct {
	Err string
	Email database.CampaignEmail
}