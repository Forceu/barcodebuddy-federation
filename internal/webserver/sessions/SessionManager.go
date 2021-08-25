package sessionmanager

import (
	"BarcodeServer/internal/configuration"
	"BarcodeServer/internal/helper"
	models "BarcodeServer/internal/webserver/sessions/model"
	"net/http"
	"time"
)

// If no login occurred during this time, the admin session will be deleted. Default 30 days
const cookieLifeAdmin = 60 * 24 * time.Hour

// IsValidSession checks if the user is submitting a valid session token
// If valid session is found, useSession will be called
// Returns true if authenticated, otherwise false
func IsValidSession(w http.ResponseWriter, r *http.Request) bool {
	cookie, err := r.Cookie("session_token")
	if err == nil {
		sessionString := cookie.Value
		if sessionString != "" {
			sessions := configuration.GetSessions()
			defer configuration.UnlockSession()
			_, ok := (*sessions)[sessionString]
			if ok {
				return useSession(w, sessionString, sessions)
			}
		}
	}
	return false
}

// useSession checks if a session is still valid. It Changes the session string
// if it has // been used for more than an hour to limit session hijacking
// Returns true if session is still valid
// Returns false if session is invalid (and deletes it)
func useSession(w http.ResponseWriter, sessionString string, sessions *map[string]models.Session) bool {
	session := (*sessions)[sessionString]
	if session.ValidUntil < time.Now().Unix() {
		delete(*sessions, sessionString)
		return false
	}
	if session.RenewAt < time.Now().Unix() {
		CreateSession(w, sessions)
		delete(*sessions, sessionString)
	}
	return true
}

// CreateSession creates a new session - called after login with correct username / password
func CreateSession(w http.ResponseWriter, sessions *map[string]models.Session) {
	if sessions == nil {
		sessions = configuration.GetSessions()
		defer configuration.SaveSessions()
	}
	sessionString := helper.GenerateRandomString(60)
	(*sessions)[sessionString] = models.Session{
		RenewAt:    time.Now().Add(time.Hour).Unix(),
		ValidUntil: time.Now().Add(cookieLifeAdmin).Unix(),
	}
	writeSessionCookie(w, sessionString, time.Now().Add(cookieLifeAdmin))
}

// LogoutSession logs out user and deletes session
func LogoutSession(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_token")
	if err == nil {
		sessions := configuration.GetSessions()
		delete(*sessions, cookie.Value)
		configuration.SaveSessions()
	}
	writeSessionCookie(w, "", time.Now())
}

// Writes session cookie to browser
func writeSessionCookie(w http.ResponseWriter, sessionString string, expiry time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:    "session_token",
		Value:   sessionString,
		Expires: expiry,
	})
}
