package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"htmx-calendar/components"
	"htmx-calendar/models"
	"htmx-calendar/services"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/resend/resend-go/v2"
	"github.com/supabase-community/postgrest-go"
	"github.com/supabase-community/supabase-go"
)

var dateLayout = "2006-01-02"

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
	} else {
		fmt.Println("Loaded .env file")
	}
	today := time.Now().Format(dateLayout)
	userId := os.Getenv("SUPABASE_USER_ID")

	channels := make([]chan []models.EventData, 14)

	for idx := range channels {
		channels[idx] = make(chan []models.EventData)
	}

	go GetCurrentDayDataForUser(userId, today, channels[0])
	go GetDailyDataForUser(userId, today, channels[1])
	go GetWeeklyDataForUserWithExactDate(userId, today, channels[2])
	go GetWeeklyDataForUserSaturday(userId, today, channels[3])
	go GetEveryTwoWeeksDataForUserWithExactDate(userId, today, channels[4])
	go GetEveryTwoWeeksDataForUserSaturday(userId, today, channels[5])
	go GetMonthlyDataForUserWithExactDate(userId, today, channels[6])
	go GetMonthlyDataForUserFirstSaturday(userId, today, channels[7])
	go GetQuarterlyDataForUserWithExactDate(userId, today, channels[8])
	go GetQuarterlyDataForUserFirstSaturday(userId, today, channels[9])
	go GetHalfYearlyDataForUserWithExactDate(userId, today, channels[10])
	go GetHalfYearlyDataForUserFirstSaturday(userId, today, channels[11])
	go GetYearlyDataForUserWithExactDate(userId, today, channels[12])
	go GetYearlyDataForUserFirstSaturday(userId, today, channels[13])

	consolidatedData := []models.EventData{}
	dataIdIncludedHash := services.NewHashSet[string]()
	for _, channel := range channels {
		for data := range channel {
			for _, event := range data {
				if !dataIdIncludedHash.Contains(event.Id) {
					dataIdIncludedHash.Add(event.Id)
					consolidatedData = append(consolidatedData, event)
				}
			}
		}
	}
	if len(consolidatedData) == 0 {
		fmt.Println("No events found for today")
		return
	}
	component := components.EmailTemplate(consolidatedData)
	emailStrBuffer := new(bytes.Buffer)
	component.Render(context.Background(), emailStrBuffer)
	emailChannel := make(chan string)
	defer close(emailChannel)
	go sendEmail(os.Getenv("REMINDER_EMAIL_TO"), os.Getenv("REMINDER_EMAIL_FROM"), emailStrBuffer.String(), os.Getenv("REMINDER_EMAIL_SUBJECT"), os.Getenv("RESEND_API_KEY"), emailChannel)
	fmt.Println(<-emailChannel)

	// Below is for debugging purposes, to print the consolidated data
	// fmt.Println("")
	// for _, data := range consolidatedData {
	// 	fmt.Printf("Task : %v, Frequency : %v\n", data.Task, data.Frequency)
	// }
}
func GetCurrentDayDataForUser(userId string, currDateStr string, channel chan<- []models.EventData) {
	defer close(channel)
	serviceRole := os.Getenv("SUPABASE_SERVICE_ROLE")
	apiUrl := os.Getenv("SUPABASE_API_URL")
	response := []models.EventData{}
	client, err := supabase.NewClient(apiUrl, serviceRole, &supabase.ClientOptions{})
	if err != nil {
		fmt.Printf("Error connecting to supabase %v\n", err.Error())
		channel <- response
		return
	}
	data, _, err := client.From("calendar").Select("id, task, frequency, date, stopAfter, exact", "exact", false).Eq("user_id", userId).Eq("date", currDateStr).Order("created_at", &postgrest.OrderOpts{Ascending: true}).Execute()
	if err != nil {
		fmt.Printf("Error executing query %v\n", err.Error())
		channel <- response
		return
	}
	if err := json.Unmarshal(data, &response); err != nil {
		fmt.Printf("Error unmarshalling results %v\n", err.Error())
	}
	channel <- response
}
func GetDailyDataForUser(userId string, currDateStr string, channel chan<- []models.EventData) {
	defer close(channel)
	serviceRole := os.Getenv("SUPABASE_SERVICE_ROLE")
	apiUrl := os.Getenv("SUPABASE_API_URL")
	allResponse := []models.EventData{}
	ftedResponse := []models.EventData{}
	client, err := supabase.NewClient(apiUrl, serviceRole, &supabase.ClientOptions{})
	if err != nil {
		fmt.Printf("Error connecting to supabase %v\n", err.Error())
		channel <- ftedResponse
		return
	}

	data, _, err := client.From("calendar").Select("id, task, frequency, date, stopAfter, exact", "exact", false).Eq("user_id", userId).Eq("frequency", "Daily").Order("created_at", &postgrest.OrderOpts{Ascending: true}).Execute()

	if err != nil {
		fmt.Printf("Error executing query %v\n", err.Error())
		channel <- ftedResponse
		return
	}
	if err := json.Unmarshal(data, &allResponse); err != nil {
		fmt.Printf("Error unmarshalling results %v\n", err.Error())
	}
	currDate, err := time.Parse(dateLayout, currDateStr)
	if err != nil {
		fmt.Printf("Error parsing current date %v\n", err.Error())
		channel <- ftedResponse
		return
	}
	for _, event := range allResponse {
		eventDate, err := time.Parse(dateLayout, event.Date)
		if err != nil {
			fmt.Printf("Error parsing date %v\n", err.Error())
			continue
		}
		diff := int(currDate.Sub(eventDate).Hours() / 24)
		if diff < 0 {
			fmt.Printf("Skipping future event %v in daily\n", event.Task)
			continue
		}
		stopAfterDate, err := time.Parse(dateLayout, event.StopAfter)
		if err == nil {
			if stopAfterDate.Before(currDate) {
				fmt.Printf("Skipping event %v in daily due to StopAfter\n", event.Task)
				continue
			}
		}
		ftedResponse = append(ftedResponse, event)
	}
	channel <- ftedResponse
}
func GetWeeklyDataForUserWithExactDate(userId string, currDateStr string, channel chan<- []models.EventData) {
	defer close(channel)
	serviceRole := os.Getenv("SUPABASE_SERVICE_ROLE")
	apiUrl := os.Getenv("SUPABASE_API_URL")
	allResponse := []models.EventData{}
	ftedResponse := []models.EventData{}
	client, err := supabase.NewClient(apiUrl, serviceRole, &supabase.ClientOptions{})
	if err != nil {
		fmt.Printf("Error connecting to supabase %v\n", err.Error())
		channel <- ftedResponse
		return
	}

	data, _, err := client.From("calendar").Select("id, task, frequency, date, stopAfter, exact", "exact", false).Eq("user_id", userId).Eq("exact", "yes").Eq("frequency", "Weekly").Order("created_at", &postgrest.OrderOpts{Ascending: true}).Execute()

	if err != nil {
		fmt.Printf("Error executing query %v\n", err.Error())
		channel <- ftedResponse
		return
	}
	if err := json.Unmarshal(data, &allResponse); err != nil {
		fmt.Printf("Error unmarshalling results %v\n", err.Error())
	}
	currDate, err := time.Parse(dateLayout, currDateStr)
	if err != nil {
		fmt.Printf("Error parsing current date %v\n", err.Error())
		channel <- ftedResponse
		return
	}
	for _, event := range allResponse {
		eventDate, err := time.Parse(dateLayout, event.Date)
		if err != nil {
			fmt.Printf("Error parsing date %v\n", err.Error())
			continue
		}
		diff := int(currDate.Sub(eventDate).Hours() / 24)
		if diff < 0 {
			fmt.Printf("Skipping future event %v in weekly\n", event.Task)
			continue
		} else if diff%7 == 0 {
			stopAfterDate, err := time.Parse(dateLayout, event.StopAfter)
			if err == nil {
				if stopAfterDate.Before(currDate) {
					fmt.Printf("Skipping event %v in weekly due to StopAfter\n", event.Task)
					continue
				}
			}
			ftedResponse = append(ftedResponse, event)
		}
	}
	channel <- ftedResponse
}

