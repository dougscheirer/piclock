package main

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"golang.org/x/net/context"
)

type httpConfigService struct {
	srv     http.Server
	handler *myHandler
}

func (h *httpConfigService) launch(handler *myHandler, addr string) {
	h.handler = handler
	// start a web server that handles JSON and static content
	r := mux.NewRouter()

	// start a web server that handles JSON and static content
	// auth middleware
	r.Use(handler.BasicAuth)
	// static server
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
	// api server
	r.HandleFunc("/api/{cmd}", handler.apiHandler)
	// root handler
	r.HandleFunc("/", handler.rootHandler)
	// give the mux to http
	http.Handle("/", r)

	srv := &http.Server{Addr: ":8080"}

	// add to the wg
	wg.Add(1)

	// launch the server
	go func() {
		defer wg.Done()
		log.Println("starting config service http server")
		err := srv.ListenAndServe()
		log.Print(err)
		log.Print("Exiting config service")
	}()
}

func (h *httpConfigService) stop() {
	h.srv.Shutdown(context.Background())
}

func (h *httpConfigService) setSecret(s string) {
	h.handler.secret = s
}
