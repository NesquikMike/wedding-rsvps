package controllers

import (
	"errors"
	"fmt"
	"github.com/nesquikmike/wedding-rsvps/internal/cookies"
	"github.com/nesquikmike/wedding-rsvps/internal/models"
	"html/template"
	"log"
	"net/http"
	"regexp"
	"strings"
)

var inviteeStore = map[string]*models.Invitee{
	"Maria-1St": &models.Invitee{
		Name: "Maria",
		Code: "Maria-1St",
	},
	"Luis-2nD": &models.Invitee{
		Name: "Luis",
		Code: "Luis-2nD",
	},
	"Beth-3Rd": &models.Invitee{
		Name: "Beth",
		Code: "Beth-3Rd",
	},
}

type Controller struct {
	isProd bool
	tpl             *template.Template
	inviteeStore    map[string]*models.Invitee
	logger          *log.Logger
	viewData        *models.ViewData
	secretCookieKey []byte
}

func NewController(isProd bool, t *template.Template, logger *log.Logger, viewData *models.ViewData, secretCookieKey []byte) *Controller {
	return &Controller{
		isProd: isProd,
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
			i, ok := c.inviteeStore[inviteeCode]
			if !ok || inviteeCode == "" {
				invalidInviteeCookie := cookies.GenerateCookie(cookies.SessionTokenName, models.InvalidInviteeKey, c.isProd)
				if err := cookies.WriteEncrypted(w, invalidInviteeCookie, c.secretCookieKey); err != nil {
					c.logger.Printf("could not write invalid invitee cookie: %v\n", err)
				}
				http.Redirect(w, req, "/", http.StatusFound)
				return
			}

			c.viewData.Invitee = i
			invitee = i
		} else {
			c.logger.Println(err)
			http.Redirect(w, req, "/reset-invitee", http.StatusFound)
			return
		}
	}

	inviteeCookie := cookies.GenerateCookie(cookies.SessionTokenName, invitee.Code, c.isProd)
	if err := cookies.WriteEncrypted(w, inviteeCookie, c.secretCookieKey); err != nil {
		c.logger.Printf("for invitee %v could not write invitee cookie: %v\n", invitee.Code, err)
	}

	attendance := req.FormValue("attendance")
	if attendance == "true" {
		invitee.Attendance = true
	} else {
		invitee.Attendance = false
		invitee.FormCompleted = true
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

	invitee, ok := c.inviteeStore[inviteeCode]
	if !ok {
		return &models.InvalidInvitee, fmt.Errorf("inviteeCode %s: %w", inviteeCode, InvalidInviteeError)
	}

	return invitee, nil
}

func (c Controller) InviteeDetails(w http.ResponseWriter, req *http.Request) {
	invitee, err := c.getInvitee(w, req)
	if err != nil {
		c.logger.Println(err)
		if err == http.ErrNoCookie || errors.Unwrap(err) == InvalidInviteeError {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	email := req.FormValue("email")
	if !strings.Contains(email, "@") || !strings.Contains(email, ".") || len(email) < 6 {
		c.logger.Println(fmt.Sprintf("email %s for inviteeCode %s is invalid", email, invitee.Code))
		invitee.InvalidDetails = true
		w.WriteHeader(http.StatusBadRequest)
		http.Redirect(w, req, "/", http.StatusFound)
		return
	}

	phoneNumber := req.FormValue("phone-number")
	re := regexp.MustCompile(`^[0-9+][0-9]+$`)
	if !re.MatchString(phoneNumber) {
		c.logger.Println(fmt.Sprintf("phoneNumber %s for inviteeCode %s is invalid", phoneNumber, invitee.Code))
		invitee.InvalidDetails = true
		w.WriteHeader(http.StatusBadRequest)
		http.Redirect(w, req, "/", http.StatusFound)

		return
	}
	dietaryRequirements := req.FormValue("dietary-requirements")

	invitee.Email = email
	invitee.PhoneNumber = phoneNumber
	invitee.DietaryRequirements = dietaryRequirements
	invitee.DetailsProvided = true
	invitee.FormCompleted = true

	c.viewData.Invitee = invitee

	http.Redirect(w, req, "/", http.StatusFound)
}

func (c Controller) ChangeDetails(w http.ResponseWriter, req *http.Request) {
	invitee, err := c.getInvitee(w, req)
	if err != nil {
		c.logger.Println(err)
		if err == http.ErrNoCookie || errors.Unwrap(err) == InvalidInviteeError {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
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
		c.logger.Println(err)
		if err == http.ErrNoCookie || errors.Unwrap(err) == InvalidInviteeError {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
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
			c.logger.Println(err)
			w.WriteHeader(http.StatusBadRequest)
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
			invitee.InvalidDetails = false
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
