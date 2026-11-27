package handlers

import (
	"bytes"
	"context"
	"datastar-calendar/internal/models"
	"datastar-calendar/internal/views/components"
	"datastar-calendar/services"
	"datastar-calendar/services/db"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/starfederation/datastar-go/datastar"
)

type ActionHandler struct {
	dbService *db.DBService
	uiSidMap  *sync.Map
}

func NewActionHandler(dbService *db.DBService, uidSidMap *sync.Map) *ActionHandler {
	return &ActionHandler{
		dbService: dbService,
		uiSidMap:  uidSidMap,
	}
}

func (a *ActionHandler) Save(responseWriter http.ResponseWriter, request *http.Request) {
	if strings.ToUpper(request.Method) != "POST" {
		responseWriter.WriteHeader(405)
		return
	}
	token := request.Context().Value(services.TokenKey).(string)
	var clientSignals models.ClientSignals
	datastar.ReadSignals(request, &clientSignals)
	// fmt.Println("Received signals: ", ClientSignals)
	key := services.GenerateUserSessionKey(token, clientSignals.UiSid)

	dateLayout := "2006-01-02"
	if clientSignals.EventId != "" {
		dateLayout = "01/02/2006"
	}
	stopAfterDateLayout := "01/02/2006"

	task := strings.TrimSpace(clientSignals.Task)
	date := clientSignals.Date
	frequency := clientSignals.Frequency
	stopAfter := clientSignals.StopAfter
	exact := clientSignals.Exact

	var dateParsed time.Time
	var stopAfterParsed time.Time
	var dateParseError error
	var stopAfterParseError error

	errors := []string{}

	if date == "" {
		errors = append(errors, "Date is required")
	} else {
		dateParsed, dateParseError = time.Parse(dateLayout, date)
		if dateParseError != nil {
			errors = append(errors, "Date is not in correct format")
		} else if clientSignals.EventId != "" {
			if time.Until(dateParsed).Hours() < -24 {
				errors = append(errors, "Date should not be in the past")
			}
		}

	}
	if len(task) <= 3 {
		errors = append(errors, "Task should be more than 3 characters")
	}
	frequencyAllowed := false
	for _, frequencyFromArray := range services.AllowedFrequencies {
		if frequencyFromArray == frequency {
			frequencyAllowed = true
			break
		}
	}

	if !frequencyAllowed {
		errors = append(errors, "Frequency is not allowed")
	} else if frequency == "Only once" && stopAfter != "" {
		errors = append(errors, "Stop after is not allowed for only once frequency")
	} else if frequency != "Only once" {
		if stopAfter != "" {
			stopAfterParsed, stopAfterParseError = time.Parse(stopAfterDateLayout, stopAfter)
			// fmt.Printf("Parsed stop after date: %v, error: %v\n", stopAfterParsed, err)
			if stopAfterParseError != nil {
				errors = append(errors, "Stop After is not in correct format")
			} else if stopAfterParsed.Sub(dateParsed).Hours() < 24 {
				errors = append(errors, "Stop After date should be after Event date")
			}
		}
	}
	if frequency == "Only once" && exact == "yes" {
		errors = append(errors, "Exact date is not allowed for only once frequency")
	}
	if len(errors) > 0 {
		dataBuffer := new(bytes.Buffer)
		dataShowAttr := "!$_addingTask"
		if clientSignals.EventId != "" {
			dataShowAttr = "!$_savingTask"
		}
		components.SaveEventValidationError(errors, dataShowAttr).Render(context.Background(), dataBuffer)
		if userSession, userSessionExists := a.uiSidMap.Load(key); userSessionExists {
			userSession.(chan models.LongSSEData) <- models.LongSSEData{
				Content: dataBuffer.String(),
				Mode:    datastar.WithModeOuter(),
			}
		}
		return
	}
	var dataFromDbSave models.EventData
	parsedToken, _, tokenParseErr := jwt.NewParser().ParseUnverified(token, jwt.MapClaims{})
	var functionalityErrored bool
	if tokenParseErr != nil {
		fmt.Printf("Error parsing accesstoken %v\n", tokenParseErr)
		functionalityErrored = true
	}
	if !functionalityErrored {
		sub, subjectErr := parsedToken.Claims.GetSubject()
		if subjectErr != nil {
			fmt.Printf("Error get subject from claims %v\n", subjectErr)
			functionalityErrored = true
		}

		if subjectErr == nil {
			channel := make(chan models.EventData)
			if stopAfterParseError == nil && stopAfter != "" {
				// fmt.Printf("Before Parsed stop after: %v\n", stopAfter)
				stopAfter = stopAfterParsed.Format("2006-01-02")
			} else {
				stopAfter = ""
			}
			// fmt.Printf("Parsed stop after: %v\n", stopAfter)
			dataToDB := models.EventData{
				Task:      task,
				Frequency: frequency,
				Date:      dateParsed.Format("2006-01-02"),
				UserId:    sub,
				Exact:     exact,
				StopAfter: stopAfter,
			}
			if clientSignals.EventId != "" {
				dataToDB.Id = clientSignals.EventId
				go a.dbService.UpdateData(token, dataToDB, channel)

			} else {
				go a.dbService.AddData(token, dataToDB, channel)
			}
			dataFromDbSave = <-channel
			if dataFromDbSave.Id == "" {
				functionalityErrored = true
			}
		}
	}
	dataBuffer := new(bytes.Buffer)
	if functionalityErrored {
		if clientSignals.EventId != "" {
			components.EditEventResult(false, task).Render(context.Background(), dataBuffer)
		} else {
			components.AddEventResult(false, task).Render(context.Background(), dataBuffer)
		}
	} else {
		if clientSignals.EventId != "" {
			components.EditEventResult(true, task).Render(context.Background(), dataBuffer)
		} else {
			components.AddEventResult(true, task).Render(context.Background(), dataBuffer)
		}
	}
	if userSession, userSessionExists := a.uiSidMap.Load(key); userSessionExists {
		userSession.(chan models.LongSSEData) <- models.LongSSEData{
			Content: dataBuffer.String(),
			Mode:    datastar.WithModeOuter(),
		}
		if !functionalityErrored {
			eventTableDataUIBuffer := new(bytes.Buffer)
			components.CalendarTableEventItem(dataFromDbSave).Render(context.Background(), eventTableDataUIBuffer)
			switch {
			case clientSignals.EventId == "":
				userSession.(chan models.LongSSEData) <- models.LongSSEData{
					Content:   "{eventId:'',task:'',frequency:'Only once',stopAfter:'',exact:'no'}",
					IsSignals: true,
				}
				userSession.(chan models.LongSSEData) <- models.LongSSEData{
					Content:  eventTableDataUIBuffer.String(),
					Selector: "#eventsContainer-" + dateParsed.Format("2006-01-02"),
					Mode:     datastar.WithModeAppend(),
				}
			case clientSignals.EventId != "" && !clientSignals.DateChanged:
				userSession.(chan models.LongSSEData) <- models.LongSSEData{
					Content: eventTableDataUIBuffer.String(),
				}
			case clientSignals.EventId != "" && clientSignals.DateChanged:
				userSession.(chan models.LongSSEData) <- models.LongSSEData{
					IsRemove: true,
					Selector: "#event-" + clientSignals.EventId,
				}
				userSession.(chan models.LongSSEData) <- models.LongSSEData{
					Content:  eventTableDataUIBuffer.String(),
					Selector: "#eventsContainer-" + dateParsed.Format("2006-01-02"),
					Mode:     datastar.WithModeAppend(),
				}
			}

		}
	}
}

