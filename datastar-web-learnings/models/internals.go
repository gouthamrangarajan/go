package models

type Functionality int

const (
	_ Functionality = iota
	LANDING_PAGE_UI
	LOAD_MORE_FUNCTIONALITY
	SEARCH_FUNCTIONALITY
	CLEAR_SEARCH_FUNCTIONALITY
	INVALID_SEARCH_FUNCTIONALITY
	NO_DATA_FOUND_FUNCTIONALITY
	ADD_PAGE_UI
	TAGS_UI
	ADD_VIDEO_VALIDATION_ERROR_FUNCTIONALITY
	ADD_VIDEO_SUCCESS_FUNCTIONALITY
	ADD_VIDEO_ERROR_FUNCTIONALITY
	DELETE_VIDEO_SUCCESS_FUNCTIONALITY
	DELETE_VIDEO_ERROR_FUNCTIONALITY
)

type LongSSEData struct {
	FunctionalityVal      Functionality
	SearchTxt             string
	OffsetVal             int
	Data                  []VideoResponse
	UserAgent             string
	Tags                  []string
	AddVideoErrorMessages []string
	AddVideoErrorsignals  string
	VideoDeleted          string
}
