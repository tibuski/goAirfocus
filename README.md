# Airfocus API Tools

A Go application that provides tools to interact with the Airfocus API, specifically designed to help users manage workspaces, users, and fields. The application is containerized using Docker for easy deployment and updates.

## Features

- **User Management**
  - List all users with their roles
  - Get user workspace access with hierarchical group display
  - View user permissions across workspaces

- **Workspace Management**
  - List all workspaces
  - Search workspaces by name
  - Get workspace details including permissions
  - View workspace hierarchy with groups

- **Field Management**
  - List all fields
  - Search fields by workspace
  - View field usage across workspaces

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
2. Enter your Airfocus API key in the provided field
3. Use the tools to:
   - View and manage user workspace access
   - Get workspace details and hierarchy
   - Search and manage fields

### User Workspace Access Display

The application provides a hierarchical view of user workspace access, showing:

- Workspaces grouped by their workspace groups
- Permission levels for each workspace (full, write, read, comment)
- Clear visual hierarchy with group paths
- Color-coded permission badges

Example:
```
Group A > Subgroup 1
  - Workspace 1 [full]
  - Workspace 2 [write]

Group B
  - Workspace 3 [read]
  - Workspace 4 [comment]

Ungrouped Workspaces
  - Workspace 5 [full]
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
