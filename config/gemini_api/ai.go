package gemini_api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

type AIClient struct {
	APIURL string
}

func NewAIClient() *AIClient {
	return &AIClient{
		APIURL: os.Getenv("GEMINI_API_URL"),
	}
}

type Part struct {
	Text string `json:"text"`
}

type Content struct {
	Parts []Part `json:"parts"`
}

type Candidate struct {
	Content Content `json:"content"`
}

type AIResponse struct {
	Candidates []Candidate `json:"candidates"`
}

func (client *AIClient) GetResponse(input string) (string, error) {
	apiKey := GetNextAPIKey()

	payload := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{
					{"text": input},
				},
			},
		},
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", client.APIURL+"?key="+apiKey, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		RemoveAPIKey(apiKey)
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		RemoveAPIKey(apiKey)
		return "", fmt.Errorf("API lỗi: %d", resp.StatusCode)
	}

	var result AIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if len(result.Candidates) > 0 && len(result.Candidates[0].Content.Parts) > 0 {
		return result.Candidates[0].Content.Parts[0].Text, nil
	}

	return "", fmt.Errorf("unexpected response format: %+v", result)
}
