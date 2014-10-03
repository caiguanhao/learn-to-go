package main

import (
	"crypto/hmac"
	"crypto/sha1"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type (
	Opts struct {
		Secret  string
		Command struct {
			Available bool
			Name      string
			Args      []string
		}
	}
)

var (
	Options Opts
)

func verifySignature(message, messageMAC, key []byte) bool {
	if len(message) == 0 || len(messageMAC) == 0 {
		return false
	}
	mac := hmac.New(sha1.New, key)
	mac.Write(message)
	expectedMAC := []byte(fmt.Sprintf("%x", mac.Sum(nil)))
	return hmac.Equal(messageMAC, expectedMAC)
}

func handleGitHubWebhookRequest(res http.ResponseWriter, req *http.Request) {
	for req.Method == "POST" {
		event := req.Header.Get("x-github-event")
		if event == "" {
			break
		}
		ip := req.Header.Get("x-real-ip")
		if ip == "" {
			ip = req.RemoteAddr
		}
		log.Printf("[%s] %s", ip, event)
		var signature []byte
		fmt.Sscanf(req.Header.Get("x-hub-signature"), "sha1=%s", &signature)
		body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			log.Print(err)
			break
		}
		req.Body.Close()
		if verifySignature(body, signature, []byte(Options.Secret)) {
			if Options.Command.Available {
				go func() {
					name := Options.Command.Name
					cmd := exec.Command(name, Options.Command.Args...)
					args := strings.Join(Options.Command.Args, " ")
					cmd.Start()
					log.Printf("[%s:%p:%s] %s %s", ip, &cmd, "RUN", name, args)
					cmd.Wait()
					log.Printf("[%s:%p:%s] %s %s", ip, &cmd, "FIN", name, args)
				}()
			} else {
				log.Printf("[%s] authenticated, but nothing to do", ip)
			}
			fmt.Fprintln(res, "OK")
			return
		} else {
			log.Printf("[%s] failed to authenticate", ip)
		}
		break
	}
	http.Error(res, "404 page not found", http.StatusNotFound)
}

func init() {
	flag.StringVar(&Options.Secret, "secret", "", "")
	flag.Usage = func() {
		n := filepath.Base(os.Args[0])
		fmt.Printf(`Usage:   %s [--secret <secret>] <command>

Note:    secret must be provided by either --secret option or
         environment variable WEBHOOKSECRET

Example: %s --secret "mypass" grunt make
`, n, n)
	}
	flag.Parse()
	if Options.Secret == "" {
		Options.Secret = os.Getenv("WEBHOOKSECRET")
	}
	if Options.Secret == "" {
		fmt.Fprintln(os.Stderr, "You must specify secret. See --help.")
		os.Exit(1)
	}
	if flag.NArg() > 0 {
		args := flag.Args()
		Options.Command.Available = true
		Options.Command.Name = args[0]
		Options.Command.Args = args[1:]
	}
	if !Options.Command.Available {
		fmt.Fprintln(os.Stderr,
			"Warning: You haven't specified any command to execute.")
	}
}

func main() {
	http.HandleFunc("/webhook", handleGitHubWebhookRequest)
	addr := "127.0.0.1:52142"
	log.Printf("Listening on %s", addr)
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		log.Fatal(err)
	}
}
