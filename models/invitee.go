package models

type Invitee struct {
	Name                string
	Code                string
	Email               string
	PhoneNumber         string
	DietaryRequirements string
	Attendance          bool
	DetailsProvided     bool
}
