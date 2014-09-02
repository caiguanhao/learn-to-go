package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path"
	"time"
)

var (
	GET                         = "GET"
	AUTHORIZATION               = "Authorization"
	NO_NOTIFICATIONS_YET        = "No notifications yet."
	NO_NEW_NOTIFICATIONS_YET    = "No new notifications yet."
	GITHUB_NOTIFY_CHECKSUM_FILE = ".github.notify.checksum"
	GITHUB_NOTIFY_TOKEN_FILE    = ".github.notify.token"
	GITHUB_NOTIFICATIONS_API    = "https://api.github.com/notifications" +
		"?all=true&participating=true"
)

type Subject struct {
	Type             string `json:"type"`
	Title            string `json:"title"`
	LatestCommentUrl string `json:"latest_comment_url"`
}

type Notifications struct {
	ID      string  `json:"id"`
	Subject Subject `json:"subject"`
}

type User struct {
	Login string `json:"login"`
}

type Commit struct {
	HtmlUrl string `json:"html_url"`
}

var accessToken string
var saveToken bool
var removeToken bool
var installOnMac bool
var userHomeDir string

func getOpts() {
	currentUser, _ := user.Current()
	userHomeDir = currentUser.HomeDir

	flag.StringVar(&accessToken, "token", "", "<token>  GitHub access token")
	flag.BoolVar(&saveToken, "save", false, "          Save token to file")
	flag.BoolVar(&removeToken, "remove", false, "        Remove token file and exit")
	flag.BoolVar(&installOnMac, "install", false, "       Install app on Mac OS X")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTION]\n\n", path.Base(os.Args[0]))
		flag.VisitAll(func(flag *flag.Flag) {
			switch flag.DefValue {
			case "true", "false", "":
				fmt.Fprintf(os.Stderr, "  --%s %s\n", flag.Name, flag.Usage)
			default:
				fmt.Fprintf(os.Stderr, "  --%s %s, default is %s\n",
					flag.Name, flag.Usage, flag.DefValue)
			}
		})
	}
	flag.Parse()

	tokenFile := path.Join(userHomeDir, GITHUB_NOTIFY_TOKEN_FILE)

	if installOnMac {
		installAppOnMac()
		os.Exit(0)
	}

	if removeToken {
		err := os.Remove(tokenFile)
		if err == nil {
			fmt.Fprintf(os.Stderr, "Removed token file %s\n", tokenFile)
		} else {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(0)
	}

	if accessToken == "" {
		content, err := ioutil.ReadFile(tokenFile)
		if err == nil {
			accessToken = string(content)
		}
		if accessToken != "" {
			fmt.Fprintf(os.Stderr, "Read token from %s\n", tokenFile)
		}
	}

	if accessToken == "" {
		fmt.Fprintln(os.Stderr, "Warning: No access token specified!")
	}

	if saveToken {
		if accessToken == "" {
			fmt.Fprintln(os.Stderr, "Please use this option with --token!")
			os.Exit(1)
		}
		ioutil.WriteFile(tokenFile, []byte(accessToken), 0600)
		fmt.Fprintf(os.Stderr, "Token has been saved to %s\n", tokenFile)
		fmt.Fprintf(os.Stderr, "You can run `%s` without --token next time!\n", path.Base(os.Args[0]))
	}
}

func installAppOnMac() {
	file, _ := ioutil.TempFile("", "")
	filename := file.Name()
	file.WriteString(INSTALL_SHELL_SCRIPT)
	file.Chmod(500)
	file.Close()
	fmt.Printf("Running shell script %s\n", filename)
	exec.Command(filename).Run()
}

func get(url string) ([]byte, error) {
	client := &http.Client{}

	var err error
	var req *http.Request
	var res *http.Response

	req, err = http.NewRequest(GET, url, nil)

	if err != nil {
		return nil, err
	}

	req.Header.Add(AUTHORIZATION, fmt.Sprintf("token %s", accessToken))
	res, err = client.Do(req)

	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		if lastStatusString != res.Status {
			lastStatusString = res.Status
		}
		log(&lastStatusString)
		return nil, err
	}

	defer res.Body.Close()
	return ioutil.ReadAll(res.Body)
}

var lastTimeMsg *string
var lastStatusString string
var lastTimeDot bool

func log(message *string) {
	if lastTimeMsg == message {
		fmt.Print(".")
		lastTimeDot = true
	} else {
		if lastTimeDot {
			fmt.Println()
		}
		fmt.Println(*message)
		lastTimeDot = false
	}
	lastTimeMsg = message
}

func check() {
	var body []byte
	var err error

	body, err = get(GITHUB_NOTIFICATIONS_API)

	if body == nil || err != nil {
		return
	}

	var notifications []Notifications
	json.Unmarshal(body, &notifications)

	if len(notifications) == 0 {
		log(&NO_NOTIFICATIONS_YET)
		return
	}

	check := []byte(fmt.Sprintf("%v", notifications))
	checksum := []byte(fmt.Sprintf("%x", sha1.Sum(check)))

	checksumFile := path.Join(userHomeDir, GITHUB_NOTIFY_CHECKSUM_FILE)

	content, err := ioutil.ReadFile(checksumFile)

	if err == nil {
		if bytes.Equal(checksum, content) {
			log(&NO_NEW_NOTIFICATIONS_YET)
			return
		}
	}

	ioutil.WriteFile(checksumFile, checksum, 0600)

	body, err = get(notifications[0].Subject.LatestCommentUrl)

	if err != nil {
		return
	}

	commit := &Commit{}

	json.Unmarshal(body, &commit)

	exec.Command("open", commit.HtmlUrl).Run()

	openMsg := fmt.Sprintf("Opened %s", commit.HtmlUrl)
	log(&openMsg)
}

func main() {
	getOpts()
	for {
		check()
		time.Sleep(8 * time.Second)
	}
}
