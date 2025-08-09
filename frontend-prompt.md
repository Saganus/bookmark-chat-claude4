# Frontend System Implementation Prompt

## Project Overview
Build a responsive web interface for a bookmark chat system using vanilla JavaScript with jQuery, providing an intuitive chat interface to interact with scraped bookmarks through natural language queries.

## Core Requirements

### Technology Stack
- **HTML5** with semantic markup
- **CSS3** with CSS Grid/Flexbox for layouts
- **JavaScript ES6+** with jQuery for DOM manipulation
- **No build tools required** (pure client-side code)
- **Responsive Design** - Mobile-first approach
- **Local Storage** for client-side preferences
- **Fetch API** for HTTP requests (with jQuery.ajax as fallback)

## Project Structure
```
bookmark-chat-frontend/
├── index.html                    # Main application page
├── css/
│   ├── main.css                 # Main stylesheet
│   ├── components/              
│   │   ├── chat.css             # Chat interface styles
│   │   ├── bookmarks.css       # Bookmark list/grid styles
│   │   ├── search.css          # Search interface styles
│   │   └── modal.css           # Modal dialog styles
│   └── responsive.css          # Media queries and responsive styles
├── js/
│   ├── app.js                  # Main application logic
│   ├── api/
│   │   └── client.js           # API client wrapper
│   ├── components/
│   │   ├── chat.js             # Chat component logic
│   │   ├── bookmarks.js        # Bookmark management
│   │   ├── search.js           # Search functionality
│   │   └── import.js           # Import functionality
│   ├── utils/
│   │   ├── dom.js              # DOM manipulation helpers
│   │   ├── storage.js          # Local storage wrapper
│   │   └── format.js           # Formatting utilities
│   └── config.js               # Configuration constants
├── assets/
│   ├── icons/                  # SVG icons
│   └── images/                 # Static images
└── README.md
```

## UI Components Design

### 1. Main Layout Structure
```html
<!-- Mobile-first responsive layout -->
<div class="app-container">
    <header class="app-header">
        <!-- Logo, navigation, user menu -->
    </header>
    
    <nav class="sidebar" id="sidebar">
        <!-- Collapsible on mobile -->
        <div class="sidebar-section">
            <!-- Quick actions -->
        </div>
        <div class="sidebar-section">
            <!-- Bookmark folders -->
        </div>
    </nav>
    
    <main class="main-content">
        <div class="chat-container" id="chatView">
            <!-- Primary chat interface -->
        </div>
        <div class="bookmarks-container" id="bookmarksView" style="display:none;">
            <!-- Bookmark management view -->
        </div>
    </main>
    
    <div class="mobile-nav">
        <!-- Bottom navigation for mobile -->
    </div>
</div>
```

### 2. Chat Interface Component
```html
<div class="chat-interface">
    <div class="chat-header">
        <h2 class="chat-title">Chat with Your Bookmarks</h2>
        <div class="chat-actions">
            <button class="btn-new-chat">New Chat</button>
            <button class="btn-history">History</button>
        </div>
    </div>
    
    <div class="messages-container" id="messagesContainer">
        <!-- Scrollable message list -->
        <div class="message message--user">
            <div class="message-content">User message</div>
        </div>
        <div class="message message--assistant">
            <div class="message-content">
                <p>Assistant response</p>
                <div class="message-sources">
                    <!-- Bookmark citations -->
                </div>
            </div>
        </div>
    </div>
    
    <div class="chat-input-container">
        <div class="search-suggestions" id="searchSuggestions">
            <!-- Dynamic suggestions -->
        </div>
        <form class="chat-form" id="chatForm">
            <textarea 
                class="chat-input" 
                placeholder="Ask about your bookmarks..."
                rows="1"
            ></textarea>
            <button type="submit" class="btn-send">Send</button>
        </form>
    </div>
</div>
```

