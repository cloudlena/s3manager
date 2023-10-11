package s3manager

import "encoding/base64"

func decodeVariable(input string) (string, error) {
	decodedInput, decodeError := base64.StdEncoding.DecodeString(input)
	if decodeError != nil {
		return "", decodeError
	}
	return string(decodedInput), nil
}
