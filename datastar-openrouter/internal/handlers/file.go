package handlers

import (
	"bytes"
	"context"
	"datastar-openrouter/internal/models"
	"datastar-openrouter/internal/views/components"
	"datastar-openrouter/services"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"

	"sync"

	"github.com/starfederation/datastar-go/datastar"
)

type FileHandler struct {
	uisidMap      *sync.Map
	helperService *services.HelperService
	dbService     *services.DBService
	userIdKey     string
	imgRegex      *regexp.Regexp
	pdfRegex      *regexp.Regexp
}

func NewFileHandler(uisidMap *sync.Map, helperService *services.HelperService, dbService *services.DBService) *FileHandler {
	imgRegex, err := regexp.Compile(os.Getenv("IMG_REGEX"))
	if err != nil {
		fmt.Printf("Error compiling IMG_REGEX: %v\n", err)
	}
	pdfRegex, err := regexp.Compile(os.Getenv("PDF_REGEX"))
	if err != nil {
		fmt.Printf("Error compiling PDF_REGEX: %v\n", err)
	}
	return &FileHandler{
		uisidMap:      uisidMap,
		helperService: helperService,
		dbService:     dbService,
		userIdKey:     os.Getenv("USER_ID_KEY"),
		imgRegex:      imgRegex,
		pdfRegex:      pdfRegex,
	}
}

func (h *FileHandler) HandleGetImage(responseWriter http.ResponseWriter, request *http.Request) {
	userId := request.Context().Value(h.userIdKey).(string)
	var clientSignal models.ClientSignals
	datastar.ReadSignals(request, &clientSignal)
	userSessionKey := h.helperService.GenerateUserSessionKey(userId, clientSignal.UiSid)

	fileDataChannel := make(chan models.ChatConversation)
	go h.dbService.GetChatConversationFileData(models.GetConversationRequest{SessionId: clientSignal.SessionId,
		ConversationId: clientSignal.MessageIdToFetchImage, UserId: userId}, fileDataChannel)

	converstationWithFileData := <-fileDataChannel
	if converstationWithFileData.FileData != "" {
		imageDataBuffer := new(bytes.Buffer)
		components.ChatMessageImageDisplayOnHover(converstationWithFileData).Render(context.Background(), imageDataBuffer)
		if userSession, userSessionExists := h.uisidMap.Load(userSessionKey); userSessionExists {
			userSession.(chan models.LongSSEData) <- models.LongSSEData{
				Content: imageDataBuffer.String(),
			}
			userSession.(chan models.LongSSEData) <- models.LongSSEData{
				IsSignal: true,
				Content: `{showImage_` + strconv.Itoa(clientSignal.MessageIdToFetchImage) + `:true,
							imageFetched_` + strconv.Itoa(clientSignal.MessageIdToFetchImage) + `:true}`,
			}
		}
	}
}

func (h *FileHandler) HandleFileUpload(responseWriter http.ResponseWriter, request *http.Request) {
	userId := request.Context().Value(h.userIdKey).(string)
	var clientSignal models.ClientSignals
	datastar.ReadSignals(request, &clientSignal)
	if len(clientSignal.FileData) != 1 {
		http.Error(responseWriter, "Bad Request", http.StatusBadRequest)
		return
	}
	fileDataForRegex := "data:" + clientSignal.FileData[0].Mime + ";base64," + clientSignal.FileData[0].Contents
	fileName := clientSignal.FileData[0].Name
	imgMatches := h.imgRegex.FindStringSubmatch(fileDataForRegex)
	pdfMatches := h.pdfRegex.FindStringSubmatch(fileDataForRegex)

	userSessionKey := h.helperService.GenerateUserSessionKey(userId, clientSignal.UiSid)
	if userSession, userSessionExists := h.uisidMap.Load(userSessionKey); userSessionExists {
		if (clientSignal.FileData[0].Mime == "application/pdf" && len(pdfMatches) != 2) ||
			(clientSignal.FileData[0].Mime != "application/pdf" && len(imgMatches) != 4) {
			userSession.(chan models.LongSSEData) <- models.LongSSEData{
				Content:  "{fileData:'',fileUploading:false}",
				IsSignal: true,
			}
			fileName = ""
			userSession.(chan models.LongSSEData) <- models.LongSSEData{
				Content: "Invalid file type. Please upload an file with type (JPG, PNG, WEBP, GIF, PDF)",
				IsError: true,
			}
			return
		}
		decodedBytes, err := base64.StdEncoding.DecodeString(clientSignal.FileData[0].Contents)
		if err != nil || len(decodedBytes) > 6*1024*1024 {
			userSession.(chan models.LongSSEData) <- models.LongSSEData{
				Content:  "{fileData:'',fileUploading:false}",
				IsSignal: true,
			}
			userSession.(chan models.LongSSEData) <- models.LongSSEData{
				Content: "File too large. Please upload a file smaller than 6 MB.",
				IsError: true,
			}
			fileName = ""
			return
		}
		bytesBuffer := new(bytes.Buffer)
		components.FileAttachmentDisplay(fileName).Render(context.Background(), bytesBuffer)
		userSession.(chan models.LongSSEData) <- models.LongSSEData{
			Content:           bytesBuffer.String(),
			UseViewTransition: true,
		}
		userSession.(chan models.LongSSEData) <- models.LongSSEData{
			Content:  "{fileUploading:false}",
			IsSignal: true,
		}
	}
}

func (h *FileHandler) HandleRemoveUploadedFile(responseWriter http.ResponseWriter, request *http.Request) {
	userId := request.Context().Value(h.userIdKey).(string)
	var clientSignal models.ClientSignals
	datastar.ReadSignals(request, &clientSignal)
	bytesBuffer := new(bytes.Buffer)

	userSessionKey := h.helperService.GenerateUserSessionKey(userId, clientSignal.UiSid)
	if userSession, userSessionExists := h.uisidMap.Load(userSessionKey); userSessionExists {
		components.FileAttachmentDisplay("").Render(context.Background(), bytesBuffer)
		userSession.(chan models.LongSSEData) <- models.LongSSEData{
			Content:           bytesBuffer.String(),
			UseViewTransition: true,
		}
		userSession.(chan models.LongSSEData) <- models.LongSSEData{
			Content:  "{fileData:'',fileUploading:false}",
			IsSignal: true,
		}
	}
}
