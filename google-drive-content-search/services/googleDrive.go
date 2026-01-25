package services

import (
	"context"
	"encoding/json"
	"fmt"
	"google-drive-content-search/models"
	"io"
	"net/http"
	"os"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

func GetFilesFromDrive(channel chan<- []models.FileData) {
	filesToReturn := []models.FileData{}
	driverService, err := createDriverService(false)
	if err != nil {
		channel <- filesToReturn
		return
	}
	folderId := os.Getenv("DRIVE_FOLDER_ID")

	var allFiles []*drive.File

	allFiles, err = recursiveGetFilesData(driverService, folderId, allFiles)
	if err != nil && strings.Contains(err.Error(), "Token has been expired or revoked") {
		fmt.Printf("OAuth2 token expired, generating new token...\n")
		driverService, err = createDriverService(true)
		if err != nil {
			channel <- filesToReturn
			return
		}
		allFiles, err = recursiveGetFilesData(driverService, folderId, allFiles)
		if err != nil {
			channel <- filesToReturn
			return
		}
	}
	if len(allFiles) == 0 {
		channel <- filesToReturn
		return
	}
	// if len(allFiles) != 0 {
	// 	fmt.Printf("Total files found in Drive folder and subfolders: %v\n", len(allFiles))
	// }

	// Limiting to first n files for demo purposes
	// filesData.Files = filesData.Files[0:2]
	extractDataChannel := make(chan models.FileData, len(allFiles))
	defer close(extractDataChannel)
	for _, file := range allFiles {
		// fmt.Printf("Found file: %s (ID: %s, MimeType: %s)\n", file.Name, file.Id, file.MimeType)
		switch file.MimeType {
		case "application/vnd.google-apps.spreadsheet",
			"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":
			{
				go func() {
					fileDataToAppend, _ := extractSpreedSheetData(driverService, file)
					extractDataChannel <- fileDataToAppend
				}()
			}
		case "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
			"application/msword", "application/pdf",
			"image/png", "image/jpeg", "image/jpg":
			{
				go func() {
					fileDataToAppend, _ := extractWordOrPDFData(folderId, driverService, file)
					extractDataChannel <- fileDataToAppend
				}()
			}
		default:
			{
				fmt.Printf("Skipping unsupported file type: %s (MimeType: %s)\n", file.Name, file.MimeType)
				extractDataChannel <- models.FileData{}
			}
		}
	}
	for range len(allFiles) {
		fileData := <-extractDataChannel
		if strings.TrimSpace(fileData.ExtractedText) != "" {
			filesToReturn = append(filesToReturn, fileData)
		}
	}
	channel <- filesToReturn
}

func generateTokenFromConfig(oauthConfig *oauth2.Config) {
	authURL := oauthConfig.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n\n", authURL)
	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		fmt.Println("Error reading authorization code:", err)
		return
	}
	token, err := oauthConfig.Exchange(context.Background(), authCode)
	if err != nil {
		fmt.Println("Error exchanging authorization code for token:", err)
		return
	}
	fileToWrite, err := os.OpenFile(TOKEN_FILE, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		fmt.Println("Error creating token file:", err)
		fmt.Printf("token: %v\n", token)
	}
	defer fileToWrite.Close()
	err = json.NewEncoder(fileToWrite).Encode(token)
	if err != nil {
		fmt.Println("Error encoding token to file:", err)
	}
}
func getClient(oauthConfig *oauth2.Config) *http.Client {
	file, err := os.Open(TOKEN_FILE)
	if err != nil {
		fmt.Println("Error opening token file:", err)
		return nil
	}
	defer file.Close()
	token := &oauth2.Token{}
	err = json.NewDecoder(file).Decode(token)
	if err != nil {
		fmt.Println("Error decoding token from file:", err)
		return nil
	}
	return oauthConfig.Client(context.Background(), token)
}
func createDriverService(tokenExpired bool) (*drive.Service, error) {
	clientSecret, err := os.ReadFile(CLIENT_SECRET_FILE)
	if err != nil {
		fmt.Println("Error reading client secret file:", err)
		return nil, err
	}
	oauthConfig, err := google.ConfigFromJSON(clientSecret, drive.DriveScope)
	if err != nil {
		fmt.Println("Error creating OAuth2 config:", err)
		return nil, err
	}
	oauthClient := getClient(oauthConfig)
	if oauthClient == nil || tokenExpired {
		fmt.Println("Error obtaining OAuth2 client")
		generateTokenFromConfig(oauthConfig)
		oauthClient = getClient(oauthConfig)
	}
	if oauthClient == nil {
		fmt.Println("Error obtaining OAuth2 client after token generation")
		return nil, err
	}
	driverService, err := drive.NewService(context.Background(), option.WithHTTPClient(oauthClient))
	if err != nil {
		fmt.Println("Error creating Drive service:", err)
		return nil, err
	}
	return driverService, nil
}
func recursiveGetFilesData(driverService *drive.Service, folderId string, allFiles []*drive.File) ([]*drive.File, error) {
	filesData, err := driverService.Files.List().Q(fmt.Sprintf("'%s' in parents", folderId)).Do()
	if err != nil {
		fmt.Println("Error retrieving files:", err)
		return allFiles, err
	}
	allFiles = append(allFiles, filesData.Files...)
	for _, file := range filesData.Files {
		if file.MimeType == "application/vnd.google-apps.folder" {
			allFiles, err = recursiveGetFilesData(driverService, file.Id, allFiles)
		}
	}
	return allFiles, err
}

