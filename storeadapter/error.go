package storeadapter

import (
	"errors"
)

var (
	ErrorKeyNotFound        = errors.New("The requested key could not be found")
	ErrorNodeIsDirectory    = errors.New("Node is a directory, not a leaf")
	ErrorNodeIsNotDirectory = errors.New("Node is a leaf, not a directory")
	ErrorTimeout            = errors.New("Store request timed out")
	ErrorInvalidFormat      = errors.New("Node has invalid format")
)
