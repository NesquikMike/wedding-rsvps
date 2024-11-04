package controllers

import (
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/nesquikmike/wedding-rsvps/internal/cookies"
	"github.com/nesquikmike/wedding-rsvps/internal/database"
	"github.com/nesquikmike/wedding-rsvps/internal/models"
)

type Controller struct {
	isProd          bool
	tpl             *template.Template
	guestStore      database.GuestStore
	logger          *log.Logger
	viewData        *models.ViewData
	secretCookieKey []byte
	apiKey          string
	s3AssetsBucket  string
}

func NewController(isProd bool, t *template.Template, guestStore database.GuestStore, logger *log.Logger, viewData *models.ViewData, secretCookieKey []byte, apiKey, s3AssetsBucket string) *Controller {
	return &Controller{
		isProd:          isProd,
		tpl:             t,
		guestStore:      guestStore,
		logger:          logger,
		viewData:        viewData,
		secretCookieKey: secretCookieKey,
		apiKey:          apiKey,
		s3AssetsBucket:  s3AssetsBucket,
	}
}

var ErrInvalidGuest error = errors.New("guestCode is invalid")

func (c Controller) RSVP(w http.ResponseWriter, req *http.Request) {
	var guest *models.Guest
	guest, err := c.getGuestFromCookie(w, req)
	if err != nil {
		if err == http.ErrNoCookie || err == ErrInvalidGuest {
			guestCode := req.FormValue("guest-code")
			re := regexp.MustCompile(`^[A-Z][a-z]+-[A-Za-z0-9]+$`)
			if len(guestCode) != 12 || !re.MatchString(guestCode) {
				c.logger.Printf("invalid code %s was used\n", guestCode)
				invalidGuestCookie := cookies.GenerateCookie(cookies.SessionTokenName, models.InvalidGuestKey, c.isProd)
				if err := cookies.WriteEncrypted(w, invalidGuestCookie, c.secretCookieKey); err != nil {
					c.logger.Printf("could not write invalid guest cookie: %v\n", err)
				}
				http.Redirect(w, req, "/", http.StatusFound)
				return
			}

			i, err := c.guestStore.GetGuest(guestCode)
			if i == nil || guestCode == "" {
				c.logger.Printf("invalid code %s was used\n", guestCode)
				invalidGuestCookie := cookies.GenerateCookie(cookies.SessionTokenName, models.InvalidGuestKey, c.isProd)
				if err := cookies.WriteEncrypted(w, invalidGuestCookie, c.secretCookieKey); err != nil {
					c.logger.Printf("could not write invalid guest cookie: %v\n", err)
				}
				http.Redirect(w, req, "/", http.StatusFound)
				return
			}
			if err != nil {
				c.logger.Printf("could not get guest: %v\n", err)
				w.WriteHeader(http.StatusBadRequest)
				http.Redirect(w, req, "/error", http.StatusFound)
				return
			}

			c.viewData.Guest = i
			guest = i
		} else {
			c.logger.Println(err)
			http.Redirect(w, req, "/error", http.StatusFound)
			return
		}
	}

	guestCookie := cookies.GenerateCookie(cookies.SessionTokenName, guest.Code, c.isProd)
	if err := cookies.WriteEncrypted(w, guestCookie, c.secretCookieKey); err != nil {
		c.logger.Printf("for guest %v could not write guest cookie: %v\n", guest.Code, err)
	}

	attendance := req.FormValue("attendance")
	if attendance == "true" {
		c.guestStore.UpdateGuestAttendance(guest.Code, true, guest.FormCompleted)
	} else {
		c.guestStore.UpdateGuestAttendance(guest.Code, false, true)
	}

	http.Redirect(w, req, "/", http.StatusFound)
}

