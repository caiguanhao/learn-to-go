package main

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/json"
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
	Webhook struct {
		Respository struct {
			Name     string `json:"name"`
			FullName string `json:"full_name"`
		} `json:"repository"`
	}
)

var (
	Configs Conf
)

func verify(message, messageMAC, key []byte) (bool, string) {
	if len(message) == 0 || len(messageMAC) == 0 {
		return false, "empty body or signature"
	}

	mac := hmac.New(sha1.New, key)
	mac.Write(message)
	expectedMAC := []byte(fmt.Sprintf("%x", mac.Sum(nil)))
	isSignatureValid := hmac.Equal(messageMAC, expectedMAC)
	if !isSignatureValid {
		return false, "failed to authenticate"
	}

	return true, ""
}

func check(body []byte, event string) (bool, string) {
	repositoryToCheck, valid := Configs.Get("repository")
	if valid && repositoryToCheck != "" {
		hook := &Webhook{}
		err := json.Unmarshal(body, &hook)
		if err == nil {
			if strings.Contains(repositoryToCheck, "/") {
				if hook.Respository.FullName != repositoryToCheck {
					return false, "authenticated, but repository does not match full name"
				}
			} else {
				if hook.Respository.Name != repositoryToCheck {
					return false, "authenticated, but repository does not match name"
				}
			}
		} else {
			return false, "authenticated, but failed to unmarshal body"
		}
	}

	command, valid := Configs.GetByEvent("command", event)
	if !valid {
		return false, "authenticated, but nothing to do"
	}

	return true, command
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
		verified, notVerifiedReason := verify(body, signature, []byte(secret))
		if valid && verified {
			ok, ret := check(body, event)
			if ok {
				go func() {
					directory, valid := Configs.GetByEvent("directory", event)
					cmd := exec.Command("bash", "-c", ret)
					if valid {
						cmd.Dir = directory
						log.Printf("[%s:%p:DIR] %s", ip, &cmd, directory)
					}
					err := cmd.Start()
					if err != nil {
						log.Printf("[%s] failed to start: %s", ip, ret)
						return
					}
					log.Printf("[%s:%p:RUN] %s", ip, &cmd, ret)
					err = cmd.Wait()
					if err == nil {
						log.Printf("[%s:%p:FIN] successful", ip, &cmd)
					} else {
						log.Printf("[%s:%p:ERR] exit with %s", ip, &cmd, err)
					}
				}()
				fmt.Fprintln(res, "OK")
			} else {
				fmt.Fprintln(res, ret)
				log.Printf("[%s] %s", ip, ret)
			}
			return
		} else {
			log.Printf("[%s] %s", ip, notVerifiedReason)
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
