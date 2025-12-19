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
make build
```

Or use Go directly:

```bash
git clone https://github.com/DevNergis/staticup.git
cd staticup
go build -o staticup
```

### Using Make

The project includes a Makefile for common tasks:

```bash
make help    # Show available commands
make build   # Build the binary
make install # Install to /usr/local/bin (requires sudo)
make clean   # Remove built binary
make fmt     # Format the code
make vet     # Run go vet
```

## Usage

### Basic Upload

Upload a file directly:

```bash
staticup /path/to/your/file.txt
```

Or using the `--file` flag:

```bash
staticup --file /path/to/your/file.txt
```

### Custom API Endpoint

Specify a custom API endpoint:

```bash
staticup /path/to/your/file.txt --api https://your-api.example.com
```

### With Authentication

If the API requires authentication:

```bash
staticup /path/to/your/file.txt --key YOUR_API_KEY
```

### Verbose Output

Enable verbose logging to see detailed information:

```bash
staticup /path/to/your/file.txt --verbose
```

## Command-Line Options

- `<file-path>` (positional): Path to the file to upload
- `--file`: Path to the file to upload (alternative to positional argument)
- `--api`: API endpoint URL (default: https://static.a85labs.net)
- `--key`: API key for authentication (optional)
- `--verbose`: Enable verbose output
- `--version`: Show version information

## Examples

```bash
# Upload an image (positional argument)
staticup photo.jpg

# Upload with verbose output
staticup document.pdf --verbose

# Upload to custom endpoint with API key
staticup data.json --api https://custom-static.example.com --key abc123

# Check version
staticup --version
```

## API

This tool is designed to work with the API at https://static.a85labs.net/openapi.json

The tool expects the API to:
- Accept POST requests to `/upload` endpoint
- Support multipart/form-data file uploads
- Return a JSON response with the uploaded file URL

## License

AGPL-3.0