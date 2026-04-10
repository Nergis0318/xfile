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
	defaultAPIEndpoint = "https://file.xeon.kr"
	version            = "1.1.0"
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
		fmt.Printf("xfile version %s\n", version)
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

	// Try to shorten the uploaded URL using lru.kr API
	shortURL, err := shortenWithLRU(url, *verbose)
	if err != nil {
		if *verbose {
			fmt.Fprintf(os.Stderr, "Warning: failed to shorten URL: %v\n", err)
		}
	}

	// Show QR code for the shortened URL if available, otherwise use original
	if *showQR {
		qrTarget := url
		if shortURL != "" {
			qrTarget = shortURL
		}

		fmt.Println("\nQR Code:")
		config := qrterminal.Config{
			Level:     qrterminal.H,
			Writer:    os.Stdout,
			BlackChar: qrterminal.BLACK,
			WhiteChar: qrterminal.WHITE,
			QuietZone: 1,
		}
		qrterminal.GenerateWithConfig(qrTarget, config)
	}

	fmt.Printf("\n")
	fmt.Printf("URL: %s\n", url)
	if shortURL != "" {
		fmt.Printf("\n")
		fmt.Printf("Short URL: %s\n", shortURL)
	}
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

// shortenWithLRU calls lru.kr /api/shorten to create a short link for the given URL.
func shortenWithLRU(orig string, verbose bool) (string, error) {
	api := "https://lru.kr/api/shorten"

	payload := map[string]string{"url": orig}
	b, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	client := &http.Client{Timeout: 20 * time.Second}
	req, err := http.NewRequest("POST", api, bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	if verbose {
		fmt.Printf("Shorten request to: %s\n", api)
		fmt.Printf("Request body: %s\n", string(b))
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if verbose {
		fmt.Printf("Shorten response status: %d\n", resp.StatusCode)
		fmt.Printf("Shorten response body: %s\n", string(body))
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("shorten request failed: %d %s", resp.StatusCode, string(body))
	}

	// Try to parse response as JSON and extract a short URL
	var data any
	if err := json.Unmarshal(body, &data); err != nil {
		// fallback: if body is plain text URL
		s := strings.TrimSpace(string(body))
		if strings.HasPrefix(s, "http") {
			return s, nil
		}
		return "", fmt.Errorf("failed to parse shorten response: %w", err)
	}

	// Walk JSON to find a plausible short URL or key
	if m, ok := data.(map[string]any); ok {
		// common keys: "short_url", "url", "result", "data", "key", "short_key"
		if v := findStringURLInMap(m); v != "" {
			return v, nil
		}
	}

	// try to find any string in nested structures that looks like a short path or url
	var found string
	var walk func(any)
	walk = func(x any) {
		if found != "" {
			return
		}
		switch v := x.(type) {
		case string:
			s := strings.TrimSpace(v)
			if strings.HasPrefix(s, "http") {
				found = s
				return
			}
			// short key (alphanumeric, length 2-20)
			if len(s) >= 2 && len(s) <= 20 && !strings.ContainsAny(s, " \n\r\t") {
				// assume it's a key
				found = joinURL("https://lru.kr", s)
				return
			}
		case map[string]any:
			for _, vv := range v {
				walk(vv)
				if found != "" {
					return
				}
			}
		case []any:
			for _, vv := range v {
				walk(vv)
				if found != "" {
					return
				}
			}
		}
	}
	walk(data)
	if found != "" {
		return found, nil
	}

	return "", fmt.Errorf("no short url found in response")
}

func findStringURLInMap(m map[string]any) string {
	// prefer obvious full URLs
	for k, v := range m {
		lk := strings.ToLower(k)
		if s, ok := v.(string); ok {
			if strings.HasPrefix(s, "http") {
				return s
			}
			if lk == "short_url" || lk == "shortlink" || lk == "short" || lk == "url" {
				// url field might be a path
				if strings.HasPrefix(s, "http") {
					return s
				}
				return joinURL("https://lru.kr", s)
			}
		}
	}
	// check nested maps for keys like key/short_key
	for k, v := range m {
		if vv, ok := v.(map[string]any); ok {
			if s := findStringURLInMap(vv); s != "" {
				return s
			}
		} else if arr, ok := v.([]any); ok {
			for _, item := range arr {
				if im, ok := item.(map[string]any); ok {
					if s := findStringURLInMap(im); s != "" {
						return s
					}
				}
			}
		} else if ks, ok := v.(string); ok {
			lk := strings.ToLower(k)
			if lk == "key" || lk == "short_key" || lk == "id" {
				if strings.HasPrefix(ks, "http") {
					return ks
				}
				return joinURL("https://lru.kr", ks)
			}
		}
	}
	return ""
}
