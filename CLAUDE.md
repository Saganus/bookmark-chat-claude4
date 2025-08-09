# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a bookmark chat system that allows users to interact with their bookmarks through natural language. It consists of:
- Go backend API using Echo framework with OpenAPI-driven development
- Frontend planned to use vanilla JavaScript with jQuery
- Semantic search capabilities using embeddings and vector databases
- Chat interface with RAG (Retrieval-Augmented Generation)

## Key Commands

### Backend Development
```bash
# Generate server code from OpenAPI spec
oapi-codegen -package api -generate types,server,spec api/openapi.yaml > api/generated/server.gen.go

# Run the server
go run cmd/server/main.go

# Build the server
go build -o bin/server cmd/server/main.go

# Run tests
go test ./...

# Update dependencies
go mod tidy
```

## Architecture Overview

### Backend Structure
The backend follows a clean architecture pattern with OpenAPI-first development:

1. **API Layer** (`api/`)
   - OpenAPI specification defines all endpoints
   - Code generation creates type-safe handlers
   - Endpoints handle bookmarks, search, chat, and system operations

2. **Handler Layer** (`internal/handlers/`)
   - Implements the OpenAPI-generated interface
   - Currently contains stub implementations
   - Should coordinate between services

3. **Service Layer** (to be implemented in `internal/services/`)
   - Bookmark parsing (Firefox JSON, Chrome HTML)
   - Web scraping for content extraction
   - Embedding generation via OpenAI
   - Vector search using libSQL
   - Chat/RAG implementation

4. **Database Layer** (to be implemented)
   - libSQL with vector extensions
   - SQLite FTS5 for keyword search
   - Migrations for schema management

### API Endpoints
The system exposes these main endpoint groups:
- `/api/bookmarks/*` - CRUD operations and import
- `/api/search` - Hybrid search (semantic + keyword)
- `/api/chat/*` - Conversational interface
- `/api/health`, `/api/stats` - System monitoring

### Key Technologies
- **Go + Echo**: High-performance API server
- **OpenAPI + oapi-codegen**: Type-safe API development
- **libSQL**: SQLite with vector extensions for embeddings
- **OpenAI API**: Embeddings (text-embedding-3-small) and chat (GPT-4)

## Current State

The project is in early development with:
- OpenAPI specification complete
- Basic server structure set up
- Handler stubs generated
- Frontend and service implementations pending

## Development Notes

When implementing features:
1. Start with the OpenAPI spec - it's the source of truth
2. Regenerate code after spec changes
3. Implement handlers by adding service layer logic
4. Use the prompt files as detailed implementation guides
5. Follow Go idioms and Echo framework patterns