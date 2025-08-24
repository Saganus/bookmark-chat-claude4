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

# Run storage layer tests specifically
go test ./internal/storage -v

# Run storage benchmarks
go test ./internal/storage -bench=. -benchmem

# Run storage example
go run examples/storage_example.go

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

3. **Service Layer** (`internal/services/`)
   - Bookmark parsing (Firefox JSON, Chrome HTML) - implemented
   - Web scraping for content extraction - implemented 
   - Import service with batch processing - implemented
   - Embedding generation via OpenAI (integration needed)
   - Chat/RAG implementation (to be implemented)

4. **Storage Layer** (`internal/storage/`)
   - Complete libSQL implementation with vector extensions
   - SQLite FTS5 for keyword search
   - Hybrid search combining semantic and keyword search
   - Batch operations for efficient processing
   - Comprehensive error handling and transactions

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
- **goquery**: HTML parsing and content extraction
- **FTS5**: Full-text search with BM25 ranking

## Current State

The project has a solid foundation with:
- OpenAPI specification complete with comprehensive endpoint definitions
- Server structure set up with Echo framework integration
- Handler stubs generated from OpenAPI spec
- Storage layer fully implemented with libSQL, vector embeddings, and hybrid search
- Service layer partially implemented (parsing, scraping, import services)
- Frontend structure in place but needs integration with backend
- OpenAI integration needed for embeddings and chat functionality

## Development Notes

When implementing features:
1. Start with the OpenAPI spec - it's the source of truth
2. Regenerate code after spec changes using the oapi-codegen command
3. Implement handlers by coordinating between services and storage layers
4. Use the prompt files (`backend-prompt.md`, `frontend-prompt.md`) as detailed implementation guides
5. Follow Go idioms and Echo framework patterns
6. The storage layer is feature-complete - use it for all data persistence
7. Service layer has parsing and scraping - build upon these for import workflows
8. Frontend uses vanilla JavaScript with jQuery - no build system required
9. Do not add "🤖 Generated with [Claude Code](https://claude.ai/code) Co-Authored-By: Claude <noreply@anthropic.com>" when doing git commit

### Storage Layer Usage
The storage layer (`internal/storage/`) provides:
- Complete CRUD operations for bookmarks and content
- Vector embeddings storage with F32_BLOB(1536) for text-embedding-3-small
- Hybrid search combining semantic similarity and FTS5 keyword search
- Batch operations for efficient bulk processing
- See `internal/storage/README.md` for comprehensive usage examples

### Testing Strategy
- Run `go test ./...` for all tests
- Storage layer has comprehensive test coverage in `storage_test.go`
- Use `examples/storage_example.go` to understand storage layer capabilities
- Test files include specific parsers and scrapers