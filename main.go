package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
)

const (
	openAIURL = "https://api.openai.com/v1/engines/gpt-3.5-turbo-instruct/completions"
)

type openAIRequest struct {
	Prompt      string  `json:"prompt"`
	MaxTokens   int     `json:"max_tokens"`
	Temperature float64 `json:"temperature"`
}

type openAIResponse struct {
	Choices []struct {
		Text string `json:"text"`
	} `json:"choices"`
}

func main() {
	// Load the OpenAI API key from an environment variable
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("The OPENAI_API_KEY environment variable is not set.")
		os.Exit(1)
	}

	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		os.Exit(1)
	}
	defer listener.Close()
	fmt.Println("Listening on :8080")

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting:", err.Error())
			continue
		}

		go handleRequest(conn, apiKey)
	}
}

func handleRequest(conn net.Conn, apiKey string) {
	reader := bufio.NewReader(conn)

	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				fmt.Println("Client closed the connection")
			} else {
				fmt.Printf("Error reading from connection: %s\n", err)
			}
			break
		}

		response, err := askChatGPT(strings.TrimSpace(message), apiKey)
		if err != nil {
			fmt.Printf("Error asking ChatGPT: %s\n", err)
			fmt.Fprintln(conn, "Error: "+err.Error()) // Send error message to client
			continue
		}

		fmt.Fprintln(conn, response)

		// Check for a quit command or similar if you want to provide a way to close the connection
		if strings.TrimSpace(message) == "quit" {
			fmt.Println("Client sent quit command")
			break
		}
	}

	conn.Close() // Now close the connection
}

func askChatGPT(prompt string, apiKey string) (string, error) {
	requestBody := openAIRequest{
		Prompt:      prompt,
		MaxTokens:   150,
		Temperature: 0.7, // You can adjust the creativity level
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", openAIURL, strings.NewReader(string(jsonBody)))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var response openAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", err
	}

	if len(response.Choices) > 0 {
		return strings.TrimSpace(response.Choices[0].Text), nil
	}

	return "", fmt.Errorf("no response from ChatGPT")
}