### 3. Bookmark Management Interface
```html
<div class="bookmarks-manager">
    <div class="bookmarks-header">
        <div class="bookmarks-actions">
            <button class="btn-import">Import Bookmarks</button>
            <button class="btn-view-mode" data-mode="grid">
                <!-- Toggle grid/list view -->
            </button>
        </div>
        <div class="bookmarks-search">
            <input type="search" placeholder="Search bookmarks...">
            <select class="filter-dropdown">
                <option>All Folders</option>
                <!-- Dynamic folder options -->
            </select>
        </div>
    </div>
    
    <div class="bookmarks-stats">
        <!-- Statistics bar -->
        <span class="stat">Total: <strong id="totalCount">0</strong></span>
        <span class="stat">Indexed: <strong id="indexedCount">0</strong></span>
    </div>
    
    <div class="bookmarks-grid" id="bookmarksGrid">
        <!-- Bookmark cards -->
        <article class="bookmark-card">
            <div class="bookmark-favicon">
                <img src="" alt="">
            </div>
            <h3 class="bookmark-title"></h3>
            <p class="bookmark-description"></p>
            <div class="bookmark-actions">
                <button class="btn-icon" title="View">👁️</button>
                <button class="btn-icon" title="Edit">✏️</button>
                <button class="btn-icon" title="Delete">🗑️</button>
            </div>
        </article>
    </div>
</div>
```

### 4. Import Modal
```html
<div class="modal" id="importModal">
    <div class="modal-content">
        <div class="modal-header">
            <h2>Import Bookmarks</h2>
            <button class="modal-close">&times;</button>
        </div>
        <div class="modal-body">
            <div class="import-options">
                <div class="import-option">
                    <input type="radio" name="importType" value="firefox" id="firefox">
                    <label for="firefox">
                        <strong>Firefox</strong>
                        <small>JSON format from bookmark backup</small>
                    </label>
                </div>
                <div class="import-option">
                    <input type="radio" name="importType" value="chrome" id="chrome">
                    <label for="chrome">
                        <strong>Chrome</strong>
                        <small>HTML format from bookmark manager</small>
                    </label>
                </div>
            </div>
            
            <div class="file-upload-area" id="uploadArea">
                <input type="file" id="fileInput" accept=".json,.html">
                <label for="fileInput" class="file-upload-label">
                    <span>Drop file here or click to browse</span>
                </label>
            </div>
            
            <div class="import-progress" style="display:none;">
                <div class="progress-bar">
                    <div class="progress-fill"></div>
                </div>
                <p class="progress-text">Importing...</p>
            </div>
        </div>
    </div>
</div>
```

## CSS Design System

### Color Palette
```css
:root {
    /* Light mode */
    --color-primary: #2563eb;
    --color-primary-dark: #1d4ed8;
    --color-secondary: #10b981;
    --color-background: #ffffff;
    --color-surface: #f9fafb;
    --color-text: #111827;
    --color-text-secondary: #6b7280;
    --color-border: #e5e7eb;
    --color-error: #ef4444;
    --color-success: #10b981;
    
    /* Spacing */
    --spacing-xs: 0.25rem;
    --spacing-sm: 0.5rem;
    --spacing-md: 1rem;
    --spacing-lg: 1.5rem;
    --spacing-xl: 2rem;
    
    /* Typography */
    --font-sans: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
    --font-mono: "SF Mono", Monaco, "Cascadia Code", monospace;
    
    /* Breakpoints */
    --breakpoint-sm: 640px;
    --breakpoint-md: 768px;
    --breakpoint-lg: 1024px;
}

/* Dark mode */
@media (prefers-color-scheme: dark) {
    :root {
        --color-background: #111827;
        --color-surface: #1f2937;
        --color-text: #f9fafb;
        --color-text-secondary: #9ca3af;
        --color-border: #374151;
    }
}
```

### Responsive Grid System
```css
.container {
    width: 100%;
    max-width: 1280px;
    margin: 0 auto;
    padding: 0 var(--spacing-md);
}

.grid {
    display: grid;
    gap: var(--spacing-md);
    grid-template-columns: repeat(auto-fill, minmax(250px, 1fr));
}

/* Mobile-first responsive */
@media (min-width: 640px) {
    .grid { grid-template-columns: repeat(2, 1fr); }
}

@media (min-width: 768px) {
    .grid { grid-template-columns: repeat(3, 1fr); }
}

@media (min-width: 1024px) {
    .grid { grid-template-columns: repeat(4, 1fr); }
}
```

## JavaScript Architecture

