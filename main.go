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
		Url:               envVars["URL"],
		PartnerOne:        envVars["PARTNER_ONE"],
		PartnerTwo:        envVars["PARTNER_TWO"],
		Date:              envVars["DATE"],
		Venue:             envVars["VENUE"],
		TimeStart:         envVars["TIME_START"],
		BankName:          envVars["BANK_NAME"],
		BankAccountName:   envVars["BANK_ACCOUNT_NAME"],
		BankSortCode:      envVars["BANK_SORT_CODE"],
		BankAccountNumber: envVars["BANK_ACCOUNT_NUMBER"],
		FooterMessage:     template.HTML(envVars["FOOTER_MESSAGE"]),
	}

	c := controllers.NewController(tpl, log.Default(), &viewData)

	http.HandleFunc("/", c.Index)
	http.HandleFunc("/rsvp", c.RSVP)
	http.HandleFunc("/invitee-details", c.InviteeDetails)
	http.HandleFunc("/change-details", c.ChangeDetails)
	http.HandleFunc("/change-attendance-response", c.ChangeAttendanceResponse)
	http.Handle("/favicon.ico", http.NotFoundHandler())
	http.ListenAndServe(":8080", nil)
}
