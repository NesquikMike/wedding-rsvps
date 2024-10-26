package controllers

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

type UserRequest struct {
	GuestName string `json:"name"`
}

func (c Controller) AddGuest(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var userReq UserRequest
	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	if err := json.Unmarshal(body, &userReq); err != nil {
		http.Error(w, "Bad Request: Invalid JSON", http.StatusBadRequest)
		return
	}

	name := userReq.GuestName
	c.logger.Printf("/add-guest request for name %v", name)

	err = c.guestStore.InsertGuest(name)
	if err != nil {
		c.logger.Printf("error inserting guest %v: %v\n", name, err)
		http.Error(w, "Error inserting guest", http.StatusInternalServerError)
	}

	code, err := c.guestStore.GetGuestCode(name)
	if err != nil {
		c.logger.Printf("error getting guest code for guest %v: %v\n", name, err)
		http.Error(w, "Error returning guest code", http.StatusInternalServerError)
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("guest %v added with code %v", name, code)))
}

func (c Controller) GetGuest(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var userReq UserRequest
	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	if err := json.Unmarshal(body, &userReq); err != nil {
		http.Error(w, "Bad Request: Invalid JSON", http.StatusBadRequest)
		return
	}

	name := userReq.GuestName
	c.logger.Printf("/get-guest request for name %v", name)

	code, err := c.guestStore.GetGuestCode(name)
	if err != nil {
		c.logger.Printf("error getting guest code %v: %v\n", name, err)
		http.Error(w, "Error returning guest code", http.StatusInternalServerError)
	}

	guest, err := c.guestStore.GetGuest(code)
	if err != nil {
		c.logger.Printf("error getting guest %v: %v\n", code, err)
		http.Error(w, "Error returning guest", http.StatusInternalServerError)
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("guest %+v", guest)))
}

func (c Controller) GetRSVPs(w http.ResponseWriter, r *http.Request) {
	c.logger.Printf("/get-rsvps request")

	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// Example query (customize as needed)
	rows, err := c.guestStore.GetRSVPs()
	if err != nil {
		c.logger.Printf("Query error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Prepare CSV writer to write to the response
	w.Header().Set("Content-Disposition", "attachment;filename=rsvps.csv")
	w.Header().Set("Content-Type", "text/csv")
	csvWriter := csv.NewWriter(w)

	// Write CSV headers
	headers := []string{
		"ID",
		"Name",
		"Code",
		"Email",
		"Phone Number",
		"Meal Choice",
		"Dietary Requirements",
		"Attendance",
		"Invalid Details",
		"Details Provided",
		"Form Started",
		"Form Completed",
	}
	if err := csvWriter.Write(headers); err != nil {
		c.logger.Printf("CSV header error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Write the rows to the CSV
	for rows.Next() {
		var id int
		var name, code string
		var email, phoneNumber, mealChoice, dietaryRequirements sql.NullString
		var attendance, invalidDetails, detailsProvided, formStarted, formCompleted sql.NullBool
		if err := rows.Scan(
			&id,
			&name,
			&code,
			&email,
			&phoneNumber,
			&mealChoice,
			&dietaryRequirements,
			&attendance,
			&invalidDetails,
			&detailsProvided,
			&formStarted,
			&formCompleted,
		); err != nil {
			c.logger.Printf("Row scan error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		var emailStr, phoneNumberStr, mealChoiceStr, dietaryRequirementsStr string
		var attendanceBool, invalidDetailsBool, detailsProvidedBool, formStartedBool, formCompletedBool bool

		if email.Valid {
			emailStr = email.String
		}
		if phoneNumber.Valid {
			phoneNumberStr = phoneNumber.String
		}
		if mealChoice.Valid {
			mealChoiceStr = mealChoice.String
		}
		if dietaryRequirements.Valid {
			dietaryRequirementsStr = dietaryRequirements.String
		}
		if attendance.Valid {
			attendanceBool = attendance.Bool
		}
		if invalidDetails.Valid {
			invalidDetailsBool = invalidDetails.Bool
		}
		if detailsProvided.Valid {
			detailsProvidedBool = detailsProvided.Bool
		}
		if formCompleted.Valid {
			formCompletedBool = formCompleted.Bool
		}

		record := []string{
			strconv.Itoa(id),
			name,
			code,
			emailStr,
			phoneNumberStr,
			mealChoiceStr,
			dietaryRequirementsStr,
			strconv.FormatBool(attendanceBool),
			strconv.FormatBool(invalidDetailsBool),
			strconv.FormatBool(detailsProvidedBool),
			strconv.FormatBool(formStartedBool),
			strconv.FormatBool(formCompletedBool),
		}
		if err := csvWriter.Write(record); err != nil {
			c.logger.Printf("CSV write error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}

	// Flush the CSV writer and send the response
	csvWriter.Flush()
	if err := csvWriter.Error(); err != nil {
		c.logger.Printf("CSV flush error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func (c Controller) GetVisitsData(w http.ResponseWriter, r *http.Request) {
	c.logger.Printf("/get-visits-data request")

	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// Example query (customize as needed)
	rows, err := c.guestStore.GetVisitsData()
	if err != nil {
		c.logger.Printf("Query error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Prepare CSV writer to write to the response
	w.Header().Set("Content-Disposition", "attachment;filename=visit_data.csv")
	w.Header().Set("Content-Type", "text/csv")
	csvWriter := csv.NewWriter(w)

	// Write CSV headers
	headers := []string{
		"ID",
		"Page Name",
		"Visit Count",
		"First Visit Time",
		"Latest Visit Time",
	}
	if err := csvWriter.Write(headers); err != nil {
		c.logger.Printf("CSV header error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Write the rows to the CSV
	for rows.Next() {
		var id, visitCount int
		var pageName, firstVisitTime, latestVisitTime string
		if err := rows.Scan(
			&id,
			&pageName,
			&visitCount,
			&firstVisitTime,
			&latestVisitTime,
		); err != nil {
			c.logger.Printf("Row scan error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		record := []string{
			strconv.Itoa(id),
			pageName,
			strconv.Itoa(visitCount),
			firstVisitTime,
			latestVisitTime,
		}
		if err := csvWriter.Write(record); err != nil {
			c.logger.Printf("CSV write error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}

	// Flush the CSV writer and send the response
	csvWriter.Flush()
	if err := csvWriter.Error(); err != nil {
		c.logger.Printf("CSV flush error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
