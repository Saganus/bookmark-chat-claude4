// Scraping functionality for bookmark content extraction

class ScrapingManager {
    constructor(api) {
        this.api = api;
        this.bookmarks = [];
        this.selectedBookmarks = new Set();
        this.scrapingStatus = 'idle'; // idle, running, paused, stopped
        this.statusCheckInterval = null;
        this.container = $('#scrapingTree');
        this.init();
    }

    async init() {
        await this.loadBookmarks();
        this.bindEvents();
        this.updateUI();
    }

    bindEvents() {
        // Control buttons
        $('#startScrapingBtn').on('click', () => this.startScraping());
        $('#pauseBtn').on('click', () => this.togglePause());
        $('#stopBtn').on('click', () => this.stopScraping());
        
        // Select all checkbox
        $('#selectAllBtn').on('change', (e) => this.toggleSelectAll(e.target.checked));

        // Individual bookmark checkboxes
        this.container.on('change', '.scraping-checkbox', (e) => {
            this.handleBookmarkSelection($(e.target));
        });

        // Folder toggle functionality
        this.container.on('click', '.folder-toggle', (e) => {
            e.preventDefault();
            e.stopPropagation();
            this.toggleFolder($(e.target));
        });

        // Tree item clicks for folder expansion
        this.container.on('click', '.tree-item-content', (e) => {
            const $target = $(e.target);
            
            // Don't toggle if clicking on checkbox, link, status, or scrape button
            if ($target.is('input, a, .bookmark-status, .scrape-bookmark-btn') || $target.closest('input, a, .bookmark-status, .scrape-bookmark-btn').length) {
                return;
            }

            const $toggle = $(e.currentTarget).find('.folder-toggle');
            if ($toggle.length) {
                this.toggleFolder($toggle);
            }
        });

        // Individual scrape button clicks
        this.container.on('click', '.scrape-bookmark-btn', async (e) => {
            e.preventDefault();
            e.stopPropagation();
            await this.scrapeIndividualBookmark($(e.target));
        });
    }

    async loadBookmarks() {
        try {
            // Use the same data source as bookmark manager
            if (window.bookmarkManager && window.bookmarkManager.bookmarks) {
                this.bookmarks = window.bookmarkManager.bookmarks;
            } else {
                const response = await this.api.getBookmarks();
                this.bookmarks = response.bookmarks || response || [];
            }
            this.render();
        } catch (error) {
            console.error('Failed to load bookmarks for scraping:', error);
            this.showError('Failed to load bookmarks. Please try importing bookmarks first.');
        }
    }

    render() {
        if (!this.bookmarks || this.bookmarks.length === 0) {
            this.container.html('<p class="empty-state">No bookmarks available for scraping. Please import bookmarks first.</p>');
            return;
        }

        const treeHtml = this.renderScrapingTree(this.bookmarks);
        this.container.html(`<ul class="tree-list">${treeHtml}</ul>`);
        this.updateSelectAllState();
    }

    renderScrapingTree(items) {
        if (!items || !Array.isArray(items)) {
            return '';
        }

        return items.map(item => {
            const isFolder = item.type === 'folder' || (item.children && item.children.length > 0);
            return this.renderScrapingTreeItem(item, isFolder);
        }).join('');
    }

    renderScrapingTreeItem(item, isFolder = false) {
        const nodeId = item.id || generateId();
        const hasChildren = item.children && item.children.length > 0;
        
        let html = `
            <li class="tree-item" data-id="${nodeId}">
                <div class="tree-item-content">
        `;
        
        // Checkbox for both folders and bookmarks
        const isSelected = this.selectedBookmarks.has(nodeId);
        html += `<input type="checkbox" class="scraping-checkbox" data-id="${nodeId}" ${isSelected ? 'checked' : ''}>`;
        
        if (isFolder && hasChildren) {
            html += `<button class="folder-toggle" aria-label="Expand folder">▶</button>`;
        } else {
            html += `<span class="tree-icon"></span>`;
        }
        
        html += `<span class="tree-icon ${isFolder ? 'folder' : 'bookmark'}">${isFolder ? '📁' : '🔖'}</span>`;
        
        if (isFolder) {
            html += `<span class="tree-text folder-name">${escapeHtml(item.title || item.name || 'Untitled Folder')}</span>`;
        } else {
            const url = item.url || '#';
            const title = item.title || item.name || 'Untitled Bookmark';
            html += `<span class="tree-text"><a href="${escapeHtml(url)}" target="_blank" rel="noopener">${escapeHtml(title)}</a></span>`;
            
            // Add status indicator for bookmarks
            const status = item.scrapeStatus || 'not-scraped';
            const tooltip = this.getStatusTooltip(status, item.scrapeError);
            html += `<span class="bookmark-status tooltip status-${status}" data-tooltip="${tooltip}"></span>`;
            
            // Add individual scrape button
            html += `<button class="btn btn-small scrape-bookmark-btn" data-id="${nodeId}" title="Scrape this bookmark">⟳</button>`;
        }
        
        html += `</div>`;
        
        if (hasChildren) {
            html += `<ul class="tree-children tree-list collapsed">`;
            html += this.renderScrapingTree(item.children);
            html += `</ul>`;
        }
        
        html += `</li>`;
        
        return html;
    }

