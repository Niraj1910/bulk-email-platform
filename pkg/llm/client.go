package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type GroqClient struct {
	apiKey  string
	model   string
	baseURL string
	http    *http.Client
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type RequestBody struct {
	Messages    []Message `json:"messages"`
	Model       string    `json:"model"`
	Temperature float64   `json:"temperature"`
	MaxTokens   int       `json:"max_tokens"`
}

type ResponseBody struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func NewGroqClient(apiKey string) *GroqClient {
	return &GroqClient{
		apiKey:  apiKey,
		model:   "llama-3.1-8b-instant",
		baseURL: "https://api.groq.com/openai/v1/chat/completions",
		http: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (gc *GroqClient) GetModel() string {
	return gc.model
}

func (gc *GroqClient) GenerateEmail(description, context string) (string, string, error) {
	prompt := fmt.Sprintf(`You are an expert email writer. Generate a professional email based on:

Description of recipient: %s
Context: %s

Return ONLY a JSON object with this exact format:
{
  "subject": "the email subject line (keep it concise)",
  "message": "the email body (keep it professional and friendly, 3-4 sentences max)"
}

Do not include any other text, markdown, or explanation.`, description, context)

	requestBody := RequestBody{
		Messages: []Message{
			{Role: "system", Content: "You are an email writing assistant. Always respond with valid JSON only."},

			{Role: "user", Content: prompt},
		},
		Model:       gc.model,
		Temperature: 1.0,
		MaxTokens:   300,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", "", fmt.Errorf("failed to create request: %w", err)
	}

	req, err := http.NewRequest("POST", gc.baseURL, bytes.NewReader(jsonData))
	if err != nil {
		return "", "", fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+gc.apiKey)

	resp, err := gc.http.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("failed to send request to Groq: %w", err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("Groq API error (status %d): %s", resp.StatusCode, string(body))
	}

	var groqResp ResponseBody
	err = json.Unmarshal(body, &groqResp)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse Groq response: %w", err)
	}

	if len(groqResp.Choices) == 0 {
		return "", "", fmt.Errorf("Groq returned no choices")
	}

	content := groqResp.Choices[0].Message.Content

	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var emailContent struct {
		Subject string `json:"subject"`
		Message string `json:"message"`
	}

	err = json.Unmarshal([]byte(content), &emailContent)
	if err != nil {
		return gc.extractManually(content)
	}

	return emailContent.Subject, emailContent.Message, nil
}

func (c *GroqClient) extractManually(content string) (string, string, error) {

	subjectRegex := regexp.MustCompile(`(?i)subject:\s*(.+?)(?:\n|$)`)
	messageRegex := regexp.MustCompile(`(?i)message:\s*([\s\S]+?)(?:\n\s*\n|$)`)

	subjectMatch := subjectRegex.FindStringSubmatch(content)
	messageMatch := messageRegex.FindStringSubmatch(content)

	subject := ""
	message := ""

	if len(subjectMatch) > 1 {
		subject = strings.TrimSpace(subjectMatch[1])
	}

	if len(messageMatch) > 1 {
		message = strings.TrimSpace(messageMatch[1])
		// Clean up extra spaces
		message = regexp.MustCompile(`\s+`).ReplaceAllString(message, " ")
	}

	if subject != "" && message != "" {
		fmt.Println("Gettin 'subject and message' from the extractManually()")
		return subject, message, nil
	}

	return "Message from our team", content, nil
}
