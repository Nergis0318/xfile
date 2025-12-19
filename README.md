# staticup

A simple CLI tool for uploading files to static file hosting services.

## Installation

### From Source

```bash
go install github.com/DevNergis/staticup@latest
```

Or clone and build:

```bash
git clone https://github.com/DevNergis/staticup.git
cd staticup
go build -o staticup
```

## Usage

### Basic Upload

Upload a file to the default API endpoint:

```bash
staticup -file /path/to/your/file.txt
```

### Custom API Endpoint

Specify a custom API endpoint:

```bash
staticup -file /path/to/your/file.txt -api https://your-api.example.com
```

### With Authentication

If the API requires authentication:

```bash
staticup -file /path/to/your/file.txt -key YOUR_API_KEY
```

### Verbose Output

Enable verbose logging to see detailed information:

```bash
staticup -file /path/to/your/file.txt -verbose
```

## Command-Line Options

- `-file` (required): Path to the file to upload
- `-api`: API endpoint URL (default: https://static.a85labs.net)
- `-key`: API key for authentication (optional)
- `-verbose`: Enable verbose output
- `-version`: Show version information

## Examples

```bash
# Upload an image
staticup -file photo.jpg

# Upload with verbose output
staticup -file document.pdf -verbose

# Upload to custom endpoint with API key
staticup -file data.json -api https://custom-static.example.com -key abc123

# Check version
staticup -version
```

## API

This tool is designed to work with the API at https://static.a85labs.net/openapi.json

The tool expects the API to:
- Accept POST requests to `/upload` endpoint
- Support multipart/form-data file uploads
- Return a JSON response with the uploaded file URL

## License

AGPL-3.0