### API Client Module
```javascript
// js/api/client.js
class APIClient {
    constructor(baseURL = '/api') {
        this.baseURL = baseURL;
    }
    
    async request(endpoint, options = {}) {
        const url = `${this.baseURL}${endpoint}`;
        const config = {
            ...options,
            headers: {
                'Content-Type': 'application/json',
                ...options.headers,
            },
        };
        
        try {
            const response = await fetch(url, config);
            if (!response.ok) throw new Error(`HTTP ${response.status}`);
            return await response.json();
        } catch (error) {
            console.error('API request failed:', error);
            throw error;
        }
    }
    
    // Bookmark methods
    async getBookmarks(params = {}) { /* ... */ }
    async importBookmarks(file, type) { /* ... */ }
    async deleteBookmark(id) { /* ... */ }
    
    // Chat methods
    async sendMessage(message, conversationId) { /* ... */ }
    async getConversations() { /* ... */ }
    
    // Search methods
    async search(query, options) { /* ... */ }
}
```

### Chat Component
```javascript
// js/components/chat.js
class ChatComponent {
    constructor(container, api) {
        this.container = container;
        this.api = api;
        this.conversationId = null;
        this.init();
    }
    
    init() {
        this.bindEvents();
        this.loadConversation();
    }
    
    bindEvents() {
        $('#chatForm').on('submit', (e) => this.handleSubmit(e));
        $('.chat-input').on('input', (e) => this.handleInput(e));
        // Auto-resize textarea
        // Show suggestions
        // Handle keyboard shortcuts
    }
    
    async handleSubmit(e) {
        e.preventDefault();
        const message = $('.chat-input').val().trim();
        if (!message) return;
        
        // Add user message to UI
        this.addMessage(message, 'user');
        
        // Clear input
        $('.chat-input').val('').trigger('input');
        
        // Show typing indicator
        this.showTypingIndicator();
        
        try {
            // Send to API
            const response = await this.api.sendMessage(message, this.conversationId);
            
            // Update conversation ID
            this.conversationId = response.conversationId;
            
            // Add assistant response
            this.addMessage(response.reply, 'assistant', response.sources);
        } catch (error) {
            this.showError('Failed to send message');
        } finally {
            this.hideTypingIndicator();
        }
    }
    
    addMessage(content, role, sources = []) {
        const messageHTML = this.renderMessage(content, role, sources);
        $('#messagesContainer').append(messageHTML);
        this.scrollToBottom();
    }
    
    renderMessage(content, role, sources) {
        // Generate message HTML with markdown support
        // Include source citations if provided
    }
}
```

### Bookmark Manager
```javascript
// js/components/bookmarks.js
class BookmarkManager {
    constructor(container, api) {
        this.container = container;
        this.api = api;
        this.bookmarks = [];
        this.viewMode = 'grid';
        this.init();
    }
    
    async init() {
        await this.loadBookmarks();
        this.bindEvents();
        this.render();
    }
    
    bindEvents() {
        // Import button
        $('.btn-import').on('click', () => this.showImportModal());
        
        // View mode toggle
        $('.btn-view-mode').on('click', (e) => this.toggleViewMode(e));
        
        // Search input
        $('.bookmarks-search input').on('input', debounce((e) => {
            this.filterBookmarks(e.target.value);
        }, 300));
        
        // Bookmark actions (delegation)
        $(this.container).on('click', '.bookmark-card', (e) => {
            this.handleBookmarkAction(e);
        });
    }
    
    async loadBookmarks() {
        try {
            this.bookmarks = await this.api.getBookmarks();
            this.updateStats();
        } catch (error) {
            this.showError('Failed to load bookmarks');
        }
    }
    
    render() {
        const html = this.bookmarks.map(bookmark => 
            this.renderBookmarkCard(bookmark)
        ).join('');
        $('#bookmarksGrid').html(html);
    }
}
```

### Import Handler
```javascript
// js/components/import.js
class ImportHandler {
    constructor(api) {
        this.api = api;
        this.init();
    }
    
    init() {
        this.setupDragDrop();
        this.bindEvents();
    }
    
    setupDragDrop() {
        const dropArea = $('#uploadArea');
        
        ['dragenter', 'dragover', 'dragleave', 'drop'].forEach(eventName => {
            dropArea.on(eventName, (e) => {
                e.preventDefault();
                e.stopPropagation();
            });
        });
        
        dropArea.on('dragenter dragover', () => {
            dropArea.addClass('drag-active');
        });
        
        dropArea.on('dragleave drop', () => {
            dropArea.removeClass('drag-active');
        });
        
        dropArea.on('drop', (e) => {
            const files = e.originalEvent.dataTransfer.files;
            this.handleFiles(files);
        });
    }
    
    async handleFiles(files) {
        if (files.length === 0) return;
        
        const file = files[0];
        const type = $('input[name="importType"]:checked').val();
        
        if (!type) {
            alert('Please select browser type');
            return;
        }
        
        // Show progress
        $('.import-progress').show();
        
        try {
            const result = await this.api.importBookmarks(file, type);
            this.showSuccess(`Imported ${result.count} bookmarks`);
            this.closeModal();
            // Refresh bookmark list
            window.bookmarkManager.loadBookmarks();
        } catch (error) {
            this.showError('Import failed: ' + error.message);
        } finally {
            $('.import-progress').hide();
        }
    }
}
```

