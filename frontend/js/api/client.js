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

        try {
            const response = await fetch(url, config);
            
            if (!response.ok) {
                const errorText = await response.text();
                throw new Error(`HTTP ${response.status}: ${errorText}`);
            }
            
            const contentType = response.headers.get('content-type');
            if (contentType && contentType.includes('application/json')) {
                return await response.json();
            }
            
            return await response.text();
        } catch (error) {
            console.error('API request failed:', error);
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
     * Search bookmarks
     * @param {string} query - Search query
     * @param {Object} options - Search options
     * @returns {Promise} Search results
     */
    async search(query, options = {}) {
        const params = new URLSearchParams({
            q: query,
            ...options,
        });
        
        return await this.request(`/search?${params}`);
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