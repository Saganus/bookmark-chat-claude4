// Bookmark management functionality

class BookmarkManager {
    constructor(api) {
        this.api = api;
        this.bookmarks = [];
        this.container = $('#bookmarksTree');
        this.init();
    }

    async init() {
        await this.loadBookmarks();
        this.bindEvents();
    }

    bindEvents() {
        // Handle folder toggle clicks
        this.container.on('click', '.folder-toggle', (e) => {
            e.preventDefault();
            e.stopPropagation();
            this.toggleFolder($(e.target));
        });

        // Handle tree item clicks (for folder expansion)
        this.container.on('click', '.tree-item-content', (e) => {
            const $target = $(e.target);
            
            // Don't toggle if clicking on a link or scrape button
            if ($target.is('a, .scrape-bookmark-btn') || $target.closest('a, .scrape-bookmark-btn').length) {
                return;
            }

            const $toggle = $(e.currentTarget).find('.folder-toggle');
            if ($toggle.length) {
                this.toggleFolder($toggle);
            }
        });

        // Handle scrape button clicks
        this.container.on('click', '.scrape-bookmark-btn', async (e) => {
            e.preventDefault();
            e.stopPropagation();
            await this.scrapeBookmark($(e.target));
        });

        // Handle bookmark link clicks
        this.container.on('click', 'a[href]', (e) => {
            // Let the default behavior handle opening links
            // Just track the click for analytics if needed
            const url = $(e.target).attr('href');
            console.log('Bookmark clicked:', url);
        });
    }

    async loadBookmarks() {
        try {
            const response = await this.api.getBookmarks();
            this.bookmarks = response.bookmarks || response || [];
            this.render();
        } catch (error) {
            console.error('Failed to load bookmarks:', error);
            this.showError('Failed to load bookmarks. Please try again.');
        }
    }

    render() {
        if (!this.bookmarks || this.bookmarks.length === 0) {
            this.container.html('<p class="empty-state">No bookmarks imported yet. Click "Import Bookmarks" to get started.</p>');
            return;
        }

        const treeHtml = this.renderBookmarkTree(this.bookmarks);
        this.container.html(`<ul class="tree-list">${treeHtml}</ul>`);
    }

    renderBookmarkTree(items) {
        if (!items || !Array.isArray(items)) {
            return '';
        }

        return items.map(item => {
            const isFolder = item.type === 'folder' || (item.children && item.children.length > 0);
            return this.renderTreeItem(item, isFolder);
        }).join('');
    }

    renderTreeItem(item, isFolder = false) {
        const nodeId = item.id || generateId();
        const hasChildren = item.children && item.children.length > 0;
        
        let html = `
            <li class="tree-item" data-id="${nodeId}">
                <div class="tree-item-content">
        `;
        
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
            
            // Add scrape button for bookmarks
            html += `<button class="btn btn-small scrape-bookmark-btn" data-id="${nodeId}" title="Scrape this bookmark">⟳</button>`;
        }
        
        html += `</div>`;
        
        if (hasChildren) {
            html += `<ul class="tree-children tree-list collapsed">`;
            html += this.renderBookmarkTree(item.children);
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

    showError(message) {
        this.container.html(`
            <div class="error-state">
                <p>${escapeHtml(message)}</p>
                <button class="btn btn-primary" onclick="window.bookmarkManager.loadBookmarks()">Retry</button>
            </div>
        `);
    }

    async scrapeBookmark($button) {
        const bookmarkId = $button.data('id');
        
        try {
            // Disable the button during scraping
            $button.prop('disabled', true).addClass('loading').text('⟳');
            
            // Call the API to rescrape the bookmark
            const result = await this.api.rescrapeBookmark(bookmarkId);
            
            // Show success message
            const title = result.title || 'Bookmark';
            showToast(`Successfully scraped "${title}"`, 'success');
            
            // Optionally refresh the bookmark data
            await this.loadBookmarks();
            
        } catch (error) {
            console.error('Failed to scrape bookmark:', error);
            showToast(`Failed to scrape bookmark: ${error.message}`, 'error');
            
        } finally {
            // Re-enable the button
            $button.prop('disabled', false).removeClass('loading').text('⟳');
        }
    }

    // Expand all folders
    expandAll() {
        this.container.find('.tree-children').removeClass('collapsed');
        this.container.find('.folder-toggle').addClass('expanded').attr('aria-label', 'Collapse folder');
    }

    // Collapse all folders
    collapseAll() {
        this.container.find('.tree-children').addClass('collapsed');
        this.container.find('.folder-toggle').removeClass('expanded').attr('aria-label', 'Expand folder');
    }

    // Search bookmarks (simple text search)
    searchBookmarks(query) {
        if (!query.trim()) {
            this.render();
            return;
        }

        const filteredBookmarks = this.filterBookmarks(this.bookmarks, query.toLowerCase());
        
        if (filteredBookmarks.length === 0) {
            this.container.html('<p class="empty-state">No bookmarks found matching your search.</p>');
        } else {
            const treeHtml = this.renderBookmarkTree(filteredBookmarks);
            this.container.html(`<ul class="tree-list">${treeHtml}</ul>`);
            
            // Expand all folders in search results
            this.expandAll();
        }
    }

    filterBookmarks(items, query) {
        const filtered = [];
        
        for (const item of items) {
            const isFolder = item.type === 'folder' || (item.children && item.children.length > 0);
            const title = (item.title || item.name || '').toLowerCase();
            const url = (item.url || '').toLowerCase();
            
            if (isFolder) {
                // For folders, check if title matches or if any children match
                const filteredChildren = this.filterBookmarks(item.children || [], query);
                
                if (title.includes(query) || filteredChildren.length > 0) {
                    filtered.push({
                        ...item,
                        children: filteredChildren
                    });
                }
            } else {
                // For bookmarks, check title and URL
                if (title.includes(query) || url.includes(query)) {
                    filtered.push(item);
                }
            }
        }
        
        return filtered;
    }

    // Get all bookmark URLs for scraping
    getAllBookmarkIds() {
        const ids = [];
        
        const extractIds = (items) => {
            for (const item of items) {
                if (item.type !== 'folder' && item.url) {
                    ids.push(item.id || item.url);
                }
                
                if (item.children && item.children.length > 0) {
                    extractIds(item.children);
                }
            }
        };
        
        extractIds(this.bookmarks);
        return ids;
    }
}

// Export for use in other modules
window.BookmarkManager = BookmarkManager;