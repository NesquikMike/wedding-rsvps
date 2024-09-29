package database

import (
	"fmt"
	"strings"
	"database/sql"
	"log"
	"math/rand"

	_ "github.com/mattn/go-sqlite3"

	"github.com/nesquikmike/wedding-rsvps/internal/models"
)

const (
	charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	codeLen = 12
)

type InviteeStore struct {
	db *sql.DB
}

func NewInviteeStore(db *sql.DB) InviteeStore {
	return InviteeStore{db: db}
}

func (i InviteeStore) SetupDatabase(inviteeNames [][]string) error {
	createTableQuery := `CREATE TABLE IF NOT EXISTS invitees (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        name TEXT NOT NULL,
		code TEXT NOT NULL,
		email TEXT,
		phone_number TEXT,
        dietary_requirements TEXT,
        attendance BOOLEAN,
        invalid_details BOOLEAN,
        details_provided BOOLEAN,
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
	if tableCount < len(inviteeNames) {
		missingInviteesCount := len(inviteeNames) - tableCount
		for idx := 0; idx < missingInviteesCount; idx++ {
			err = i.insertInvitee(inviteeNames[tableCount + idx][0])
			if err != nil {
				return err
			}
		}
	}

	log.Println("Database and table set up and populated successfully!")
	return nil
}

func (i InviteeStore) getTableCount() (int, error) {
    var count int
    query := "SELECT COUNT(*) FROM invitees"
    err := i.db.QueryRow(query).Scan(&count)
    if err != nil {
        return -1, err
    }
    return count, nil
}

func (i InviteeStore) insertInvitee(name string) error {
	tableCount, err := i.getTableCount()

	inviteeKey := tableCount + 1

	firstName := name
	if strings.Contains(name, " ") {
		firstName = strings.Split(name, " ")[0]
	}

	randCharsLen := codeLen - len(firstName) - 1
	code := firstName + "-" + generatePseudorandomString(inviteeKey, randCharsLen)
	
    insertQuery := `INSERT INTO invitees (name, code) VALUES (?, ?)`
    _, err = i.db.Exec(insertQuery, name, code)
    return err
}

func (i InviteeStore) UpdateInviteeDetails(code, email, phoneNumber, dietaryRequirements string) error {
    query := `UPDATE invitees
              SET
				email = ?,
				phone_number = ?, 
				dietary_requirements = ?, 
				invalid_details = false, 
				details_provided = true, 
				form_completed = true 
              WHERE code = ?`

    result, err := i.db.Exec(query, email, phoneNumber, dietaryRequirements, code)
    if err != nil {
        return fmt.Errorf("failed to update invitee: %v", err)
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

func (i InviteeStore) UpdateInviteeAttendance(code string, attendance, formCompleted bool) error {
    query := `UPDATE invitees
              SET attendance = ?, form_completed = ?
              WHERE code = ?`

    result, err := i.db.Exec(query, attendance, formCompleted, code)
    if err != nil {
        return fmt.Errorf("failed to update invitee: %v", err)
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

func (i InviteeStore) UpdateInviteeInvalidDetails(code string, invalidDetails bool) error {
    query := `UPDATE invitees
              SET invalid_details = ?
              WHERE code = ?`

    result, err := i.db.Exec(query, invalidDetails, code)
    if err != nil {
        return fmt.Errorf("failed to update invitee: %v", err)
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

func (i InviteeStore) GetInvitee(code string) (*models.Invitee, error) {
    // Prepare the query to select a guest by ID
    query := `SELECT
		name, 
		code, 
		email, 
		phone_number, 
		dietary_requirements,
		attendance,
		invalid_details,
		details_provided,
		form_completed 
	FROM invitees WHERE code = ?`
    
    // Execute the query
    row := i.db.QueryRow(query, code)

    // Create a Guest variable to hold the result
    var invitee models.Invitee

	var email sql.NullString
	var phoneNumber sql.NullString
	var dietaryRequirements sql.NullString
	var attendance sql.NullBool
	var invalidDetails sql.NullBool
	var detailsProvided sql.NullBool
	var formCompleted sql.NullBool

    // Scan the result into the guest struct
    err := row.Scan(
		&invitee.Name, 
		&invitee.Code, 
		&email, 
		&phoneNumber, 
		&dietaryRequirements, 
		&attendance, 
		&invalidDetails, 
		&detailsProvided,
		&formCompleted,
	)

	if email.Valid {
		invitee.Email = email.String
	}
	if phoneNumber.Valid {
		invitee.PhoneNumber = phoneNumber.String
	}
	if dietaryRequirements.Valid {
		invitee.DietaryRequirements = dietaryRequirements.String
	}
	if attendance.Valid {
		invitee.Attendance = attendance.Bool
	}
	if invalidDetails.Valid {
		invitee.InvalidDetails = invalidDetails.Bool
	}
	if detailsProvided.Valid {
		invitee.DetailsProvided = detailsProvided.Bool
	}
	if formCompleted.Valid {
		invitee.FormCompleted = formCompleted.Bool
	}
	fmt.Println("invitee: ", invitee)
	fmt.Println("err: ", err)
    
    if err == sql.ErrNoRows {
        return nil, nil // No guest found with the given ID
    } else if err != nil {
        return nil, err // Return error for other scan errors
    }

    return &invitee, nil // Return the guest struct
}
