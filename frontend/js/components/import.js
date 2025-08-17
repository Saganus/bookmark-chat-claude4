// Import functionality for bookmark files

class ImportHandler {
    constructor(api) {
        this.api = api;
        this.modal = $('#importModal');
        this.uploadArea = $('#uploadArea');
        this.fileInput = $('#fileInput');
        this.progressElement = $('.import-progress');
        this.init();
    }

    init() {
        this.setupDragDrop();
        this.bindEvents();
    }

    setupDragDrop() {
        const events = ['dragenter', 'dragover', 'dragleave', 'drop'];
        
        // Prevent default drag behaviors
        events.forEach(eventName => {
            this.uploadArea.on(eventName, (e) => {
                e.preventDefault();
                e.stopPropagation();
            });
        });

        // Handle drag enter/over
        this.uploadArea.on('dragenter dragover', () => {
            this.uploadArea.addClass('drag-active');
        });

        // Handle drag leave
        this.uploadArea.on('dragleave', () => {
            this.uploadArea.removeClass('drag-active');
        });

        // Handle file drop
        this.uploadArea.on('drop', (e) => {
            this.uploadArea.removeClass('drag-active');
            const files = e.originalEvent.dataTransfer.files;
            this.handleFiles(files);
        });
    }

    bindEvents() {
        // Show modal when import button is clicked
        $('#importBtn').on('click', () => {
            this.showModal();
        });

        // Close modal
        $('.modal-close').on('click', () => {
            this.hideModal();
        });

        // Close modal when clicking outside
        this.modal.on('click', (e) => {
            if (e.target === this.modal[0]) {
                this.hideModal();
            }
        });

        // Handle file input change
        this.fileInput.on('change', (e) => {
            this.handleFiles(e.target.files);
        });

        // Close modal on Escape key
        $(document).on('keydown', (e) => {
            if (e.key === 'Escape' && this.modal.hasClass('show')) {
                this.hideModal();
            }
        });
    }

    showModal() {
        this.modal.addClass('show');
        this.resetForm();
        
        // Focus on the first radio button
        $('input[name="importType"]:first').focus();
    }

    hideModal() {
        this.modal.removeClass('show');
        this.resetForm();
    }

    resetForm() {
        this.fileInput.val('');
        this.progressElement.hide();
        $('.progress-fill').css('width', '0%');
        this.uploadArea.removeClass('drag-active');
    }

    async handleFiles(files) {
        if (!files || files.length === 0) {
            return;
        }

        const file = files[0];
        const selectedType = $('input[name="importType"]:checked').val();

        if (!selectedType) {
            showToast('Please select browser type first', 'error');
            return;
        }

        // Validate file type
        if (!this.validateFile(file, selectedType)) {
            return;
        }

        await this.uploadFile(file, selectedType);
    }

    validateFile(file, type) {
        // Get valid extensions from config or use defaults
        const validExtensions = (window.CONFIG && window.CONFIG.UPLOAD.ACCEPTED_TYPES) || {
            chrome: ['.html', '.htm'],
            firefox: ['.html', '.htm']
        };

        const fileExtension = '.' + file.name.split('.').pop().toLowerCase();
        const allowedExtensions = validExtensions[type] || [];

        if (!allowedExtensions.includes(fileExtension)) {
            showToast(`Invalid file type. Expected ${allowedExtensions.join(' or ')} for ${type}`, 'error');
            return false;
        }

        // Check file size using config or default
        const maxSize = (window.CONFIG && window.CONFIG.UPLOAD.MAX_SIZE) || (10 * 1024 * 1024); // 10MB
        if (file.size > maxSize) {
            showToast(`File too large. Maximum size is ${formatFileSize(maxSize)}`, 'error');
            return false;
        }

        return true;
    }

    async uploadFile(file, type) {
        try {
            // Show progress
            this.progressElement.show();
            this.setProgress(0, 'Preparing upload...');

            // Simulate upload progress
            const progressInterval = setInterval(() => {
                const currentWidth = parseFloat($('.progress-fill').css('width')) || 0;
                const containerWidth = $('.progress-bar').width();
                const percentage = (currentWidth / containerWidth) * 100;
                
                if (percentage < 90) {
                    this.setProgress(percentage + 10, 'Uploading...');
                }
            }, 200);

            // Upload file
            const result = await this.api.importBookmarks(file, type);
            
            clearInterval(progressInterval);
            this.setProgress(100, 'Import complete!');

            // Show success message
            showToast(`Successfully imported ${result.count || 'unknown number of'} bookmarks`, 'success');

            // Close modal after a short delay
            setTimeout(() => {
                this.hideModal();
                
                // Trigger bookmark list refresh
                if (window.bookmarkManager) {
                    window.bookmarkManager.loadBookmarks();
                }
                
                if (window.scrapingManager) {
                    window.scrapingManager.loadBookmarks();
                }
            }, 1500);

        } catch (error) {
            console.error('Import failed:', error);
            this.progressElement.hide();
            showToast(`Import failed: ${error.message}`, 'error');
        }
    }

    setProgress(percentage, text) {
        $('.progress-fill').css('width', `${percentage}%`);
        $('.progress-text').text(text);
    }
}

// Export for use in other modules
window.ImportHandler = ImportHandler;