func (a *ActionHandler) Delete(responseWriter http.ResponseWriter, request *http.Request) {
	token := request.Context().Value(services.TokenKey).(string)
	var clientSignals models.ClientSignals
	datastar.ReadSignals(request, &clientSignals)
	// fmt.Println("Received signals: ", ClientSignals)
	if clientSignals.EventId == "" {
		responseWriter.WriteHeader(400)
		return
	}
	channel := make(chan bool)
	go a.dbService.DeleteEvent(models.DeleteEvent{AccessToken: token, Id: clientSignals.EventId}, channel)
	ret := <-channel
	key := services.GenerateUserSessionKey(token, clientSignals.UiSid)
	if userSession, userSessionExists := a.uiSidMap.Load(key); userSessionExists {
		dataBuffer := new(bytes.Buffer)
		if !ret {
			components.DeleteEventResult(false).Render(context.Background(), dataBuffer)
			userSession.(chan models.LongSSEData) <- models.LongSSEData{
				Content: dataBuffer.String(),
				Mode:    datastar.WithModeOuter(),
			}

		} else {
			components.DeleteEventResult(true).Render(context.Background(), dataBuffer)
			userSession.(chan models.LongSSEData) <- models.LongSSEData{
				Content: dataBuffer.String(),
				Mode:    datastar.WithModeOuter(),
			}

			userSession.(chan models.LongSSEData) <- models.LongSSEData{
				IsRemove: true,
				Selector: "#event-" + clientSignals.EventId,
			}
			userSession.(chan models.LongSSEData) <- models.LongSSEData{
				Content:   "{showDeleteConfirm:false}",
				IsSignals: true,
			}
		}
	}

}

