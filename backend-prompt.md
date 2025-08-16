# Backend System Implementation Prompt

## Project Overview
Build a Go-based backend API server for a bookmark chat system that allows users to have conversational interactions with their scraped bookmarks using semantic search and natural language processing.

## Core Requirements

### Technology Stack
- **Language**: Go (latest stable version)
- **Web Framework**: Echo framework (v4)
- **Database**: libSQL with vector extensions for embeddings storage
- **Search**: SQLite FTS5 extension for hybrid search (semantic + BM25)
- **API Schema**: OpenAPI 3.0 specification with oapi-codegen for code generation
- **Embeddings**: OpenAI embeddings API (text-embedding-3-small model recommended)
- **LLM Integration**: OpenAI GPT-4 for chat completions

## Project Structure
```
/
├── api/
│   ├── openapi.yaml              # OpenAPI specification
│   └── generated/                # Generated code from oapi-codegen
├── cmd/
│   └── server/
│       └── main.go               # Application entry point
├── internal/
│   ├── config/
│   │   └── config.go             # Configuration management
│   ├── database/
│   │   ├── connection.go         # Database connection management
│   │   ├── migrations/           # SQL migration files
│   │   └── models.go             # Database models
│   ├── handlers/
│   │   ├── bookmarks.go          # Bookmark CRUD handlers
│   │   ├── chat.go               # Chat interaction handlers
│   │   ├── import.go             # Bookmark import handlers
│   │   └── search.go             # Search handlers
│   ├── services/
│   │   ├── bookmark_parser.go    # Parse Firefox/Chrome bookmarks
│   │   ├── scraper.go            # Web content scraper
│   │   ├── embeddings.go         # Embedding generation service
│   │   ├── vectordb.go           # Vector database operations
│   │   ├── search.go             # Hybrid search implementation
│   │   └── chat.go               # Chat/LLM service
│   └── middleware/
│       ├── cors.go               # CORS configuration
│       ├── auth.go               # Authentication (if needed)
│       └── logging.go            # Request/response logging
├── pkg/
│   ├── utils/                   # Utility functions
│   └── types/                   # Shared types
├── scripts/
│   ├── generate-api.sh          # Script to run oapi-codegen
│   └── setup-db.sh              # Database setup script
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

## API Endpoints Design

### OpenAPI Schema Structure
Define the following endpoints in `api/openapi.yaml`:

#### 1. Bookmark Management
- `POST /api/bookmarks/import` - Import bookmarks from file
  - Request: multipart/form-data with bookmark file
  - Response: Import status and statistics
  
- `GET /api/bookmarks` - List all bookmarks
  - Query params: page, limit, filter, sort
  - Response: Paginated bookmark list
  
- `GET /api/bookmarks/{id}` - Get bookmark details
  - Response: Full bookmark with content and metadata
  
- `PUT /api/bookmarks/{id}` - Update bookmark
  - Request: Bookmark update payload
  - Response: Updated bookmark
  
- `DELETE /api/bookmarks/{id}` - Delete bookmark
  - Response: Success status

- `POST /api/bookmarks/{id}/rescrape` - Re-scrape bookmark content
  - Response: Updated bookmark with new content

#### 2. Search Endpoints
- `POST /api/search` - Hybrid search
  - Request: { query: string, limit: number, searchType: "semantic" | "keyword" | "hybrid" }
  - Response: Ranked search results with relevance scores

#### 3. Chat Endpoints
- `POST /api/chat` - Send chat message
  - Request: { message: string, conversationId?: string, context?: string[] }
  - Response: { reply: string, sources: Bookmark[], conversationId: string }
  
- `GET /api/chat/conversations` - List conversations
  - Response: List of conversation summaries
  
- `GET /api/chat/conversations/{id}` - Get conversation history
  - Response: Full conversation with messages

#### 4. System Endpoints
- `GET /api/health` - Health check
- `GET /api/stats` - System statistics (bookmark count, index status, etc.)

## Database Schema

### Tables Structure

```sql
-- Main bookmarks table
CREATE TABLE bookmarks (
    id TEXT PRIMARY KEY,
    url TEXT UNIQUE NOT NULL,
    title TEXT,
    description TEXT,
    content TEXT,
    scraped_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    folder_path TEXT,
    favicon_url TEXT,
    tags TEXT -- JSON array
);

