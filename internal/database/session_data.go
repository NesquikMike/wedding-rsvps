package database

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/nesquikmike/wedding-rsvps/internal/models"
)

func (i GuestStore) createSessionDataTable() error {
	createTableQuery := `CREATE TABLE IF NOT EXISTS session_data (
        code TEXT PRIMARY KEY,
		invalid_email BOOLEAN,
		invalid_phone_number BOOLEAN,
		invalid_dietary_requirements BOOLEAN
    );`

	_, err := i.db.Exec(createTableQuery)
	if err != nil {
		return err
	}

	log.Println("session_data table set up successfully!")
	return nil
}

func (i GuestStore) UpdateSessionInvalidEmail(code string, invalid bool) error {
	query := `
	INSERT INTO session_data (code, invalid_email) 
	VALUES (?, ?)
	ON CONFLICT(code)
	DO UPDATE SET invalid_email = excluded.invalid_email;
	`

	result, err := i.db.Exec(query, code, invalid)
	if err != nil {
		return fmt.Errorf("failed to save session data: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to retrieve affected rows: %v", err)
	}

	if rowsAffected != 1 {
		return fmt.Errorf("rowsAffected %v for code %v with invalid email address was not 1", rowsAffected, code)
	}

	return nil
}

func (i GuestStore) UpdateSessionInvalidPhoneNumber(code string, invalid bool) error {
	query := `
	INSERT INTO session_data (code, invalid_phone_number) 
	VALUES (?, ?)
	ON CONFLICT(code)
	DO UPDATE SET invalid_phone_number = excluded.invalid_phone_number;
	`

	result, err := i.db.Exec(query, code, invalid)
	if err != nil {
		return fmt.Errorf("failed to save session data: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to retrieve affected rows: %v", err)
	}

	if rowsAffected != 1 {
		return fmt.Errorf("rowsAffected %v for code %v with invalid phone number was not 1", rowsAffected, code)
	}

	return nil
}

func (i GuestStore) UpdateSessionInvalidDietaryRequirements(code string, invalid bool) error {
	query := `
	INSERT INTO session_data (code, invalid_dietary_requirements) 
	VALUES (?, ?)
	ON CONFLICT(code)
	DO UPDATE SET invalid_dietary_requirements = excluded.invalid_dietary_requirements;
	`

	result, err := i.db.Exec(query, code, invalid)
	if err != nil {
		return fmt.Errorf("failed to save session data: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to retrieve affected rows: %v", err)
	}

	if rowsAffected != 1 {
		return fmt.Errorf("rowsAffected %v for code %v with invalid dietary requirements was not 1", rowsAffected, code)
	}

	return nil
}

func (i GuestStore) GetSessionData(code string) (*models.SessionData, error) {
	query := `SELECT
		invalid_email, invalid_phone_number, invalid_dietary_requirements 
	FROM session_data WHERE code = ?`

	row := i.db.QueryRow(query, code)

	var invalidEmail sql.NullBool
	var invalidPhoneNumber sql.NullBool
	var invalidDietaryRequirements sql.NullBool

	err := row.Scan(
		&invalidEmail,
		&invalidPhoneNumber,
		&invalidDietaryRequirements,
	)
	if err != nil {
		return nil, err
	}

	var invalidEmailBool bool
	var invalidPhoneNumberBool bool
	var invalidDietaryRequirementsBool bool

	if invalidEmail.Valid {
		invalidEmailBool = invalidEmail.Bool
	}
	if invalidPhoneNumber.Valid {
		invalidPhoneNumberBool = invalidPhoneNumber.Bool
	}
	if invalidDietaryRequirements.Valid {
		invalidDietaryRequirementsBool = invalidDietaryRequirements.Bool
	}

	sessionData := &models.SessionData{
		Code:                       code,
		InvalidEmail:               invalidEmailBool,
		InvalidPhoneNumber:         invalidPhoneNumberBool,
		InvalidDietaryRequirements: invalidDietaryRequirementsBool,
	}

	return sessionData, nil
}