    toggleFolder($toggle) {
        const $item = $toggle.closest('.tree-item');
        const $children = $item.find('> .tree-children');
        
        if ($children.length === 0) {
            return;
        }

        const isCollapsed = $children.hasClass('collapsed');
        
        if (isCollapsed) {
            $children.removeClass('collapsed');
            $toggle.addClass('expanded').attr('aria-label', 'Collapse folder');
        } else {
            $children.addClass('collapsed');
            $toggle.removeClass('expanded').attr('aria-label', 'Expand folder');
        }
    }

    handleBookmarkSelection($checkbox) {
        const id = $checkbox.data('id');
        const isChecked = $checkbox.prop('checked');
        const $item = $checkbox.closest('.tree-item');
        
        if (isChecked) {
            this.selectedBookmarks.add(id);
        } else {
            this.selectedBookmarks.delete(id);
        }

        // If this is a folder, select/deselect all children
        const $children = $item.find('.tree-children .scraping-checkbox');
        $children.each((_, child) => {
            const $child = $(child);
            const childId = $child.data('id');
            
            $child.prop('checked', isChecked);
            
            if (isChecked) {
                this.selectedBookmarks.add(childId);
            } else {
                this.selectedBookmarks.delete(childId);
            }
        });

        this.updateSelectAllState();
        this.updateStartButtonState();
    }

    toggleSelectAll(selectAll) {
        this.container.find('.scraping-checkbox').each((_, checkbox) => {
            const $checkbox = $(checkbox);
            const id = $checkbox.data('id');
            
            $checkbox.prop('checked', selectAll);
            
            if (selectAll) {
                this.selectedBookmarks.add(id);
            } else {
                this.selectedBookmarks.delete(id);
            }
        });

        this.updateStartButtonState();
    }

    updateSelectAllState() {
        const totalCheckboxes = this.container.find('.scraping-checkbox').length;
        const checkedCheckboxes = this.container.find('.scraping-checkbox:checked').length;
        
        const $selectAll = $('#selectAllBtn');
        
        if (checkedCheckboxes === 0) {
            $selectAll.prop('checked', false).prop('indeterminate', false);
        } else if (checkedCheckboxes === totalCheckboxes) {
            $selectAll.prop('checked', true).prop('indeterminate', false);
        } else {
            $selectAll.prop('checked', false).prop('indeterminate', true);
        }
    }

    updateStartButtonState() {
        const hasSelection = this.selectedBookmarks.size > 0;
        const isIdle = this.scrapingStatus === 'idle' || this.scrapingStatus === 'stopped';
        
        $('#startScrapingBtn').prop('disabled', !hasSelection || !isIdle);
    }

    async startScraping() {
        if (this.selectedBookmarks.size === 0) {
            showToast('Please select bookmarks to scrape', 'error');
            return;
        }

        try {
            // Get only bookmark IDs (not folder IDs)
            const bookmarkIds = this.getSelectedBookmarkIds();
            
            if (bookmarkIds.length === 0) {
                showToast('No bookmarks selected for scraping', 'error');
                return;
            }

            await this.api.startScraping(bookmarkIds);
            
            this.scrapingStatus = 'running';
            this.updateUI();
            this.startStatusChecking();
            
            showToast(`Started scraping ${bookmarkIds.length} bookmarks`, 'success');
            
        } catch (error) {
            console.error('Failed to start scraping:', error);
            showToast(`Failed to start scraping: ${error.message}`, 'error');
        }
    }

    async togglePause() {
        try {
            if (this.scrapingStatus === 'running') {
                await this.api.pauseScraping();
                this.scrapingStatus = 'paused';
                showToast('Scraping paused', 'info');
            } else if (this.scrapingStatus === 'paused') {
                await this.api.resumeScraping();
                this.scrapingStatus = 'running';
                showToast('Scraping resumed', 'info');
            }
            
            this.updateUI();
            
        } catch (error) {
            console.error('Failed to toggle pause:', error);
            showToast(`Failed to ${this.scrapingStatus === 'running' ? 'pause' : 'resume'} scraping: ${error.message}`, 'error');
        }
    }

    async stopScraping() {
        try {
            await this.api.stopScraping();
            this.scrapingStatus = 'stopped';
            this.updateUI();
            this.stopStatusChecking();
            
            showToast('Scraping stopped', 'info');
            
        } catch (error) {
            console.error('Failed to stop scraping:', error);
            showToast(`Failed to stop scraping: ${error.message}`, 'error');
        }
    }

    startStatusChecking() {
        this.stopStatusChecking(); // Clear any existing interval
        
        const interval = (window.CONFIG && window.CONFIG.UI.STATUS_CHECK_INTERVAL) || 2000;
        
        this.statusCheckInterval = setInterval(async () => {
            try {
                const status = await this.api.getScrapingStatus();
                this.updateScrapingProgress(status);
                
                // Stop checking if scraping is complete
                if (status.status === 'completed' || status.status === 'stopped') {
                    this.stopStatusChecking();
                    this.scrapingStatus = 'idle';
                    this.updateUI();
                }
                
            } catch (error) {
                console.error('Failed to get scraping status:', error);
                this.stopStatusChecking();
            }
        }, interval);
    }

