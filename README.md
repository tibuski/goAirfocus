# Airfocus API Tools

A simple web application that provides tools to interact with the Airfocus API, specifically designed to help users find workspace and field IDs easily.

## Features

- 🔑 Secure API key management (client-side only)
- 🔍 Workspace ID lookup by name or dropdown selection
- 📋 Field ID lookup by name or dropdown selection
- 🎯 Workspace alias display
- 💻 Modern, responsive UI using Tailwind CSS
- 🔒 Context-aware API requests with proper error handling
- 🔄 Automatic filtering of fields created on 2025-03-20 with empty updatedAt
- 📊 Team field workspace count display
- 🏢 Workspace name display for non-team fields

## Prerequisites

- Go 1.21 or later
- An Airfocus API key with appropriate permissions

## Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/yourusername/goAirfocus.git
   cd goAirfocus
   ```

2. Install dependencies:
   ```bash
   go mod download
   ```

3. Run the application:
   ```bash
   go run .
   ```

The server will start on `http://localhost:8080`

## Usage

1. Open your web browser and navigate to `http://localhost:8080`
2. Enter your Airfocus API key in the provided field
3. Use the tools to:
   - Get workspace IDs by name or from a dropdown list
   - Get field IDs by name or from a dropdown list
   - View workspace aliases alongside workspace names
   - See workspace names for non-team fields
   - View the number of workspaces for team fields

## Field Display Format

- Team Fields: `Field Name (Team Field) - Used in X workspaces`
- Non-team Fields: `Field Name (Workspace1, Workspace2, ...)`

## API Endpoints

The application provides the following endpoints:

- `POST /api/workspaces` - List all available workspaces
- `POST /api/workspace/id` - Get workspace ID by name
- `POST /api/fields` - List all available fields
- `POST /api/field/id` - Get field ID by name

## Project Structure

```
.
├── main.go           # Main application entry point
├── airfocus/         # Airfocus API client package
│   └── client.go     # API client implementation
├── templates/        # HTML templates
│   └── index.html    # Main application template
├── static/          # Static assets
├── go.mod           # Go module definition
└── README.md        # This file
```

## Development

### Building

To build the application:

```bash
go build -o airfocus-tools
```

### Testing

To run the tests:

```bash
go test ./...
```

## Security

- API keys are never stored on the server
- All API requests are made with proper context handling
- HTTPS is recommended for production deployment

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgments

- [Airfocus API Documentation](https://developer.airfocus.com/)
- [Tailwind CSS](https://tailwindcss.com/)
- [Go Programming Language](https://golang.org/) 