func GetWeeklyDataForUserSaturday(userId string, currDateStr string, channel chan<- []models.EventData) {
	defer close(channel)
	serviceRole := os.Getenv("SUPABASE_SERVICE_ROLE")
	apiUrl := os.Getenv("SUPABASE_API_URL")
	allResponse := []models.EventData{}
	ftedResponse := []models.EventData{}
	client, err := supabase.NewClient(apiUrl, serviceRole, &supabase.ClientOptions{})
	if err != nil {
		fmt.Printf("Error connecting to supabase %v\n", err.Error())
		channel <- ftedResponse
		return
	}

	data, _, err := client.From("calendar").Select("id, task, frequency, date, stopAfter, exact", "exact", false).Eq("user_id", userId).Eq("frequency", "Weekly").Neq("exact", "yes").Order("created_at", &postgrest.OrderOpts{Ascending: true}).Execute()

	if err != nil {
		fmt.Printf("Error executing query %v\n", err.Error())
		channel <- ftedResponse
		return
	}
	if err := json.Unmarshal(data, &allResponse); err != nil {
		fmt.Printf("Error unmarshalling results %v\n", err.Error())
	}

	currDate, err := time.Parse(dateLayout, currDateStr)
	if err != nil {
		fmt.Printf("Error parsing current date %v\n", err.Error())
		channel <- ftedResponse
		return
	}
	if currDate.Weekday() == time.Saturday {
		currDate, _ := time.Parse(dateLayout, currDateStr)
		for _, event := range allResponse {
			eventDate, err := time.Parse(dateLayout, event.Date)
			if err != nil {
				fmt.Printf("Error parsing date %v\n", err.Error())
				continue
			}
			diff := int(currDate.Sub(eventDate).Hours() / 24)
			if diff < 0 {
				fmt.Printf("Skipping future event %v in weekly\n", event.Task)
				continue
			}
			stopAfterDate, err := time.Parse(dateLayout, event.StopAfter)
			if err == nil {
				if stopAfterDate.Before(currDate) {
					fmt.Printf("Skipping event %v in weekly due to StopAfter\n", event.Task)
					continue
				}
			}
			ftedResponse = append(ftedResponse, event)
		}
	}
	channel <- ftedResponse
}
func GetEveryTwoWeeksDataForUserWithExactDate(userId string, currDateStr string, channel chan<- []models.EventData) {
	defer close(channel)
	serviceRole := os.Getenv("SUPABASE_SERVICE_ROLE")
	apiUrl := os.Getenv("SUPABASE_API_URL")
	allResponse := []models.EventData{}
	ftedResponse := []models.EventData{}
	client, err := supabase.NewClient(apiUrl, serviceRole, &supabase.ClientOptions{})
	if err != nil {
		fmt.Printf("Error connecting to supabase %v\n", err.Error())
		channel <- ftedResponse
		return
	}

	data, _, err := client.From("calendar").Select("id, task, frequency, date, stopAfter, exact", "exact", false).Eq("user_id", userId).Eq("exact", "yes").Eq("frequency", "Every two weeks").Order("created_at", &postgrest.OrderOpts{Ascending: true}).Execute()

	if err != nil {
		fmt.Printf("Error executing query %v\n", err.Error())
		channel <- ftedResponse
		return
	}
	if err := json.Unmarshal(data, &allResponse); err != nil {
		fmt.Printf("Error unmarshalling results %v\n", err.Error())
	}

	currDate, err := time.Parse(dateLayout, currDateStr)
	if err != nil {
		fmt.Printf("Error parsing current date %v\n", err.Error())
		channel <- ftedResponse
		return
	}
	for _, event := range allResponse {
		eventDate, err := time.Parse(dateLayout, event.Date)
		if err != nil {
			fmt.Printf("Error parsing date %v\n", err.Error())
			continue
		}
		diff := int(currDate.Sub(eventDate).Hours() / 24)
		if diff < 0 {
			fmt.Printf("Skipping future event %v in every two weeks\n", event.Task)
			continue
		}
		if diff%14 == 0 {
			stopAfterDate, err := time.Parse(dateLayout, event.StopAfter)
			if err == nil {
				if stopAfterDate.Before(currDate) {
					fmt.Printf("Skipping event %v in every two weeks due to StopAfter\n", event.Task)
					continue
				}
			}
			ftedResponse = append(ftedResponse, event)
		}
	}
	channel <- ftedResponse
}
func GetAlternateSaturdaysInAYear(year int) services.HashSet[string] {
	alternateSaturdays := services.NewHashSet[string]()
	firstDate := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)

	daysToAddToReachSaturday := time.Saturday - firstDate.Weekday()
	saturdayToAddToSet := firstDate.AddDate(0, 0, int(daysToAddToReachSaturday))
	for saturdayToAddToSet.Year() == year {
		alternateSaturdays.Add(saturdayToAddToSet.Format(dateLayout))
		saturdayToAddToSet = saturdayToAddToSet.AddDate(0, 0, 14) // Move to the next alternate Saturday
	}
	return alternateSaturdays

}
func GetEveryTwoWeeksDataForUserSaturday(userId string, currDateStr string, channel chan<- []models.EventData) {
	defer close(channel)
	serviceRole := os.Getenv("SUPABASE_SERVICE_ROLE")
	apiUrl := os.Getenv("SUPABASE_API_URL")
	allResponse := []models.EventData{}
	ftedResponse := []models.EventData{}
	client, err := supabase.NewClient(apiUrl, serviceRole, &supabase.ClientOptions{})
	if err != nil {
		fmt.Printf("Error connecting to supabase %v\n", err.Error())
		channel <- ftedResponse
		return
	}

	data, _, err := client.From("calendar").Select("id, task, frequency, date, stopAfter, exact", "exact", false).Eq("user_id", userId).Eq("frequency", "Every two weeks").Neq("exact", "yes").Order("created_at", &postgrest.OrderOpts{Ascending: true}).Execute()

	if err != nil {
		fmt.Printf("Error executing query %v\n", err.Error())
		channel <- ftedResponse
		return
	}
	if err := json.Unmarshal(data, &allResponse); err != nil {
		fmt.Printf("Error unmarshalling results %v\n", err.Error())
	}

	currDate, err := time.Parse(dateLayout, currDateStr)
	if err != nil {
		fmt.Printf("Error parsing current date %v\n", err.Error())
		channel <- ftedResponse
		return
	}
	if currDate.Weekday() == time.Saturday {
		alternateSaturdays := GetAlternateSaturdaysInAYear(currDate.Year())

		for _, event := range allResponse {
			eventDate, err := time.Parse(dateLayout, event.Date)
			if err != nil {
				fmt.Printf("Error parsing date %v\n", err.Error())
				continue
			}
			diff := int(currDate.Sub(eventDate).Hours() / 24)
			if diff < 0 {
				fmt.Printf("Skipping future event %v in every two weeks\n", event.Task)
				continue
			}
			if alternateSaturdays.Contains(event.Date) {
				stopAfterDate, err := time.Parse(dateLayout, event.StopAfter)
				if err == nil {
					if stopAfterDate.Before(currDate) {
						fmt.Printf("Skipping event %v in alternate saturdays due to StopAfter\n", event.Task)
						continue
					}
				}
				ftedResponse = append(ftedResponse, event)
			}
		}
	}
	channel <- ftedResponse
}
func GetMonthlyDataForUserWithExactDate(userId string, currDateStr string, channel chan<- []models.EventData) {
	defer close(channel)
	serviceRole := os.Getenv("SUPABASE_SERVICE_ROLE")
	apiUrl := os.Getenv("SUPABASE_API_URL")
	allResponse := []models.EventData{}
	ftedResponse := []models.EventData{}
	client, err := supabase.NewClient(apiUrl, serviceRole, &supabase.ClientOptions{})
	if err != nil {
		fmt.Printf("Error connecting to supabase %v\n", err.Error())
		channel <- ftedResponse
		return
	}

	data, _, err := client.From("calendar").Select("id, task, frequency, date, stopAfter, exact", "exact", false).Eq("user_id", userId).Eq("exact", "yes").Eq("frequency", "Monthly").Order("created_at", &postgrest.OrderOpts{Ascending: true}).Execute()

	if err != nil {
		fmt.Printf("Error executing query %v\n", err.Error())
		channel <- ftedResponse
		return
	}
	if err := json.Unmarshal(data, &allResponse); err != nil {
		fmt.Printf("Error unmarshalling results %v\n", err.Error())
	}

	currDate, err := time.Parse(dateLayout, currDateStr)
	if err != nil {
		fmt.Printf("Error parsing current date %v\n", err.Error())
		channel <- ftedResponse
		return
	}
	dateToInclude := currDate
	appStartDate := time.Date(2023, time.Month(1), 1, 0, 0, 0, 0, currDate.Location())
	allMonthDates := services.NewHashSet[string]()
	for {
		if dateToInclude.Before(appStartDate) {
			break
		}
		allMonthDates.Add(dateToInclude.Format(dateLayout))
		dateToInclude = dateToInclude.AddDate(0, -1, 0)
	}
	for _, event := range allResponse {
		if allMonthDates.Contains(event.Date) {
			eventDate, err := time.Parse(dateLayout, event.Date)
			if err != nil {
				fmt.Printf("Error parsing date %v\n", err.Error())
				continue
			}
			diff := int(currDate.Sub(eventDate).Hours() / 24)
			if diff < 0 {
				fmt.Printf("Skipping future event %v in monthly\n", event.Task)
				continue
			}
			stopAfter, err := time.Parse(dateLayout, event.StopAfter)
			if err == nil {
				if stopAfter.Before(currDate) {
					fmt.Printf("Skipping event %v in monthly due to StopAfter\n", event.Task)
					continue
				}
			}
			ftedResponse = append(ftedResponse, event)
		}
	}
	channel <- ftedResponse
}
func GetMonthlyDataForUserFirstSaturday(userId string, currDateStr string, channel chan<- []models.EventData) {
	defer close(channel)
	serviceRole := os.Getenv("SUPABASE_SERVICE_ROLE")
	apiUrl := os.Getenv("SUPABASE_API_URL")
	allResponse := []models.EventData{}
	ftedResponse := []models.EventData{}
	client, err := supabase.NewClient(apiUrl, serviceRole, &supabase.ClientOptions{})
	if err != nil {
		fmt.Printf("Error connecting to supabase %v\n", err.Error())
		channel <- ftedResponse
		return
	}

	data, _, err := client.From("calendar").Select("id, task, frequency, date, stopAfter, exact", "exact", false).Eq("user_id", userId).Neq("exact", "yes").Eq("frequency", "Monthly").Order("created_at", &postgrest.OrderOpts{Ascending: true}).Execute()

	if err != nil {
		fmt.Printf("Error executing query %v\n", err.Error())
		channel <- ftedResponse
		return
	}
	if err := json.Unmarshal(data, &allResponse); err != nil {
		fmt.Printf("Error unmarshalling results %v\n", err.Error())
	}

	currDate, err := time.Parse(dateLayout, currDateStr)
	if err != nil {
		fmt.Printf("Error parsing current date %v\n", err.Error())
		channel <- ftedResponse
		return
	}
	if currDate.Weekday() == time.Saturday && currDate.Day() <= 7 {
		for _, event := range allResponse {
			eventDate, err := time.Parse(dateLayout, event.Date)
			if err != nil {
				fmt.Printf("Error parsing date %v\n", err.Error())
				continue
			}
			diff := int(currDate.Sub(eventDate).Hours() / 24)
			if diff < 0 {
				fmt.Printf("Skipping future event %v in monthly\n", event.Task)
				continue
			}
			stopAfterDate, err := time.Parse(dateLayout, event.StopAfter)
			if err == nil {
				if stopAfterDate.Before(currDate) {
					fmt.Printf("Skipping event %v in monthly due to StopAfter\n", event.Task)
					continue
				}
			}
			ftedResponse = append(ftedResponse, event)
		}
	}
	channel <- ftedResponse
}
func GetQuarterlyDataForUserWithExactDate(userId string, currDateStr string, channel chan<- []models.EventData) {
	defer close(channel)
	serviceRole := os.Getenv("SUPABASE_SERVICE_ROLE")
	apiUrl := os.Getenv("SUPABASE_API_URL")
	allResponse := []models.EventData{}
	ftedResponse := []models.EventData{}
	client, err := supabase.NewClient(apiUrl, serviceRole, &supabase.ClientOptions{})
	if err != nil {
		fmt.Printf("Error connecting to supabase %v\n", err.Error())
		channel <- ftedResponse
		return
	}

	data, _, err := client.From("calendar").Select("id, task, frequency, date, stopAfter, exact", "exact", false).Eq("user_id", userId).Eq("exact", "yes").Eq("frequency", "Quarterly").Order("created_at", &postgrest.OrderOpts{Ascending: true}).Execute()

	if err != nil {
		fmt.Printf("Error executing query %v\n", err.Error())
		channel <- ftedResponse
		return
	}
	if err := json.Unmarshal(data, &allResponse); err != nil {
		fmt.Printf("Error unmarshalling results %v\n", err.Error())
	}

	currDate, err := time.Parse(dateLayout, currDateStr)
	if err != nil {
		fmt.Printf("Error parsing current date %v\n", err.Error())
		channel <- ftedResponse
		return
	}
	for _, event := range allResponse {
		eventDate, err := time.Parse(dateLayout, event.Date)
		if err != nil {
			fmt.Printf("Error parsing date %v\n", err.Error())
			continue
		}

		diff := int(currDate.Sub(eventDate).Hours() / 24)
		if diff < 0 {
			fmt.Printf("Skipping future event %v in quarterly\n", event.Task)
			continue
		}
		if int(currDate.Sub(eventDate).Hours()/24/30)%3 == 0 {
			stopAfterDate, err := time.Parse(dateLayout, event.StopAfter)
			if err == nil {
				if stopAfterDate.Before(currDate) {
					fmt.Printf("Skipping event %v in quarterly due to StopAfter\n", event.Task)
					continue
				}
			}
			ftedResponse = append(ftedResponse, event)
		}
	}
	channel <- ftedResponse
}
func GetQuarterlyDataForUserFirstSaturday(userId string, currDateStr string, channel chan<- []models.EventData) {
	defer close(channel)
	serviceRole := os.Getenv("SUPABASE_SERVICE_ROLE")
	apiUrl := os.Getenv("SUPABASE_API_URL")
	allResponse := []models.EventData{}
	ftedResponse := []models.EventData{}
	client, err := supabase.NewClient(apiUrl, serviceRole, &supabase.ClientOptions{})
	if err != nil {
		fmt.Printf("Error connecting to supabase %v\n", err.Error())
		channel <- ftedResponse
		return
	}

	data, _, err := client.From("calendar").Select("id, task, frequency, date, stopAfter, exact", "exact", false).Eq("user_id", userId).Neq("exact", "yes").Eq("frequency", "Quarterly").Order("created_at", &postgrest.OrderOpts{Ascending: true}).Execute()

	if err != nil {
		fmt.Printf("Error executing query %v\n", err.Error())
		channel <- ftedResponse
		return
	}
	if err := json.Unmarshal(data, &allResponse); err != nil {
		fmt.Printf("Error unmarshalling results %v\n", err.Error())
	}

	currDate, err := time.Parse(dateLayout, currDateStr)
	if err != nil {
		fmt.Printf("Error parsing current date %v\n", err.Error())
		channel <- ftedResponse
		return
	}
	if currDate.Weekday() == time.Saturday && currDate.Day() <= 7 && currDate.Month()%3 == 1 {
		for _, event := range allResponse {
			eventDate, err := time.Parse(dateLayout, event.Date)
			if err != nil {
				fmt.Printf("Error parsing date %v\n", err.Error())
				continue
			}
			diff := int(currDate.Sub(eventDate).Hours() / 24)
			if diff < 0 {
				fmt.Printf("Skipping future event %v in quarterly\n", event.Task)
				continue
			}
			stopAfterDate, err := time.Parse(dateLayout, event.StopAfter)
			if err == nil {
				if stopAfterDate.Before(currDate) {
					fmt.Printf("Skipping event %v in quarterly due to StopAfter\n", event.Task)
					continue
				}
			}
			ftedResponse = append(ftedResponse, event)
		}
	}
	channel <- ftedResponse
}
func GetHalfYearlyDataForUserWithExactDate(userId string, currDateStr string, channel chan<- []models.EventData) {
	defer close(channel)
	serviceRole := os.Getenv("SUPABASE_SERVICE_ROLE")
	apiUrl := os.Getenv("SUPABASE_API_URL")
	allResponse := []models.EventData{}
	ftedResponse := []models.EventData{}
	client, err := supabase.NewClient(apiUrl, serviceRole, &supabase.ClientOptions{})
	if err != nil {
		fmt.Printf("Error connecting to supabase %v\n", err.Error())
		channel <- ftedResponse
		return
	}

	data, _, err := client.From("calendar").Select("id, task, frequency, date, stopAfter, exact", "exact", false).Eq("user_id", userId).Eq("exact", "yes").Eq("frequency", "Half yearly").Order("created_at", &postgrest.OrderOpts{Ascending: true}).Execute()

	if err != nil {
		fmt.Printf("Error executing query %v\n", err.Error())
		channel <- ftedResponse
		return
	}
	if err := json.Unmarshal(data, &allResponse); err != nil {
		fmt.Printf("Error unmarshalling results %v\n", err.Error())
	}

	currDate, err := time.Parse(dateLayout, currDateStr)
	if err != nil {
		fmt.Printf("Error parsing current date %v\n", err.Error())
		channel <- ftedResponse
		return
	}
	for _, event := range allResponse {
		eventDate, err := time.Parse(dateLayout, event.Date)
		if err != nil {
			fmt.Printf("Error parsing date %v\n", err.Error())
			continue
		}
		diff := int(currDate.Sub(eventDate).Hours() / 24)
		if diff < 0 {
			fmt.Printf("Skipping future event %v in half yearly\n", event.Task)
			continue
		}
		if int(currDate.Sub(eventDate).Hours()/24/30)%6 == 0 {
			stopAfterDate, err := time.Parse(dateLayout, event.StopAfter)
			if err == nil {
				if stopAfterDate.Before(currDate) {
					fmt.Printf("Skipping event %v in half yearly due to StopAfter\n", event.Task)
					continue
				}
			}
			ftedResponse = append(ftedResponse, event)
		}
	}
	channel <- ftedResponse
}
func GetHalfYearlyDataForUserFirstSaturday(userId string, currDateStr string, channel chan<- []models.EventData) {
	defer close(channel)
	serviceRole := os.Getenv("SUPABASE_SERVICE_ROLE")
	apiUrl := os.Getenv("SUPABASE_API_URL")
	allResponse := []models.EventData{}
	ftedResponse := []models.EventData{}
	client, err := supabase.NewClient(apiUrl, serviceRole, &supabase.ClientOptions{})
	if err != nil {
		fmt.Printf("Error connecting to supabase %v\n", err.Error())
		channel <- ftedResponse
		return
	}

	data, _, err := client.From("calendar").Select("id, task, frequency, date, stopAfter, exact", "exact", false).Eq("user_id", userId).Neq("exact", "yes").Eq("frequency", "Half yearly").Order("created_at", &postgrest.OrderOpts{Ascending: true}).Execute()

	if err != nil {
		fmt.Printf("Error executing query %v\n", err.Error())
		channel <- ftedResponse
		return
	}
	if err := json.Unmarshal(data, &allResponse); err != nil {
		fmt.Printf("Error unmarshalling results %v\n", err.Error())
	}

	currDate, err := time.Parse(dateLayout, currDateStr)
	if err != nil {
		fmt.Printf("Error parsing current date %v\n", err.Error())
		channel <- ftedResponse
		return
	}
	if currDate.Weekday() == time.Saturday && currDate.Day() <= 7 && currDate.Month()%6 == 1 {
		for _, event := range allResponse {
			eventDate, err := time.Parse(dateLayout, event.Date)
			if err != nil {
				fmt.Printf("Error parsing date %v\n", err.Error())
				continue
			}
			diff := int(currDate.Sub(eventDate).Hours() / 24)
			if diff < 0 {
				fmt.Printf("Skipping future event %v in half yearly\n", event.Task)
				continue
			}
			stopAfterDate, err := time.Parse(dateLayout, event.StopAfter)
			if err == nil {
				if stopAfterDate.Before(currDate) {
					fmt.Printf("Skipping event %v in half yearly due to StopAfter\n", event.Task)
					continue
				}
			}
			ftedResponse = append(ftedResponse, event)
		}
	}
	channel <- ftedResponse
}
func GetYearlyDataForUserWithExactDate(userId string, currDateStr string, channel chan<- []models.EventData) {
	defer close(channel)
	serviceRole := os.Getenv("SUPABASE_SERVICE_ROLE")
	apiUrl := os.Getenv("SUPABASE_API_URL")
	allResponse := []models.EventData{}
	ftedResponse := []models.EventData{}
	client, err := supabase.NewClient(apiUrl, serviceRole, &supabase.ClientOptions{})
	if err != nil {
		fmt.Printf("Error connecting to supabase %v\n", err.Error())
		channel <- ftedResponse
		return
	}

	data, _, err := client.From("calendar").Select("id, task, frequency, date, stopAfter, exact", "exact", false).Eq("user_id", userId).Eq("exact", "yes").Eq("frequency", "Yearly").Order("created_at", &postgrest.OrderOpts{Ascending: true}).Execute()

	if err != nil {
		fmt.Printf("Error executing query %v\n", err.Error())
		channel <- ftedResponse
		return
	}
	if err := json.Unmarshal(data, &allResponse); err != nil {
		fmt.Printf("Error unmarshalling results %v\n", err.Error())
	}

	currDate, err := time.Parse(dateLayout, currDateStr)
	if err != nil {
		fmt.Printf("Error parsing current date %v\n", err.Error())
		channel <- ftedResponse
		return
	}
	for _, event := range allResponse {
		eventDate, err := time.Parse(dateLayout, event.Date)
		if err != nil {
			fmt.Printf("Error parsing date %v\n", err.Error())
			continue
		}
		diff := int(currDate.Sub(eventDate).Hours() / 24)
		if diff < 0 {
			fmt.Printf("Skipping future event %v in yearly\n", event.Task)
			continue
		}
		if int(currDate.Sub(eventDate).Hours()/24/30)%12 == 0 {
			stopAfterDate, err := time.Parse(dateLayout, event.StopAfter)
			if err == nil {
				if stopAfterDate.Before(currDate) {
					fmt.Printf("Skipping event %v in yearly due to StopAfter\n", event.Task)
					continue
				}
			}
			ftedResponse = append(ftedResponse, event)
		}
	}
	channel <- ftedResponse
}
func GetYearlyDataForUserFirstSaturday(userId string, currDateStr string, channel chan<- []models.EventData) {
	defer close(channel)
	serviceRole := os.Getenv("SUPABASE_SERVICE_ROLE")
	apiUrl := os.Getenv("SUPABASE_API_URL")
	allResponse := []models.EventData{}
	ftedResponse := []models.EventData{}
	client, err := supabase.NewClient(apiUrl, serviceRole, &supabase.ClientOptions{})
	if err != nil {
		fmt.Printf("Error connecting to supabase %v\n", err.Error())
		channel <- ftedResponse
		return
	}

	data, _, err := client.From("calendar").Select("id, task, frequency, date, stopAfter, exact", "exact", false).Eq("user_id", userId).Neq("exact", "yes").Eq("frequency", "Yearly").Order("created_at", &postgrest.OrderOpts{Ascending: true}).Execute()

	if err != nil {
		fmt.Printf("Error executing query %v\n", err.Error())
		channel <- ftedResponse
		return
	}
	if err := json.Unmarshal(data, &allResponse); err != nil {
		fmt.Printf("Error unmarshalling results %v\n", err.Error())
	}

	currDate, err := time.Parse(dateLayout, currDateStr)
	if err != nil {
		fmt.Printf("Error parsing current date %v\n", err.Error())
		channel <- ftedResponse
		return
	}
	if currDate.Weekday() == time.Saturday && currDate.Day() <= 7 && currDate.Month() == 1 {
		for _, event := range allResponse {
			eventDate, err := time.Parse(dateLayout, event.Date)
			if err != nil {
				fmt.Printf("Error parsing date %v\n", err.Error())
				continue
			}
			diff := int(currDate.Sub(eventDate).Hours() / 24)
			if diff < 0 {
				fmt.Printf("Skipping future event %v in yearly\n", event.Task)
				continue
			}
			stopAfterDate, err := time.Parse(dateLayout, event.StopAfter)
			if err == nil {
				if stopAfterDate.Before(currDate) {
					fmt.Printf("Skipping event %v in yearly due to StopAfter\n", event.Task)
					continue
				}
			}
			ftedResponse = append(ftedResponse, event)
		}
	}
	channel <- ftedResponse
}
func sendEmail(to string, from string, htmlBody string, subject string, apiKey string, channel chan<- string) {
	client := resend.NewClient(apiKey)
	params := &resend.SendEmailRequest{
		From:    from,
		To:      []string{to},
		Html:    htmlBody,
		Subject: subject,
	}

	_, err := client.Emails.Send(params)
	if err != nil {
		fmt.Printf("Error sending Email %v\n", err.Error())
		channel <- "Error sending Email"
		return
	}
	channel <- "Email successfully sent"
}

//RG below does not work figure out why and better wayx``
// .And("(current_date-to_date(date,'YYYY-MM-DD'))%7=0", "")
// .And("frequency='Weekly'", "")
//rawFilter := fmt.Sprintf("((current_date - to_date(\"date\", 'YYYY-MM-DD')) %% 7) = 0")
// rawFilter := "\"frequency\"='Weekly'"
