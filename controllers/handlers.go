package controllers

import (
	"errors"
	"fmt"
	"github.com/nesquikmike/wedding-rsvps/models"
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
}

type Controller struct {
	tpl          *template.Template
	inviteeStore map[string]*models.Invitee
	logger       *log.Logger
	viewData     *models.ViewData
}

func NewController(t *template.Template, logger *log.Logger, viewData *models.ViewData) *Controller {
	return &Controller{
		tpl:          t,
		inviteeStore: inviteeStore,
		logger:       logger,
		viewData:     viewData,
	}
}

var InvalidInviteeError error = errors.New("inviteeCode is invalid")

func (c Controller) RSVP(w http.ResponseWriter, req *http.Request) {
	var invitee *models.Invitee
	invitee, err := c.getInvitee(w, req)
	if err != nil {
		if err == http.ErrNoCookie {
			inviteeCode := req.FormValue("invitee-code")
			i, ok := c.inviteeStore[inviteeCode]
			if !ok || inviteeCode == "" {
				w.WriteHeader(http.StatusUnauthorized)
				c.tpl.ExecuteTemplate(w, "invalid_invitee.gohtml", c.viewData)
				return
			}

			http.SetCookie(w, &http.Cookie{
				Name:  "session_token",
				Value: inviteeCode,
			})

			c.viewData.Invitee = i
			invitee = i
		} else {
			c.logger.Println(err)
			w.WriteHeader(http.StatusBadRequest)
			c.tpl.ExecuteTemplate(w, "invalid_invitee.gohtml", c.viewData)
			return
		}
	}

	attendance := req.FormValue("attendance")
	if attendance == "true" {
		invitee.Attendance = true

		c.tpl.ExecuteTemplate(w, "invitee_details.gohtml", c.viewData)
		return
	}

	invitee.Attendance = false

	c.tpl.ExecuteTemplate(w, "invitee_declined.gohtml", c.viewData)
}

func (c Controller) getInvitee(w http.ResponseWriter, req *http.Request) (*models.Invitee, error) {
	var invitee *models.Invitee

	cookie, err := req.Cookie("session_token")
	if err != nil {
		return invitee, err
	}
	inviteeCode := cookie.Value

	invitee, ok := c.inviteeStore[inviteeCode]
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return invitee, fmt.Errorf("inviteeCode %s: %w", inviteeCode, InvalidInviteeError)
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
		w.WriteHeader(http.StatusBadRequest)
		c.tpl.ExecuteTemplate(w, "invalid_details.gohtml", c.viewData)
		return
	}

	phoneNumber := req.FormValue("phone-number")
	re := regexp.MustCompile(`^[0-9+][0-9]+$`)
	if !re.MatchString(phoneNumber) {
		c.logger.Println(fmt.Sprintf("phoneNumber %s for inviteeCode %s is invalid", phoneNumber, invitee.Code))
		w.WriteHeader(http.StatusBadRequest)
		c.tpl.ExecuteTemplate(w, "invalid_details.gohtml", c.viewData)
		return
	}
	dietaryRequirements := req.FormValue("dietary-requirements")

	invitee.Email = email
	invitee.PhoneNumber = phoneNumber
	invitee.DietaryRequirements = dietaryRequirements
	invitee.DetailsProvided = true

	c.viewData.Invitee = invitee

	c.tpl.ExecuteTemplate(w, "invitee_accepted.gohtml", c.viewData)
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

	invitee.DetailsProvided = false
	c.viewData.Invitee = invitee

	c.tpl.ExecuteTemplate(w, "invitee_details.gohtml", c.viewData)
}

func (c Controller) ChangeAttendanceResponse(w http.ResponseWriter, req *http.Request) {
	_, err := c.getInvitee(w, req)
	if err != nil {
		c.logger.Println(err)
		if err == http.ErrNoCookie || errors.Unwrap(err) == InvalidInviteeError {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	c.tpl.ExecuteTemplate(w, "change_attendance_response.gohtml", c.viewData)
}

func (c Controller) Index(w http.ResponseWriter, req *http.Request) {
	invitee, err := c.getInvitee(w, req)
	if err != nil {
		if err != http.ErrNoCookie && errors.Unwrap(err) != InvalidInviteeError {
			c.logger.Println(err)
			w.WriteHeader(http.StatusBadRequest)
			c.tpl.ExecuteTemplate(w, "invalid_invitee.gohtml", c.viewData)
			return
		}
	}

	if invitee != nil {
		c.logger.Println(fmt.Sprintf("invitee %s has already used site", invitee.Code))
		c.viewData.Invitee = invitee

		if invitee.Attendance && invitee.DetailsProvided {
			c.tpl.ExecuteTemplate(w, "invitee_accepted.gohtml", c.viewData)
			return
		} else if invitee.Attendance && !invitee.DetailsProvided {
			c.tpl.ExecuteTemplate(w, "invitee_details.gohtml", c.viewData)
			return
		} else {
			c.tpl.ExecuteTemplate(w, "invitee_declined.gohtml", c.viewData)
			return
		}
	}

	c.tpl.ExecuteTemplate(w, "index.gohtml", c.viewData)
}
