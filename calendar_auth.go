package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path/filepath"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
)

// getClient uses a Context and Config to retrieve a Token
// then generate a Client. It returns the generated Client.
func getClient(ctx context.Context, config *oauth2.Config, prompt bool) *http.Client {
	cacheFile, err := tokenCacheFile()
	if err != nil {
		log.Printf("Unable to get path to cached credential file. %v", err)
	}
	tok, err := tokenFromFile(cacheFile)
	if err != nil {
		if prompt {
			tok = getTokenFromWeb(config)
			saveToken(cacheFile, tok)
		} else {
			// run with -oauth to generate the token
			return nil
		}
	} else {
		if prompt != false {
			log.Println("OAUTH has a valid token in " + cacheFile)
		}
	}

	return config.Client(ctx, tok)
}

// getTokenFromWeb uses Config to request a Token.
// It returns the retrieved Token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var code string
	if _, err := fmt.Scan(&code); err != nil {
		log.Printf("Unable to read authorization code %v", err)
	}

	tok, err := config.Exchange(oauth2.NoContext, code)
	if err != nil {
		log.Printf("Unable to retrieve token from web %v", err)
	}
	return tok
}

// tokenCacheFile generates credential file path/filename.
// It returns the generated credential path/filename.
func tokenCacheFile() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	tokenCacheDir := filepath.Join(usr.HomeDir, ".credentials")
	os.MkdirAll(tokenCacheDir, 0700)
	return filepath.Join(tokenCacheDir,
		url.QueryEscape("piclock.json")), err
}

// tokenFromFile retrieves a Token from a given file path.
// It returns the retrieved Token and any read error encountered.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.OpenFile(file, os.O_RDONLY, 0600)
	if err != nil {
		return nil, err
	}
	t := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(t)
	defer f.Close()
	return t, err
}

// saveToken uses a file path to create a file and store the
// token in it.
func saveToken(file string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", file)
	f, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Printf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func getCalenderService(settings *configSettings, prompt bool) (*calendar.Service, error) {
	// if' we're pretending, skip the check
	if settings.GetBool("cachedAlarms") {
		return nil, nil
	}

	ctx := context.Background()

	b, err := ioutil.ReadFile(settings.GetString("secretPath") + "/client_secret.json")
	if err != nil {
		log.Printf("Unable to read client secret file: %v", err)
		return nil, err
	}

	// If modifying these scopes, delete your previously saved credentials
	// at ~/.credentials/piclock.json
	config, err := google.ConfigFromJSON(b, calendar.CalendarReadonlyScope)
	if err != nil {
		log.Printf("Unable to parse client secret file to config: %v", err)
		return nil, err
	}
	client := getClient(ctx, config, prompt)

	srv, err := calendar.New(client)
	if err != nil {
		log.Printf("Unable to retrieve calendar Client %v", err)
		return nil, err
	}

	return srv, nil
}

// a looping function to get the Oauth key before anything else
func confirmCalendarAuth(settings *configSettings) {
	defer func() {
	}()

	// if we're pretending, skip the check
	// TODO: move this logic into the test framework
	if settings.GetBool("cachedAlarms") {
		return
	}

	_, err := getCalenderService(settings, true)
	if err == nil {
		// success!
		return
	}

	log.Println(err)
}
