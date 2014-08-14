run: weather
	./weather

weather:
	go build -o weather src/weather/main.go

clean:
	rm -f weather
