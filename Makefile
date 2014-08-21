github-notify:
	go run src/github-notify/notify.go --token $(ACCESSTOKEN)

weather:
	go run src/weather/main.go
