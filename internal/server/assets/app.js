// Audiobook Builder TTS - Web Client

class App {
    constructor() {
        this.ws = null;
        this.jobs = new Map();
        this.providers = [];
        this.voices = [];
        this.selectedFile = null;
        this.currentPreview = null;
        this.reconnectAttempts = 0;
        this.maxReconnectAttempts = 10;
        this.reconnectDelay = 1000;
        this.settings = {};

        this.init();
    }

    async init() {
        this.bindElements();
        this.bindEvents();
        await this.loadConfig();
        await this.loadProviders();
        await this.loadJobs();
        this.connectWebSocket();
    }

    bindElements() {
        // Upload form elements
        this.dropZone = document.getElementById('drop-zone');
        this.fileInput = document.getElementById('file-input');
        this.selectedFileEl = document.getElementById('selected-file');
        this.fileNameEl = document.getElementById('file-name');
        this.fileSizeEl = document.getElementById('file-size');
        this.removeFileBtn = document.getElementById('remove-file');
        this.providerSelect = document.getElementById('provider');
        this.languageSelect = document.getElementById('language');
        this.modelSelect = document.getElementById('model');
        this.voiceSelect = document.getElementById('voice');
        this.speedInput = document.getElementById('speed');
        this.speedValue = document.getElementById('speed-value');
        this.pitchInput = document.getElementById('pitch');
        this.pitchValue = document.getElementById('pitch-value');
        this.previewBtn = document.getElementById('preview-btn');
        this.uploadBtn = document.getElementById('upload-btn');

        // Preview section elements
        this.previewSection = document.getElementById('preview-section');
        this.closePreviewBtn = document.getElementById('close-preview');
        this.previewCover = document.getElementById('preview-cover');
        this.previewTitle = document.getElementById('preview-title');
        this.previewAuthor = document.getElementById('preview-author');
        this.previewDescription = document.getElementById('preview-description');
        this.previewChapters = document.getElementById('preview-chapters');
        this.previewWords = document.getElementById('preview-words');
        this.previewDuration = document.getElementById('preview-duration');
        this.previewCosts = document.getElementById('preview-costs');
        this.previewChapterList = document.getElementById('preview-chapter-list');
        this.confirmConvertBtn = document.getElementById('confirm-convert');

        // Jobs list
        this.jobsList = document.getElementById('jobs-list');
        this.jobsCount = document.getElementById('jobs-count');

        // Connection status
        this.statusDot = document.getElementById('status-dot');
        this.statusText = document.getElementById('status-text');

        // Toast container
        this.toastContainer = document.getElementById('toast-container');

        // Settings modal elements
        this.settingsBtn = document.getElementById('settings-btn');
        this.settingsModal = document.getElementById('settings-modal');
        this.settingsClose = document.getElementById('settings-close');
        this.settingsCancel = document.getElementById('settings-cancel');
        this.settingsSave = document.getElementById('settings-save');
        this.settingsTabs = document.querySelectorAll('.tab-btn');
        this.tabContents = document.querySelectorAll('.tab-content');
        this.testAbsBtn = document.getElementById('test-abs-connection');
        this.absConnectionResult = document.getElementById('abs-connection-result');
    }

    bindEvents() {
        // Drop zone events
        this.dropZone.addEventListener('click', () => this.fileInput.click());
        this.dropZone.addEventListener('dragover', (e) => this.handleDragOver(e));
        this.dropZone.addEventListener('dragleave', () => this.handleDragLeave());
        this.dropZone.addEventListener('drop', (e) => this.handleDrop(e));
        this.fileInput.addEventListener('change', (e) => this.handleFileSelect(e));
        this.removeFileBtn.addEventListener('click', () => this.clearSelectedFile());

        // Cascading dropdown changes
        this.providerSelect.addEventListener('change', () => {
            this.loadLanguages();
            this.updateCostEstimate();
        });
        this.languageSelect.addEventListener('change', () => this.loadModels());
        this.modelSelect.addEventListener('change', () => this.loadVoices());

        // Range inputs
        this.speedInput.addEventListener('input', () => {
            this.speedValue.textContent = this.speedInput.value + 'x';
        });
        this.pitchInput.addEventListener('input', () => {
            this.pitchValue.textContent = this.pitchInput.value + 'x';
        });

        // Preview and Upload buttons
        this.previewBtn.addEventListener('click', () => this.previewFile());
        this.uploadBtn.addEventListener('click', () => this.uploadFile());

        // Preview section events
        this.closePreviewBtn.addEventListener('click', () => this.closePreview());
        this.confirmConvertBtn.addEventListener('click', () => this.confirmConvert());

        // Settings modal events
        this.settingsBtn.addEventListener('click', () => this.openSettings());
        this.settingsClose.addEventListener('click', () => this.closeSettings());
        this.settingsCancel.addEventListener('click', () => this.closeSettings());
        this.settingsSave.addEventListener('click', () => this.saveSettings());
        this.settingsModal.querySelector('.modal-overlay').addEventListener('click', () => this.closeSettings());
        
        // Settings tabs
        this.settingsTabs.forEach(tab => {
            tab.addEventListener('click', () => this.switchTab(tab.dataset.tab));
        });

        // Test Audiobookshelf connection
        if (this.testAbsBtn) {
            this.testAbsBtn.addEventListener('click', () => this.testAudiobookshelfConnection());
        }
    }

