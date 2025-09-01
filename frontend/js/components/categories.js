// Categories component for AI-powered bookmark categorization

class CategoriesComponent {
    constructor(apiClient) {
        console.log('🏷️ CategoriesComponent constructor called with apiClient:', apiClient);
        this.apiClient = apiClient;
        this.isProcessing = false;
        this.currentOperation = null;
        this.init();
    }

    init() {
        console.log('🏷️ CategoriesComponent init called');
        this.bindEvents();
        this.loadCategories();
    }

    bindEvents() {
        console.log('🏷️ Binding categories events...');
        console.log('🏷️ jQuery available:', typeof $);
        console.log('🏷️ bulkCategorizeBtn element found:', $('#bulkCategorizeBtn').length);
        console.log('🏷️ refreshCategoriesBtn element found:', $('#refreshCategoriesBtn').length);
        
        // Bulk categorize button
        const $bulkBtn = $('#bulkCategorizeBtn');
        if ($bulkBtn.length === 0) {
            console.error('🏷️ bulkCategorizeBtn element not found in DOM');
        } else {
            $bulkBtn.on('click', () => {
                console.log('🤖 Bulk categorize button clicked!');
                this.startBulkCategorization();
            });
            console.log('🏷️ Bulk categorize click handler bound');
        }

        // Refresh categories button  
        const $refreshBtn = $('#refreshCategoriesBtn');
        if ($refreshBtn.length === 0) {
            console.error('🏷️ refreshCategoriesBtn element not found in DOM');
        } else {
            $refreshBtn.on('click', () => {
                console.log('🔄 Refresh categories button clicked!');
                this.loadCategories();
            });
            console.log('🏷️ Refresh categories click handler bound');
        }
        
        // Handle individual bookmark categorization from other tabs
        $(document).on('click', '.categorize-bookmark-btn', (e) => {
            const bookmarkId = $(e.target).data('bookmark-id');
            console.log('🤖 Individual categorization requested for:', bookmarkId);
            this.categorizeBookmark(bookmarkId);
        });

        console.log('🏷️ Categories events bound successfully');
    }

    async loadCategories() {
        console.log('🏷️ Loading categories...');
        
        try {
            const categories = await this.apiClient.getCategories();
            console.log('✅ Categories loaded:', categories);
            this.renderCategories(categories);
        } catch (error) {
            console.error('❌ Failed to load categories:', error);
            this.showError('Failed to load categories: ' + error.message);
            this.renderCategories([]);
        }
    }

    renderCategories(categories) {
        console.log('🏷️ Rendering', categories.length, 'categories');
        
        const container = $('#categoryList');
        container.empty();
        
        if (categories.length === 0) {
            container.html('<p class="empty-state">No categories yet. Start categorizing bookmarks to see them here!</p>');
            return;
        }
        
        // Sort categories by usage count (descending) and then by name
        const sortedCategories = categories.sort((a, b) => {
            if (b.usage_count !== a.usage_count) {
                return b.usage_count - a.usage_count;
            }
            return a.name.localeCompare(b.name);
        });

        const html = sortedCategories.map(category => {
            const categoryName = category.parent_category 
                ? `${category.parent_category}/${category.name}` 
                : category.name;
            
            return `
                <div class="category-item" data-category-id="${category.id}">
                    <div class="category-info">
                        <span class="category-name" style="background-color: ${category.color || '#e0e0e0'}">
                            ${categoryName}
                        </span>
                        <span class="category-count">${category.usage_count} bookmark${category.usage_count !== 1 ? 's' : ''}</span>
                    </div>
                    <div class="category-meta">
                        <small class="category-date">Created ${this.formatDate(category.created_at)}</small>
                    </div>
                </div>
            `;
        }).join('');
        
        container.html(html);
    }

    async categorizeBookmark(bookmarkId) {
        console.log('🤖 Categorizing individual bookmark:', bookmarkId);
        
        if (this.isProcessing) {
            console.log('⏳ Already processing, skipping request');
            return;
        }
        
        this.isProcessing = true;
        this.updateProcessingState(true, 'Categorizing bookmark...');
        
        try {
            const result = await this.apiClient.categorizeBookmark(bookmarkId);
            console.log('✅ Bookmark categorized successfully:', result);
            
            // Show categorization result to user
            this.showCategorizationResult(bookmarkId, result);
            
            // Refresh categories list to show updated counts
            await this.loadCategories();
            
        } catch (error) {
            console.error('❌ Categorization failed:', error);
            this.showError('Failed to categorize bookmark: ' + error.message);
        } finally {
            this.isProcessing = false;
            this.updateProcessingState(false);
        }
    }

