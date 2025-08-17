// DOM utility functions

/**
 * Debounce function to limit the rate of function calls
 * @param {Function} func - Function to debounce
 * @param {number} wait - Wait time in milliseconds
 * @returns {Function} Debounced function
 */
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

/**
 * Throttle function to limit the rate of function calls
 * @param {Function} func - Function to throttle
 * @param {number} limit - Limit time in milliseconds
 * @returns {Function} Throttled function
 */
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

/**
 * Show a toast notification
 * @param {string} message - Message to show
 * @param {string} type - Type of notification (success, error, info)
 */
function showToast(message, type = 'info') {
    // Remove existing toasts
    $('.toast').remove();
    
    const toast = $(`
        <div class="toast toast-${type}">
            <span>${message}</span>
            <button class="toast-close">&times;</button>
        </div>
    `);
    
    $('body').append(toast);
    
    // Auto-remove after configured duration
    const duration = (window.CONFIG && window.CONFIG.UI.TOAST_DURATION) || 5000;
    setTimeout(() => {
        toast.fadeOut(() => toast.remove());
    }, duration);
    
    // Manual close
    toast.find('.toast-close').on('click', () => {
        toast.fadeOut(() => toast.remove());
    });
}

/**
 * Format file size to human readable format
 * @param {number} bytes - File size in bytes
 * @returns {string} Formatted file size
 */
function formatFileSize(bytes) {
    if (bytes === 0) return '0 Bytes';
    
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

/**
 * Generate a unique ID
 * @returns {string} Unique ID
 */
function generateId() {
    return 'id_' + Math.random().toString(36).substr(2, 9);
}

/**
 * Escape HTML to prevent XSS
 * @param {string} text - Text to escape
 * @returns {string} Escaped text
 */
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

/**
 * Create a tree node element
 * @param {Object} item - Tree item data
 * @param {boolean} isFolder - Whether the item is a folder
 * @param {boolean} forScraping - Whether this is for the scraping tab
 * @returns {jQuery} Tree node element
 */
function createTreeNode(item, isFolder = false, forScraping = false) {
    const nodeId = generateId();
    const hasChildren = item.children && item.children.length > 0;
    
    let nodeHtml = `
        <li class="tree-item" data-id="${item.id || nodeId}">
            <div class="tree-item-content">
    `;
    
    if (forScraping) {
        nodeHtml += `<input type="checkbox" class="scraping-checkbox" data-id="${item.id || nodeId}">`;
    }
    
    if (isFolder && hasChildren) {
        nodeHtml += `<button class="folder-toggle" aria-label="Expand folder">▶</button>`;
    } else {
        nodeHtml += `<span class="tree-icon"></span>`;
    }
    
    nodeHtml += `
        <span class="tree-icon ${isFolder ? 'folder' : 'bookmark'}">${isFolder ? '📁' : '🔖'}</span>
    `;
    
    if (isFolder) {
        nodeHtml += `<span class="tree-text folder-name">${escapeHtml(item.title || item.name || 'Untitled Folder')}</span>`;
    } else {
        const url = item.url || '#';
        const title = item.title || item.name || 'Untitled Bookmark';
        nodeHtml += `<span class="tree-text"><a href="${escapeHtml(url)}" target="_blank" rel="noopener">${escapeHtml(title)}</a></span>`;
    }
    
    if (forScraping && !isFolder) {
        const status = item.scrapeStatus || 'not-scraped';
        const tooltip = getStatusTooltip(status, item.scrapeError);
        nodeHtml += `<span class="bookmark-status tooltip status-${status}" data-tooltip="${tooltip}"></span>`;
    }
    
    nodeHtml += `</div>`;
    
    if (hasChildren) {
        nodeHtml += `<ul class="tree-children tree-list collapsed">`;
        item.children.forEach(child => {
            const childIsFolder = child.type === 'folder' || (child.children && child.children.length > 0);
            nodeHtml += createTreeNode(child, childIsFolder, forScraping).prop('outerHTML');
        });
        nodeHtml += `</ul>`;
    }
    
    nodeHtml += `</li>`;
    
    return $(nodeHtml);
}

/**
 * Get tooltip text for bookmark status
 * @param {string} status - Bookmark status
 * @param {string} error - Error message if any
 * @returns {string} Tooltip text
 */
function getStatusTooltip(status, error = '') {
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

// Add toast CSS if not already present
if (!$('#toast-styles').length) {
    $('head').append(`
        <style id="toast-styles">
            .toast {
                position: fixed;
                top: 20px;
                right: 20px;
                background: var(--color-surface);
                border: 1px solid var(--color-border);
                border-radius: var(--radius-md);
                padding: var(--spacing-md);
                box-shadow: var(--shadow-lg);
                z-index: 10000;
                display: flex;
                align-items: center;
                gap: var(--spacing-md);
                max-width: 300px;
                animation: slideIn 0.3s ease;
            }
            
            .toast-success {
                border-left: 4px solid var(--color-success);
            }
            
            .toast-error {
                border-left: 4px solid var(--color-error);
            }
            
            .toast-info {
                border-left: 4px solid var(--color-primary);
            }
            
            .toast-close {
                background: none;
                border: none;
                font-size: 1.2rem;
                cursor: pointer;
                color: var(--color-text-secondary);
                padding: 0;
                margin-left: auto;
            }
            
            .toast-close:hover {
                color: var(--color-text);
            }
            
            @media (max-width: 640px) {
                .toast {
                    top: 10px;
                    right: 10px;
                    left: 10px;
                    max-width: none;
                }
            }
        </style>
    `);
}