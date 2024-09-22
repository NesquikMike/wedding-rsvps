package main

import (
	"bufio"
	"fmt"
	"github.com/nesquikmike/wedding-rsvps/controllers"
	"github.com/nesquikmike/wedding-rsvps/models"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
)

var tpl *template.Template

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

	c := controllers.NewController(tpl, log.Default(), &viewData)

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
