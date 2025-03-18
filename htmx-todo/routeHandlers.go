package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"htmx-todo/components"
	"htmx-todo/models"
	"net/http"
	"os"
	"strconv"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func MainPage(w http.ResponseWriter, r *http.Request) {
	valid := validateUserIdInCookie(r)
	if !valid {
		components.MainElForLogin(false).Render(r.Context(), w)
	} else {
		authToken := os.Getenv("TURSO_AUTH_TOKEN")
		databaseUrl := os.Getenv("TURSO_DATABASE_URL")
		sort := r.URL.Query().Get("sort")
		groceries := GetGroceryData(databaseUrl, authToken, sort)
		items, _ := tranformGroceries(groceries, false)
		components.MainEl(items, sort).Render(r.Context(), w)
	}
}

func Login(w http.ResponseWriter, r *http.Request) {
	hashedTokenFromConfig := os.Getenv("TOKEN")
	token := r.FormValue("token")
	compareErr := bcrypt.CompareHashAndPassword([]byte(hashedTokenFromConfig), []byte(token))
	if compareErr != nil {
		components.SectionElForLogin(true).Render(r.Context(), w)
	} else {
		authToken := os.Getenv("TURSO_AUTH_TOKEN")
		databaseUrl := os.Getenv("TURSO_DATABASE_URL")
		groceries := GetGroceryData(databaseUrl, authToken, "")
		items, _ := tranformGroceries(groceries, true)
		cookie := generateUserIdCookie()
		http.SetCookie(w, &cookie)
		components.SectionEl(items, "").Render(r.Context(), w)
	}
}
func generateUserIdCookie() http.Cookie {
	secure := true
	if os.Getenv("ENV") == "Development" {
		secure = false
	}
	cookieSecretKey := os.Getenv("COOKIE_SECRET")
	cookieName := "id"
	userId := os.Getenv("USER_ID")

	mac := hmac.New(sha256.New, []byte(cookieSecretKey))
	mac.Write([]byte(cookieName))
	mac.Write([]byte(userId))
	signature := mac.Sum(nil)

	cookieValueSignedBytes := append(signature, []byte(userId)...)
	cookieValueSignedStr := base64.URLEncoding.EncodeToString(cookieValueSignedBytes)

	cookie := http.Cookie{
		Name:     cookieName,
		Value:    cookieValueSignedStr,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		Expires:  time.Now().Add(365 * 24 * time.Hour),
		SameSite: http.SameSiteStrictMode,
	}
	return cookie
}
func validateUserIdInCookie(r *http.Request) bool {
	cookieName := "id"
	userIdFromConfig := os.Getenv("USER_ID")
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		return false
	}
	cookieValueBase64Encoded := cookie.Value
	cookieValueSignedStr, err := base64.URLEncoding.DecodeString(cookieValueBase64Encoded)
	if err != nil {
		return false
	}

	cookieValueSignedBytes := []byte(cookieValueSignedStr)
	signature := cookieValueSignedBytes[:sha256.Size]

	userIdFromCookie := cookieValueSignedBytes[sha256.Size:]

	cookieSecretKey := os.Getenv("COOKIE_SECRET")
	mac := hmac.New(sha256.New, []byte(cookieSecretKey))
	mac.Write([]byte(cookieName))
	mac.Write([]byte(userIdFromConfig))
	expectedSignature := mac.Sum(nil)

	if !hmac.Equal(signature, expectedSignature) {
		return false
	}
	return string(userIdFromCookie) == userIdFromConfig
}
func AddGroceryItem(w http.ResponseWriter, r *http.Request) {
	authToken := os.Getenv("TURSO_AUTH_TOKEN")
	databaseUrl := os.Getenv("TURSO_DATABASE_URL")
	newItem := components.Item{Name: r.FormValue("item"), Quantity: 1, AnimationClass: "animate-slide-down"}
	newItem.Id = InsertGrocery(databaseUrl, authToken, newItem.Name, newItem.Quantity)
	sort := r.FormValue("sort")
	groceries := GetGroceryData(databaseUrl, authToken, sort)
	items, idToIndexMap := tranformGroceries(groceries, false)
	items[idToIndexMap[newItem.Id]].AnimationClass = newItem.AnimationClass
	components.ItemsUl(items).Render(r.Context(), w)
}
func IncrementGroceryItemQuantity(w http.ResponseWriter, r *http.Request) {
	authToken := os.Getenv("TURSO_AUTH_TOKEN")
	databaseUrl := os.Getenv("TURSO_DATABASE_URL")
	id, IdErr := strconv.Atoi(r.FormValue("id"))
	currentQuantity, QuantityErr := strconv.Atoi(r.FormValue("currentQuantity"))
	if IdErr != nil || QuantityErr != nil {
		components.ItemQuantityDisplay(id, currentQuantity, false).Render(r.Context(), w)
	} else {
		currentQuantity += 1
		rowsAffected := UpdateQuantity(databaseUrl, authToken, id, currentQuantity)
		if rowsAffected != 0 {
			components.ItemQuantityDisplay(id, currentQuantity, true).Render(r.Context(), w)
		} else {
			components.ItemQuantityDisplay(id, currentQuantity, false).Render(r.Context(), w)
		}
	}
}
func DecrementGroceryItemQuantity(w http.ResponseWriter, r *http.Request) {
	authToken := os.Getenv("TURSO_AUTH_TOKEN")
	databaseUrl := os.Getenv("TURSO_DATABASE_URL")
	id, IdErr := strconv.Atoi(r.FormValue("id"))
	currentQuantity, QuantityErr := strconv.Atoi(r.FormValue("currentQuantity"))
	if IdErr != nil || QuantityErr != nil {
		components.ItemQuantityDisplay(id, currentQuantity, false).Render(r.Context(), w)
	} else {
		currentQuantity -= 1
		if currentQuantity < 1 {
			currentQuantity = 1
		}
		rowsAffected := UpdateQuantity(databaseUrl, authToken, id, currentQuantity)
		if rowsAffected != 0 {
			components.ItemQuantityDisplay(id, currentQuantity, true).Render(r.Context(), w)
		} else {
			components.ItemQuantityDisplay(id, currentQuantity, false).Render(r.Context(), w)
		}
	}
}
func DeleteGroceryItem(w http.ResponseWriter, r *http.Request) {
	authToken := os.Getenv("TURSO_AUTH_TOKEN")
	databaseUrl := os.Getenv("TURSO_DATABASE_URL")
	sort := r.FormValue("sort")
	groceries := GetGroceryData(databaseUrl, authToken, sort)
	items, idToIndexMap := tranformGroceries(groceries, false)
	id, err := strconv.Atoi(r.FormValue("id"))
	if err == nil {
		rowsAffected := DeleteGrocery(databaseUrl, authToken, id)
		if rowsAffected != 0 {
			items = findAndRemoveItem(items, id, idToIndexMap)
		}
	}
	components.ItemsUl(items).Render(r.Context(), w)
}
func ToggleCompleteGroceryItem(w http.ResponseWriter, r *http.Request) {
	authToken := os.Getenv("TURSO_AUTH_TOKEN")
	databaseUrl := os.Getenv("TURSO_DATABASE_URL")
	id, err := strconv.Atoi(r.FormValue("id"))
	if err == nil {
		groceryModelItem := GetGroceryItemData(databaseUrl, authToken, id)
		item := transformGrocery(groceryModelItem)
		rowsAffected := UpdateCompleted(databaseUrl, authToken, id, !groceryModelItem.Completed)
		if rowsAffected != 0 {
			groceryModelItem.Completed = !groceryModelItem.Completed
			item.Completed = !item.Completed
		}
		components.ItemNameDisplay(item).Render(r.Context(), w)
	}
}
func findAndRemoveItem(items []components.Item, id int, idToIndexMap map[int]int) []components.Item {
	if idToIndexMap != nil {
		index, ok := idToIndexMap[id]
		if ok {
			items = append(items[:index], items[index+1:]...)
			idToIndexMap = nil
			return items
		}
	}
	for i, item := range items {
		if item.Id == id {
			items = append(items[:i], items[i+1:]...)
			break
		}
	}

	return items
}

func tranformGroceries(list []models.Grocery, animateAllItems bool) ([]components.Item, map[int]int) {
	animationClass := ""
	if animateAllItems {
		animationClass = "animate-slide-down"
	}
	items := []components.Item{}
	itemIdToIndexMap := make(map[int]int)
	for _, grocery := range list {
		item := components.Item{Id: grocery.Id, Name: grocery.Description, Quantity: grocery.Quantity, Completed: grocery.Completed, AnimationClass: animationClass}
		items = append(items, item)
		itemIdToIndexMap[grocery.Id] = len(items) - 1
	}
	return items, itemIdToIndexMap
}

func transformGrocery(grocery models.Grocery) components.Item {
	return components.Item{Id: grocery.Id, Name: grocery.Description, Quantity: grocery.Quantity, Completed: grocery.Completed, AnimationClass: ""}
}