    async startBulkCategorization() {
        console.log('🚀 Starting bulk categorization...');
        
        if (this.isProcessing) {
            console.log('⏳ Already processing, skipping bulk categorization');
            return;
        }
        
        // First, get uncategorized bookmarks
        try {
            const bookmarks = await this.apiClient.getBookmarks();
            
            // Filter for bookmarks that likely need categorization
            // For now, we'll categorize all bookmarks - in production you might want to filter
            const uncategorizedBookmarks = bookmarks.bookmarks || [];
            
            if (uncategorizedBookmarks.length === 0) {
                this.showInfo('No bookmarks found that need categorization.');
                return;
            }
            
            console.log(`📋 Found ${uncategorizedBookmarks.length} bookmarks for bulk categorization`);
            
            this.isProcessing = true;
            this.updateProcessingState(true, `Starting bulk categorization of ${uncategorizedBookmarks.length} bookmarks...`);
            
            // Extract bookmark IDs
            const bookmarkIds = uncategorizedBookmarks.map(bookmark => bookmark.id);
            
            // Start bulk categorization with auto-apply for high confidence results
            const results = await this.apiClient.bulkCategorize({
                bookmark_ids: bookmarkIds,
                auto_apply: true,
                confidence_threshold: 0.8
            });
            
            console.log('✅ Bulk categorization completed:', results);
            
            // Show results summary
            this.showBulkResults(results);
            
            // Refresh categories to show updated counts
            await this.loadCategories();
            
        } catch (error) {
            console.error('❌ Bulk categorization failed:', error);
            this.showError('Bulk categorization failed: ' + error.message);
        } finally {
            this.isProcessing = false;
            this.updateProcessingState(false);
        }
    }

    showCategorizationResult(bookmarkId, result) {
        console.log('🏷️ Showing categorization result for bookmark:', bookmarkId, result);
        
        const resultHtml = `
            <div class="categorization-result">
                <div class="result-header">
                    <strong>Bookmark Categorized</strong>
                    <small>Confidence: ${(result.confidence_score * 100).toFixed(0)}%</small>
                </div>
                <div class="result-content">
                    <div class="primary-category">
                        <strong>Primary:</strong> <span class="category-tag">${result.primary_category}</span>
                    </div>
                    ${result.secondary_categories && result.secondary_categories.length > 0 ? `
                        <div class="secondary-categories">
                            <strong>Secondary:</strong> 
                            ${result.secondary_categories.map(cat => `<span class="category-tag secondary">${cat}</span>`).join(' ')}
                        </div>
                    ` : ''}
                    ${result.tags && result.tags.length > 0 ? `
                        <div class="tags">
                            <strong>Tags:</strong> 
                            ${result.tags.map(tag => `<span class="tag">#${tag}</span>`).join(' ')}
                        </div>
                    ` : ''}
                    ${result.reasoning ? `
                        <div class="reasoning">
                            <strong>Reasoning:</strong> <em>${result.reasoning}</em>
                        </div>
                    ` : ''}
                </div>
            </div>
        `;
        
        const recentContainer = $('#recentCategorizations');
        if (recentContainer.find('.empty-state').length > 0) {
            recentContainer.empty();
        }
        recentContainer.prepend(resultHtml);
        
        // Keep only the last 5 results
        recentContainer.find('.categorization-result').slice(5).remove();
    }

    showBulkResults(results) {
        console.log('📊 Showing bulk categorization results:', results);
        
        const message = `
            Bulk categorization completed:
            • ${results.total_processed} bookmarks processed
            • ${results.total_applied} automatically applied (high confidence)
            • ${results.total_processed - results.total_applied} require manual review
        `;
        
        this.showSuccess(message);
        
        // Show some individual results in the recent categorizations
        if (results.results && results.results.length > 0) {
            results.results.slice(0, 3).forEach(result => {
                if (result.categorization) {
                    this.showCategorizationResult(result.bookmark_id, result.categorization);
                }
            });
        }
    }

    updateProcessingState(isProcessing, message = '') {
        const progressContainer = $('.categorization-progress');
        const bulkButton = $('#bulkCategorizeBtn');
        
        if (isProcessing) {
            progressContainer.show();
            $('.progress-text').text(message);
            bulkButton.prop('disabled', true);
            
            // Simple progress animation
            let progress = 0;
            this.currentOperation = setInterval(() => {
                progress += 2;
                if (progress > 90) progress = 90; // Don't go to 100% until actually done
                $('.categorization-progress .progress-fill').css('width', progress + '%');
            }, 500);
            
        } else {
            progressContainer.hide();
            $('.progress-fill').css('width', '0%');
            bulkButton.prop('disabled', false);
            
            if (this.currentOperation) {
                clearInterval(this.currentOperation);
                this.currentOperation = null;
            }
        }
    }

    showSuccess(message) {
        console.log('✅ Success:', message);
        // You could integrate with a toast notification system here
        alert('Success: ' + message);
    }

    showInfo(message) {
        console.log('ℹ️ Info:', message);
        alert('Info: ' + message);
    }

    showError(message) {
        console.error('❌ Error:', message);
        alert('Error: ' + message);
    }

    formatDate(dateString) {
        const date = new Date(dateString);
        const now = new Date();
        const diffTime = Math.abs(now - date);
        const diffDays = Math.ceil(diffTime / (1000 * 60 * 60 * 24));
        
        if (diffDays === 1) {
            return 'yesterday';
        } else if (diffDays <= 7) {
            return `${diffDays} days ago`;
        } else {
            return date.toLocaleDateString();
        }
    }
}

// Export for global access
window.CategoriesComponent = CategoriesComponent;