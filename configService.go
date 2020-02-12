package main

import (
	"crypto/subtle"
	"encoding/json"
	"log"
	"net/http"
)

func init() {
	wg.Add(1)
}

// TODO: figure this out
type configResponse struct {
	Response string  `json:"response"`
	Error    string  `json:"error"`
	Alarms   []alarm `json:"alarms"`
}

type configSvcMsg struct {
	secret string
}

type apiHandler struct {
	rt     runtimeConfig
	secret string
	user   string
	realm  string
}

func NewHandler(rt runtimeConfig) apiHandler {
	return apiHandler{
		rt:     rt,
		secret: rt.clock.Now().String(),
		user:   "piclock",
		realm:  "piclock",
	}
}

// BasicAuth binds to a object instance, and without accessors it
// will bind the string values instead of references
func (m *apiHandler) getUser() string {
	return m.user
}

func (m *apiHandler) getSecret() string {
	return m.secret
}

func (m *apiHandler) getRealm() string {
	return m.realm
}

func (m *apiHandler) BasicAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		// log.Printf("cur: %s / %s, got: %s / %s", m.getUser(), m.getSecret(), user, pass)
		if !ok || subtle.ConstantTimeCompare([]byte(user), []byte(m.user)) != 1 || subtle.ConstantTimeCompare([]byte(pass), []byte(m.secret)) != 1 {
			w.Header().Set("WWW-Authenticate", `Basic realm="`+m.getRealm()+`"`)
			w.WriteHeader(401)
			w.Write([]byte("Unauthorised.\n"))
			return
		} else {
			next.ServeHTTP(w, r)
		}
	})
}

func (m *apiHandler) getStatus() configResponse {
	// run a getAlarmsFromService
	alarms, err := getAlarmsFromService(m.rt)
	if err != nil {
		return configResponse{Response: "BAD", Error: err.Error()}
	} else {
		// return the alarms list too
		return configResponse{Response: "OK", Alarms: alarms}
	}
}

func writeAnswer(w http.ResponseWriter, cr configResponse) {
	output, _ := json.Marshal(cr)
	w.Write(output)
}

func (m *apiHandler) apiSecret(w http.ResponseWriter, r *http.Request) {
	m.apiError(w, r)
}

func (m *apiHandler) apiOauth(w http.ResponseWriter, r *http.Request) {

}

func (m *apiHandler) apiStatus(w http.ResponseWriter, r *http.Request) {
	writeAnswer(w, m.getStatus())
}

func (m *apiHandler) apiError(w http.ResponseWriter, r *http.Request) {
	// default is to return (?500))
	w.WriteHeader(500)
	w.Write([]byte("Error\n"))
}

func (m *apiHandler) rootHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/static/index.html", 301)
}

func runConfigService(rt runtimeConfig) {
	defer wg.Done()

	handler := NewHandler(rt)

	rt.configService.launch(&handler, ":8080")

	log.Println("starting config service comms loop")
	comms := rt.comms

	// comms loop, listen for secrets
	for true {
		select {
		case <-comms.quit:
			log.Printf("quit from config service")
			// stop the server
			rt.configService.stop()
			return
		case msg := <-comms.configSvc:
			// we only accept secret strings
			log.Printf("Got a new secret: %s", msg.secret)
			handler.secret = msg.secret
		default:
			rt.clock.Sleep(dAlarmSleep) // should we just have a default?
		}
	}
}