    stopStatusChecking() {
        if (this.statusCheckInterval) {
            clearInterval(this.statusCheckInterval);
            this.statusCheckInterval = null;
        }
    }

    updateScrapingProgress(status) {
        const { current = 0, total = 1, current_url = '', progress = 0 } = status;
        
        // Update progress bar
        $('.progress-fill').css('width', `${progress}%`);
        
        // Update status text
        let statusText = 'Idle';
        if (this.scrapingStatus === 'running') {
            statusText = `Scraping ${current} of ${total}`;
            if (current_url) {
                statusText += `: ${current_url}`;
            }
        } else if (this.scrapingStatus === 'paused') {
            statusText = `Paused (${current} of ${total})`;
        }
        
        $('.progress-status').text(statusText);

        // Update individual bookmark statuses if provided
        if (status.bookmark_statuses) {
            this.updateBookmarkStatuses(status.bookmark_statuses);
        }
    }

    updateBookmarkStatuses(statuses) {
        for (const [bookmarkId, status] of Object.entries(statuses)) {
            const $statusElement = this.container.find(`[data-id="${bookmarkId}"]`).closest('.tree-item').find('.bookmark-status');
            
            if ($statusElement.length) {
                // Remove old status classes
                $statusElement.removeClass('status-not-scraped status-in-progress status-scraped status-error');
                
                // Add new status class
                $statusElement.addClass(`status-${status.status}`);
                
                // Update tooltip
                const tooltip = this.getStatusTooltip(status.status, status.error);
                $statusElement.attr('data-tooltip', tooltip);
            }
        }
    }

    updateUI() {
        const isRunning = this.scrapingStatus === 'running';
        const isPaused = this.scrapingStatus === 'paused';
        const isIdle = this.scrapingStatus === 'idle' || this.scrapingStatus === 'stopped';

        // Update button states
        $('#startScrapingBtn').prop('disabled', !isIdle || this.selectedBookmarks.size === 0);
        $('#pauseBtn').prop('disabled', !isRunning && !isPaused);
        $('#stopBtn').prop('disabled', isIdle);

        // Update pause button text
        $('#pauseBtn').text(isPaused ? 'Resume' : 'Pause');

        // Update progress if idle
        if (isIdle) {
            $('.progress-fill').css('width', '0%');
            $('.progress-status').text('Idle');
        }
    }

    getSelectedBookmarkIds() {
        const bookmarkIds = [];
        
        this.selectedBookmarks.forEach(id => {
            const $item = this.container.find(`[data-id="${id}"]`).closest('.tree-item');
            const isBookmark = $item.find('.bookmark-status').length > 0;
            
            if (isBookmark) {
                bookmarkIds.push(id);
            }
        });
        
        return bookmarkIds;
    }

    getStatusTooltip(status, error = '') {
        switch (status) {
            case 'not-scraped':
                return 'Ready to scrape';
            case 'in-progress':
                return 'Scraping...';
            case 'scraped':
                return `Scraped successfully on ${new Date().toLocaleDateString()}`;
            case 'error':
                return `Error: ${error || 'Unknown error'}`;
            default:
                return 'Unknown status';
        }
    }

    showError(message) {
        this.container.html(`
            <div class="error-state">
                <p>${escapeHtml(message)}</p>
                <button class="btn btn-primary" onclick="window.scrapingManager.loadBookmarks()">Retry</button>
            </div>
        `);
    }

    async scrapeIndividualBookmark($button) {
        const bookmarkId = $button.data('id');
        const $statusElement = $button.siblings('.bookmark-status');
        
        try {
            // Update status to in-progress
            $statusElement.removeClass('status-not-scraped status-scraped status-error')
                         .addClass('status-in-progress')
                         .attr('data-tooltip', 'Scraping...');
            
            // Disable the button during scraping
            $button.prop('disabled', true).addClass('loading');
            
            // Call the API to rescrape the bookmark
            const result = await this.api.rescrapeBookmark(bookmarkId);
            
            // Update status to success
            $statusElement.removeClass('status-in-progress')
                         .addClass('status-scraped')
                         .attr('data-tooltip', `Scraped successfully`);
            
            // Show success message
            const title = result.title || 'Bookmark';
            showToast(`Successfully scraped "${title}"`, 'success');
            
        } catch (error) {
            console.error('Failed to scrape bookmark:', error);
            
            // Update status to error
            $statusElement.removeClass('status-in-progress')
                         .addClass('status-error')
                         .attr('data-tooltip', `Error: ${error.message}`);
            
            // Show error message
            showToast(`Failed to scrape bookmark: ${error.message}`, 'error');
            
        } finally {
            // Re-enable the button
            $button.prop('disabled', false).removeClass('loading');
        }
    }

    // Clean up when component is destroyed
    destroy() {
        this.stopStatusChecking();
    }
}

// Export for use in other modules
window.ScrapingManager = ScrapingManager;