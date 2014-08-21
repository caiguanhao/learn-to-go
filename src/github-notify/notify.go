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

func getOpts() {
	flag.StringVar(&accessToken, "token", "", "<token>     GitHub access token")
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
}

func get(url string) ([]byte, error) {
	client := &http.Client{}

	var err error
	var req *http.Request
	var res *http.Response

	req, err = http.NewRequest("GET", url, nil)

	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", fmt.Sprintf("token %s", accessToken))
	res, err = client.Do(req)

	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		if lastTimeMsg == 10 {
			fmt.Print(".")
		} else {
			fmt.Println(res.Status)
		}
		lastTimeMsg = 10
		return nil, err
	}

	defer res.Body.Close()
	return ioutil.ReadAll(res.Body)
}

var lastTimeMsg = -1

func check() {
	var body []byte
	var err error

	body, err = get("https://api.github.com/notifications?all=true&participating=true")

	if body == nil || err != nil {
		return
	}

	var notifications []Notifications
	json.Unmarshal(body, &notifications)

	if len(notifications) == 0 {
		if lastTimeMsg == 1 {
			fmt.Print(".")
		} else {
			fmt.Println("No notifications yet.")
		}
		lastTimeMsg = 1
		return
	}

	check := []byte(fmt.Sprintf("%v", notifications))
	checksum := []byte(fmt.Sprintf("%x", sha1.Sum(check)))
	currentUser, _ := user.Current()
	checksumFile := path.Join(currentUser.HomeDir, ".github.notify.checksum")

	content, err := ioutil.ReadFile(checksumFile)

	if err == nil {
		if bytes.Equal(checksum, content) {
			if lastTimeMsg == 2 {
				fmt.Print(".")
			} else {
				fmt.Println("No new notifications yet.")
			}
			lastTimeMsg = 2
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
}

func main() {
	getOpts()
	for {
		check()
		time.Sleep(8 * time.Second)
	}
}
