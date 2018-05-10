package main

import (
	"bufio"
	"crypto/sha1"
	"fmt"
	"net/http"
	"strings"
	"sync"
)

const (
	// base URL for the HaveIBeenPwnd API
	pwndAPIBase string = "https://api.pwnedpasswords.com"
)

func passContainsUser(u string, p string) bool {
	fmt.Printf("Checking to see if password contains username %s.", u)
	return strings.Contains(p, u)
}

// isPwnd performs a lookup against the HaveIBeenPwnd range lookup API
// to determine if a password has been included in a public data breach
func isPwnd(p string) bool {

	pSha1 := strings.ToUpper(fmt.Sprintf("%x", sha1.Sum([]byte(p))))
	fmt.Printf("Performing HaveIBeenPwned lookup on hash: %s", pSha1)

	// The API uses the first 5 characters of the sha1 hash hex string to
	// return a list of possible hash matches. This prevents exposure
	// of the full unsalted sha1 hash of the password
	resp, err := http.Get(fmt.Sprintf("%s/range/%s", pwndAPIBase, pSha1[:5]))
	if err != nil {
		fmt.Println("Request failure while accessing HaveIBeenPwnd.")
		return false
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		// HaveIBeenPwnd returns the hash suffixes with a count of the
		// number of times that they have been compromised. Because of
		// this, we only check everything after idx 5
		if strings.Contains(scanner.Text(), pSha1[5:]) {
			fmt.Printf("%s has been compromised.", pSha1)
			return true
		}
	}

	fmt.Printf("%s is not in a public data breach.", pSha1)
	return false
}

func checkpass(user string, pass string, banlist *sync.Map) bool {
	// by default, passwords are considered OK
	passOk := true

	// convert the username and password to lowercase
	user = strings.ToLower(user)
	pass = strings.ToLower(pass)

	// determine if the username is in the password anywhere
	if passContainsUser(user, pass) {
		passOk = false
	}

	// determine if password is in HaveIBeenPwned
	// since this is an external network call, it may make sense to
	// eventually do this aynchronously
	if isPwnd(pass) {
		passOk = false
	}

	// check password against locally defined banlist of passwords
	if _, ok := banlist.Load(pass); ok {
		fmt.Println("Password appears in banlist.")
		passOk = false
	}

	return passOk
}
