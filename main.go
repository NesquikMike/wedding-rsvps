package main

import (
	"io"
	"bufio"
	"database/sql"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"github.com/nesquikmike/wedding-rsvps/internal/controllers"
	"github.com/nesquikmike/wedding-rsvps/internal/database"
	"github.com/nesquikmike/wedding-rsvps/internal/models"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

var tpl *template.Template

const (
	requiredLenSecretCookieKey = 32
	csvPath                    = "./names.csv"
)

func init() {
	tpl = template.Must(template.ParseGlob("templates/*.gohtml"))
	template.Must(tpl.ParseGlob("templates/form/*.gohtml"))
}

func main() {
	file, err := os.Open(".env")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	envVars := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		l := scanner.Text()
		k, v, ok := strings.Cut(l, "=")
		if !ok {
			log.Fatal(fmt.Errorf("no equal sign to split env key-value pair"))
		}
		v = strings.Trim(v, "\"'")

		envVars[k] = v
	}

	isProd := envVars["ENVIRONMENT"] == "production"

	logFile, err := os.OpenFile("server.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening log file: %v", err)
	}
	defer logFile.Close()

	multiWriter := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(multiWriter)

	secretCookieKey, err := hex.DecodeString(envVars["SECRET_COOKIE_KEY"])
	if err != nil {
		log.Fatal(err)
	}

	if len(secretCookieKey) != requiredLenSecretCookieKey {
		log.Fatal(fmt.Errorf("secretCookieKey is %v bytes long when it should be %v", len(secretCookieKey), requiredLenSecretCookieKey))
	}

	// Open the SQLite database (creates the file if it doesn't exist)
	db, err := sql.Open("sqlite3", "./invitees.db")
	if err != nil {
		log.Fatal("Error opening up database: ", err)
	}
	defer db.Close()

	rows, err := readCSV(csvPath)
	if err != nil {
		log.Fatal("Error reading csv: ", err)
	}

	inviteeStore := database.NewInviteeStore(db)
	err = inviteeStore.SetupDatabase(rows)
	if err != nil {
		log.Fatal("Error setting up database: ", err)
	}

	viewData := models.ViewData{
		Url:                   envVars["URL"],
		PartnerOne:            envVars["PARTNER_ONE"],
		PartnerTwo:            envVars["PARTNER_TWO"],
		Date:                  envVars["DATE"],
		VenueVague:            envVars["VENUE_VAGUE"],
		VenueAddress:          template.HTML(strings.ReplaceAll(envVars["VENUE_ADDRESS"], "\\", "")),
		VenueTravelDetails:    template.HTML(strings.ReplaceAll(envVars["VENUE_TRAVEL_DETAILS"], "\\", "")),
		TimeStart:             envVars["TIME_START"],
		TimeArrival:           envVars["TIME_ARRIVAL"],
		MainPhotoFileName:     envVars["MAIN_PHOTO_FILE_NAME"],
		PostCeremonyItinerary: template.HTML(strings.ReplaceAll(envVars["POST_CEREMONY_ITINERARY"], "\\", "")),
		BankName:              envVars["BANK_NAME"],
		BankAccountName:       envVars["BANK_ACCOUNT_NAME"],
		BankSortCode:          envVars["BANK_SORT_CODE"],
		BankAccountNumber:     envVars["BANK_ACCOUNT_NUMBER"],
		FooterMessage:         template.HTML(strings.ReplaceAll(envVars["FOOTER_MESSAGE"], "\\", "")),
	}

	c := controllers.NewController(isProd, tpl, inviteeStore, log.Default(), &viewData, secretCookieKey)

	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("./assets"))))
	http.HandleFunc("/", c.Index)
	http.HandleFunc("/rsvp", c.RSVP)
	http.HandleFunc("/invitee-details", c.InviteeDetails)
	http.HandleFunc("/change-details", c.ChangeDetails)
	http.HandleFunc("/change-attendance-response", c.ChangeAttendanceResponse)
	http.HandleFunc("/reset-invitee", c.ResetInvitee)
	http.Handle("/favicon.ico", http.NotFoundHandler())
	http.ListenAndServe(":8080", nil)
}

func readCSV(filePath string) ([][]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	return records, nil
}
