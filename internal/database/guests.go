package database

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"strings"

	_ "github.com/mattn/go-sqlite3"

	"github.com/nesquikmike/wedding-rsvps/internal/models"
)

const (
	charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	codeLen = 12
)

type GuestStore struct {
	db *sql.DB
}

func NewGuestStore(db *sql.DB) GuestStore {
	return GuestStore{db: db}
}

func (i GuestStore) SetupDatabase(guestNames [][]string) error {
	err := i.createGuestsTable(guestNames)
	if err != nil {
		return err
	}

	err = i.createPageVisitsTable()
	if err != nil {
		return err
	}

	err = i.createSessionDataTable()
	if err != nil {
		return err
	}

	log.Println("Database and tables set up and populated successfully!")
	return nil
}

func (i GuestStore) createGuestsTable(guestNames [][]string) error {
	createTableQuery := `CREATE TABLE IF NOT EXISTS guests (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        name TEXT NOT NULL,
		code TEXT NOT NULL,
		email TEXT,
		phone_number TEXT,
		meal_choice BOOLEAN,
        dietary_requirements TEXT,
        attendance BOOLEAN,
        invalid_details BOOLEAN,
        details_provided BOOLEAN,
        form_started BOOLEAN NOT NULL,
        form_completed BOOLEAN
    );`

	_, err := i.db.Exec(createTableQuery)
	if err != nil {
		return err
	}

	tableCount, err := i.getTableCount()
	if err != nil {
		return err
	}

	// This assumes that any new additions are appended to the end of the csv
	if tableCount < len(guestNames) {
		missingGuestsCount := len(guestNames) - tableCount
		for idx := 0; idx < missingGuestsCount; idx++ {
			err = i.InsertGuest(guestNames[tableCount+idx][0])
			if err != nil {
				return err
			}
		}
	}

	log.Println("guests table set up and populated successfully!")
	return nil
}

func (i GuestStore) getTableCount() (int, error) {
	var count int
	query := "SELECT COUNT(*) FROM guests"
	err := i.db.QueryRow(query).Scan(&count)
	if err != nil {
		return -1, err
	}
	return count, nil
}

func (i GuestStore) InsertGuest(name string) error {
	tableCount, err := i.getTableCount()

	guestKey := tableCount + 1

	firstName := name
	if strings.Contains(name, " ") {
		firstName = strings.Split(name, " ")[0]
	}

	randCharsLen := codeLen - len(firstName) - 1
	code := firstName + "-" + generatePseudorandomString(guestKey, randCharsLen)

	insertQuery := `INSERT INTO guests (name, code, form_started) VALUES (?, ?, false)`
	_, err = i.db.Exec(insertQuery, name, code)
	return err
}

