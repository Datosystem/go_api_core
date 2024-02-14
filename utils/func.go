package utils

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/Datosystem/go_api_core/message"
)

// Convert the input data to a byte array
func ConvertArrayByte(data interface{}) []byte {
	reqBodyBytes := new(bytes.Buffer)
	json.NewEncoder(reqBodyBytes).Encode(data)
	return reqBodyBytes.Bytes()
}

/*
	It makes an HTTP call and returns the response body:

*       ${ @method } The method to use in the call, e.g., http.MethodGet
*       ${ @url } URL to call
*       ${ @header } Headers to set in the request
*       ${ @jsonData } The JSON to send to the destination
*/
func MakeHTTPCall(method, url string, headers *http.Header, jsonData []byte) ([]byte, error) {
	client := &http.Client{}
	req, _ := http.NewRequest(method, url, bytes.NewBuffer(jsonData))
	if headers != nil {
		req.Header = *headers
	}
	if headers == nil || headers.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(res.Body)

	if err != nil {
		return nil, err
	}
	if res.StatusCode >= 400 && res.StatusCode <= 500 {
		return nil, &message.Msg{Status: res.StatusCode, Message: buf.String()}
	}
	return buf.Bytes(), nil
}

func StringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}
