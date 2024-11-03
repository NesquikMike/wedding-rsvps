package models

type SessionData struct {
	Code                       string
	InvalidEmail               bool
	InvalidPhoneNumber         bool
	InvalidDietaryRequirements bool
}
