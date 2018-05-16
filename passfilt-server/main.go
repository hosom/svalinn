package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	socketName = "/var/run/passfilt/passfilt.socket"
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
		fmt.Fprint(w, "OK")
		a.allowed++
		return
	}

	// failed password evaluations receive a 403 response
	http.Error(w, "False", http.StatusForbidden)
	a.rejected++
	return
}

func main() {
	fmt.Println("Starting password filter server...")
	banlist := flag.String("banlist", "./banlist.txt", "file path to banlist file")
	flag.Parse()

	if _, err := os.Stat(socketName); err == nil {
		// socket exists, clean it up
		fmt.Println("unix socket file already exists, cleaning up...")
		if err := os.Remove(socketName); err != nil {
			fmt.Println("Failed to remove stale socket file exiting.")
			os.Exit(1)
		}
	}

	listener, err := net.Listen("unix", socketName)
	if err != nil {
		log.Fatal(err)
	}

	passfilt := newAPI(*banlist)
	servMux := http.NewServeMux()
	servMux.Handle("/", passfilt)

	serv := &http.Server{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  5 * time.Second,
		Handler:      servMux,
	}

	// simple signal handler to cleanup unix socket file
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		fmt.Println("Received shutdown signal: ", sig)
		fmt.Println("Cleaning up unix socket...")
		serv.Close()
		os.Exit(0)
	}()

	if err := serv.Serve(listener); err != nil {
		fmt.Println(err)
	}
}
