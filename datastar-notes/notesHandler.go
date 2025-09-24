package main

import (
	"datastar-notes/components"
	"datastar-notes/models"
	"datastar-notes/services"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/starfederation/datastar-go/datastar"
)

func getNotesHandler(responseWriter http.ResponseWriter, request *http.Request) {
	accessToken := request.Context().Value(services.UserTokenKey).(string)
	channel := make(chan []models.NoteData)
	defer close(channel)
	go services.GetAllNotes(
		accessToken,
		channel,
	)
	data := <-channel
	if request.Header.Get("datastar-request") == "true" {
		sse := datastar.NewSSE(responseWriter, request)
		sse.PatchElementTempl(components.Section(data), datastar.WithUseViewTransitions(true))
		sse.PatchElementTempl(components.ReorderButton(), datastar.WithUseViewTransitions(true))
		sse.PatchElementTempl(components.AddButton(), datastar.WithUseViewTransitions(true))
	} else {
		components.Main(data).Render(request.Context(), responseWriter)
	}
}

func updateNoteHandler(responseWriter http.ResponseWriter, request *http.Request) {
	accessToken := request.Context().Value(services.UserTokenKey).(string)
	channel := make(chan bool)
	defer close(channel)
	var uiRequest models.UINote
	requestBody, err := io.ReadAll(request.Body)
	if err != nil {
		fmt.Printf("Error reading request body in update note: %v\n", err)
		return
	}
	err = json.Unmarshal(requestBody, &uiRequest)
	if err != nil {
		fmt.Printf("Error unmarshalling request body in update note: %v\n", err)
		return
	}
	go services.UpdateContentFromEditorJs(accessToken, uiRequest, channel)
	success := <-channel
	if success {
		replacedId := components.ReplaceHypenInId(uiRequest.Id)
		sse := datastar.NewSSE(responseWriter, request)
		sse.PatchElementTempl(components.CancelEditorChangesButton(models.NoteData{Id: uiRequest.Id, Content: uiRequest.Content}, replacedId))
	}
}

func getTitleEditUIHandler(responseWriter http.ResponseWriter, request *http.Request) {
	var uiRequest models.UINote
	dataStarQueryString := request.URL.Query().Get("datastar")
	err := json.Unmarshal([]byte(dataStarQueryString), &uiRequest)

	if err != nil || uiRequest.Id == "" {
		fmt.Printf("Error unmarshalling request body in title edit ui: %v\n", err)
		return
	}
	replacedId := components.ReplaceHypenInId(uiRequest.Id)
	sse := datastar.NewSSE(responseWriter, request)
	sse.PatchElementTempl(components.NoteTitleEditUI(models.NoteData{
		Id: uiRequest.Id, Title: uiRequest.Title,
	}, replacedId), datastar.WithUseViewTransitions(true))
}

func saveTitleHandler(responseWriter http.ResponseWriter, request *http.Request) {
	accessToken := request.Context().Value(services.UserTokenKey).(string)
	channel := make(chan bool)
	defer close(channel)
	var uiRequest models.UINote
	requestBody, err := io.ReadAll(request.Body)
	if err != nil {
		fmt.Printf("Error reading request body in update note title: %v\n", err)
		return
	}
	err = json.Unmarshal(requestBody, &uiRequest)
	if err != nil {
		fmt.Printf("Error unmarshalling request body in update note title: %v\n", err)
		return
	}
	go services.UpdateTitle(accessToken, uiRequest, channel)
	<-channel
	sse := datastar.NewSSE(responseWriter, request)
	sse.PatchElementTempl(components.NoteTitleUI(models.NoteData{Id: uiRequest.Id, Title: uiRequest.Title}, components.ReplaceHypenInId(uiRequest.Id)), datastar.WithUseViewTransitions(true))
}

