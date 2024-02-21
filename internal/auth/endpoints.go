package auth

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/gorilla/securecookie"
	"golang.org/x/oauth2"
)

type AuthEndpoints struct {
	sc           *securecookie.SecureCookie
	oauth2Config *oauth2.Config
}

func NewAuthEndpoints(sc *securecookie.SecureCookie, oauth2Config *oauth2.Config) AuthEndpoints {
	return AuthEndpoints{
		sc:           sc,
		oauth2Config: oauth2Config,
	}
}

func (endpoints *AuthEndpoints) login(w http.ResponseWriter, r *http.Request) {
	var tokenBytes [255]byte
	if _, err := rand.Read(tokenBytes[:]); err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	state := hex.EncodeToString(tokenBytes[:])

	http.SetCookie(w, &http.Cookie{
		Name:     "state",
		Value:    state,
		HttpOnly: true,
	})

	http.Redirect(w, r, endpoints.oauth2Config.AuthCodeURL(state), http.StatusTemporaryRedirect)
}

func (endpoints *AuthEndpoints) callback(w http.ResponseWriter, r *http.Request) {
	state := r.FormValue("state")
	storedState, err := r.Cookie("state")
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if storedState.Value != state {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	_, err = endpoints.oauth2Config.Exchange(r.Context(), r.FormValue("code"), oauth2.SetAuthURLParam("user_info", ""))
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	value := map[string]string{
		"username": "bar",
	}
	if encoded, err := endpoints.sc.Encode("session", value); err == nil {
		cookie := &http.Cookie{
			Name:     "session",
			Value:    encoded,
			Path:     "/",
			Secure:   true,
			HttpOnly: true,
		}
		http.SetCookie(w, cookie)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (endpoints *AuthEndpoints) GetRouter() *chi.Mux {
	r := chi.NewRouter()
	r.Get("/", endpoints.login)
	r.Get("/callback", endpoints.callback)
	return r
}
