# AGENTS.md

This file contains essential information to help agents navigate and work effectively in this repository.

## Project Overview
This is a Go-based backend application with a Vue.js frontend, using Docker for deployment. The backend handles file management with user authentication, sharing, and database operations.

## Development Commands
- `make up-d` - Start the development environment with docker compose
- `make up-d-build` - Start with build (rebuilds images)
- `make down` - Stop the development environment  
- `make run` - Run the Go backend directly without Docker
- `make lint` - Run linter (golangci-lint)
- `make build-api` - Build the Go API binary
- `make build-ui` - Build the frontend

## Environment Variables
### Backend
- `APP_ENV` - Set to "dev" for debug mode and non-secure cookies, "production" otherwise
- `UPLOAD_DIR` - Directory where uploaded files are stored (default: ./avenuectl/temp/)
- `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_DATABASE` - Database connection
- `ROOT_USER_EMAIL`, `ROOT_USER_PASSWORD` - Admin account credentials
- `MAX_FILE_BYTE_SIZE` - Maximum upload size per file (default: 200MB)
- `REGISTRATION_ENABLED` - Enable public self-registration (default: false)
- `ENABLE_FILE_SHARING`, `ENABLE_FOLDER_SHARING` - Enable share links for files/folders

### Frontend
- `VITE_APP_API_URL` - API URL (default: http://localhost:8080/)

## Project Structure
- `backend/` - Go source code
- `frontend/` - Vue.js frontend 
- `api-tests/` - Integration tests for backend API
- `docker-compose.yml` - Development environment setup
- `Dockerfile` - Build configuration

## Key Endpoints
- `/login`, `/register`, `/logout` - Authentication endpoints
- `/files`, `/folders` - File management endpoints  
- `/users` - User management endpoints

## Testing
- Run all API tests: `go test ./api-tests/...`
- Run specific test (e.g. auth): `go test -v ./api-tests/auth_test.go`
- Use the docker-compose environment for integration testing

## Build Process
- Backend built with Go (1.26.1)
- Frontend built with Vite + Vue 3
- Production build uses "prod" build tag in Go

## Special Considerations  
- The application requires an active PostgreSQL database to run
- File uploads are stored in the directory specified by `UPLOAD_DIR`
- The root admin user is created on first boot or can be reset with `ROOT_USER_RESET=true`
- Docker volumes are used for persistent storage of uploads and database data

## Key Files
- `main.go` - Entry point; sets up database, migrations, handlers, and server
- `handlers/` - HTTP endpoint handlers for API routes
- `db/` - Database connection, migrations, and model definitions
- `shared/` - Shared utilities and configuration helpers
