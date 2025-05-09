package repository

import "errors"

// Common repository errors
var (
	// ErrBoardNotFound is returned when a board is not found
	ErrBoardNotFound = errors.New("board not found")
	
	// ErrLabelNotFound is returned when a label is not found
	ErrLabelNotFound = errors.New("label not found")
)