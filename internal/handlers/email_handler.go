package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
)

// SendEmailViaResend sends email using Resend API
func SendEmailViaResend(name, email, to, subject, body string) error {
	apiKey := os.Getenv("RESEND_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("RESEND_API_KEY not set")
	}

	payload := map[string]interface{}{
		"from":     "Book Library Support <onboarding@resend.dev>",
		"to":       []string{to},
		"subject":  subject,
		"text":     body,
		"reply_to": email, // Allow replies to go to the sender
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal json: %v", err)
	}

	req, err := http.NewRequest("POST", "https://api.resend.com/emails", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 202 {
		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		return fmt.Errorf("resend API error %d: %v", resp.StatusCode, result)
	}

	return nil
}

// ContactRequest represents the incoming contact form request
type ContactRequest struct {
	Name    string `json:"name" binding:"required,min=2,max=50"`
	Email   string `json:"email" binding:"required,email"`
	Subject string `json:"subject" binding:"required,min=2,max=100"`
	Message string `json:"message" binding:"required"`
}

// ContactHandler handles contact form submissions
func ContactHandler(c *gin.Context) {
	var req ContactRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Debug: Check if API key exists
	apiKey := os.Getenv("RESEND_API_KEY")
	fmt.Printf("RESEND_API_KEY exists: %v\n", apiKey != "")
	fmt.Printf("RESEND_API_KEY length: %d\n", len(apiKey))

	body := fmt.Sprintf("From: %s <%s>\n\n%s", req.Name, req.Email, req.Message)
	recipient := "himelsd117@gmail.com"

	if err := SendEmailViaResend(req.Name, req.Email, recipient, req.Subject, body); err != nil {
		fmt.Printf("Email error: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to send email",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Email sent successfully!"})
}
