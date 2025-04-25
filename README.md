# message-service

A microservice for message scheduling and delivery with a RESTful API interface.

## Overview

message-service is a Go-based backend service that provides functionality to send messages, manage a scheduler for automated message delivery, and track sent messages. The application exposes a RESTful API with Swagger documentation.

## Features

- **Message Sending:** Send messages to recipients via API
- **Automated Scheduler:** Start/stop a scheduler for automated message delivery
- **Message Tracking:** Retrieve a history of sent messages
- **API Documentation:** Swagger UI for interactive API documentation

## Tech Stack

- **Go (Golang):** Core programming language
- **Gin:** HTTP web framework
- **PostgreSQL:** Primary database for message storage
- **Redis:** Used for caching and message queue
- **Swagger:** API documentation

## API Endpoints

### Messages
- **POST /api/messages/send:** Send a message to a recipient
  - Request body contains message content, ID, and recipient phone
- **GET /api/messages/sent:** Retrieve a list of sent messages

### Scheduler
- **POST /api/scheduler/start:** Start the automatic message sending process
- **POST /api/scheduler/stop:** Stop the automatic message sending process

### Documentation
- **GET /swagger/*any:** Access Swagger UI documentation

## Project Structure

- **main.go:** Application entry point
- **docs/:** Auto-generated Swagger documentation
- **internal/:** Internal application code
  - **config/:** Configuration management
  - **handler/:** HTTP request handlers
  - **mpostgres/:** PostgreSQL database operations
  - **pkg/gpostgresql/:** PostgreSQL connection utilities
  - **service/:** Business logic implementation

## Setup & Configuration

### Prerequisites
- Go 1.x
- PostgreSQL
- Redis

### Environment Variables
Copy `env.example` to `.env` and configure the required settings.

### Installation & Running with Docker
docker compose down -v
docker compose up --build


## Dependencies

Major dependencies include:
- github.com/gin-gonic/gin
- github.com/swaggo/swag
- github.com/useinsider/go-pkg/inslogger
- github.com/useinsider/go-pkg/insredis

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
