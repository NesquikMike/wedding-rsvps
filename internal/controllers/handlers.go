package controllers

import (
	"errors"
	"fmt"
	"github.com/nesquikmike/wedding-rsvps/internal/cookies"
	"github.com/nesquikmike/wedding-rsvps/internal/models"
	"github.com/nesquikmike/wedding-rsvps/internal/database"
	"html/template"
	"log"
	"net/http"
	"regexp"
	"strings"
)

type Controller struct {
	isProd          bool
	tpl             *template.Template
	inviteeStore    database.InviteeStore
	logger          *log.Logger
	viewData        *models.ViewData
	secretCookieKey []byte
}

func NewController(isProd bool, t *template.Template, inviteeStore database.InviteeStore, logger *log.Logger, viewData *models.ViewData, secretCookieKey []byte) *Controller {
	return &Controller{
		isProd:          isProd,
		tpl:             t,
		inviteeStore:    inviteeStore,
		logger:          logger,
		viewData:        viewData,
		secretCookieKey: secretCookieKey,
	}
}

var InvalidInviteeError error = errors.New("inviteeCode is invalid")

func (c Controller) RSVP(w http.ResponseWriter, req *http.Request) {
	var invitee *models.Invitee
	invitee, err := c.getInvitee(w, req)
	if err != nil {
		if err == http.ErrNoCookie || err == InvalidInviteeError {
			inviteeCode := req.FormValue("invitee-code")
			re := regexp.MustCompile(`^[A-Z][a-z]+-[A-Za-z0-9]+$`)
			if len(inviteeCode) != 12 || !re.MatchString(inviteeCode) {
				invalidInviteeCookie := cookies.GenerateCookie(cookies.SessionTokenName, models.InvalidInviteeKey, c.isProd)
				if err := cookies.WriteEncrypted(w, invalidInviteeCookie, c.secretCookieKey); err != nil {
					c.logger.Printf("could not write invalid invitee cookie: %v\n", err)
				}
				http.Redirect(w, req, "/", http.StatusFound)
				return
			}

			i, err := c.inviteeStore.GetInvitee(inviteeCode)
			if i == nil || inviteeCode == "" {
				invalidInviteeCookie := cookies.GenerateCookie(cookies.SessionTokenName, models.InvalidInviteeKey, c.isProd)
				if err := cookies.WriteEncrypted(w, invalidInviteeCookie, c.secretCookieKey); err != nil {
					c.logger.Printf("could not write invalid invitee cookie: %v\n", err)
				}
				http.Redirect(w, req, "/", http.StatusFound)
				return
			}
			if err != nil {
				c.logger.Printf("could not get invitee: %v\n", err)
				w.WriteHeader(http.StatusBadRequest)
				http.Redirect(w, req, "/error", http.StatusFound)
				return
			}

			c.viewData.Invitee = i
			invitee = i
		} else {
			c.logger.Println(err)
			http.Redirect(w, req, "/error", http.StatusFound)
			return
		}
	}

	inviteeCookie := cookies.GenerateCookie(cookies.SessionTokenName, invitee.Code, c.isProd)
	if err := cookies.WriteEncrypted(w, inviteeCookie, c.secretCookieKey); err != nil {
		c.logger.Printf("for invitee %v could not write invitee cookie: %v\n", invitee.Code, err)
	}

	attendance := req.FormValue("attendance")
	if attendance == "true" {
		c.inviteeStore.UpdateInviteeAttendance(invitee.Code, true, invitee.FormCompleted)
	} else {
		c.inviteeStore.UpdateInviteeAttendance(invitee.Code, false, true)
	}

	http.Redirect(w, req, "/", http.StatusFound)
}

func (c Controller) getInvitee(w http.ResponseWriter, req *http.Request) (*models.Invitee, error) {
	var invitee *models.Invitee

	inviteeCode, err := cookies.ReadEncrypted(req, cookies.SessionTokenName, c.secretCookieKey)
	if err != nil {
		return &models.InvalidInvitee, err
	}

	// Stop an unneccessary read of the DB
	if inviteeCode == models.InvalidInviteeKey {
		return &models.InvalidInvitee, InvalidInviteeError
	}

	invitee, err = c.inviteeStore.GetInvitee(inviteeCode)
	if err != nil {
		return nil, err
	} else if invitee == nil {
		return &models.InvalidInvitee, fmt.Errorf("inviteeCode %s: %w", inviteeCode, InvalidInviteeError)
	}

	return invitee, nil
}

