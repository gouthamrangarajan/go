package models

type Functionality int

const (
	_ Functionality = iota
	LOAD_MORE_FUNCTIONALITY
	SEARCH_FUNCTIONALITY
	CLEAR_SEARCH_FUNCTIONALITY
	INVALID_SEARCH_FUNCTIONALITY
	NO_DATA_FOUND_FUNCTIONALITY
	ADD_PAGE_UI
	LANDING_PAGE_UI
	TAGS_UI
)

type LongSSEData struct {
	FunctionalityVal Functionality
	SearchTxt        string
	OffsetVal        int
	Data             []VideoResponse
	UserAgent        string
	Tags             []string
}