## Responsive Design Patterns

### Mobile-First Approach
```css
/* Base mobile styles */
.app-container {
    display: flex;
    flex-direction: column;
    min-height: 100vh;
}

.sidebar {
    position: fixed;
    left: -100%;
    transition: left 0.3s ease;
    z-index: 1000;
}

.sidebar.active {
    left: 0;
}

/* Tablet and up */
@media (min-width: 768px) {
    .app-container {
        flex-direction: row;
    }
    
    .sidebar {
        position: relative;
        left: 0;
        width: 250px;
    }
    
    .mobile-nav {
        display: none;
    }
}

/* Desktop */
@media (min-width: 1024px) {
    .sidebar {
        width: 300px;
    }
    
    .chat-container {
        max-width: 800px;
        margin: 0 auto;
    }
}
```

### Touch-Friendly Interactions
```css
/* Minimum touch target size */
button, .btn, .clickable {
    min-height: 44px;
    min-width: 44px;
    padding: var(--spacing-sm) var(--spacing-md);
}

/* Prevent text selection on buttons */
button {
    user-select: none;
    -webkit-tap-highlight-color: transparent;
}

/* Smooth scrolling */
.messages-container {
    overflow-y: auto;
    -webkit-overflow-scrolling: touch;
    scroll-behavior: smooth;
}
```

## Performance Optimizations

### Lazy Loading
```javascript
// Intersection Observer for lazy loading bookmarks
const observerOptions = {
    root: null,
    rootMargin: '50px',
    threshold: 0.01
};

const imageObserver = new IntersectionObserver((entries) => {
    entries.forEach(entry => {
        if (entry.isIntersecting) {
            const img = entry.target;
            img.src = img.dataset.src;
            imageObserver.unobserve(img);
        }
    });
}, observerOptions);

// Observe all bookmark favicon images
document.querySelectorAll('.bookmark-favicon img').forEach(img => {
    imageObserver.observe(img);
});
```

### Debouncing and Throttling
```javascript
function debounce(func, wait) {
    let timeout;
    return function executedFunction(...args) {
        const later = () => {
            clearTimeout(timeout);
            func(...args);
        };
        clearTimeout(timeout);
        timeout = setTimeout(later, wait);
    };
}

function throttle(func, limit) {
    let inThrottle;
    return function(...args) {
        if (!inThrottle) {
            func.apply(this, args);
            inThrottle = true;
            setTimeout(() => inThrottle = false, limit);
        }
    };
}
```

## Accessibility Features
- Semantic HTML5 elements
- ARIA labels and roles
- Keyboard navigation support
- Focus management
- Screen reader compatibility
- High contrast mode support
- Reduced motion preferences

## Error Handling
- User-friendly error messages
- Retry mechanisms for failed requests
- Offline detection and handling
- Graceful degradation
- Loading states and skeleton screens

## Local Storage Schema
```javascript
// Store user preferences
localStorage.setItem('userPrefs', JSON.stringify({
    theme: 'auto', // 'light', 'dark', 'auto'
    viewMode: 'grid', // 'grid', 'list'
    sidebarCollapsed: false,
    fontSize: 'medium' // 'small', 'medium', 'large'
}));

// Cache recent searches
localStorage.setItem('recentSearches', JSON.stringify([
    { query: 'javascript tutorials', timestamp: Date.now() }
]));

// Store conversation drafts
localStorage.setItem('chatDraft', 'Unsent message...');
```

## Progressive Enhancement
1. Core HTML functionality works without JavaScript
2. CSS provides basic styling and layout
3. JavaScript adds enhanced interactions
4. Features degrade gracefully
5. Network requests have fallbacks

## Testing Approach
- Cross-browser testing (Chrome, Firefox, Safari, Edge)
- Mobile device testing (iOS, Android)
- Accessibility testing with screen readers
- Performance testing with Lighthouse
- Manual testing of all user flows