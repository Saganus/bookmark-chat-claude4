// Main application entry point

class BookmarkApp {
    constructor() {
        this.api = new APIClient();
        this.currentTab = 'bookmarks';
        this.init();
    }

    init() {
        this.setupTabNavigation();
        this.initializeComponents();
        this.bindGlobalEvents();
        
        // Show initial tab
        this.showTab('bookmarks');
    }

    setupTabNavigation() {
        $('.tab-btn').on('click', (e) => {
            const tabName = $(e.target).data('tab');
            this.showTab(tabName);
        });
    }

    showTab(tabName) {
        // Update tab buttons
        $('.tab-btn').removeClass('active');
        $(`.tab-btn[data-tab="${tabName}"]`).addClass('active');
        
        // Update tab content
        $('.tab-content').removeClass('active');
        $(`#${tabName}Tab`).addClass('active');
        
        this.currentTab = tabName;
        
        // Refresh data when switching tabs
        this.refreshTabData(tabName);
        
        // Update URL hash for deep linking
        if (history.replaceState) {
            history.replaceState(null, null, `#${tabName}`);
        }
    }

    refreshTabData(tabName) {
        // Refresh data for the active tab to ensure components show current database state
        switch (tabName) {
            case 'bookmarks':
                if (window.bookmarkManager) {
                    window.bookmarkManager.loadBookmarks();
                }
                break;
            case 'scraping':
                if (window.scrapingManager) {
                    window.scrapingManager.loadBookmarks();
                }
                break;
            case 'search':
                // Search tab doesn't need data refresh, it loads on demand
                break;
            case 'categories':
                if (window.categoriesManager) {
                    window.categoriesManager.loadCategories();
                }
                break;
        }
    }

    initializeComponents() {
        // Initialize components
        window.importHandler = new ImportHandler(this.api);
        window.bookmarkManager = new BookmarkManager(this.api);
        window.scrapingManager = new ScrapingManager(this.api);
        window.searchManager = new SearchComponent(this.api);
        window.categoriesManager = new CategoriesComponent(this.api);
        
        console.log('Application components initialized');
    }

    bindGlobalEvents() {
        // Handle browser back/forward buttons
        $(window).on('hashchange', () => {
            const hash = window.location.hash.substring(1);
            if (hash && ['bookmarks', 'scraping', 'search', 'categories'].includes(hash)) {
                this.showTab(hash);
            }
        });

        // Handle initial hash on page load
        const initialHash = window.location.hash.substring(1);
        if (initialHash && ['bookmarks', 'scraping', 'search', 'categories'].includes(initialHash)) {
            this.showTab(initialHash);
        }

        // Keyboard shortcuts
        $(document).on('keydown', (e) => {
            // Tab switching with Ctrl/Cmd + 1/2/3
            if ((e.ctrlKey || e.metaKey) && !e.shiftKey && !e.altKey) {
                switch (e.which) {
                    case 49: // 1
                        e.preventDefault();
                        this.showTab('bookmarks');
                        break;
                    case 50: // 2
                        e.preventDefault();
                        this.showTab('scraping');
                        break;
                    case 51: // 3
                        e.preventDefault();
                        this.showTab('search');
                        break;
                    case 52: // 4
                        e.preventDefault();
                        this.showTab('categories');
                        break;
                }
            }
            
            // Import shortcut: Ctrl/Cmd + I
            if ((e.ctrlKey || e.metaKey) && e.which === 73 && !e.shiftKey && !e.altKey) {
                e.preventDefault();
                $('#importBtn').click();
            }
        });

        // Handle errors globally
        $(document).on('ajaxError', (event, jqXHR, ajaxSettings, thrownError) => {
            console.error('AJAX Error:', thrownError);
            
            // Don't show error for status polling (it's expected to fail sometimes)
            if (!ajaxSettings.url.includes('/scraping/status')) {
                showToast('Network error occurred. Please check your connection.', 'error');
            }
        });

        // Handle window beforeunload if scraping is in progress
        $(window).on('beforeunload', () => {
            if (window.scrapingManager && 
                (window.scrapingManager.scrapingStatus === 'running' || 
                 window.scrapingManager.scrapingStatus === 'paused')) {
                return 'Scraping is in progress. Are you sure you want to leave?';
            }
        });

        // Clean up resources when page unloads
        $(window).on('unload', () => {
            if (window.scrapingManager) {
                window.scrapingManager.destroy();
            }
        });
    }

    // Public methods for external access
    refreshBookmarks() {
        if (window.bookmarkManager) {
            window.bookmarkManager.loadBookmarks();
        }
        if (window.scrapingManager) {
            window.scrapingManager.loadBookmarks();
        }
    }

    getCurrentTab() {
        return this.currentTab;
    }

    // Health check method
    async checkHealth() {
        try {
            const health = await this.api.getHealth();
            console.log('System health:', health);
            return health;
        } catch (error) {
            console.error('Health check failed:', error);
            showToast('Unable to connect to server. Please check if the backend is running.', 'error');
            return null;
        }
    }
}

// Initialize application when DOM is ready
$(document).ready(() => {
    // Check if jQuery is loaded
    if (typeof $ === 'undefined') {
        console.error('jQuery is not loaded');
        return;
    }

    // Check if required classes are loaded
    const requiredClasses = ['APIClient', 'ImportHandler', 'BookmarkManager', 'ScrapingManager', 'SearchComponent', 'CategoriesComponent'];
    const missingClasses = requiredClasses.filter(className => typeof window[className] === 'undefined');
    
    if (missingClasses.length > 0) {
        console.error('Missing required classes:', missingClasses);
        showToast('Failed to load application components. Please refresh the page.', 'error');
        return;
    }

    // Initialize the application
    try {
        window.app = new BookmarkApp();
        console.log('Bookmark Application initialized successfully');
        
        // Perform initial health check
        window.app.checkHealth();
        
    } catch (error) {
        console.error('Failed to initialize application:', error);
        showToast('Failed to initialize application. Please refresh the page.', 'error');
    }
});

// Export for debugging and external access
window.BookmarkApp = BookmarkApp;