func extractSpreedSheetData(driverService *drive.Service, file *drive.File) (models.FileData, error) {
	retVal := models.FileData{
		Name:     file.Name,
		Id:       file.Id,
		MimeType: file.MimeType,
	}
	downloadResult, err := driverService.Files.Export(file.Id, "text/csv").Download()
	if err != nil {
		fmt.Println("Error downloading spreadsheet file:", err)
		return retVal, err
	}
	respBody, err := io.ReadAll(downloadResult.Body)
	if err != nil {
		fmt.Println("Error reading download body:", err)
		return retVal, err
	}
	retVal.Data = respBody
	retVal.ExtractedText = string(respBody)
	if strings.TrimSpace(retVal.ExtractedText) != "" {
		openRouterChannel := make(chan models.OpenRouterResponse)
		defer close(openRouterChannel)
		openRouterRequest := models.OpenRouterRequest{
			Messages: []models.OpenRouterRequestMessage{
				{
					Role:    "user",
					Content: fmt.Sprintf(OPEN_ROUTER_GENERAL_PROMPT+OPEN_ROUTER_CSV_PROMPT, retVal.ExtractedText),
				},
			}}
		go CallOpenRouter(openRouterRequest, openRouterChannel)
		openRouterResponse := <-openRouterChannel
		if len(openRouterResponse.Choices) > 0 {
			retVal.ExtractedTextMarkdown = openRouterResponse.Choices[0].Message.Content
		}
	}
	return retVal, err
}
func extractWordOrPDFData(folderIdToSaveNewFile string, driverService *drive.Service, file *drive.File) (models.FileData, error) {
	retVal := models.FileData{
		Name:     file.Name,
		Id:       file.Id,
		MimeType: file.MimeType,
	}
	downloadResult, err := driverService.Files.Get(file.Id).Download()
	if err != nil {
		fmt.Println("Error downloading file:", err)
		return retVal, err
	}
	respBody, err := io.ReadAll(downloadResult.Body)
	if err != nil {
		fmt.Println("Error reading download body:", err)
		return retVal, err
	}
	retVal.Data = respBody
	newFile, err := driverService.Files.Copy(file.Id, &drive.File{
		Name:     file.Name + "_ocr",
		MimeType: "application/vnd.google-apps.document",
		Parents:  []string{folderIdToSaveNewFile},
		DriveId:  file.DriveId,
	}).Do()
	if err != nil {
		fmt.Printf("Error creating OCR File for %v , error:%v\n", file.Name, err)
		return retVal, err
	}
	ocrDownloadResult, err := driverService.Files.Export(newFile.Id, "text/plain").Download()
	if err != nil {
		fmt.Printf("Error downloading OCR file for %v error:%v\n", file.Name, err)
		return retVal, err
	}
	ocrRespBody, err := io.ReadAll(ocrDownloadResult.Body)
	if err != nil {
		fmt.Println("Error reading OCR download body:", err)
		return retVal, err
	}
	// fmt.Printf("Generated OCR Text for file %s\n", file.Name)
	retVal.OCRText = string(ocrRespBody)
	err = driverService.Files.Delete(newFile.Id).Do()
	if err != nil {
		fmt.Println("Error deleting OCR file:", err)
	}
	retVal.ExtractedText = retVal.OCRText
	if strings.TrimSpace(retVal.ExtractedText) != "" {
		openRouterChannel := make(chan models.OpenRouterResponse)
		defer close(openRouterChannel)
		openRouterRequest := models.OpenRouterRequest{
			Messages: []models.OpenRouterRequestMessage{
				{
					Role:    "user",
					Content: fmt.Sprintf(OPEN_ROUTER_GENERAL_PROMPT+OPEN_ROUTER_OCR_PROMPT, retVal.ExtractedText),
				},
			}}
		go CallOpenRouter(openRouterRequest, openRouterChannel)
		openRouterResponse := <-openRouterChannel
		if len(openRouterResponse.Choices) > 0 {
			retVal.ExtractedTextMarkdown = openRouterResponse.Choices[0].Message.Content
		}
	}
	return retVal, err
}
