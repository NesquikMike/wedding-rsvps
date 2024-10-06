package controllers

import (
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"regexp"
	"strings"

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
}

func NewController(isProd bool, t *template.Template, guestStore database.GuestStore, logger *log.Logger, viewData *models.ViewData, secretCookieKey []byte) *Controller {
	return &Controller{
		isProd:          isProd,
		tpl:             t,
		guestStore:      guestStore,
		logger:          logger,
		viewData:        viewData,
		secretCookieKey: secretCookieKey,
	}
}

var InvalidGuestError error = errors.New("guestCode is invalid")

func (c Controller) RSVP(w http.ResponseWriter, req *http.Request) {
	var guest *models.Guest
	guest, err := c.getGuest(w, req)
	if err != nil {
		if err == http.ErrNoCookie || err == InvalidGuestError {
			guestCode := req.FormValue("guest-code")
			re := regexp.MustCompile(`^[A-Z][a-z]+-[A-Za-z0-9]+$`)
			if len(guestCode) != 12 || !re.MatchString(guestCode) {
				invalidGuestCookie := cookies.GenerateCookie(cookies.SessionTokenName, models.InvalidGuestKey, c.isProd)
				if err := cookies.WriteEncrypted(w, invalidGuestCookie, c.secretCookieKey); err != nil {
					c.logger.Printf("could not write invalid guest cookie: %v\n", err)
				}
				http.Redirect(w, req, "/", http.StatusFound)
				return
			}

			i, err := c.guestStore.GetGuest(guestCode)
			if i == nil || guestCode == "" {
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

func (c Controller) getGuest(w http.ResponseWriter, req *http.Request) (*models.Guest, error) {
	var guest *models.Guest

	guestCode, err := cookies.ReadEncrypted(req, cookies.SessionTokenName, c.secretCookieKey)
	if err != nil {
		return &models.InvalidGuest, err
	}

	// Stop an unneccessary read of the DB
	if guestCode == models.InvalidGuestKey {
		return &models.InvalidGuest, InvalidGuestError
	}

	guest, err = c.guestStore.GetGuest(guestCode)
	if err != nil {
		return nil, err
	} else if guest == nil {
		return &models.InvalidGuest, fmt.Errorf("guestCode %s: %w", guestCode, InvalidGuestError)
	}

	return guest, nil
}

func (c Controller) GuestDetails(w http.ResponseWriter, req *http.Request) {
	guest, err := c.getGuest(w, req)
	if err != nil || guest == nil {
		if err == http.ErrNoCookie || errors.Unwrap(err) == InvalidGuestError {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		c.logger.Printf("could not get guest: %v\n", err)
		w.WriteHeader(http.StatusBadRequest)
		http.Redirect(w, req, "/error", http.StatusFound)
		return
	}

	email := req.FormValue("email")
	if !strings.Contains(email, "@") || !strings.Contains(email, ".") || len(email) < 6 {
		c.logger.Println(fmt.Sprintf("email %s for guestCode %s is invalid", email, guest.Code))
		c.guestStore.UpdateGuestInvalidDetails(guest.Code, true)
		w.WriteHeader(http.StatusBadRequest)
		http.Redirect(w, req, "/", http.StatusFound)
		return
	}

	phoneNumber := req.FormValue("phone-number")
	re := regexp.MustCompile(`^[0-9+][0-9]+$`)
	if !re.MatchString(phoneNumber) {
		c.logger.Println(fmt.Sprintf("phoneNumber %s for guestCode %s is invalid", phoneNumber, guest.Code))
		c.guestStore.UpdateGuestInvalidDetails(guest.Code, true)
		w.WriteHeader(http.StatusBadRequest)
		http.Redirect(w, req, "/", http.StatusFound)

		return
	}
	dietaryRequirements := req.FormValue("dietary-requirements")

	c.guestStore.UpdateGuestDetails(guest.Code, email, phoneNumber, dietaryRequirements)

	http.Redirect(w, req, "/", http.StatusFound)
}

func (c Controller) ChangeDetails(w http.ResponseWriter, req *http.Request) {
	guest, err := c.getGuest(w, req)
	if err != nil {
		if err == http.ErrNoCookie || errors.Unwrap(err) == InvalidGuestError {
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
	c.guestStore.UpdatePageVisit(guest.ID, "change-details")

	c.tpl.ExecuteTemplate(w, "guest_details.gohtml", c.viewData)
}

func (c Controller) ChangeAttendanceResponse(w http.ResponseWriter, req *http.Request) {
	guest, err := c.getGuest(w, req)
	if err != nil {
		if err == http.ErrNoCookie || errors.Unwrap(err) == InvalidGuestError {
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
	guest, err := c.getGuest(w, req)
	if guest == nil || guest.Code == models.InvalidGuestKey {
		c.viewData.Guest = nil
	}
	if err != nil {
		switch {
		case err == http.ErrNoCookie:
			c.tpl.ExecuteTemplate(w, "index.gohtml", c.viewData)
			return
		case err == InvalidGuestError || errors.Unwrap(err) == InvalidGuestError:
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

		switch {
		case !guest.Attendance:
			c.guestStore.UpdatePageVisit(guest.ID, "guest-declined")
			c.tpl.ExecuteTemplate(w, "guest_declined.gohtml", c.viewData)
			return
		case guest.InvalidDetails:
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
	c.tpl.ExecuteTemplate(w, "index.gohtml", c.viewData)
}
