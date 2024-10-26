package models

const InvalidGuestKey = "invalid_guest"

type Guest struct {
	ID                  int
	Name                string
	Code                string
	Email               string
	PhoneNumber         string
	MealChoice          string
	DietaryRequirements string
	Attendance          bool
	InvalidDetails      bool
	DetailsProvided     bool
	FormStarted         bool
	FormCompleted       bool
}

var InvalidGuest = Guest{
	Code:           InvalidGuestKey,
	InvalidDetails: true,
}
