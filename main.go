package main

import (
	"bufio"
	"database/sql"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/nesquikmike/wedding-rsvps/internal/backup"
	"github.com/nesquikmike/wedding-rsvps/internal/controllers"
	"github.com/nesquikmike/wedding-rsvps/internal/database"
	"github.com/nesquikmike/wedding-rsvps/internal/models"

	_ "github.com/mattn/go-sqlite3"
)

var tpl *template.Template

const (
	requiredLenSecretCookieKey = 32
	csvPath                    = "./names.csv"
	guestsDBFilePath           = "./guests.db"
	backupTimeInterval         = 24 * time.Hour
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

	logFile, err := setNewLogFile(nil)
	if err != nil {
		log.Fatalf("error opening log file: %v", err)
	}
	defer logFile.Close()

	secretCookieKey, err := hex.DecodeString(envVars["SECRET_COOKIE_KEY"])
	if err != nil {
		log.Fatal(err)
	}

	if len(secretCookieKey) != requiredLenSecretCookieKey {
		log.Fatal(fmt.Errorf("secretCookieKey is %v bytes long when it should be %v", len(secretCookieKey), requiredLenSecretCookieKey))
	}

	// Open the SQLite database (creates the file if it doesn't exist)
	db, err := sql.Open("sqlite3", guestsDBFilePath)
	if err != nil {
		log.Fatal("Error opening up database: ", err)
	}
	defer db.Close()

	rows, err := readCSV(csvPath)
	if err != nil {
		log.Fatal("Error reading csv: ", err)
	}

	guestStore := database.NewGuestStore(db)
	err = guestStore.SetupDatabase(rows)
	if err != nil {
		log.Fatal("Error setting up database: ", err)
	}

	s3BucketBackups := envVars["S3_BUCKET_BACKUPS"]
	var s3Uploader *backup.S3Uploader
	if s3BucketBackups != "" {
		s3Uploader, err = backup.NewS3Uploader(s3BucketBackups, isProd)
		if err != nil {
			log.Fatal("Error setting up S3Uploader: ", err)
		}
	}

	go startMidnightTicker(s3Uploader, logFile)

	apiKey := envVars["API_KEY"]

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

	s3BucketAssets := envVars["S3_BUCKET_ASSETS"]

	c := controllers.NewController(isProd, tpl, guestStore, log.Default(), &viewData, secretCookieKey, apiKey, s3BucketAssets)
	if s3BucketAssets != "" {
		http.HandleFunc("/assets/", c.StaticHandler)
	} else {
		http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("./assets"))))
	}
	http.HandleFunc("/", c.Index)
	http.HandleFunc("/rsvp", c.RSVP)
	http.HandleFunc("/guest-details", c.GuestDetails)
	http.HandleFunc("/change-details", c.ChangeDetails)
	http.HandleFunc("/change-attendance-response", c.ChangeAttendanceResponse)
	http.HandleFunc("/reset-guest", c.ResetGuest)
	http.HandleFunc("/api/add-guest", c.ApiKeyMiddleware(c.AddGuest))
	http.HandleFunc("/api/get-guest", c.ApiKeyMiddleware(c.GetGuest))
	http.HandleFunc("/api/get-rsvps", c.ApiKeyMiddleware(c.GetRSVPs))
	http.HandleFunc("/api/get-visits-data", c.ApiKeyMiddleware(c.GetVisitsData))
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

func setNewLogFile(oldFile *os.File) (*os.File, error) {
	oldFile.Close()

	err := os.MkdirAll("logs", os.ModePerm)
	if err != nil {
		log.Println("error creating logs folder: ", err)
		return nil, err
	}

	logFileName := fmt.Sprintf("logs/server_%s.log", time.Now().Format("2006-01-02"))
	logFile, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return logFile, err
	}

	multiWriter := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(multiWriter)

	log.Println("new logger set")

	return logFile, nil
}

func startMidnightTicker(s3Uploader *backup.S3Uploader, oldLogFile *os.File) {
	serverStart := time.Now()
	firstMidnight := serverStart.Truncate(backupTimeInterval).Add(backupTimeInterval)
	durationUntilFirstMidnight := firstMidnight.Sub(serverStart)

	time.Sleep(durationUntilFirstMidnight)

	ticker := time.NewTicker(backupTimeInterval)
	defer ticker.Stop()

	logFile, err := setNewLogFile(oldLogFile)
	if err != nil {
		log.Fatalf("error opening log file: %v", err)
	}

	if s3Uploader != nil {
		err = performBackups(s3Uploader)
		if err != nil {
			log.Printf("error performing backups: %v", err)
		}
	}

	for range ticker.C {
		logFile, err = setNewLogFile(logFile)
		if err != nil {
			log.Printf("error opening log file: %v", err)
		}

		if s3Uploader != nil {
			err = performBackups(s3Uploader)
			if err != nil {
				log.Printf("error performing backups: %v", err)
			}
		}
	}

	defer logFile.Close()
}

func performBackups(s3Uploader *backup.S3Uploader) error {
	ydayDate := time.Now().Add(backupTimeInterval).Format("2006-01-02")
	oldLogFileName := fmt.Sprintf("logs/server_%s.log", ydayDate)
	err := s3Uploader.UploadFile(oldLogFileName, oldLogFileName)
	if err != nil {
		log.Printf("error uploading log file %s to s3: %v", oldLogFileName, err)
	}

	dbBackupFileName := fmt.Sprintf("guests_%s.db", ydayDate)
	dbBackupFilePath := fmt.Sprintf("/tmp/%s", dbBackupFileName)
	err = backup.BackupDatabaseLocally(guestsDBFilePath, dbBackupFilePath)
	if err != nil {
		log.Printf("error creating db backup file %s: %v", dbBackupFileName, err)
	}

	dbBackupS3FilePath := fmt.Sprintf("backup_dbs/%s", dbBackupFileName)
	err = s3Uploader.UploadFile(dbBackupFilePath, dbBackupS3FilePath)
	if err != nil {
		log.Printf("error uploading db backup file %s to s3: %v", dbBackupS3FilePath, err)
	}

	err = os.Remove(dbBackupFilePath)
	if err != nil {
		log.Printf("error removing db backup file %s locally: %v", dbBackupFilePath, err)
	}

	return nil
}