func (c Controller) InviteeDetails(w http.ResponseWriter, req *http.Request) {
	invitee, err := c.getInvitee(w, req)
	if err != nil || invitee == nil {
		if err == http.ErrNoCookie || errors.Unwrap(err) == InvalidInviteeError {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		c.logger.Printf("could not get invitee: %v\n", err)
		w.WriteHeader(http.StatusBadRequest)
		http.Redirect(w, req, "/error", http.StatusFound)
		return
	}

	email := req.FormValue("email")
	if !strings.Contains(email, "@") || !strings.Contains(email, ".") || len(email) < 6 {
		c.logger.Println(fmt.Sprintf("email %s for inviteeCode %s is invalid", email, invitee.Code))
		c.inviteeStore.UpdateInviteeInvalidDetails(invitee.Code, true)
		w.WriteHeader(http.StatusBadRequest)
		http.Redirect(w, req, "/", http.StatusFound)
		return
	}

	phoneNumber := req.FormValue("phone-number")
	re := regexp.MustCompile(`^[0-9+][0-9]+$`)
	if !re.MatchString(phoneNumber) {
		c.logger.Println(fmt.Sprintf("phoneNumber %s for inviteeCode %s is invalid", phoneNumber, invitee.Code))
		c.inviteeStore.UpdateInviteeInvalidDetails(invitee.Code, true)
		w.WriteHeader(http.StatusBadRequest)
		http.Redirect(w, req, "/", http.StatusFound)

		return
	}
	dietaryRequirements := req.FormValue("dietary-requirements")

	c.inviteeStore.UpdateInviteeDetails(invitee.Code, email, phoneNumber, dietaryRequirements)

	http.Redirect(w, req, "/", http.StatusFound)
}

func (c Controller) ChangeDetails(w http.ResponseWriter, req *http.Request) {
	invitee, err := c.getInvitee(w, req)
	if err != nil {
		if err == http.ErrNoCookie || errors.Unwrap(err) == InvalidInviteeError {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		c.logger.Printf("could not get invitee: %v\n", err)
		w.WriteHeader(http.StatusBadRequest)
		http.Redirect(w, req, "/error", http.StatusFound)
		return
	}
	inviteeCookie := cookies.GenerateCookie(cookies.SessionTokenName, invitee.Code, c.isProd)
	if err := cookies.WriteEncrypted(w, inviteeCookie, c.secretCookieKey); err != nil {
		c.logger.Printf("for invitee %v could not write invitee cookie: %v\n", invitee.Code, err)
	}

	c.viewData.Invitee = invitee

	c.tpl.ExecuteTemplate(w, "invitee_details.gohtml", c.viewData)
}

func (c Controller) ChangeAttendanceResponse(w http.ResponseWriter, req *http.Request) {
	invitee, err := c.getInvitee(w, req)
	if err != nil {
		if err == http.ErrNoCookie || errors.Unwrap(err) == InvalidInviteeError {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		c.logger.Printf("could not get invitee: %v\n", err)
		w.WriteHeader(http.StatusBadRequest)
		http.Redirect(w, req, "/error", http.StatusFound)
		return
	}
	inviteeCookie := cookies.GenerateCookie(cookies.SessionTokenName, invitee.Code, c.isProd)
	if err := cookies.WriteEncrypted(w, inviteeCookie, c.secretCookieKey); err != nil {
		c.logger.Printf("for invitee %v could not write invitee cookie: %v\n", invitee.Code, err)
	}

	c.tpl.ExecuteTemplate(w, "change_attendance_response.gohtml", c.viewData)
}

func (c Controller) Index(w http.ResponseWriter, req *http.Request) {
	invitee, err := c.getInvitee(w, req)
	if invitee == nil || invitee.Code == models.InvalidInviteeKey {
		c.viewData.Invitee = nil
	}
	if err != nil {
		switch {
		case err == http.ErrNoCookie:
			c.tpl.ExecuteTemplate(w, "index.gohtml", c.viewData)
			return
		case err == InvalidInviteeError || errors.Unwrap(err) == InvalidInviteeError:
			c.tpl.ExecuteTemplate(w, "invalid_invitee.gohtml", c.viewData)
			return
		default:
			c.logger.Printf("could not get invitee: %v\n", err)
			w.WriteHeader(http.StatusBadRequest)
			http.Redirect(w, req, "/error", http.StatusFound)
			return
		}
	}

	if invitee != nil {
		inviteeCookie := cookies.GenerateCookie(cookies.SessionTokenName, invitee.Code, c.isProd)
		if err := cookies.WriteEncrypted(w, inviteeCookie, c.secretCookieKey); err != nil {
			c.logger.Printf("for invitee %v could not write invitee cookie: %v\n", invitee.Code, err)
		}
		c.viewData.Invitee = invitee

		switch {
		case invitee.Attendance && invitee.DetailsProvided:
			c.tpl.ExecuteTemplate(w, "invitee_accepted.gohtml", c.viewData)
			return
		case invitee.Attendance && invitee.InvalidDetails:
			c.tpl.ExecuteTemplate(w, "invalid_details.gohtml", c.viewData)
			return
		case invitee.Attendance && !invitee.DetailsProvided:
			c.tpl.ExecuteTemplate(w, "invitee_details.gohtml", c.viewData)
			return
		default:
			c.tpl.ExecuteTemplate(w, "invitee_declined.gohtml", c.viewData)
			return
		}
	}

}

func (c Controller) ResetInvitee(w http.ResponseWriter, req *http.Request) {
	blankCookie := cookies.GenerateBlankCookie(cookies.SessionTokenName, c.isProd)
	http.SetCookie(w, blankCookie)
	c.viewData.Invitee = nil

	http.Redirect(w, req, "/", http.StatusFound)
	c.tpl.ExecuteTemplate(w, "index.gohtml", c.viewData)
}
