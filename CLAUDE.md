# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a bookmark chat system that allows users to interact with their bookmarks through natural language. It consists of:
- Go backend API using Echo framework with OpenAPI-driven development
- Frontend using vanilla JavaScript with jQuery (fully implemented)
- Semantic search capabilities using embeddings and vector databases
- Chat interface with RAG (Retrieval-Augmented Generation)
- Complete web scraping and content extraction pipeline
- Hybrid search combining semantic similarity and keyword search

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
- `/api/bookmarks/*` - CRUD operations, import, and individual scraping
- `/api/scraping/*` - Bulk scraping operations (start, pause, resume, stop, status)
- `/api/search` - Hybrid search (semantic + keyword, fully implemented)
- `/api/chat/*` - Conversational interface (stub implementation)
- `/api/health`, `/api/stats` - System monitoring

### Key Technologies
- **Go + Echo**: High-performance API server
- **OpenAPI + oapi-codegen**: Type-safe API development
- **libSQL**: SQLite with vector extensions for embeddings
- **OpenAI API**: Embeddings (text-embedding-3-small) and chat (GPT-4)
- **goquery**: HTML parsing and content extraction
- **FTS5**: Full-text search with BM25 ranking

## Current State

The project is now feature-complete with a working bookmark management system:

### ✅ Fully Implemented Features:
- **Complete Frontend Interface**: 3-tab web application (Bookmarks, Scraping, Search)
- **Bookmark Import**: Firefox JSON and Chrome HTML format support with validation
- **Content Scraping**: Individual and bulk scraping with status tracking and progress monitoring  
- **Hybrid Search**: Semantic + keyword search with relevance scoring and content snippets
- **Data Persistence**: All bookmarks, content, and metadata stored in libSQL database
- **Web Scraping Pipeline**: goquery-based content extraction with readability algorithms
- **Real-time Updates**: Live scraping progress, tab switching, and data refresh
- **Error Handling**: Comprehensive error handling throughout the stack
- **Logging & Debugging**: Extensive logging for troubleshooting

### 🔧 Infrastructure Components:
- OpenAPI specification with all endpoint definitions implemented
- Echo framework server with all handlers implemented  
- Storage layer with libSQL, vector embeddings, and FTS5 search
- Service layer with parsing, scraping, import, and content processing
- Frontend with jQuery-based components and responsive design
- API client with comprehensive error handling and logging

### ⚠️ Current Limitations:
- **OpenAI Integration**: Requires OPENAI_API_KEY for embeddings (falls back to keyword-only search)
- **Chat Interface**: Stub implementation only (endpoints exist but return mock data)
- **Authentication**: Not implemented (all endpoints are public)

### 🚀 Ready for Production:
- Bookmark import and management
- Content scraping (individual and bulk)
- Search functionality (hybrid when API key provided, keyword-only fallback)
- Real-time scraping progress monitoring
- Responsive web interface

## Development Notes

### New Feature Development:
1. **Backend**: All major handlers are implemented - focus on chat/RAG integration
2. **Frontend**: 3-tab interface is complete - focus on chat UI integration
3. **Search**: Fully working - enhance with filters, saved searches, or advanced query syntax
4. **OpenAI**: Add OPENAI_API_KEY environment variable to enable semantic search and embeddings

### Development Workflow:
1. Start with the OpenAPI spec - it's the source of truth for new endpoints
2. Regenerate code after spec changes using the oapi-codegen command
3. Implement handlers by coordinating between services and storage layers
4. Use the prompt files (`backend-prompt.md`, `frontend-prompt.md`) as detailed implementation guides
5. Follow Go idioms and Echo framework patterns
6. Frontend uses vanilla JavaScript with jQuery - no build system required
7. Do not add "🤖 Generated with [Claude Code](https://claude.ai/code) Co-Authored-By: Claude <noreply@anthropic.com>" when doing git commit

### Frontend Architecture:
- **3-Tab Interface**: Bookmarks (tree view), Scraping (bulk operations), Search (hybrid search)
- **Components**: Modular JavaScript components with event handling and API integration
- **Real-time Updates**: Tab switching triggers data refresh, scraping shows live progress
- **Responsive Design**: Works on desktop and mobile with CSS Grid and Flexbox
- **API Integration**: Comprehensive API client with detailed logging and error handling

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
- **Frontend Testing**: Use browser console to debug with extensive logging (🔍 search, 🌐 API, 📊 scraping)
- **Manual Testing**: Import bookmarks → Scrape content → Search bookmarks workflow

### Troubleshooting
- **Search not working**: Check browser console for 🔍 and 🌐 log messages
- **Scraping tab empty**: Check if bookmarks are imported and tab refresh is working
- **No search results**: Without OPENAI_API_KEY, only keyword search works - scrape content first
- **Frontend issues**: All components have extensive console logging for debugging