-- Vector embeddings table (libSQL vector)
CREATE TABLE bookmark_embeddings (
    id TEXT PRIMARY KEY,
    bookmark_id TEXT NOT NULL,
    embedding VECTOR(1536), -- Adjust dimension based on model
    model_version TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (bookmark_id) REFERENCES bookmarks(id) ON DELETE CASCADE
);

-- FTS5 virtual table for full-text search
CREATE VIRTUAL TABLE bookmarks_fts USING fts5(
    title, 
    description, 
    content,
    content=bookmarks
);

-- Conversations table
CREATE TABLE conversations (
    id TEXT PRIMARY KEY,
    title TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Chat messages table
CREATE TABLE messages (
    id TEXT PRIMARY KEY,
    conversation_id TEXT NOT NULL,
    role TEXT CHECK(role IN ('user', 'assistant')),
    content TEXT NOT NULL,
    bookmark_refs TEXT, -- JSON array of bookmark IDs
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE
);
```

## Implementation Guidelines

### 1. Bookmark Import Service
- Support Firefox JSON format (bookmarks export)
- Support Chrome HTML format (bookmark manager export)
- Parse nested folder structures
- Extract metadata (title, URL, add_date, tags)

### 2. Content Scraping Service
- Use Go's built-in HTML parser or goquery
- Extract main content (use readability algorithm)
- Clean and normalize text
- Handle rate limiting and retries
- Store both raw HTML and cleaned text

### 3. Embedding Generation
- Batch process bookmarks for efficiency
- Use OpenAI's text-embedding-3-small model
- Store embeddings in libSQL vector column
- Implement chunking for long content (max 8191 tokens)
- Cache embeddings to avoid regeneration

### 4. Hybrid Search Implementation
```go
// Pseudo-code structure
type SearchService struct {
    // 1. Semantic search using vector similarity
    // 2. Keyword search using FTS5
    // 3. Combine results with weighted scoring
    // 4. Re-rank based on recency and relevance
}
```

### 5. Chat Service
- Implement RAG (Retrieval-Augmented Generation)
- Search relevant bookmarks based on user query
- Build context from top-k results
- Generate response using GPT-4 with context
- Track conversation history
- Include source citations in responses

### 6. Configuration Management
```go
type Config struct {
    Server struct {
        Port string
        Host string
    }
    Database struct {
        URL string
        MaxConnections int
    }
    OpenAI struct {
        APIKey string
        EmbeddingModel string
        ChatModel string
    }
    Search struct {
        SemanticWeight float64
        KeywordWeight float64
        MaxResults int
    }
}
```

## Error Handling
- Implement consistent error response format
- Use proper HTTP status codes
- Log errors with context
- Implement retry logic for external services
- Graceful degradation when services are unavailable

## Performance Considerations
- Implement connection pooling for database
- Use goroutines for parallel processing
- Cache frequently accessed data
- Implement pagination for large result sets
- Use batch operations for embeddings
- Implement request debouncing for search

## Security Considerations
- Validate and sanitize all inputs
- Implement rate limiting
- Use prepared statements for SQL queries
- Secure API key storage (environment variables)
- CORS configuration for frontend access
- Optional: Add JWT authentication

## Testing Strategy
- Unit tests for each service
- Integration tests for API endpoints
- Mock external services (OpenAI, web scraping)
- Test bookmark parsing with various formats
- Load testing for search performance

## Deployment Considerations
- Use environment variables for configuration
- Implement graceful shutdown
- Health check endpoint for monitoring
- Structured logging (JSON format)
- Metrics collection (Prometheus format)
- Docker containerization support

## Development Workflow
1. Define complete OpenAPI specification
2. Generate server stubs using oapi-codegen
3. Implement database layer with migrations
4. Build bookmark import and parsing logic
5. Implement scraping service
6. Set up embedding generation pipeline
7. Build hybrid search functionality
8. Implement chat/RAG system
9. Add middleware and error handling
10. Write comprehensive tests

## Example Code Generation Commands
```bash
# Generate server code from OpenAPI spec
oapi-codegen -package api -generate types,server,spec api/openapi.yaml > api/generated/server.gen.go

# Run migrations
migrate -path internal/database/migrations -database "sqlite://bookmarks.db" up

# Build and run
go build -o bin/server cmd/server/main.go
./bin/server
```

## Additional Features to Consider
- Bookmark deduplication
- Automatic tagging using NLP
- Scheduled re-scraping of bookmarks
- Export functionality
- Bookmark sharing capabilities
- Search history and analytics
- Smart summarization of bookmark content