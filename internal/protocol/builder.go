package protocol

func BuildMessage(messageType MessageType, body []byte, headers map[string]string) Message {
	if headers == nil {
		headers = map[string]string{}
	}

	return Message{
		Response: Response{
			Headers: headers,
			Status:  StatusOK,
			Error:   "",
		},
		Type: messageType,
		Body: body,
	}
}

func BuildErrorMessage(messageType MessageType, err error) *Message {
	headers := map[string]string{}
	errorMessage := ""

	if err != nil {
		errorMessage = err.Error()
	}

	return &Message{
		Response: Response{
			Headers: headers,
			Status:  StatusError,
			Error:   errorMessage,
		},
		Type: messageType,
		Body: nil,
	}
}