func (i GuestStore) UpdateGuestEmail(code, email string) error {
	query := `UPDATE guests
              SET
				email = ?
              WHERE code = ?`

	result, err := i.db.Exec(query, email, code)
	if err != nil {
		return fmt.Errorf("failed to update guest %v email %v: %v", code, email, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to retrieve affected rows: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no guest found with code %s", code)
	}

	return nil
}

func (i GuestStore) UpdateGuestPhoneNumber(code, phoneNumber string) error {
	query := `UPDATE guests
              SET
				phone_number = ? 
              WHERE code = ?`

	result, err := i.db.Exec(query, phoneNumber, code)
	if err != nil {
		return fmt.Errorf("failed to update guest %v phone number %v: %v", code, phoneNumber, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to retrieve affected rows: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no guest found with code %s", code)
	}

	return nil
}

func (i GuestStore) UpdateGuestMealChoice(code, mealChoice string) error {
	query := `UPDATE guests
              SET
				meal_choice = ? 
              WHERE code = ?`

	result, err := i.db.Exec(query, mealChoice, code)
	if err != nil {
		return fmt.Errorf("failed to update guest %v meal choice %v: %v", code, mealChoice, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to retrieve affected rows: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no guest found with code %s", code)
	}

	return nil
}

func (i GuestStore) UpdateGuestDietaryRequirements(code, dietaryRequirements string) error {
	query := `UPDATE guests
              SET
				dietary_requirements = ? 
              WHERE code = ?`

	result, err := i.db.Exec(query, dietaryRequirements, code)
	if err != nil {
		return fmt.Errorf("failed to update guest %v dietary requirements %v: %v", code, dietaryRequirements, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to retrieve affected rows: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no guest found with code %s", code)
	}

	return nil
}

func (i GuestStore) UpdateGuestDetailsProvidedSuccessfully(code string) error {
	query := `UPDATE guests
              SET
				invalid_details = false, 
				details_provided = true, 
				form_completed = true 
              WHERE code = ?`

	result, err := i.db.Exec(query, code)
	if err != nil {
		return fmt.Errorf("failed to update guest: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to retrieve affected rows: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no guest found with code %s", code)
	}

	return nil
}

func (i GuestStore) UpdateGuestAttendance(code string, attendance, formCompleted bool) error {
	query := `UPDATE guests
              SET attendance = ?, form_started = true, form_completed = ?
              WHERE code = ?`

	result, err := i.db.Exec(query, attendance, formCompleted, code)
	if err != nil {
		return fmt.Errorf("failed to update guest: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to retrieve affected rows: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no guest found with code %s", code)
	}

	return nil
}

func (i GuestStore) UpdateGuestInvalidDetails(code string, invalidDetails bool) error {
	query := `UPDATE guests
              SET invalid_details = ?
              WHERE code = ?`

	result, err := i.db.Exec(query, invalidDetails, code)
	if err != nil {
		return fmt.Errorf("failed to update guest: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to retrieve affected rows: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no guest found with code %s", code)
	}

	return nil
}

func generatePseudorandomString(seed int, length int) string {
	source := rand.New(rand.NewSource(int64(seed)))

	result := make([]byte, length)

	for i := range result {
		randomIndex := source.Intn(len(charset))
		result[i] = charset[randomIndex]
	}

	return string(result)
}

func (i GuestStore) GetGuest(code string) (*models.Guest, error) {
	query := `SELECT
		id, 
		name, 
		code, 
		email, 
		phone_number, 
		meal_choice,
		dietary_requirements,
		attendance,
		invalid_details,
		details_provided,
		form_started,
		form_completed 
	FROM guests WHERE code = ?`

	row := i.db.QueryRow(query, code)

	var guest models.Guest

	var email sql.NullString
	var phoneNumber sql.NullString
	var mealChoice sql.NullString
	var dietaryRequirements sql.NullString
	var attendance sql.NullBool
	var invalidDetails sql.NullBool
	var detailsProvided sql.NullBool
	var formCompleted sql.NullBool

	// Scan the result into the guest struct
	err := row.Scan(
		&guest.ID,
		&guest.Name,
		&guest.Code,
		&email,
		&phoneNumber,
		&mealChoice,
		&dietaryRequirements,
		&attendance,
		&invalidDetails,
		&detailsProvided,
		&guest.FormStarted,
		&formCompleted,
	)

	if email.Valid {
		guest.Email = email.String
	}
	if phoneNumber.Valid {
		guest.PhoneNumber = phoneNumber.String
	}
	if mealChoice.Valid {
		guest.MealChoice = mealChoice.String
	}
	if dietaryRequirements.Valid {
		guest.DietaryRequirements = dietaryRequirements.String
	}
	if attendance.Valid {
		guest.Attendance = attendance.Bool
	}
	if invalidDetails.Valid {
		guest.InvalidDetails = invalidDetails.Bool
	}
	if detailsProvided.Valid {
		guest.DetailsProvided = detailsProvided.Bool
	}
	if formCompleted.Valid {
		guest.FormCompleted = formCompleted.Bool
	}

	if err == sql.ErrNoRows {
		return nil, nil // No guest found with the given ID
	} else if err != nil {
		return nil, err // Return error for other scan errors
	}

	return &guest, nil // Return the guest struct
}

func (i GuestStore) GetGuestCode(name string) (string, error) {
	query := `SELECT
		code 
	FROM guests WHERE name = ?`

	// Execute the query
	row := i.db.QueryRow(query, name)

	var code sql.NullString

	err := row.Scan(
		&code,
	)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("no guest found with the given name %v", name)
	} else if err != nil {
		return "", err // Return error for other scan errors
	}

	if !code.Valid {
		return "", fmt.Errorf("code is invalid for name %v", name)
	}

	return code.String, nil
}

func (i GuestStore) GetRSVPs() (*sql.Rows, error) {
	query := `SELECT
		id, 
		name, 
		code, 
		email, 
		phone_number, 
		meal_choice,
		dietary_requirements,
		attendance,
		invalid_details,
		details_provided,
		form_started,
		form_completed 
	FROM guests`

	rows, err := i.db.Query(query)
	if err != nil {
		return nil, err
	}

	return rows, nil
}
