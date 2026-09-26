package handlers

import (
	"bytes"
	"context"
	"datastar-openrouter/internal/models"
	"datastar-openrouter/internal/views/components/modals"
	"datastar-openrouter/services"
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/starfederation/datastar-go/datastar"
)

type SyncHandler struct {
	uisidMap             *sync.Map
	userIdCodeMap        *sync.Map
	userIdKey            string
	helperService        *services.HelperService
	authorizationService *services.AuthorizationService
	dbService            *services.DBService
}

func NewSyncHandler(uisidMap *sync.Map, helperService *services.HelperService,
	userIdCodeMap *sync.Map, authorizationService *services.AuthorizationService,
	dbService *services.DBService) *SyncHandler {
	return &SyncHandler{
		uisidMap:             uisidMap,
		userIdKey:            os.Getenv("USER_ID_KEY"),
		helperService:        helperService,
		userIdCodeMap:        userIdCodeMap,
		authorizationService: authorizationService,
		dbService:            dbService,
	}
}

func (s *SyncHandler) HandleInitSync(responseWriter http.ResponseWriter, request *http.Request) {
	userId := request.Context().Value(s.userIdKey).(string)
	bytesBuffer := new(bytes.Buffer)
	modals.SyncDeviceModalDefaultActions().Render(context.Background(), bytesBuffer)
	var clientSignal models.ClientSignals
	datastar.ReadSignals(request, &clientSignal)
	userSessionKey := s.helperService.GenerateUserSessionKey(userId, clientSignal.UiSid)
	if userSession, userSessionExists := s.uisidMap.Load(userSessionKey); userSessionExists {
		userSession.(chan models.LongSSEData) <- models.LongSSEData{
			Content:           bytesBuffer.String(),
			Mode:              datastar.WithModeOuter(),
			UseViewTransition: true,
		}
		userSession.(chan models.LongSSEData) <- models.LongSSEData{
			IsSignal: true,
			Content:  "{syncCode:''}",
		}
	}
}

func (s *SyncHandler) HandleGenerateSyncCode(responseWriter http.ResponseWriter, request *http.Request) {
	userId := request.Context().Value(s.userIdKey).(string)
	bytesBuffer := new(bytes.Buffer)
	var val string
	var err error
	if val, err = s.helperService.GenerateManualCode(8); err != nil {
		val = "12345768"
	}
	_, loaded := s.userIdCodeMap.LoadOrStore(userId, val)
	if !loaded {
		fmt.Printf("Unable to store Generated new sync code for user %s:\n", userId)
	}
	modals.SyncDeviceModalDisplayCode(val).Render(context.Background(), bytesBuffer)
	var clientSignal models.ClientSignals
	datastar.ReadSignals(request, &clientSignal)
	userSessionKey := s.helperService.GenerateUserSessionKey(userId, clientSignal.UiSid)
	if userSession, userSessionExists := s.uisidMap.Load(userSessionKey); userSessionExists {
		userSession.(chan models.LongSSEData) <- models.LongSSEData{
			Content:           bytesBuffer.String(),
			Mode:              datastar.WithModeOuter(),
			UseViewTransition: true,
		}
	}
}

func (s *SyncHandler) HandleEntersyncForm(responseWriter http.ResponseWriter, request *http.Request) {
	userId := request.Context().Value(s.userIdKey).(string)
	bytesBuffer := new(bytes.Buffer)
	modals.SyncDeviceModalEnterCode().Render(context.Background(), bytesBuffer)
	var clientSignal models.ClientSignals
	datastar.ReadSignals(request, &clientSignal)
	userSessionKey := s.helperService.GenerateUserSessionKey(userId, clientSignal.UiSid)
	if userSession, userSessionExists := s.uisidMap.Load(userSessionKey); userSessionExists {
		userSession.(chan models.LongSSEData) <- models.LongSSEData{
			Content:           bytesBuffer.String(),
			Mode:              datastar.WithModeOuter(),
			UseViewTransition: true,
		}
	}
}

func (s *SyncHandler) HandleVerifySyncCode(responseWriter http.ResponseWriter, request *http.Request) {
	userIdInRequest := request.Context().Value(s.userIdKey).(string)
	clientSignal := models.ClientSignals{}
	datastar.ReadSignals(request, &clientSignal)
	userSessionKey := s.helperService.GenerateUserSessionKey(userIdInRequest, clientSignal.UiSid)

	// fmt.Printf("userId and synccode %v:%v\n", userIdInRequest, clientSignal.SyncCode)

	userIdWhoGeneratedCode := ""
	s.userIdCodeMap.Range(func(userIdInMap, value any) bool {
		if value.(string) == clientSignal.SyncCode {
			userIdWhoGeneratedCode = userIdInMap.(string)
			return false
		}
		return true
	})

	if userIdWhoGeneratedCode == "" || clientSignal.SyncCode == "" {
		http.Error(responseWriter, "Invalid Request.", http.StatusBadRequest)
		return
	}

	// fmt.Printf("from userId and to userId %v:%v\n", userIdInRequest, userIdWhoGeneratedCode)
	if userIdWhoGeneratedCode == userIdInRequest {
		if userSession, userSessionExists := s.uisidMap.Load(userSessionKey); userSessionExists {
			userSession.(chan models.LongSSEData) <- models.LongSSEData{
				Content: "You cannot sync with the same device.",
				IsError: true,
			}
		}
		return
	}

	userIdUpdateChannel := make(chan int)
	go s.dbService.UpdateChatSessionsUserId(userIdInRequest, userIdWhoGeneratedCode, userIdUpdateChannel)
	updatedRecords := <-userIdUpdateChannel

	if userSession, userSessionExists := s.uisidMap.Load(userSessionKey); userSessionExists {
		if updatedRecords == 0 {
			userSession.(chan models.LongSSEData) <- models.LongSSEData{
				Content: "Error Syncing. Pleasse try again later",
				IsError: true,
			}
			userSession.(chan models.LongSSEData) <- models.LongSSEData{
				Content:           "{showSyncDeviceModal:false}",
				IsSignal:          true,
				UseViewTransition: true,
			}
			return
		}
		cookie, err := s.authorizationService.GenerateUserIdCookie(userIdWhoGeneratedCode)
		if err == nil {
			http.SetCookie(responseWriter, &cookie)
		}
		// flush everything to browser so that window.location.reload works properly
		if f, ok := responseWriter.(http.Flusher); ok {
			// fmt.Printf("reached here\n")
			responseWriter.WriteHeader(http.StatusOK)
			f.Flush()
		}
		userSession.(chan models.LongSSEData) <- models.LongSSEData{
			Content:  "window.location.reload()",
			IsScript: true,
		}
	}

}
