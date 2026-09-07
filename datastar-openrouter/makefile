dev:
	npx concurrently -k -p "name" -n "TAILWIND,TEMPL" -c "blue,magenta" \
		"npx tailwindcss -i ./assets/css/input.css -o ./assets/css/nexus.css --watch" \
		"templ generate --watch --proxy=\"http://localhost:3000\" --cmd=\"go run ./cmd/server\""


build:
	npm ci
	npx tailwindcss -i ./assets/css/input.css -o ./assets/css/nexus.css --minify
	go run github.com/a-h/templ/cmd/templ@v0.3.1020 generate
	go build -ldflags="-s -w" -o bin/datastar-openrouter ./cmd/server
	