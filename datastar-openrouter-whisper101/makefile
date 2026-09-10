dev:
	npx concurrently -k -p "name" -n "TAILWIND,TEMPL" -c "blue,magenta" \
		"npx tailwindcss -i ./assets/css/input.css -o ./assets/css/stream-scribe.css --watch" \
		"templ generate --watch --proxy=\"http://localhost:3000\" --cmd=\"go run ./cmd/server\""


build:
	npx tailwindcss -i ./assets/css/input.css -o ./assets/css/stream-scribe.css --minify
	templ generate
	go build -ldflags="-s -w" -o bin/datastar-openrouter-whisper101 ./cmd/server
	