func (a *ActionHandler) UpdateStatus(responseWriter http.ResponseWriter, request *http.Request) {
	if strings.ToUpper(request.Method) != "POST" {
		responseWriter.WriteHeader(405)
		return
	}

	token := request.Context().Value(services.TokenKey).(string)
	var clientSignals models.ClientSignals
	datastar.ReadSignals(request, &clientSignals)
	clientSignals.EventId = strings.TrimSpace(clientSignals.EventId)
	if clientSignals.EventId == "" {
		responseWriter.WriteHeader(400)
		return
	}
	// fmt.Println("Received signals: ", ClientSignals)
	key := services.GenerateUserSessionKey(token, clientSignals.UiSid)

	updateChannel := make(chan bool)
	go a.dbService.UpdateIsActive(models.UpdateIsActiveEvent{
		Id:          clientSignals.EventId,
		IsActive:    clientSignals.IsActive,
		AccessToken: token,
	}, updateChannel)

	success := <-updateChannel
	var manageEventsPageData models.ManageEventsPageData
	if success {
		getDataChannel := make(chan []models.EventData)
		go a.dbService.GetAllNotOnlyOnceFrequencyData(token, getDataChannel)
		eventsData := <-getDataChannel
		manageEventsPageData = services.ConstructManageEventsPageData(eventsData, clientSignals.ManagePageFilter)
	}
	if userSession, userSessionExists := a.uiSidMap.Load(key); userSessionExists {
		switch success {
		case true:
			toggleButtonBuffer := new(bytes.Buffer)
			components.ToggleEventStatusButton(clientSignals.EventId, clientSignals.IsActive).Render(context.Background(), toggleButtonBuffer)
			userSession.(chan models.LongSSEData) <- models.LongSSEData{
				Content:           toggleButtonBuffer.String(),
				Mode:              datastar.WithModeOuter(),
				UseViewTransition: true,
			}
			filterActiveEventsButtonBuffer := new(bytes.Buffer)
			filterInactiveEventsButtonBuffer := new(bytes.Buffer)
			components.FilterActiveEventsButton(manageEventsPageData.ActiveDataCount).Render(context.Background(), filterActiveEventsButtonBuffer)
			components.FilterInactiveEventsButton(manageEventsPageData.InactiveDataCount).Render(context.Background(), filterInactiveEventsButtonBuffer)
			userSession.(chan models.LongSSEData) <- models.LongSSEData{
				Content: filterActiveEventsButtonBuffer.String(),
				Mode:    datastar.WithModeOuter(),
				// UseViewTransition: true,
			}
			userSession.(chan models.LongSSEData) <- models.LongSSEData{
				Content: filterInactiveEventsButtonBuffer.String(),
				Mode:    datastar.WithModeOuter(),
				// UseViewTransition: true,
			}
			if (clientSignals.IsActive && clientSignals.ManagePageFilter == models.INACTIVE_STATUS) ||
				(!clientSignals.IsActive && clientSignals.ManagePageFilter == models.ACTIVE_STATUS) {
				userSession.(chan models.LongSSEData) <- models.LongSSEData{
					IsRemove:          true,
					Selector:          "#tr-" + clientSignals.EventId,
					UseViewTransition: true,
				}
				tableFooterBuffer := new(bytes.Buffer)
				switch clientSignals.ManagePageFilter {
				case models.INACTIVE_STATUS:
					components.ManageEventsTableFooter(manageEventsPageData.InactiveDataCount).Render(context.Background(), tableFooterBuffer)
				case models.ACTIVE_STATUS:
					components.ManageEventsTableFooter(manageEventsPageData.ActiveDataCount).Render(context.Background(), tableFooterBuffer)
				}
				userSession.(chan models.LongSSEData) <- models.LongSSEData{
					Content:           tableFooterBuffer.String(),
					Mode:              datastar.WithModeOuter(),
					UseViewTransition: true,
				}
			}
		case false:
			userSession.(chan models.LongSSEData) <- models.LongSSEData{
				Content:   "{showErrorMessage:true,errorMessage:'Couldn\\'t update the reminder. Please try again.'}",
				IsSignals: true,
			}
			time.Sleep(3 * time.Second)
			userSession.(chan models.LongSSEData) <- models.LongSSEData{
				Content:   "{showErrorMessage:false}",
				IsSignals: true,
			}
		}
	}

}
