learn-to-go
===========

Learn `golang` by making some small stupid applications.
Make sure you have installed `go`.

## weather

Get weather forecast information.

```
go get -u -v github.com/caiguanhao/learn-to-go/weather
weather of Hong Kong
```

## lyrics

Get lyrics of current playing song for your iTunes.
Supports English (azlyrics.com) and Chinese songs (cn.azlyricdb.com).

```
go get -u -v github.com/caiguanhao/learn-to-go/lyrics
lyrics
```

Or search for lyrics:

```
lyrics --no-pager of Birthday by Katy Perry
```

If you have installed `lolcat`, you can view the lyrics in rainbow colors:

```
lyrics -l
```

## github-notify

Query for all new GitHub notifications about you and open related web page
in your browser.

You need to generate new token (in Settings > Applications), and select scopes:
"repo", "public_repo", "notifications" (other scopes are not necessary).
Copy the token as use it for option `--token`. To save the token, use `--save`.

```
go get -u -v github.com/caiguanhao/learn-to-go/github-notify
github-notify --token <YOUR-TOKEN-HERE> --save
```

If you are on Mac OS X, you can install the app to your Dock:

```
github-notify --token <YOUR-TOKEN-HERE> --save --install
```

## webhook

```
go get -u -v github.com/caiguanhao/learn-to-go/webhook
WEBHOOKSECRET=F5fGmWv5XYtdxeUR webhook grunt make
```

nginx configurations:

```
server {
  ...
  location = /webhook {
    proxy_pass http://webhook;
    proxy_redirect off;
    proxy_intercept_errors on;
    proxy_set_header Host $http_host;
    proxy_set_header X-Real-IP $remote_addr;
  }
}
upstream webhook {
  server 127.0.0.1:52142;
}
```

## Screenshots

weather:

![weather](https://cloud.githubusercontent.com/assets/1284703/3951341/89923244-26d4-11e4-8a4b-2e2b23963410.png)

lyrics:

![lyrics](https://cloud.githubusercontent.com/assets/1284703/4271003/e3db2620-3cd2-11e4-95d3-924436500579.png)

github-notify:

![github-notify](https://cloud.githubusercontent.com/assets/1284703/4285628/18be9ba4-3d89-11e4-941a-210db651dd92.png)
