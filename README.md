# Airfocus API Tools

A simple web application that provides tools to interact with the Airfocus API, specifically designed to help users find workspace and field IDs, and manage workspace users easily.

## Features

- 🔑 Secure API key management (client-side only)
- 🔍 Workspace ID lookup by name or dropdown selection
- 👥 **Workspace User Lookup**: Retrieve and list users with access to a selected workspace, including their roles/permissions.
- 📋 Field ID lookup by name or dropdown selection
- 🎯 Workspace alias display
- 💻 Modern, responsive UI using Tailwind CSS
- 🔒 Context-aware API requests with proper error handling
- 🔄 Automatic filtering of fields created on 2025-03-20 with empty updatedAt
- 📊 Team field workspace count display
- 🏢 Workspace name display for non-team fields
- 🌳 **Workspace Hierarchy**: View workspaces organized by their group hierarchy, with color-coded permission badges
- 🔄 **Data Caching**: Efficient API usage with automatic caching of user, workspace, and field data

## Prerequisites

- Go 1.21 or later (for local development)
- Docker and Docker Compose (for containerized deployment)
- An Airfocus API key with appropriate permissions

## Installation

### Local Development

1. Clone the repository:
   ```bash
   git clone https://github.com/tibuski/goAirfocus.git
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

### Docker Deployment

1. Clone the repository:
   ```bash
   git clone https://github.com/tibuski/goAirfocus.git
   cd goAirfocus
   ```

2. Build and start the container:
   ```bash
   docker-compose up -d
   ```

The application will be available at `http://localhost:8080`

To update to the latest version:
```bash
git pull
docker-compose up -d --build
```

To view logs:
```bash
docker-compose logs -f
```

To stop the application:
```bash
docker-compose down
```

## Usage

1. Open your web browser and navigate to `http://localhost:8080`
2. Enter your Airfocus API key in the provided field
3. Use the tools to:
   - Get workspace IDs by name or from a dropdown list
   - Get users for a selected workspace (by ID or name)
   - Get field IDs by name or from a dropdown list
   - View workspace aliases alongside workspace names
   - See workspace names for non-team fields
   - View the number of workspaces for team fields

## Field Display Format

- Team Fields: `Field Name (Team Field) - Used in X workspaces`
- Non-team Fields: `Field Name (Workspace1, Workspace2, ...)`

## API Endpoints

The application provides the following backend endpoints:

- `POST /api/workspaces` - List all available workspaces
- `POST /api/workspace/id` - Get workspace ID by name
- `POST /api/workspace/users` - Get users and their permissions for a specific workspace (by ID or name)
- `POST /api/workspace/hierarchy` - Get workspace hierarchy with group information and user permissions
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
├── Dockerfile       # Docker build instructions
├── docker-compose.yml # Docker Compose configuration
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
