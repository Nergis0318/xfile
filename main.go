package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mdp/qrterminal/v3"
	"github.com/schollz/progressbar/v3"
)

const (
	defaultAPIEndpoint = "https://static.a85labs.net"
	version            = "1.0.0"
)

// UploadResponse represents the response from the upload API
type UploadResponse struct {
	Success bool   `json:"success"`
	URL     string `json:"url"`
	Message string `json:"message"`
	Key     string `json:"key"`
	Path    string `json:"path"`
}

func main() {
	// Define command-line flags
	filePath := flag.String("file", "", "Path to the file to upload (required)")
	verbose := flag.Bool("verbose", false, "Enable verbose output")
	showVersion := flag.Bool("version", false, "Show version information")
	showQR := flag.Bool("qr", true, "Show QR code for the uploaded file URL")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s [options] <file-path>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --file <file-path> [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	// Show version and exit
	if *showVersion {
		fmt.Printf("staticup version %s\n", version)
		os.Exit(0)
	}

	// Support positional argument for file path if -file flag is not provided
	if *filePath == "" && flag.NArg() > 0 {
		*filePath = flag.Arg(0)
	}

	// Validate required flags
	if *filePath == "" {
		fmt.Fprintf(os.Stderr, "Error: file path is required\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <file-path> or %s --file <file-path>\n", os.Args[0], os.Args[0])
		flag.Usage()
		os.Exit(1)
	}

	// Check if file exists
	fileInfo, err := os.Stat(*filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Cannot access file: %v\n", err)
		os.Exit(1)
	}

	if fileInfo.IsDir() {
		fmt.Fprintf(os.Stderr, "Error: %s is a directory, not a file\n", *filePath)
		os.Exit(1)
	}

	if *verbose {
		fmt.Printf("Uploading file: %s\n", *filePath)
		fmt.Printf("File size: %d bytes\n", fileInfo.Size())
		fmt.Printf("API endpoint: %s\n", defaultAPIEndpoint)
	}

	// Upload the file
	url, err := uploadFile(*filePath, defaultAPIEndpoint, "", *verbose)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error uploading file: %v\n", err)
		os.Exit(1)
	}

	// Print success message
	fmt.Printf("\n")
	fmt.Printf("✓ File uploaded successfully!\n")

	// Show QR code if requested
	if *showQR {
		fmt.Println("\nQR Code:")
		config := qrterminal.Config{
			Level:     qrterminal.H,
			Writer:    os.Stdout,
			BlackChar: qrterminal.BLACK,
			WhiteChar: qrterminal.WHITE,
			QuietZone: 1,
		}
		qrterminal.GenerateWithConfig(url, config)
	}

	fmt.Printf("\n")
	fmt.Printf("URL: %s\n", url)
}

// uploadFile uploads a file to the static file hosting service
func uploadFile(filePath, apiEndpoint, apiKey string, verbose bool) (string, error) {
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Create a buffer to hold the multipart form data
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	// Create form file field
	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return "", fmt.Errorf("failed to create form file: %w", err)
	}

	// Copy file content to the form
	if verbose {
		fmt.Println("Reading file content...")
	}
	_, err = io.Copy(part, file)
	if err != nil {
		return "", fmt.Errorf("failed to copy file content: %w", err)
	}

	// Close the multipart writer
	err = writer.Close()
	if err != nil {
		return "", fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// Construct the upload URL
	uploadURL := fmt.Sprintf("%s/upload", apiEndpoint)
	if verbose {
		fmt.Printf("Sending request to: %s\n", uploadURL)
	}

	// Create progress bar
	bar := progressbar.DefaultBytes(
		int64(requestBody.Len()),
		"uploading",
	)

	// Create HTTP request
	req, err := http.NewRequest("POST", uploadURL, io.TeeReader(&requestBody, bar))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.ContentLength = int64(requestBody.Len())

	// Set headers
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	}

	// Send request with timeout
	client := &http.Client{
		Timeout: 5 * time.Minute,
	}

	if verbose {
		fmt.Println("Uploading...")
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if verbose {
		fmt.Printf("Response status: %d\n", resp.StatusCode)
		fmt.Printf("Response body: %s\n", string(body))
	}

	// Check response status
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var uploadResp UploadResponse
	err = json.Unmarshal(body, &uploadResp)
	if err != nil {
		// If JSON parsing fails, try to extract URL from plain text response
		// Some APIs might return just the URL as plain text
		if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
			if verbose {
				fmt.Fprintf(os.Stderr, "Warning: Received non-JSON response, treating as plain text URL\n")
			}
			plainURL := strings.TrimSpace(string(body))
			if strings.HasPrefix(plainURL, "http") {
				return plainURL, nil
			}
			return joinURL(apiEndpoint, plainURL), nil
		}
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Return the URL from the response
	if uploadResp.URL != "" {
		if strings.HasPrefix(uploadResp.URL, "http") {
			return uploadResp.URL, nil
		}
		return joinURL(apiEndpoint, uploadResp.URL), nil
	}

	// If URL is not in the response, try to construct it from path/key
	if uploadResp.Path != "" {
		return joinURL(apiEndpoint, uploadResp.Path), nil
	}

	if uploadResp.Key != "" {
		return joinURL(apiEndpoint, uploadResp.Key), nil
	}

	return "", fmt.Errorf("no URL in response")
}

// joinURL safely joins a base URL with a path, handling trailing/leading slashes
func joinURL(base, path string) string {
	baseURL, err := url.Parse(base)
	if err != nil {
		// Fallback to simple concatenation if URL parsing fails
		return strings.TrimRight(base, "/") + "/" + strings.TrimLeft(path, "/")
	}
	baseURL.Path = strings.TrimRight(baseURL.Path, "/") + "/" + strings.TrimLeft(path, "/")
	return baseURL.String()
}