func addNoteHandler(responseWriter http.ResponseWriter, request *http.Request) {
	accessToken := request.Context().Value(services.UserTokenKey).(string)
	getMaxOrderChannel := make(chan int)
	defer close(getMaxOrderChannel)
	go services.GetMaxOrder(
		accessToken,
		getMaxOrderChannel,
	)
	maxOrder := <-getMaxOrderChannel
	insertChannel := make(chan models.NoteData)
	defer close(insertChannel)
	go services.InsertNote(accessToken, maxOrder+1, insertChannel)
	note := <-insertChannel
	if note.Id != "" {
		sse := datastar.NewSSE(responseWriter, request)
		sse.PatchElementTempl(components.Editor(note), datastar.WithModeAppend(), datastar.WithSelectorID("section"), datastar.WithUseViewTransitions(true))
		replacedId := components.ReplaceHypenInId(note.Id)
		time.Sleep(200 * time.Millisecond)
		sse.ExecuteScript(fmt.Sprintf("document.getElementById('editorContainer_%v').scrollIntoView();", replacedId), datastar.WithExecuteScriptAutoRemove(true))
	}
}
func deleteNoteHandler(responseWriter http.ResponseWriter, request *http.Request) {
	accessToken := request.Context().Value(services.UserTokenKey).(string)
	channel := make(chan bool)
	defer close(channel)
	var uiRequest models.UINote
	requestBody, err := io.ReadAll(request.Body)
	if err != nil {
		fmt.Printf("Error reading request body in delete note: %v\n", err)
		return
	}
	err = json.Unmarshal(requestBody, &uiRequest)
	if err != nil {
		fmt.Printf("Error unmarshalling request body in delete note: %v\n", err)
		return
	}
	go services.DeleteNote(accessToken, uiRequest, channel)
	result := <-channel
	if result {
		sse := datastar.NewSSE(responseWriter, request)
		sse.RemoveElement("#editorContainer_"+components.ReplaceHypenInId(uiRequest.Id), datastar.WithUseViewTransitions(true))
		sse.PatchSignals([]byte("{showDeleteModal:false}"))
	}
}

func reorderNotesHandler(responseWriter http.ResponseWriter, request *http.Request) {
	accessToken := request.Context().Value(services.UserTokenKey).(string)
	channel := make(chan []models.NoteData)
	defer close(channel)
	go services.GetAllNotes(
		accessToken,
		channel,
	)
	data := <-channel
	sse := datastar.NewSSE(responseWriter, request)
	sse.PatchElementTempl(components.SectionForReorder(data), datastar.WithUseViewTransitions(true))
	sse.PatchElementTempl(components.ViewEditorsUIButton(), datastar.WithUseViewTransitions(true))
	sse.PatchElementTempl(components.AddButtonDisabled(), datastar.WithUseViewTransitions(true))
	sse.ExecuteScript("initializeSortable()", datastar.WithExecuteScriptAutoRemove(true))

}
func saveReorderedNotesHandler(responseWriter http.ResponseWriter, request *http.Request) {
	accessToken := request.Context().Value(services.UserTokenKey).(string)
	getChannel := make(chan []models.NoteData)
	defer close(getChannel)
	go services.GetAllNotes(
		accessToken,
		getChannel,
	)
	var uiRequest models.ReorderNote
	requestBody, err := io.ReadAll(request.Body)
	if err != nil {
		fmt.Printf("Error reading request body in update order: %v\n", err)
		return
	}
	err = json.Unmarshal(requestBody, &uiRequest)
	if err != nil {
		fmt.Printf("Error unmarshalling request body in update order: %v\n", err)
		return
	}
	allNotes := <-getChannel
	saveChannel := []chan bool{}
	fmt.Println(uiRequest)
	if uiRequest.Info.OldIndex < uiRequest.Info.NewIndex {
		for loopIndex := uiRequest.Info.OldIndex; loopIndex <= uiRequest.Info.NewIndex; loopIndex++ {
			note := allNotes[loopIndex]
			channel := make(chan bool)
			saveChannel = append(saveChannel, channel)
			if note.Id == uiRequest.Info.Id {
				go services.UpdateOrder(accessToken, models.ReorderNote{
					Info: models.ReorderNoteInfo{
						Id:       note.Id,
						NewIndex: uiRequest.Info.NewIndex}}, channel)
			} else {
				go services.UpdateOrder(accessToken, models.ReorderNote{
					Info: models.ReorderNoteInfo{
						Id:       note.Id,
						NewIndex: loopIndex - 1,
					}}, channel)
			}
		}
	} else if uiRequest.Info.OldIndex > uiRequest.Info.NewIndex {
		for loopIndex := uiRequest.Info.NewIndex; loopIndex <= uiRequest.Info.OldIndex; loopIndex++ {
			note := allNotes[loopIndex]
			channel := make(chan bool)
			saveChannel = append(saveChannel, channel)
			if note.Id == uiRequest.Info.Id {
				go services.UpdateOrder(accessToken, models.ReorderNote{
					Info: models.ReorderNoteInfo{
						Id:       note.Id,
						NewIndex: uiRequest.Info.NewIndex}}, channel)
			} else {
				go services.UpdateOrder(accessToken, models.ReorderNote{
					Info: models.ReorderNoteInfo{
						Id:       note.Id,
						NewIndex: loopIndex + 1,
					}}, channel)
			}
		}
	}
	for _, ch := range saveChannel {
		<-ch
		close(ch)
	}
	responseWriter.WriteHeader(http.StatusOK)
}
