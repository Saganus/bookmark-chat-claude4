// Search component for hybrid bookmark search

class SearchComponent {
    constructor(apiClient) {
        console.log('🔍 SearchComponent constructor called with apiClient:', apiClient);
        this.apiClient = apiClient;
        this.isSearching = false;
        this.init();
    }

    init() {
        console.log('🔍 SearchComponent init called');
        this.bindEvents();
    }

    bindEvents() {
        console.log('🔍 Binding search events...');
        
        // Search button click
        $('#searchBtn').on('click', () => {
            console.log('🔍 Search button clicked!');
            this.performSearch();
        });

        // Enter key in search input
        $('#searchQuery').on('keypress', (e) => {
            console.log('🔍 Key pressed in search input:', e.which);
            if (e.which === 13) { // Enter key
                console.log('🔍 Enter key pressed, performing search');
                this.performSearch();
            }
        });

        // Clear results when query is empty
        $('#searchQuery').on('input', (e) => {
            console.log('🔍 Search input changed:', e.target.value);
            if (e.target.value.trim() === '') {
                this.clearResults();
            }
        });
        
        console.log('🔍 Search events bound successfully');
        console.log('🔍 Search button element found:', $('#searchBtn').length > 0);
        console.log('🔍 Search input element found:', $('#searchQuery').length > 0);
    }

    async performSearch() {
        console.log('🔍 performSearch called, isSearching:', this.isSearching);
        
        if (this.isSearching) {
            console.log('🔍 Already searching, returning');
            return;
        }

        const query = $('#searchQuery').val().trim();
        console.log('🔍 Search query:', query);
        
        if (!query) {
            console.log('🔍 No query provided');
            this.showError('Please enter a search query');
            return;
        }

        const searchType = $('#searchType').val();
        const limit = parseInt($('#searchLimit').val());
        
        console.log('🔍 Search parameters:', { query, searchType, limit });
        console.log('🔍 API client:', this.apiClient);

        this.showLoading();
        this.isSearching = true;

        try {
            console.log(`Performing search: query="${query}", type="${searchType}", limit=${limit}`);
            
            // Call the backend search API using the proper method
            const response = await this.apiClient.searchBookmarks(query, {
                limit: limit,
                searchType: searchType
            });

            console.log('Search API response:', response);
            this.displayResults(response, query);
        } catch (error) {
            console.error('Search failed:', error);
            this.showError(`Search failed: ${error.message}`);
        } finally {
            this.isSearching = false;
        }
    }

    showLoading() {
        const $results = $('#searchResults');
        $results.html(`
            <div class="search-loading">
                <div class="loading"></div>
                <span>Searching...</span>
            </div>
        `);
    }

    displayResults(response, query) {
        const $results = $('#searchResults');
        
        if (!response.results || response.results.length === 0) {
            $results.html(`
                <div class="search-no-results">
                    <h3>No results found</h3>
                    <p>Try adjusting your search query or search type</p>
                </div>
            `);
            return;
        }

        const resultsHtml = response.results.map((result, index) => 
            this.renderSearchResult(result, index + 1)
        ).join('');

        $results.html(`
            <div class="search-results-header">
                <p class="text-sm text-secondary mb-4">
                    Found ${response.total_results} result${response.total_results !== 1 ? 's' : ''} for "${query}"
                </p>
            </div>
            ${resultsHtml}
        `);
    }

    renderSearchResult(result, index) {
        const bookmark = result.bookmark;
        const score = (result.relevance_score * 100).toFixed(1);
        const snippet = result.snippet || this.generateSnippet(bookmark.content, 150);
        
        // Format dates - check for both null and undefined due to omitempty JSON tag
        const scrapedDate = bookmark.scraped_at ? 
            new Date(bookmark.scraped_at).toLocaleDateString() : 'Not scraped';
        
        // Clean and format folder path
        const folder = bookmark.folder_path && bookmark.folder_path !== '/' ? 
            bookmark.folder_path.replace(/^\/+|\/+$/g, '') : 'Root';

        return `
            <div class="search-result-item">
                <div class="search-result-header">
                    <h3 class="search-result-title">
                        <a href="${this.escapeHtml(bookmark.url)}" target="_blank" rel="noopener">
                            ${this.escapeHtml(bookmark.title || bookmark.url)}
                        </a>
                    </h3>
                    <span class="search-result-score">
                        ${score}% match
                    </span>
                </div>
                
                <div class="search-result-url">
                    ${this.escapeHtml(bookmark.url)}
                </div>
                
                ${snippet ? `
                    <div class="search-result-snippet">
                        ${this.escapeHtml(snippet)}...
                    </div>
                ` : ''}
                
                <div class="search-result-metadata">
                    <span class="search-result-folder">
                        📁 ${this.escapeHtml(folder)}
                    </span>
                    <span class="search-result-date">
                        Scraped: ${scrapedDate}
                    </span>
                    ${bookmark.tags ? `
                        <span class="search-result-tags">
                            🏷️ ${JSON.parse(bookmark.tags).map(tag => this.escapeHtml(tag)).join(', ')}
                        </span>
                    ` : ''}
                </div>
            </div>
        `;
    }

    generateSnippet(content, maxLength) {
        if (!content) return '';
        
        const cleanContent = content.replace(/<[^>]*>/g, '').replace(/\s+/g, ' ').trim();
        if (cleanContent.length <= maxLength) return cleanContent;
        
        return cleanContent.substring(0, maxLength).replace(/\s+\S*$/, '');
    }

    escapeHtml(text) {
        if (!text) return '';
        
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    showError(message) {
        const $results = $('#searchResults');
        $results.html(`
            <div class="search-error">
                <strong>Error:</strong> ${this.escapeHtml(message)}
            </div>
        `);
    }

    clearResults() {
        const $results = $('#searchResults');
        $results.html(`
            <p class="empty-state">Enter a search query to find relevant bookmarks</p>
        `);
    }
}

// Export for global access
window.SearchComponent = SearchComponent;