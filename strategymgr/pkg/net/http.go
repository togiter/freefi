package net

import (
	"bytes"
	"io"
	"net/http"
)

func HttpPost(jsonBytes []byte, url string) ([]byte, error) {
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonBytes))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}
