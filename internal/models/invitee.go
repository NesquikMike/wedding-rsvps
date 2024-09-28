package models

const InvalidInviteeKey = "invalid_invitee"

type Invitee struct {
	Name                string
	Code                string
	Email               string
	PhoneNumber         string
	DietaryRequirements string
	Attendance          bool
	InvalidDetails      bool
	DetailsProvided     bool
	FormCompleted       bool
}

var InvalidInvitee = Invitee{
	Code:           InvalidInviteeKey,
	InvalidDetails: true,
}
