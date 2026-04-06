package models

type Functionality int

const (
	_ Functionality = iota
	LOAD_MORE_FUNCTIONALITY
	SEARCH_FUNCTIONALITY
	CLEAR_SEARCH_FUNCTIONALITY
	INVALID_SEARCH_FUNCTIONALITY
	NO_DATA_FOUND_FUNCTIONALITY
)

type LongSSEData struct {
	FunctionalityVal Functionality
	SearchTxt        string
	OffsetVal        int
	Data             []VideoResponse
}
