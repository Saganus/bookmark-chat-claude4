// Search component for hybrid bookmark search

class SearchComponent {
    constructor(apiClient) {
        this.apiClient = apiClient;
        this.isSearching = false;
        this.init();
    }

    init() {
        this.bindEvents();
    }

    bindEvents() {
        // Search button click
        $('#searchBtn').on('click', () => this.performSearch());

        // Enter key in search input
        $('#searchQuery').on('keypress', (e) => {
            if (e.which === 13) { // Enter key
                this.performSearch();
            }
        });

        // Clear results when query is empty
        $('#searchQuery').on('input', (e) => {
            if (e.target.value.trim() === '') {
                this.clearResults();
            }
        });
    }

    async performSearch() {
        if (this.isSearching) return;

        const query = $('#searchQuery').val().trim();
        if (!query) {
            this.showError('Please enter a search query');
            return;
        }

        const searchType = $('#searchType').val();
        const limit = parseInt($('#searchLimit').val());

        this.showLoading();
        this.isSearching = true;

        try {
            // Call the backend search API using the proper method
            const response = await this.apiClient.searchBookmarks(query, {
                limit: limit,
                searchType: searchType
            });

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
        
        // Format dates
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

// Initialize search component when page loads
$(document).ready(() => {
    // Wait for API client to be available
    if (window.apiClient) {
        window.searchComponent = new SearchComponent(window.apiClient);
    } else {
        // Retry after a short delay if API client isn't ready
        setTimeout(() => {
            if (window.apiClient) {
                window.searchComponent = new SearchComponent(window.apiClient);
            }
        }, 100);
    }
});