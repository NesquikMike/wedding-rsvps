package models

import (
	"html/template"
)

type ViewData struct {
	Url               string
	PartnerOne        string
	PartnerTwo        string
	Date              string
	Venue             string
	TimeStart         string
	BankName          string
	BankAccountName   string
	BankSortCode      string
	BankAccountNumber string
	FooterMessage     template.HTML
	Invitee           *Invitee
}
