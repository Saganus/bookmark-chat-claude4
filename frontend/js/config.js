// Application configuration

const CONFIG = {
    // API Configuration
    API: {
        // Base URL for the backend API
        // Using relative path since frontend and backend are served from same server
        BASE_URL: '/api',
        
        // Request timeout in milliseconds
        TIMEOUT: 30000,
        
        // Retry configuration
        MAX_RETRIES: 3,
        RETRY_DELAY: 1000
    },
    
    // UI Configuration
    UI: {
        // Status check interval for scraping (milliseconds)
        STATUS_CHECK_INTERVAL: 2000,
        
        // Toast notification auto-hide duration (milliseconds)
        TOAST_DURATION: 5000,
        
        // File upload size limit (bytes) - 10MB
        MAX_FILE_SIZE: 10 * 1024 * 1024,
        
        // Debounce delay for search (milliseconds)
        SEARCH_DEBOUNCE: 300
    },
    
    // File upload configuration
    UPLOAD: {
        // Accepted file types by browser
        ACCEPTED_TYPES: {
            chrome: ['.html', '.htm'],
            firefox: ['.html', '.htm']
        },
        
        // Maximum file size in bytes
        MAX_SIZE: 10 * 1024 * 1024 // 10MB
    },
    
    // Feature flags
    FEATURES: {
        // Enable debug logging
        DEBUG: false,
        
        // Enable offline support
        OFFLINE_SUPPORT: false,
        
        // Enable analytics
        ANALYTICS: false
    }
};

// Environment-specific overrides
if (window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1') {
    CONFIG.FEATURES.DEBUG = true;
}

// Override API base URL if needed for different environments
// For development with separate servers, uncomment and modify:
// if (window.location.hostname === 'localhost') {
//     CONFIG.API.BASE_URL = 'http://localhost:8080/api';
// }

// Make config globally available
window.CONFIG = CONFIG;

// Export for module systems
if (typeof module !== 'undefined' && module.exports) {
    module.exports = CONFIG;
}