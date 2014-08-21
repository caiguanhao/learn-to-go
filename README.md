learn-to-go
===========

Learn `golang` by making some small stupid applications.
Make sure you have installed `go`.

## weather

Get weather forecast information.

```
go get github.com/caiguanhao/learn-to-go/src/weather
weather of Hong Kong
```

## github-notify

Query for all new GitHub notifications about you and open related web page
in your browser.

You need to generate new token (in Settings > Applications), and select scopes:
"repo", "public_repo", "notifications" (other scopes are not necessary).
Copy the token as use it for option `--token`.

```
go get github.com/caiguanhao/learn-to-go/src/github-notify
github-notify --token 0e136cf2a819ac49c78c64edc416fe8f269f513c
```

## Screenshots

![Screenshot](https://cloud.githubusercontent.com/assets/1284703/3951341/89923244-26d4-11e4-8a4b-2e2b23963410.png)
