package main

import (
	"bufio"
	"crypto/sha1"
	"fmt"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	// base URL for the HaveIBeenPwnd API
	pwndAPIBase string = "https://api.pwnedpasswords.com"
	logFmt      string = "User=%s PassOK=%t UsernameInPassword=%t PasswordInBanlist=%t PasswordIsPwnd=%t PasswordEntropy=%g\n"
)

func passContainsUser(u string, p string) bool {
	return strings.Contains(p, u)
}

// isPwnd performs a lookup against the HaveIBeenPwnd range lookup API
// to determine if a password has been included in a public data breach
func isPwnd(p string) bool {

	pSha1 := strings.ToUpper(fmt.Sprintf("%x", sha1.Sum([]byte(p))))

	// The API uses the first 5 characters of the sha1 hash hex string to
	// return a list of possible hash matches. This prevents exposure
	// of the full unsalted sha1 hash of the password
	resp, err := http.Get(fmt.Sprintf("%s/range/%s", pwndAPIBase, pSha1[:5]))
	if err != nil {
		return false
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		// HaveIBeenPwnd returns the hash suffixes with a count of the
		// number of times that they have been compromised. Because of
		// this, we only check everything after idx 5
		if strings.Contains(scanner.Text(), pSha1[5:]) {
			return true
		}
	}
	return false
}

// metricEntropy tries to measure the 'randomness' of a password
// it returns the shannon entropy divided by the length of the
// password.
func metricEntropy(pass string) float64 {
	m := map[rune]float64{}
	for _, r := range pass {
		m[r]++
	}

	var hx float64
	for _, val := range m {
		hx += val * math.Log2(val)
	}

	l := float64(len(pass))
	hx = (math.Log2(l) - (hx / l))

	mEntropy := hx / l

	if math.IsNaN(mEntropy) {
		return 0.0
	}

	return mEntropy
}

func checkpass(user string, pass string, banlist *sync.Map) bool {
	// by default, passwords are considered OK
	passOk := true

	// calculate entropy before the passsword normalization occurs
	passEntropy := metricEntropy(pass)

	// convert the username and password to lowercase
	user = strings.ToLower(user)
	pass = strings.ToLower(pass)

	// determine if the username is in the password anywhere
	userInPassword := passContainsUser(user, pass)

	// determine if password is in the locally defined banlist
	_, passInBanlist := banlist.Load(pass)

	// determine if password is in HaveIBeenPwned
	// using a simple timeout routine to prevent failed calls to
	// HaveIBeenPwnd resulting in no response
	cResponse := make(chan bool, 1)
	var passIsPwnd bool
	go func() {
		cResponse <- isPwnd(pass)
	}()

	select {
	case resp := <-cResponse:
		passIsPwnd = resp
	case <-time.After(3 * time.Second):
		passIsPwnd = false
	}

	// if any one of these conditions returns true, we should say that
	// the password is NOT OK
	passOk = !(userInPassword || passInBanlist || passIsPwnd)
	fmt.Printf(logFmt, user, passOk, userInPassword,
		passInBanlist, passIsPwnd, passEntropy)

	return passOk
}
