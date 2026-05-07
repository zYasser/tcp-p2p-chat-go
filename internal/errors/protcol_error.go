package errors

import "errors"

var SerializationReadError = errors.New("failed to read incoming payload")
var SerializationError = errors.New("failed to serialize")
var FailedToConnect =errors.New("Failed To Connect")
var FailedToEncodeMessage = errors.New("Failed To encode message")
