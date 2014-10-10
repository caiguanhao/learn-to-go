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
)

type (
	Webhook struct {
		Respository struct {
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

func get(body []byte, event string) (bool, string) {
	hook := &Webhook{}
	err := json.Unmarshal(body, &hook)
	if err != nil {
		return false, "authenticated, but failed to unmarshal request body"
	}
	repo := hook.Respository.FullName
	return true, repo
}

func run(repo, event, ip string) {
	command, cvalid := Configs.GetByRepoEvent("command", repo, event)
	if !cvalid {
		log.Printf("[%s] authenticated, but nothing to do", ip)
		return
	}
	cmd := exec.Command("bash", "-c", command)
	directory, dvalid := Configs.GetByRepoEvent("directory", repo, event)
	if dvalid {
		cmd.Dir = directory
		log.Printf("[%s:%p:DIR] %s", ip, &cmd, directory)
	}
	var err error
	stdout, sovalid := Configs.GetByRepoEvent("stdout", repo, event)
	if sovalid {
		var ofile *os.File
		ofile, err = os.OpenFile(stdout, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
		if err != nil {
			log.Printf("[%s] failed to write stdout to: %s (%s)",
				ip, stdout, err)
			return
		}
		defer ofile.Close()
		cmd.Stdout = ofile
	}
	stderr, sevalid := Configs.GetByRepoEvent("stderr", repo, event)
	if sevalid {
		var efile *os.File
		efile, err = os.OpenFile(stderr, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
		if err != nil {
			log.Printf("[%s] failed to write stderr to: %s", ip, stderr)
			return
		}
		defer efile.Close()
		cmd.Stderr = efile
	}
	err = cmd.Start()
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
			ok, ret := get(body, event)
			if ok {
				go run(ret, event, ip)
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
		for _, arg := range flag.Args() {
			paths = append([]string{arg}, paths...)
		}
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
	route, _ := Configs.Get("route")
	if route == "" {
		route = "/webhook"
	}
	http.HandleFunc(route, handleGitHubWebhookRequest)
	bind, _ := Configs.Get("bind")
	port, _ := Configs.Get("port")
	if bind == "" {
		bind = "127.0.0.1"
	}
	if port == "" {
		port = "52142"
	}
	addr := fmt.Sprintf("%s:%s", bind, port)
	log.Printf("Listening on %s, route is %s", addr, route)
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		log.Fatal(err)
	}
}
