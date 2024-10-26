package models

import (
	"html/template"
)

type ViewData struct {
	Url                   string
	PartnerOne            string
	PartnerTwo            string
	Date                  string
	VenueVague            string
	VenueAddress          template.HTML
	VenueTravelDetails    template.HTML
	TimeStart             string
	TimeArrival           string
	MainPhotoFileName     string
	PostCeremonyItinerary template.HTML
	BankName              string
	BankAccountName       string
	BankSortCode          string
	BankAccountNumber     string
	FooterMessage         template.HTML
	Guest                 *Guest
	SessionData           *SessionData
}
