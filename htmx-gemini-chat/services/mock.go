package services

import (
	"net/http"
	"time"
)

func mock_html(response http.ResponseWriter) {
	str := "```html" + `		
	<!
	DOCTYPE html>
	<html>
	<head>
	<title>My Page</title>

	</head>
	<body>
	<h1>Hello, World!</h1>
	
	<p>This is a simple HTML page.</p>
	</body>
	</html>
	`
	for _, char := range str {
		sendMessageAndFlush(string(char), response)
		time.Sleep(10 * time.Millisecond)
	}
}