func (c Controller) getGuestFromCookie(w http.ResponseWriter, req *http.Request) (*models.Guest, error) {
	var guest *models.Guest

	guestCode, err := cookies.ReadEncrypted(req, cookies.SessionTokenName, c.secretCookieKey)
	if err != nil {
		return &models.InvalidGuest, err
	}

	// Stop an unneccessary read of the DB
	if guestCode == models.InvalidGuestKey {
		return &models.InvalidGuest, ErrInvalidGuest
	}

	guest, err = c.guestStore.GetGuest(guestCode)
	if err != nil {
		return nil, err
	} else if guest == nil {
		return &models.InvalidGuest, fmt.Errorf("guestCode %s: %w", guestCode, ErrInvalidGuest)
	}

	return guest, nil
}

func (c Controller) GuestDetails(w http.ResponseWriter, req *http.Request) {
	guest, err := c.getGuestFromCookie(w, req)
	if err != nil || guest == nil {
		if err == http.ErrNoCookie || errors.Unwrap(err) == ErrInvalidGuest {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		c.logger.Printf("could not get guest: %v\n", err)
		w.WriteHeader(http.StatusBadRequest)
		http.Redirect(w, req, "/error", http.StatusFound)
		return
	}

	detailsAllValid := true

	email := req.FormValue("email")
	if !strings.Contains(email, "@") || !strings.Contains(email, ".") || len(email) < 6 {
		c.logger.Println(fmt.Sprintf("email %s for guestCode %s is invalid", email, guest.Code))
		if err := c.guestStore.UpdateSessionInvalidEmail(guest.Code, true); err != nil {
			c.logger.Printf("could not update session %s that email is invalid: %v", guest.Code, err)
		}
		if err := c.guestStore.UpdateGuestInvalidDetails(guest.Code, true); err != nil {
			c.logger.Printf("could not update that guest %s details are invalid: %v", guest.Code, err)
		}
		detailsAllValid = false
	} else {
		if err := c.guestStore.UpdateSessionInvalidEmail(guest.Code, false); err != nil {
			c.logger.Printf("could not update session %s that email is valid: %v", guest.Code, err)
		}
		if err := c.guestStore.UpdateGuestEmail(guest.Code, email); err != nil {
			c.logger.Printf("could not update guest %s email: %v", guest.Code, err)
		}
	}

	phoneNumber := req.FormValue("phone-number")
	rePhoneNumber := regexp.MustCompile(`^[0-9+][0-9]+$`)
	if !rePhoneNumber.MatchString(phoneNumber) {
		c.logger.Println(fmt.Sprintf("phoneNumber %s for guestCode %s is invalid", phoneNumber, guest.Code))
		if err := c.guestStore.UpdateSessionInvalidPhoneNumber(guest.Code, true); err != nil {
			c.logger.Printf("could not update session %s that phone number is invalid: %v", guest.Code, err)
		}
		if err := c.guestStore.UpdateGuestInvalidDetails(guest.Code, true); err != nil {
			c.logger.Printf("could not update that guest %s details are invalid: %v", guest.Code, err)
		}
		detailsAllValid = false
	} else {
		if err := c.guestStore.UpdateSessionInvalidPhoneNumber(guest.Code, false); err != nil {
			c.logger.Printf("could not update session %s that phone number is valid: %v", guest.Code, err)
		}
		if err := c.guestStore.UpdateGuestPhoneNumber(guest.Code, phoneNumber); err != nil {
			c.logger.Printf("could not update guest %s phone number: %v", guest.Code, err)
		}
	}

	mealChoice := req.FormValue("meal-choice")
	if err := c.guestStore.UpdateGuestMealChoice(guest.Code, mealChoice); err != nil {
		c.logger.Printf("could not update guest %s meal choice: %v", guest.Code, err)
	}


	// Normalize the input
	dietaryRequirements := strings.ReplaceAll(req.FormValue("dietary-requirements"), "\n", " ")
	dietaryRequirements = strings.TrimSpace(dietaryRequirements)
	reDietaryRequirements := regexp.MustCompile(`^(?:(?:[A-Za-z'.,!\"#&()\-£$\d*?/~@\[\]\{\}=+_^%|]{1,100})(?:\s+|$|\.))*(?:[A-Za-z'.,!\"#&()\-£$\d*?/~@\[\]\{\}=+_^%|]{1,100})$`)
	dietaryRequirementsValid := true
	switch {
		case len(dietaryRequirements) > 500:
		dietaryRequirementsValid = false
		case len(dietaryRequirements) > 0 && !reDietaryRequirements.MatchString(dietaryRequirements):
		dietaryRequirementsValid = false
	}
	if !dietaryRequirementsValid {
		c.logger.Println(fmt.Sprintf("dietaryRequirements %s for guestCode %s is invalid", dietaryRequirements, guest.Code))
		if err := c.guestStore.UpdateSessionInvalidDietaryRequirements(guest.Code, true); err != nil {
			c.logger.Printf("could not update session %s that dietary requirements are invalid: %v", guest.Code, err)
		}
		if err := c.guestStore.UpdateGuestInvalidDetails(guest.Code, true); err != nil {
			c.logger.Printf("could not update that guest %s details are invalid: %v", guest.Code, err)
		}
		detailsAllValid = false
	} else {
		if err := c.guestStore.UpdateSessionInvalidDietaryRequirements(guest.Code, false); err != nil {
			c.logger.Printf("could not update session %s that dietary requirements are valid: %v", guest.Code, err)
		}
		if err := c.guestStore.UpdateGuestDietaryRequirements(guest.Code, dietaryRequirements); err != nil {
			c.logger.Printf("could not update guest %s dietary requirements: %v", guest.Code, err)
		}
	}

	if !detailsAllValid {
			http.Redirect(w, req, "/", http.StatusFound)
			return
	}

	if err := c.guestStore.UpdateGuestDetailsProvidedSuccessfully(guest.Code); err != nil {
		c.logger.Printf("could not update guest %s details: %v", guest.Code, err)
	}

	http.Redirect(w, req, "/", http.StatusFound)
}

func (c Controller) ChangeDetails(w http.ResponseWriter, req *http.Request) {
	guest, err := c.getGuestFromCookie(w, req)
	if err != nil {
		if err == http.ErrNoCookie || errors.Unwrap(err) == ErrInvalidGuest {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		c.logger.Printf("could not get guest: %v\n", err)
		w.WriteHeader(http.StatusBadRequest)
		http.Redirect(w, req, "/error", http.StatusFound)
		return
	}
	guestCookie := cookies.GenerateCookie(cookies.SessionTokenName, guest.Code, c.isProd)
	if err := cookies.WriteEncrypted(w, guestCookie, c.secretCookieKey); err != nil {
		c.logger.Printf("for guest %v could not write guest cookie: %v\n", guest.Code, err)
	}

	sessionData, err := c.guestStore.GetSessionData(guest.Code)
	if err != nil {
		c.logger.Printf("for guest %v could not get session data: %v\n", guest.Code, err)
	}

	c.viewData.Guest = guest
	c.viewData.SessionData = sessionData
	c.guestStore.UpdatePageVisit(guest.ID, "change-details")

	c.tpl.ExecuteTemplate(w, "guest_details.gohtml", c.viewData)
}

func (c Controller) ChangeAttendanceResponse(w http.ResponseWriter, req *http.Request) {
	guest, err := c.getGuestFromCookie(w, req)
	if err != nil {
		if err == http.ErrNoCookie || errors.Unwrap(err) == ErrInvalidGuest {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		c.logger.Printf("could not get guest: %v\n", err)
		w.WriteHeader(http.StatusBadRequest)
		http.Redirect(w, req, "/error", http.StatusFound)
		return
	}
	guestCookie := cookies.GenerateCookie(cookies.SessionTokenName, guest.Code, c.isProd)
	if err := cookies.WriteEncrypted(w, guestCookie, c.secretCookieKey); err != nil {
		c.logger.Printf("for guest %v could not write guest cookie: %v\n", guest.Code, err)
	}
	c.viewData.Guest = guest
	c.guestStore.UpdatePageVisit(guest.ID, "change-attendance-response")

	c.tpl.ExecuteTemplate(w, "change_attendance_response.gohtml", c.viewData)
}

func (c Controller) Index(w http.ResponseWriter, req *http.Request) {
	guest, err := c.getGuestFromCookie(w, req)
	if guest == nil || guest.Code == models.InvalidGuestKey {
		c.viewData.Guest = nil
	}
	if err != nil {
		switch {
		case err == http.ErrNoCookie:
			c.logger.Printf("new visitor")
			c.tpl.ExecuteTemplate(w, "index.gohtml", c.viewData)
			return
		case err == ErrInvalidGuest || errors.Unwrap(err) == ErrInvalidGuest:
			c.logger.Printf("invalid guest code")
			c.tpl.ExecuteTemplate(w, "invalid_guest.gohtml", c.viewData)
			return
		default:
			c.logger.Printf("could not get guest: %v\n", err)
			w.WriteHeader(http.StatusBadRequest)
			http.Redirect(w, req, "/error", http.StatusFound)
			return
		}
	}

	if guest != nil {
		guestCookie := cookies.GenerateCookie(cookies.SessionTokenName, guest.Code, c.isProd)
		if err := cookies.WriteEncrypted(w, guestCookie, c.secretCookieKey); err != nil {
			c.logger.Printf("for guest %v could not write guest cookie: %v\n", guest.Code, err)
		}
		c.viewData.Guest = guest
		c.logger.Printf("guest %s hit index", guest.Code)

		switch {
		case !guest.FormStarted:
			blankCookie := cookies.GenerateBlankCookie(cookies.SessionTokenName, c.isProd)
			http.SetCookie(w, blankCookie)
			c.viewData.Guest = nil

			c.tpl.ExecuteTemplate(w, "index.gohtml", c.viewData)
			return
		case !guest.Attendance:
			c.guestStore.UpdatePageVisit(guest.ID, "guest-declined")
			c.tpl.ExecuteTemplate(w, "guest_declined.gohtml", c.viewData)
			return
		case guest.InvalidDetails:
			sessionData, err := c.guestStore.GetSessionData(guest.Code)
			if err != nil {
				c.logger.Printf("for guest %v could not get session data: %v\n", guest.Code, err)
			}
			c.viewData.SessionData = sessionData
			c.guestStore.UpdatePageVisit(guest.ID, "guest-details")
			c.tpl.ExecuteTemplate(w, "invalid_details.gohtml", c.viewData)
			return
		case !guest.DetailsProvided:
			c.guestStore.UpdatePageVisit(guest.ID, "guest-details")
			c.tpl.ExecuteTemplate(w, "guest_details.gohtml", c.viewData)
			return
		default:
			c.guestStore.UpdatePageVisit(guest.ID, "guest-accepted")
			c.tpl.ExecuteTemplate(w, "guest_accepted.gohtml", c.viewData)
			return
		}
	}

}

func (c Controller) ResetGuest(w http.ResponseWriter, req *http.Request) {
	blankCookie := cookies.GenerateBlankCookie(cookies.SessionTokenName, c.isProd)
	http.SetCookie(w, blankCookie)
	c.viewData.Guest = nil

	http.Redirect(w, req, "/", http.StatusFound)
}

func (c Controller) StaticHandler(w http.ResponseWriter, req *http.Request) {
	objectKey := strings.TrimPrefix(req.URL.Path, "/assets/")
	var s3URL string
	if c.isProd {
		s3URL = fmt.Sprintf("https://s3.eu-west-2.amazonaws.com/%s/%s", c.s3AssetsBucket, objectKey)
	} else {
		s3URL = "http://localhost:4566/" + c.s3AssetsBucket + objectKey
	}

	w.Header().Set("Cache-Control", "public, max-age=86400") // Cache for 1 day
	w.Header().Set("Expires", time.Now().Add(24*time.Hour).Format(http.TimeFormat))

	http.Redirect(w, req, s3URL, http.StatusFound)
}
