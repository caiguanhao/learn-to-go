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
)

var (
	Configs Conf
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
		secret, valid := Configs.Get("secret")
		if valid && verifySignature(body, signature, []byte(secret)) {
			command, valid := Configs.GetCommandByEvent(event)
			if valid {
				go func() {
					cmd := exec.Command("bash", "-c", command)
					err := cmd.Start()
					if err != nil {
						log.Printf("[%s] failed to start: %s", ip, command)
						return
					}
					log.Printf("[%s:%p:RUN] %s", ip, &cmd, command)
					err = cmd.Wait()
					if err == nil {
						log.Printf("[%s:%p:FIN] successful", ip, &cmd)
					} else {
						log.Printf("[%s:%p:ERR] exit with %s", ip, &cmd, err)
					}
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
	noFileRead := flag.Bool("no-file-read", false, "")
	flag.Usage = func() {
		n := filepath.Base(os.Args[0])
		fmt.Printf(`USAGE:       %s [OPTION] [CONFIG FILE]

OPTION:      --no-file-read:
               Even if config specified, don't read configs from any file.
               This will force to use STDIN to get configs.

CONFIG FILE: If webhook.conf does not exist in the working
             directory, you may specify the file path to
             the config file or it will read from STDIN.
`, n)
	}
	flag.Parse()

	if !*noFileRead {
		paths := []string{"webhook.conf"}
		paths = append(paths, flag.Args()...)
		Configs.SetFilePaths(paths...)
	}
	path, read := Configs.Read()
	if read {
		if path == "" {
			path = "STDIN"
		}
		fmt.Printf("Read configs from %s.\n", path)
	} else {
		fmt.Println("No config file found.")
		os.Exit(1)
	}

	secret, valid := Configs.Get("secret")
	if !valid || secret == "" {
		fmt.Fprintln(os.Stderr, "You must specify secret. See --help.")
		os.Exit(1)
	}
}

func main() {
	http.HandleFunc("/webhook", handleGitHubWebhookRequest)
	bind, _ := Configs.Get("bind")
	port, _ := Configs.Get("port")
	if bind == "" {
		bind = "127.0.0.1"
	}
	if port == "" {
		port = "52142"
	}
	addr := fmt.Sprintf("%s:%s", bind, port)
	log.Printf("Listening on %s", addr)
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		log.Fatal(err)
	}
}
