# Airfocus API Tools

A Go application that provides tools to interact with the Airfocus API, specifically designed to help users manage workspaces, users, and fields. The application is built with a pure HTMX approach for a dynamic and responsive web interface, and it's containerized using Docker for easy deployment and updates.

## Features

- **Full HTMX Integration**: The entire frontend is now driven by HTMX, offering a seamless and consistent user experience without traditional JavaScript.
- **Improved Workspace Management**:
  - Automatically displays Workspace ID and Users when a workspace is selected from the dropdown.
  - Workspace users are grouped by permission level (Full, Write, Comment, Read) with distinct color coding for clarity.
- **User Management**
  - List all users with their roles (via "Get Users" button)
  - Get user workspace access with hierarchical group display (via "User Workspaces & Permissions" section, further tree view implementation for individual user selection is a future enhancement)
  - View user permissions across workspaces (now grouped by permission)

- **Field Management**
  - List all fields (via "Get Fields" button)
  - Get Field ID (via form submission)

- **License Information**: Accurately displays license role statistics, showing actual used seats for Admin, Editor, and Contributor roles.

## Deployment

The application is designed to be deployed using Docker. There are two deployment modes available:

### Development Mode

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

### Production Mode with Traefik

1. Ensure you have a Traefik network running:
   ```bash
   docker network create traefik_network
   ```

2. The application will use the `docker-compose.override.yml` configuration which includes:
   - Traefik labels for routing and TLS
   - External network configuration
   - Automatic HTTPS setup

3. Build and start the container:
   ```bash
   docker-compose up -d
   ```

The service will be available at `https://airfocus.yourdomain.com`

### Docker Compose Files

- `docker-compose.yml`: Base configuration for the application
- `docker-compose.override.yml`: Production configuration with Traefik settings
  ```yaml
  services:
    airfocus-tools:
      labels:
        traefik.enable: true
        traefik.http.routers.airfocus.entrypoints: websecure
        traefik.http.routers.airfocus.tls: true
        traefik.http.routers.airfocus.tls.certresolver: myresolver
        traefik.http.routers.airfocus.rule: Host(`airfocus.yourdomain.com`)
  networks:
    default:
      name: traefik_network
      external: true
  ```

### Maintenance

#### Updating the Application

To update to the latest version:
```bash
git pull
docker-compose up -d --build
```

#### Monitoring

To view logs:
```bash
docker-compose logs -f
```

#### Stopping the Application

To stop the application:
```bash
docker-compose down
```

## Usage

1. Open your web browser and navigate to:
   - Development: `http://localhost:8080`
   - Production: `https://airfocus.yourdomain.com`
2. Enter your Airfocus API key in the provided field.
3. **Workspace Management**: Click "Load Workspaces" to populate the dropdown. Then, select a workspace from the dropdown to automatically view its ID and grouped users.
4. **User Management**: Click "Get Users" to list all users with their roles. (Further functionality for user workspaces to be implemented).
5. **Field Management**: Click "Get Fields" to list all available fields. To get a specific field's ID, enter its name in the input field and click "Get Field ID".
6. **License Information**: Click "Get License Info" to view your team's license details and role statistics.

### User Workspace Access Display

The application now provides a clean, grouped view of workspace users by their permission levels:

- Users are categorized under `Full`, `Write`, `Comment`, and `Read` permissions.
- Each permission group displays the count of users and a bulleted list of their names.
- Color-coded left borders and permission badges enhance visual clarity.

Example:
```
Workspace Users (4 total)
├── Full (1 user)
│   • User with full permissions
├── Write (1 user)
│   • Luc Fourestier
├── Comment (1 user)
│   • User with comment permissions
└── Read (1 user)
    • Tony Aussems
```

## API Key

All requests require an Airfocus API key. You can obtain one from your Airfocus account settings.

## Security

- API keys are never stored on the server
- All API requests are made with proper context handling
- HTTPS is enforced in production via Traefik
- TLS certificates are automatically managed by Traefik

## License

MIT License
