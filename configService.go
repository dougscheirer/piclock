package main

import (
	"crypto/subtle"
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func init() {
	wg.Add(1)
}

// TODO: figure this out
type configResponse struct {
	Response string  `json:"response"`
	Error    error   `json:"error"`
	Alarms   []alarm `json:"alarms"`
}

type configSvcMsg struct {
	secret string
}

type myHandler struct {
	rt     runtimeConfig
	secret string
	user   string
	realm  string
}

func NewHandler(rt runtimeConfig) myHandler {
	return myHandler{
		rt:     rt,
		secret: rt.clock.Now().String(),
		user:   "piclock",
		realm:  "piclock",
	}
}

// BasicAuth binds to a object instance, and without accessors it
// will bind the string values instead of references
func (m *myHandler) getUser() string {
	return m.user
}

func (m *myHandler) getSecret() string {
	return m.secret
}

func (m *myHandler) getRealm() string {
	return m.realm
}

func (m *myHandler) BasicAuth(next http.Handler) http.Handler {
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

func (m *myHandler) getStatus() configResponse {
	// run a getAlarmsFromService
	alarms, err := getAlarmsFromService(m.rt)
	if err != nil {
		return configResponse{Response: "BAD", Error: err}
	} else {
		// return the alarms list too
		return configResponse{Response: "OK", Alarms: alarms}
	}
}

func writeAnswer(w http.ResponseWriter, cr configResponse) {
	output, _ := json.Marshal(cr)
	w.Write(output)
	w.WriteHeader(200)
}

func (m *myHandler) apiHandler(w http.ResponseWriter, r *http.Request) {
	// parse the command
	log.Printf("%s", r.URL)

	vars := mux.Vars(r)
	switch vars["cmd"] {
	case "status":
		writeAnswer(w, m.getStatus())
		return
	case "oauth":
	}

	// default is to return (?500))
	w.WriteHeader(500)
	w.Write([]byte("Error\n"))
}

func (m *myHandler) rootHandler(w http.ResponseWriter, r *http.Request) {
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
