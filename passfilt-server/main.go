package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// perform basic normalization of input files
func loadBanlist(fPath string, m *sync.Map) {
	f, err := os.Open(fPath)
	if err != nil {
		log.Fatal(err)
	}

	bannedCount := 0

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		bannedPassword := scanner.Text()
		bannedPassword = strings.ToLower(bannedPassword)
		bannedPassword = strings.TrimRight(bannedPassword, "\n")

		m.Store(bannedPassword, true)
		bannedCount++
	}

	fmt.Printf("Loaded %d items into password banlist.\n", bannedCount)
}

type api struct {
	rejected int
	allowed  int
	// sync.Map provides a goroutine safe map, so no ugly mutex
	// handling here. (requires golang 1.9+)
	banlist sync.Map
}

// fPath is the path from which to load the line delimited banlist
func newAPI(bannedPasswordsFile string) *api {

	a := &api{0, 0, sync.Map{}}

	loadBanlist(bannedPasswordsFile, &a.banlist)

	return a
}

func (a *api) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Received request for password check.")
	user, pass, ok := r.BasicAuth()

	// specify that authentication is required for this server
	if ok == false {
		// any client that sends a request without an authenticate
		// header is instructed to do so with a 401 response
		w.Header().Add("WWW-Authenticate", "Basic")
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	if checkpass(user, pass, &a.banlist) {
		// successful password evaluations receive a 200 response
		fmt.Printf("Permitted password change for user %s", user)
		fmt.Fprint(w, "OK")
		a.allowed++
		return
	}

	// failed password evaluations receive a 403 response
	fmt.Printf("Password %s failed to meet password requirements.", pass)
	http.Error(w, "False", http.StatusForbidden)
	a.rejected++
	return
}

func main() {
	fmt.Println("Starting password filter server...")

	port := flag.String("port", "443", "port to bind the passfilt server to")
	cert := flag.String("cert", "./cert", "file path to certificate file")
	key := flag.String("key", "./key", "file path to key file")
	banlist := flag.String("banlist", "./banlist.txt", "file path to banlist file")

	flag.Parse()

	passfilt := newAPI(*banlist)

	servMux := http.NewServeMux()
	servMux.Handle("/", passfilt)

	serv := &http.Server{
		Addr:         fmt.Sprintf(":%s", *port),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  5 * time.Second,
		Handler:      servMux,
	}

	log.Fatal(serv.ListenAndServeTLS(*cert, *key))
}