    // WebSocket connection
    connectWebSocket() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}/api/ws`;

        this.ws = new WebSocket(wsUrl);

        this.ws.onopen = () => {
            console.log('WebSocket connected');
            this.reconnectAttempts = 0;
            this.updateConnectionStatus(true);
        };

        this.ws.onclose = () => {
            console.log('WebSocket disconnected');
            this.updateConnectionStatus(false);
            this.scheduleReconnect();
        };

        this.ws.onerror = (error) => {
            console.error('WebSocket error:', error);
        };

        this.ws.onmessage = (event) => {
            this.handleWSMessage(event.data);
        };
    }

    scheduleReconnect() {
        if (this.reconnectAttempts < this.maxReconnectAttempts) {
            this.reconnectAttempts++;
            const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1);
            console.log(`Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts})`);
            setTimeout(() => this.connectWebSocket(), delay);
        }
    }

    updateConnectionStatus(connected) {
        this.statusDot.classList.toggle('connected', connected);
        this.statusText.textContent = connected ? 'Connected' : 'Disconnected';
    }

    handleWSMessage(data) {
        try {
            const message = JSON.parse(data);
            console.log('WS message:', message.type, message.payload);

            switch (message.type) {
                case 'job_created':
                    this.addJob(message.payload);
                    this.showToast('Job created', 'success');
                    break;
                case 'job_updated':
                case 'job_progress':
                    this.updateJob(message.payload);
                    break;
                case 'job_completed':
                    this.updateJob(message.payload);
                    this.showToast(`Conversion completed: ${message.payload.book_title || message.payload.file_name}`, 'success');
                    break;
                case 'job_failed':
                    this.updateJob(message.payload);
                    this.showToast(`Conversion failed: ${message.payload.error}`, 'error');
                    break;
                case 'job_deleted':
                    this.removeJob(message.payload.id);
                    break;
            }
        } catch (e) {
            console.error('Failed to parse WS message:', e);
        }
    }

    // API calls
    async loadConfig() {
        try {
            const response = await fetch('/api/config');
            const config = await response.json();
            this.speedInput.value = config.default_speed || 1.0;
            this.pitchInput.value = config.default_pitch || 1.0;
            this.speedValue.textContent = this.speedInput.value + 'x';
            this.pitchValue.textContent = this.pitchInput.value + 'x';
        } catch (e) {
            console.error('Failed to load config:', e);
        }
    }

    async loadProviders() {
        try {
            const response = await fetch('/api/providers');
            const data = await response.json();
            this.providers = data.providers || [];
            
            this.providerSelect.innerHTML = this.providers.map(p => 
                `<option value="${p}" ${p === data.default ? 'selected' : ''}>${p}</option>`
            ).join('');

            await this.loadLanguages();
        } catch (e) {
            console.error('Failed to load providers:', e);
        }
    }

    async loadLanguages() {
        try {
            const provider = this.providerSelect.value;
            const response = await fetch(`/api/languages?provider=${provider}`);
            const data = await response.json();
            const languages = data.languages || [];

            // Sort languages alphabetically
            languages.sort();

            if (languages.length === 0) {
                this.languageSelect.innerHTML = '<option value="">No languages available</option>';
                this.modelSelect.innerHTML = '<option value="">Select language first</option>';
                this.voiceSelect.innerHTML = '<option value="">Select model first</option>';
                return;
            }

            // Default to en-US if available, otherwise first language
            const defaultLang = languages.includes('en-US') ? 'en-US' : languages[0];

            this.languageSelect.innerHTML = languages.map(lang => 
                `<option value="${lang}" ${lang === defaultLang ? 'selected' : ''}>${lang}</option>`
            ).join('');

            await this.loadModels();
        } catch (e) {
            console.error('Failed to load languages:', e);
        }
    }

    async loadModels() {
        try {
            const provider = this.providerSelect.value;
            const response = await fetch(`/api/models?provider=${provider}`);
            const data = await response.json();
            const models = data.models || [];

            // Sort models by quality (custom order)
            const modelOrder = ['Chirp3-HD', 'Studio', 'Neural2', 'Wavenet', 'Standard'];
            models.sort((a, b) => {
                const aIdx = modelOrder.indexOf(a);
                const bIdx = modelOrder.indexOf(b);
                if (aIdx === -1 && bIdx === -1) return a.localeCompare(b);
                if (aIdx === -1) return 1;
                if (bIdx === -1) return -1;
                return aIdx - bIdx;
            });

            if (models.length === 0) {
                this.modelSelect.innerHTML = '<option value="">No models available</option>';
                this.voiceSelect.innerHTML = '<option value="">Select model first</option>';
                return;
            }

            this.modelSelect.innerHTML = models.map(model => 
                `<option value="${model}">${model}</option>`
            ).join('');

            await this.loadVoices();
        } catch (e) {
            console.error('Failed to load models:', e);
        }
    }

    async loadVoices() {
        try {
            const provider = this.providerSelect.value;
            const language = this.languageSelect.value;
            const model = this.modelSelect.value;
            
            const response = await fetch(`/api/voices?provider=${provider}&language=${language}&model=${model}`);
            const data = await response.json();
            this.voices = data.voices || [];

            if (this.voices.length === 0) {
                this.voiceSelect.innerHTML = '<option value="">No voices available</option>';
                return;
            }

            // Sort voices by name
            this.voices.sort((a, b) => a.Name.localeCompare(b.Name));

            this.voiceSelect.innerHTML = this.voices.map(v => 
                `<option value="${v.ID}" ${v.ID === data.default ? 'selected' : ''}>${v.Name}</option>`
            ).join('');
        } catch (e) {
            console.error('Failed to load voices:', e);
        }
    }

    async loadJobs() {
        try {
            const response = await fetch('/api/jobs');
            const jobs = await response.json();
            
            this.jobs.clear();
            this.jobsList.innerHTML = '';

            if (Array.isArray(jobs)) {
                // Sort by created_at descending (newest first)
                jobs.sort((a, b) => new Date(b.created_at) - new Date(a.created_at));
                jobs.forEach(job => this.addJob(job));
            }

            this.updateJobsCount();
        } catch (e) {
            console.error('Failed to load jobs:', e);
        }
    }

    // File handling
    handleDragOver(e) {
        e.preventDefault();
        this.dropZone.classList.add('dragover');
    }

    handleDragLeave() {
        this.dropZone.classList.remove('dragover');
    }

    handleDrop(e) {
        e.preventDefault();
        this.dropZone.classList.remove('dragover');
        
        const files = e.dataTransfer.files;
        if (files.length > 0) {
            this.setSelectedFile(files[0]);
        }
    }

    handleFileSelect(e) {
        const files = e.target.files;
        if (files.length > 0) {
            this.setSelectedFile(files[0]);
        }
    }

    setSelectedFile(file) {
        const ext = file.name.split('.').pop().toLowerCase();
        if (ext !== 'epub' && ext !== 'fb2') {
            this.showToast('Unsupported file format. Only .epub and .fb2 are supported.', 'error');
            return;
        }

        this.selectedFile = file;
        this.fileNameEl.textContent = file.name;
        this.fileSizeEl.textContent = this.formatFileSize(file.size);
        this.selectedFileEl.classList.add('visible');
        this.previewBtn.disabled = false;
        this.uploadBtn.disabled = false;
    }

    clearSelectedFile() {
        this.selectedFile = null;
        this.currentPreview = null;
        this.fileInput.value = '';
        this.selectedFileEl.classList.remove('visible');
        this.previewBtn.disabled = true;
        this.uploadBtn.disabled = true;
        this.closePreview();
    }

    // Preview functionality
    async previewFile() {
        if (!this.selectedFile) return;

        const formData = new FormData();
        formData.append('file', this.selectedFile);

        this.previewBtn.disabled = true;
        this.uploadBtn.disabled = true;
        
        // Stage 1: Uploading
        this.previewBtn.innerHTML = '<span class="spinner">⏳</span> Uploading file...';

        try {
            // Use XMLHttpRequest for upload progress
            const preview = await this.uploadWithProgress(formData, (stage, progress) => {
                if (stage === 'uploading') {
                    this.previewBtn.innerHTML = `<span class="spinner">⏳</span> Uploading... ${progress}%`;
                } else if (stage === 'parsing') {
                    this.previewBtn.innerHTML = '<span class="spinner">⏳</span> Parsing book...';
                }
            });

            this.currentPreview = preview;
            this.showPreview(preview);
        } catch (e) {
            this.showToast('Preview failed: ' + e.message, 'error');
        } finally {
            this.previewBtn.disabled = false;
            this.uploadBtn.disabled = !this.selectedFile;
            this.previewBtn.innerHTML = '👁️ Preview Book';
        }
    }

    // Upload file with progress tracking
    uploadWithProgress(formData, onProgress) {
        return new Promise((resolve, reject) => {
            const xhr = new XMLHttpRequest();
            
            // Track upload progress
            xhr.upload.addEventListener('progress', (e) => {
                if (e.lengthComputable) {
                    const percent = Math.round((e.loaded / e.total) * 100);
                    onProgress('uploading', percent);
                }
            });

            // When upload completes, server starts parsing
            xhr.upload.addEventListener('load', () => {
                onProgress('parsing', 100);
            });

            xhr.addEventListener('load', () => {
                if (xhr.status >= 200 && xhr.status < 300) {
                    try {
                        const response = JSON.parse(xhr.responseText);
                        resolve(response);
                    } catch (e) {
                        reject(new Error('Invalid response from server'));
                    }
                } else {
                    try {
                        const error = JSON.parse(xhr.responseText);
                        reject(new Error(error.error || 'Preview failed'));
                    } catch (e) {
                        reject(new Error('Preview failed: ' + xhr.statusText));
                    }
                }
            });

            xhr.addEventListener('error', () => {
                reject(new Error('Network error'));
            });

            xhr.addEventListener('abort', () => {
                reject(new Error('Upload cancelled'));
            });

            xhr.open('POST', '/api/preview');
            xhr.send(formData);
        });
    }

    showPreview(preview) {
        // Set cover image
        if (preview.cover_image_url) {
            this.previewCover.innerHTML = `<img src="${preview.cover_image_url}" alt="Cover">`;
        } else {
            this.previewCover.innerHTML = '<span class="no-cover">No Cover</span>';
        }

        // Set metadata
        this.previewTitle.textContent = preview.book_title || 'Unknown Title';
        this.previewAuthor.textContent = preview.book_author || 'Unknown Author';
        this.previewDescription.textContent = preview.description || 'No description available';

        // Set stats
        this.previewChapters.textContent = preview.total_chapters;
        this.previewWords.textContent = this.formatNumber(preview.total_words);
        this.previewDuration.textContent = preview.estimated_duration_formatted;

        // Update cost estimate for current selection
        this.updateCostEstimate();

        // Set chapter list
        this.previewChapterList.innerHTML = preview.chapters.map((ch, i) => `
            <div class="chapter-item">
                <span class="chapter-title">${i + 1}. ${this.escapeHtml(ch.title)}</span>
                <span class="chapter-words">${this.formatNumber(ch.word_count)} words</span>
            </div>
        `).join('');

        // Show preview section
        this.previewSection.style.display = 'block';
        this.previewSection.scrollIntoView({ behavior: 'smooth' });
    }

    closePreview() {
        this.previewSection.style.display = 'none';
    }

    async updateCostEstimate() {
        // Only update if preview is visible and we have character count
        if (!this.currentPreview || this.previewSection.style.display === 'none') {
            return;
        }

        const provider = this.providerSelect.value;
        const chars = this.currentPreview.total_characters || 0;

        if (!provider) {
            this.previewCosts.innerHTML = '<div class="cost-item"><span>Select a provider to see pricing</span></div>';
            return;
        }

        try {
            const response = await fetch(`/api/pricing?provider=${provider}&chars=${chars}`);
            const data = await response.json();

            if (!data.models || data.models.length === 0) {
                // Local provider (free)
                this.previewCosts.innerHTML = `
                    <div class="cost-item">
                        <span class="cost-provider">${provider}</span>
                        <span class="cost-value">Free</span>
                    </div>
                `;
                return;
            }

            // Display all models sorted by cost
            this.previewCosts.innerHTML = data.models.map(m => `
                <div class="cost-item">
                    <span class="cost-provider">${m.model}</span>
                    <span class="cost-value paid">$${m.cost.toFixed(2)}</span>
                </div>
            `).join('');
        } catch (e) {
            console.error('Failed to fetch pricing:', e);
            this.previewCosts.innerHTML = '<div class="cost-item"><span>Unable to fetch pricing</span></div>';
        }
    }

    async confirmConvert() {
        if (!this.selectedFile) return;
        this.closePreview();
        await this.uploadFile();
    }

    formatNumber(num) {
        if (num >= 1000000) return (num / 1000000).toFixed(1) + 'M';
        if (num >= 1000) return (num / 1000).toFixed(1) + 'K';
        return num.toString();
    }

    formatFileSize(bytes) {
        if (bytes < 1024) return bytes + ' B';
        if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
        return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
    }

    // Upload
    async uploadFile() {
        if (!this.selectedFile) return;

        const formData = new FormData();
        formData.append('file', this.selectedFile);
        formData.append('provider', this.providerSelect.value);
        formData.append('voice', this.voiceSelect.value);
        formData.append('speed', this.speedInput.value);
        formData.append('pitch', this.pitchInput.value);

        this.uploadBtn.disabled = true;
        this.previewBtn.disabled = true;

        try {
            // Use XMLHttpRequest for upload progress
            await this.uploadFileWithProgress(formData, (stage, progress) => {
                if (stage === 'uploading') {
                    this.uploadBtn.innerHTML = `<span class="spinner">⏳</span> Uploading... ${progress}%`;
                } else if (stage === 'processing') {
                    this.uploadBtn.innerHTML = '<span class="spinner">⏳</span> Processing...';
                }
            });

            this.showToast('Conversion job started!', 'success');
            this.clearSelectedFile();
        } catch (e) {
            this.showToast('Upload failed: ' + e.message, 'error');
        } finally {
            this.uploadBtn.disabled = !this.selectedFile;
            this.previewBtn.disabled = !this.selectedFile;
            this.uploadBtn.innerHTML = '📤 Start Conversion';
        }
    }

    // Upload file for conversion with progress tracking
    uploadFileWithProgress(formData, onProgress) {
        return new Promise((resolve, reject) => {
            const xhr = new XMLHttpRequest();
            
            // Track upload progress
            xhr.upload.addEventListener('progress', (e) => {
                if (e.lengthComputable) {
                    const percent = Math.round((e.loaded / e.total) * 100);
                    onProgress('uploading', percent);
                }
            });

            // When upload completes, server starts processing
            xhr.upload.addEventListener('load', () => {
                onProgress('processing', 100);
            });

            xhr.addEventListener('load', () => {
                if (xhr.status >= 200 && xhr.status < 300) {
                    resolve();
                } else {
                    try {
                        const error = JSON.parse(xhr.responseText);
                        reject(new Error(error.error || 'Upload failed'));
                    } catch (e) {
                        reject(new Error('Upload failed: ' + xhr.statusText));
                    }
                }
            });

            xhr.addEventListener('error', () => {
                reject(new Error('Network error'));
            });

            xhr.addEventListener('abort', () => {
                reject(new Error('Upload cancelled'));
            });

            xhr.open('POST', '/api/upload');
            xhr.send(formData);
        });
    }

    // Jobs management
    addJob(job) {
        this.jobs.set(job.id, job);
        
        const existingCard = document.getElementById(`job-${job.id}`);
        if (existingCard) {
            this.updateJobCard(existingCard, job);
        } else {
            const card = this.createJobCard(job);
            // Insert at the beginning (newest first)
            if (this.jobsList.firstChild) {
                this.jobsList.insertBefore(card, this.jobsList.firstChild);
            } else {
                this.jobsList.appendChild(card);
            }
        }

        this.updateJobsCount();
        this.updateEmptyState();
    }

    updateJob(job) {
        this.jobs.set(job.id, job);
        
        const card = document.getElementById(`job-${job.id}`);
        if (card) {
            this.updateJobCard(card, job);
        } else {
            this.addJob(job);
        }
    }

    removeJob(id) {
        this.jobs.delete(id);
        
        const card = document.getElementById(`job-${id}`);
        if (card) {
            card.remove();
        }

        this.updateJobsCount();
        this.updateEmptyState();
    }

    createJobCard(job) {
        const card = document.createElement('div');
        card.className = 'job-card';
        card.id = `job-${job.id}`;
        this.updateJobCard(card, job);
        return card;
    }

    updateJobCard(card, job) {
        const progress = Math.round(job.progress * 100);
        const statusClass = job.status.toLowerCase();
        
        let progressText = '';
        if (job.status === 'converting' && job.current_chapter) {
            progressText = `Chapter ${job.current_chapter_num}/${job.total_chapters}: ${job.current_chapter}`;
        } else if (job.status === 'parsing') {
            progressText = 'Parsing book...';
        } else if (job.status === 'building') {
            progressText = 'Building M4B audiobook...';
        } else if (job.status === 'uploading') {
            progressText = 'Uploading to Audiobookshelf...';
        } else if (job.status === 'pending') {
            progressText = 'Waiting in queue...';
        } else if (job.status === 'completed') {
            progressText = 'Conversion complete';
        }

        const title = job.book_title || job.file_name;
        const author = job.book_author ? ` by ${job.book_author}` : '';

        card.innerHTML = `
            <div class="job-header">
                <div>
                    <div class="job-title">${this.escapeHtml(title)}${this.escapeHtml(author)}</div>
                    <div class="job-meta">${this.escapeHtml(job.file_name)} • ${this.formatDate(job.created_at)}</div>
                </div>
                <span class="job-status ${statusClass}">
                    ${this.getStatusIcon(job.status)} ${job.status}
                </span>
            </div>
            
            ${job.status === 'converting' || job.status === 'parsing' || job.status === 'building' || job.status === 'uploading' ? `
            <div class="job-progress">
                <div class="progress-bar">
                    <div class="progress-fill" style="width: ${(job.status === 'building' || job.status === 'uploading') ? 100 : progress}%"></div>
                </div>
                <div class="progress-text">${progressText}${(job.status !== 'building' && job.status !== 'uploading') ? ` (${progress}%)` : ''}</div>
            </div>
            ` : ''}
            
            <div class="job-details">
                <div class="job-detail">
                    <span class="job-detail-label">Provider</span>
                    <span class="job-detail-value">${job.provider}</span>
                </div>
                <div class="job-detail">
                    <span class="job-detail-label">Voice</span>
                    <span class="job-detail-value">${job.voice}</span>
                </div>
                <div class="job-detail">
                    <span class="job-detail-label">Speed</span>
                    <span class="job-detail-value">${job.speed}x</span>
                </div>
                <div class="job-detail">
                    <span class="job-detail-label">Chapters</span>
                    <span class="job-detail-value">${job.total_chapters || '-'}</span>
                </div>
            </div>
            
            ${job.error ? `
            <div class="job-error">
                ⚠️ ${this.escapeHtml(job.error)}
            </div>
            ` : ''}
            
            <div class="job-actions">
                ${job.status === 'completed' ? `
                <button class="btn btn-primary" onclick="app.downloadJob('${job.id}')">
                    📥 Download
                </button>
                ` : ''}
                <button class="btn btn-danger" onclick="app.deleteJob('${job.id}')">
                    🗑️ Delete
                </button>
            </div>
        `;
    }

    getStatusIcon(status) {
        switch (status) {
            case 'pending': return '⏳';
            case 'parsing': return '📖';
            case 'converting': return '🔄';
            case 'building': return '📦';
            case 'uploading': return '☁️';
            case 'completed': return '✅';
            case 'failed': return '❌';
            case 'cancelled': return '🚫';
            default: return '❓';
        }
    }

    updateJobsCount() {
        this.jobsCount.textContent = `${this.jobs.size} job${this.jobs.size !== 1 ? 's' : ''}`;
    }

    updateEmptyState() {
        const emptyState = this.jobsList.querySelector('.empty-state');
        
        if (this.jobs.size === 0) {
            if (!emptyState) {
                const empty = document.createElement('div');
                empty.className = 'empty-state';
                empty.innerHTML = `
                    <span class="icon">📚</span>
                    <p>No conversion jobs yet</p>
                    <p>Upload a book to get started</p>
                `;
                this.jobsList.appendChild(empty);
            }
        } else if (emptyState) {
            emptyState.remove();
        }
    }

    async deleteJob(id) {
        if (!confirm('Are you sure you want to delete this job?')) return;

        try {
            const response = await fetch(`/api/jobs/${id}`, {
                method: 'DELETE'
            });

            if (!response.ok) {
                throw new Error('Failed to delete job');
            }

            this.removeJob(id);
            this.showToast('Job deleted', 'success');
        } catch (e) {
            this.showToast('Failed to delete job: ' + e.message, 'error');
        }
    }

    async downloadJob(id) {
        try {
            // Trigger file download by navigating to the download URL
            const downloadUrl = `/api/jobs/${id}/download`;
            
            // Create a temporary link and click it to trigger download
            const link = document.createElement('a');
            link.href = downloadUrl;
            link.download = ''; // Let the server set the filename
            document.body.appendChild(link);
            link.click();
            document.body.removeChild(link);
            
            this.showToast('Download started!', 'success');
        } catch (e) {
            this.showToast('Failed to download: ' + e.message, 'error');
        }
    }

    // Settings Methods
    async openSettings() {
        await this.loadSettings();
        this.populateSettingsForm();
        this.settingsModal.classList.add('active');
    }

    closeSettings() {
        this.settingsModal.classList.remove('active');
        this.absConnectionResult.textContent = '';
        this.absConnectionResult.className = 'connection-result';
    }

    switchTab(tabName) {
        // Update tab buttons
        this.settingsTabs.forEach(tab => {
            tab.classList.toggle('active', tab.dataset.tab === tabName);
        });
        
        // Update tab content
        this.tabContents.forEach(content => {
            content.classList.toggle('active', content.id === `tab-${tabName}`);
        });
    }

    async loadSettings() {
        try {
            const response = await fetch('/api/settings');
            if (response.ok) {
                this.settings = await response.json();
            }
        } catch (e) {
            console.error('Failed to load settings:', e);
            this.showToast('Failed to load settings', 'error');
        }
    }

    populateSettingsForm() {
        const s = this.settings;
        
        // General tab
        document.getElementById('cfg-server-host').value = s.server_host || '0.0.0.0';
        document.getElementById('cfg-server-port').value = s.server_port || '8080';
        document.getElementById('cfg-open-browser').checked = s.open_browser !== false;
        document.getElementById('cfg-output-dir').value = s.output_dir || './output';
        document.getElementById('cfg-temp-dir').value = s.temp_dir || './temp';
        document.getElementById('cfg-log-file').value = s.log_file || 'abb_tts.log';
        
        // TTS tab
        document.getElementById('cfg-default-provider').value = s.default_provider || 'espeak';
        document.getElementById('cfg-default-voice').value = s.default_voice || 'en-US';
        document.getElementById('cfg-default-speed').value = s.default_speed || 1.0;
        document.getElementById('cfg-default-pitch').value = s.default_pitch || 1.0;
        document.getElementById('cfg-use-default-pronunciation').checked = s.use_default_pronunciation !== false;
        document.getElementById('cfg-pronunciation-dict').value = s.pronunciation_dict_file || '';
        
        // Output tab
        document.getElementById('cfg-bit-rate').value = s.bit_rate_kbs || 128;
        document.getElementById('cfg-sample-rate').value = s.sample_rate_hz || 44100;
        document.getElementById('cfg-chapter-gap').value = s.chapter_gap_seconds || 2;
        document.getElementById('cfg-max-file-size').value = s.max_file_size_mb || 2000;
        
        // Cloud TTS tab
        document.getElementById('cfg-google-api-key').value = s.google_api_key || '';
        
        // Audiobookshelf tab
        document.getElementById('cfg-abs-url').value = s.audiobookshelf_url || '';
        document.getElementById('cfg-abs-user').value = s.audiobookshelf_user || 'admin';
        document.getElementById('cfg-abs-password').value = s.audiobookshelf_password || '';
        document.getElementById('cfg-abs-library').value = s.audiobookshelf_library || 'TTS Books';
    }

    collectSettingsForm() {
        return {
            // General
            server_host: document.getElementById('cfg-server-host').value,
            server_port: document.getElementById('cfg-server-port').value,
            open_browser: document.getElementById('cfg-open-browser').checked,
            output_dir: document.getElementById('cfg-output-dir').value,
            temp_dir: document.getElementById('cfg-temp-dir').value,
            log_file: document.getElementById('cfg-log-file').value,
            
            // TTS
            default_provider: document.getElementById('cfg-default-provider').value,
            default_voice: document.getElementById('cfg-default-voice').value,
            default_speed: parseFloat(document.getElementById('cfg-default-speed').value),
            default_pitch: parseFloat(document.getElementById('cfg-default-pitch').value),
            use_default_pronunciation: document.getElementById('cfg-use-default-pronunciation').checked,
            pronunciation_dict_file: document.getElementById('cfg-pronunciation-dict').value,
            
            // Output
            bit_rate_kbs: parseInt(document.getElementById('cfg-bit-rate').value),
            sample_rate_hz: parseInt(document.getElementById('cfg-sample-rate').value),
            chapter_gap_seconds: parseInt(document.getElementById('cfg-chapter-gap').value),
            max_file_size_mb: parseInt(document.getElementById('cfg-max-file-size').value),
            
            // Cloud TTS
            google_api_key: document.getElementById('cfg-google-api-key').value,
            
            // Audiobookshelf
            audiobookshelf_url: document.getElementById('cfg-abs-url').value,
            audiobookshelf_user: document.getElementById('cfg-abs-user').value,
            audiobookshelf_password: document.getElementById('cfg-abs-password').value,
            audiobookshelf_library: document.getElementById('cfg-abs-library').value
        };
    }

    async saveSettings() {
        try {
            const settings = this.collectSettingsForm();
            
            const response = await fetch('/api/settings', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(settings)
            });

            if (!response.ok) {
                const data = await response.json();
                throw new Error(data.error || 'Failed to save settings');
            }

            this.settings = settings;
            this.closeSettings();
            this.showToast('Settings saved successfully', 'success');
            
            // Reload config to update defaults
            await this.loadConfig();
        } catch (e) {
            this.showToast('Failed to save settings: ' + e.message, 'error');
        }
    }

    async testAudiobookshelfConnection() {
        const url = document.getElementById('cfg-abs-url').value;
        const user = document.getElementById('cfg-abs-user').value;
        const password = document.getElementById('cfg-abs-password').value;

        if (!url) {
            this.absConnectionResult.textContent = '❌ Please enter a server URL';
            this.absConnectionResult.className = 'connection-result error';
            return;
        }

        this.absConnectionResult.textContent = '⏳ Testing...';
        this.absConnectionResult.className = 'connection-result';

        try {
            const response = await fetch('/api/settings/test-audiobookshelf', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ url, user, password })
            });

            const data = await response.json();

            if (response.ok && data.success) {
                this.absConnectionResult.textContent = '✅ Connection successful!';
                this.absConnectionResult.className = 'connection-result success';
            } else {
                this.absConnectionResult.textContent = '❌ ' + (data.error || 'Connection failed');
                this.absConnectionResult.className = 'connection-result error';
            }
        } catch (e) {
            this.absConnectionResult.textContent = '❌ ' + e.message;
            this.absConnectionResult.className = 'connection-result error';
        }
    }

    // Utilities
    escapeHtml(text) {
        if (!text) return '';
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    formatDate(dateStr) {
        if (!dateStr) return '';
        const date = new Date(dateStr);
        return date.toLocaleString();
    }

    showToast(message, type = 'info') {
        const toast = document.createElement('div');
        toast.className = `toast ${type}`;
        toast.innerHTML = `
            <span>${type === 'success' ? '✅' : type === 'error' ? '❌' : 'ℹ️'}</span>
            <span>${this.escapeHtml(message)}</span>
        `;
        
        this.toastContainer.appendChild(toast);

        setTimeout(() => {
            toast.style.opacity = '0';
            setTimeout(() => toast.remove(), 300);
        }, 5000);
    }
}

// Initialize app when DOM is ready
let app;
document.addEventListener('DOMContentLoaded', () => {
    app = new App();
});
