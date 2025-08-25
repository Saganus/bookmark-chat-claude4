// API client for bookmark management and scraping

class APIClient {
    constructor(baseURL = null) {
        this.baseURL = baseURL || (window.CONFIG && window.CONFIG.API.BASE_URL) || '/api';
        this.timeout = (window.CONFIG && window.CONFIG.API.TIMEOUT) || 30000;
    }

    /**
     * Make a generic API request
     * @param {string} endpoint - API endpoint
     * @param {Object} options - Request options
     * @returns {Promise} Response data
     */
    async request(endpoint, options = {}) {
        const url = `${this.baseURL}${endpoint}`;
        const config = {
            ...options,
            headers: {
                'Content-Type': 'application/json',
                ...options.headers,
            },
        };

        console.log('🌐 Making API request:', { url, method: config.method || 'GET', body: config.body });

        try {
            const response = await fetch(url, config);
            console.log('🌐 API response status:', response.status);
            
            if (!response.ok) {
                const errorText = await response.text();
                console.error('🌐 API error response:', errorText);
                throw new Error(`HTTP ${response.status}: ${errorText}`);
            }
            
            const contentType = response.headers.get('content-type');
            console.log('🌐 Response content type:', contentType);
            
            if (contentType && contentType.includes('application/json')) {
                const jsonResponse = await response.json();
                console.log('🌐 JSON response:', jsonResponse);
                return jsonResponse;
            }
            
            const textResponse = await response.text();
            console.log('🌐 Text response:', textResponse);
            return textResponse;
        } catch (error) {
            console.error('🌐 API request failed:', error);
            throw error;
        }
    }

    /**
     * Import bookmarks from a file
     * @param {File} file - Bookmark file
     * @param {string} type - Browser type (chrome, firefox)
     * @returns {Promise} Import result
     */
    async importBookmarks(file, type) {
        const formData = new FormData();
        formData.append('file', file);
        formData.append('type', type);

        const response = await fetch(`${this.baseURL}/bookmarks/import`, {
            method: 'POST',
            body: formData,
        });

        if (!response.ok) {
            const errorText = await response.text();
            throw new Error(`Import failed: ${errorText}`);
        }

        return await response.json();
    }

    /**
     * Get all bookmarks
     * @returns {Promise} Bookmarks data
     */
    async getBookmarks() {
        return await this.request('/bookmarks');
    }

    /**
     * Get bookmark by ID
     * @param {string} id - Bookmark ID
     * @returns {Promise} Bookmark data
     */
    async getBookmark(id) {
        return await this.request(`/bookmarks/${id}`);
    }

    /**
     * Delete bookmark by ID
     * @param {string} id - Bookmark ID
     * @returns {Promise} Delete result
     */
    async deleteBookmark(id) {
        return await this.request(`/bookmarks/${id}`, {
            method: 'DELETE',
        });
    }

    /**
     * Re-scrape bookmark content by ID
     * @param {string} id - Bookmark ID
     * @returns {Promise} Updated bookmark with scraped content
     */
    async rescrapeBookmark(id) {
        return await this.request(`/bookmarks/${id}/rescrape`, {
            method: 'POST',
        });
    }

    /**
     * Start scraping selected bookmarks
     * @param {Array} bookmarkIds - Array of bookmark IDs to scrape
     * @returns {Promise} Scraping start result
     */
    async startScraping(bookmarkIds) {
        return await this.request('/scraping/start', {
            method: 'POST',
            body: JSON.stringify({ bookmark_ids: bookmarkIds }),
        });
    }

    /**
     * Pause the current scraping process
     * @returns {Promise} Pause result
     */
    async pauseScraping() {
        return await this.request('/scraping/pause', {
            method: 'POST',
        });
    }

    /**
     * Resume the paused scraping process
     * @returns {Promise} Resume result
     */
    async resumeScraping() {
        return await this.request('/scraping/resume', {
            method: 'POST',
        });
    }

    /**
     * Stop the current scraping process
     * @returns {Promise} Stop result
     */
    async stopScraping() {
        return await this.request('/scraping/stop', {
            method: 'POST',
        });
    }

    /**
     * Get current scraping status
     * @returns {Promise} Scraping status
     */
    async getScrapingStatus() {
        return await this.request('/scraping/status');
    }

    /**
     * Search bookmarks using POST method per OpenAPI spec
     * @param {string} query - Search query
     * @param {Object} options - Search options (limit, searchType)
     * @returns {Promise} Search results
     */
    async searchBookmarks(query, options = {}) {
        console.log('🌐 APIClient.searchBookmarks called:', { query, options });
        console.log('🌐 API base URL:', this.baseURL);
        
        const requestPayload = {
            query: query,
            limit: options.limit || 20,
            searchType: options.searchType || 'hybrid'
        };
        
        console.log('🌐 Search request payload:', requestPayload);
        
        try {
            const result = await this.request('/search', {
                method: 'POST',
                body: JSON.stringify(requestPayload)
            });
            console.log('🌐 Search API response:', result);
            return result;
        } catch (error) {
            console.error('🌐 Search API error:', error);
            throw error;
        }
    }

    /**
     * Legacy search method (kept for backward compatibility)
     * @param {string} query - Search query
     * @param {Object} options - Search options
     * @returns {Promise} Search results
     */
    async search(query, options = {}) {
        return this.searchBookmarks(query, options);
    }

    /**
     * Get system health status
     * @returns {Promise} Health status
     */
    async getHealth() {
        return await this.request('/health');
    }

    /**
     * Get system statistics
     * @returns {Promise} System stats
     */
    async getStats() {
        return await this.request('/stats');
    }
}

// Export for use in other modules
window.APIClient = APIClient;