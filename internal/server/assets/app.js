// Audiobook Builder TTS - Web Client

// localStorage key for TTS settings
const TTS_SETTINGS_KEY = 'abb_tts_settings';

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
        this.savedTTSSettings = this.loadTTSSettings();

        // OPDS state
        this.opdsSources = [];
        this.opdsHistory = []; // Navigation history for breadcrumbs
        this.currentOPDSBook = null;

        this.init();
    }

    // Load TTS settings from localStorage
    loadTTSSettings() {
        try {
            const saved = localStorage.getItem(TTS_SETTINGS_KEY);
            if (saved) {
                return JSON.parse(saved);
            }
        } catch (e) {
            console.error('Failed to load TTS settings from localStorage:', e);
        }
        return null;
    }

    // Save TTS settings to localStorage
    saveTTSSettings() {
        try {
            const settings = {
                provider: this.providerSelect.value,
                language: this.languageSelect.value,
                model: this.modelSelect.value,
                voice: this.voiceSelect.value
            };
            localStorage.setItem(TTS_SETTINGS_KEY, JSON.stringify(settings));
            this.savedTTSSettings = settings;
        } catch (e) {
            console.error('Failed to save TTS settings to localStorage:', e);
        }
    }

    async init() {
        this.bindElements();
        this.bindEvents();
        await this.loadConfig();
        await this.loadProviders();
        await this.loadJobs();
        await this.loadOPDSSources();
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
        this.testOpenTTSBtn = document.getElementById('test-opentts-connection');
        this.openTTSConnectionResult = document.getElementById('opentts-connection-result');
        this.testRHVoiceBtn = document.getElementById('test-rhvoice-connection');
        this.rhvoiceConnectionResult = document.getElementById('rhvoice-connection-result');

        // Main tab elements
        this.mainTabs = document.querySelectorAll('.main-tab');
        this.mainTabContents = document.querySelectorAll('.main-tab-content');

        // OPDS elements
        this.opdsSourceSelect = document.getElementById('opds-source');
        this.opdsBrowseBtn = document.getElementById('opds-browse-btn');
        this.opdsSearchContainer = document.getElementById('opds-search-container');
        this.opdsSearchType = document.getElementById('opds-search-type');
        this.opdsSearchInput = document.getElementById('opds-search-input');
        this.opdsSearchBtn = document.getElementById('opds-search-btn');
        this.opdsBreadcrumb = document.getElementById('opds-breadcrumb');
        this.opdsContent = document.getElementById('opds-content');
        this.opdsLoading = document.getElementById('opds-loading');
        
        // OPDS search state
        this.currentSearchInfo = null;

        // OPDS Book Modal
        this.opdsBookModal = document.getElementById('opds-book-modal');
        this.opdsBookTitle = document.getElementById('opds-book-title');
        this.opdsBookCover = document.getElementById('opds-book-cover');
        this.opdsBookAuthor = document.getElementById('opds-book-author');
        this.opdsBookSummary = document.getElementById('opds-book-summary');
        this.opdsBookLanguage = document.getElementById('opds-book-language');
        this.opdsBookPublisher = document.getElementById('opds-book-publisher');
        this.opdsBookCategories = document.getElementById('opds-book-categories');
        this.opdsDownloadOptions = document.getElementById('opds-download-options');
        this.opdsBookClose = document.getElementById('opds-book-close');
        this.opdsBookCancel = document.getElementById('opds-book-cancel');

        // OPDS Sources (in Settings tab)
        this.opdsSourcesList = document.getElementById('opds-sources-list');
        this.newSourceName = document.getElementById('new-source-name');
        this.newSourceUrl = document.getElementById('new-source-url');
        this.newSourceDesc = document.getElementById('new-source-desc');
        this.newSourceUsername = document.getElementById('new-source-username');
        this.newSourcePassword = document.getElementById('new-source-password');
        this.addSourceBtn = document.getElementById('add-source-btn');
    }

    bindEvents() {
        // Drop zone events
        this.dropZone.addEventListener('click', () => this.fileInput.click());
        this.dropZone.addEventListener('dragover', (e) => this.handleDragOver(e));
        this.dropZone.addEventListener('dragleave', () => this.handleDragLeave());
        this.dropZone.addEventListener('drop', (e) => this.handleDrop(e));
        this.fileInput.addEventListener('change', (e) => this.handleFileSelect(e));
        this.removeFileBtn.addEventListener('click', () => this.clearSelectedFile());

        // Cascading dropdown changes - settings are saved in loadVoices() at the end of cascade
        this.providerSelect.addEventListener('change', () => {
            this.loadLanguages();
            this.updateCostEstimate();
        });
        this.languageSelect.addEventListener('change', () => this.loadModels());
        this.modelSelect.addEventListener('change', () => this.loadVoices());
        // Save settings when voice is changed directly (without cascade)
        this.voiceSelect.addEventListener('change', () => this.saveTTSSettings());

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

        // Test OpenTTS connection
        if (this.testOpenTTSBtn) {
            this.testOpenTTSBtn.addEventListener('click', () => this.testOpenTTSConnection());
        }

        // Test RHVoice connection
        if (this.testRHVoiceBtn) {
            this.testRHVoiceBtn.addEventListener('click', () => this.testRHVoiceConnection());
        }

        // Test Voice Modal events
        this.testVoiceBtn = document.getElementById('test-voice-btn');
        this.testVoiceModal = document.getElementById('test-voice-modal');
        this.testVoiceClose = document.getElementById('test-voice-close');
        this.testVoiceDone = document.getElementById('test-voice-done');
        this.testVoiceGenerateBtn = document.getElementById('test-voice-generate-btn');
        this.testVoiceText = document.getElementById('test-voice-text');
        this.testVoiceStatus = document.getElementById('test-voice-status');
        this.testVoiceAudio = document.getElementById('test-voice-audio');
        this.testVoiceProviderDisplay = document.getElementById('test-voice-provider-display');
        this.testVoiceVoiceDisplay = document.getElementById('test-voice-voice-display');
        this.testVoiceSpeedDisplay = document.getElementById('test-voice-speed-display');
        this.testVoicePitchDisplay = document.getElementById('test-voice-pitch-display');

        if (this.testVoiceBtn) {
            this.testVoiceBtn.addEventListener('click', () => this.openTestVoiceModal());
        }
        if (this.testVoiceClose) {
            this.testVoiceClose.addEventListener('click', () => this.closeTestVoiceModal());
        }
        if (this.testVoiceDone) {
            this.testVoiceDone.addEventListener('click', () => this.closeTestVoiceModal());
        }
        if (this.testVoiceModal) {
            this.testVoiceModal.querySelector('.modal-overlay').addEventListener('click', () => this.closeTestVoiceModal());
        }
        if (this.testVoiceGenerateBtn) {
            this.testVoiceGenerateBtn.addEventListener('click', () => this.generateTestVoice());
        }

        // Main tab navigation
        this.mainTabs.forEach(tab => {
            tab.addEventListener('click', () => this.switchMainTab(tab.dataset.tab));
        });

        // OPDS events
        if (this.opdsBrowseBtn) {
            this.opdsBrowseBtn.addEventListener('click', () => this.browseOPDS());
        }
        if (this.opdsSearchBtn) {
            this.opdsSearchBtn.addEventListener('click', () => this.searchOPDS());
        }
        if (this.opdsSearchInput) {
            this.opdsSearchInput.addEventListener('keypress', (e) => {
                if (e.key === 'Enter') this.searchOPDS();
            });
        }
        // OPDS Book Modal events
        if (this.opdsBookClose) {
            this.opdsBookClose.addEventListener('click', () => this.closeBookModal());
        }
        if (this.opdsBookCancel) {
            this.opdsBookCancel.addEventListener('click', () => this.closeBookModal());
        }
        if (this.opdsBookModal) {
            this.opdsBookModal.querySelector('.modal-overlay').addEventListener('click', () => this.closeBookModal());
        }

        // OPDS Sources (in Settings tab)
        if (this.addSourceBtn) {
            this.addSourceBtn.addEventListener('click', () => this.addOPDSSource());
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
            
            // Use saved provider if available and valid, otherwise use server default
            const savedProvider = this.savedTTSSettings?.provider;
            const defaultProvider = (savedProvider && this.providers.includes(savedProvider)) 
                ? savedProvider 
                : data.default;
            
            this.providerSelect.innerHTML = this.providers.map(p => 
                `<option value="${p}" ${p === defaultProvider ? 'selected' : ''}>${p}</option>`
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

            // Use saved language if available and valid for current provider, otherwise default to en-US or first
            const savedLang = this.savedTTSSettings?.language;
            const isSameProvider = this.savedTTSSettings?.provider === provider;
            let defaultLang;
            if (isSameProvider && savedLang && languages.includes(savedLang)) {
                defaultLang = savedLang;
            } else {
                defaultLang = languages.includes('en-US') ? 'en-US' : languages[0];
            }

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

            // Use saved model if available and valid for current provider
            const savedModel = this.savedTTSSettings?.model;
            const isSameProvider = this.savedTTSSettings?.provider === provider;
            const defaultModel = (isSameProvider && savedModel && models.includes(savedModel)) 
                ? savedModel 
                : models[0];

            this.modelSelect.innerHTML = models.map(model => 
                `<option value="${model}" ${model === defaultModel ? 'selected' : ''}>${model}</option>`
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

            // Use saved voice if available and valid for current provider/language/model
            const savedVoice = this.savedTTSSettings?.voice;
            const isSameProvider = this.savedTTSSettings?.provider === provider;
            const isSameLanguage = this.savedTTSSettings?.language === language;
            const isSameModel = this.savedTTSSettings?.model === model;
            const voiceIds = this.voices.map(v => v.ID);
            let defaultVoice;
            if (isSameProvider && isSameLanguage && isSameModel && savedVoice && voiceIds.includes(savedVoice)) {
                defaultVoice = savedVoice;
            } else {
                defaultVoice = data.default || (this.voices.length > 0 ? this.voices[0].ID : '');
            }

            this.voiceSelect.innerHTML = this.voices.map(v => 
                `<option value="${v.ID}" ${v.ID === defaultVoice ? 'selected' : ''}>${v.Name}</option>`
            ).join('');
            
            // Save settings after all dropdowns are populated
            this.saveTTSSettings();
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

        // Set chapter list
        this.previewChapterList.innerHTML = preview.chapters.map((ch, i) => `
            <div class="chapter-item">
                <span class="chapter-title">${i + 1}. ${this.escapeHtml(ch.title)}</span>
                <span class="chapter-words">${this.formatNumber(ch.word_count)} words</span>
            </div>
        `).join('');

        // Show preview section first, then update cost estimate
        // (updateCostEstimate returns early if preview section is hidden)
        this.previewSection.style.display = 'block';
        this.previewSection.scrollIntoView({ behavior: 'smooth' });
        this.updateCostEstimate();
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
        if (!this.selectedFile && !this.currentPreview?.id) return;
        
        this.closePreview();
        
        // If we have a preview ID (from OPDS download), use the convert API
        if (this.currentPreview?.id && !this.selectedFile) {
            await this.convertOPDSBook();
        } else {
            await this.uploadFile();
        }
    }

    async convertOPDSBook() {
        if (!this.currentPreview?.id) return;

        this.confirmConvertBtn.disabled = true;
        this.confirmConvertBtn.innerHTML = '<span class="spinner">⏳</span> Starting...';

        try {
            const response = await fetch('/api/opds/convert', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    preview_id: this.currentPreview.id,
                    provider: this.providerSelect.value,
                    voice: this.voiceSelect.value,
                    speed: parseFloat(this.speedInput.value),
                    pitch: parseFloat(this.pitchInput.value)
                })
            });

            if (!response.ok) {
                const data = await response.json();
                throw new Error(data.error || 'Conversion failed');
            }

            this.showToast('Conversion job started!', 'success');
            this.clearSelectedFile();
        } catch (e) {
            this.showToast('Conversion failed: ' + e.message, 'error');
        } finally {
            this.confirmConvertBtn.disabled = false;
            this.confirmConvertBtn.innerHTML = '✅ Confirm & Start Conversion';
        }
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
        // Use conversion_progress for converting, build_progress for building
        const rawProgress = job.status === 'building' ? (job.build_progress || 0) : (job.conversion_progress || 0);
        const progress = Math.round(rawProgress * 100);
        const statusClass = job.status.toLowerCase();
        
        let progressText = '';
        if (job.status === 'converting' && job.current_chapter) {
            progressText = `${job.current_chapter_num}/${job.total_chapters} chapters converted`;
        } else if (job.status === 'parsing') {
            progressText = 'Parsing book...';
        } else if (job.status === 'building') {
            progressText = job.current_chapter || 'Building M4B audiobook...';
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
                    <div class="progress-fill" style="width: ${job.status === 'uploading' ? 100 : progress}%"></div>
                </div>
                <div class="progress-text">${progressText}${job.status !== 'uploading' ? ` (${progress}%)` : ''}</div>
            </div>
            ${(job.status === 'converting' || job.status === 'building') && job.worker_progress && job.worker_progress.length > 0 ? `
            <div class="worker-progress-container">
                ${job.worker_progress.map((wp, idx) => `
                    <div class="worker-progress ${wp.active ? 'active' : 'idle'}">
                        <div class="worker-label">${job.status === 'building' ? 'E' : 'W'}${idx + 1}</div>
                        <div class="worker-bar">
                            <div class="worker-fill" style="width: ${Math.round(wp.progress * 100)}%"></div>
                        </div>
                        <div class="worker-info">${wp.active ? (job.status === 'building' ? `${wp.chapter_title} ${wp.chunks_complete}%` : `Ch.${wp.chapter_index + 1} ${wp.chunks_complete}/${wp.chunks_total}`) : 'idle'}</div>
                    </div>
                `).join('')}
            </div>
            ` : ''}
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

        // Render OPDS sources when switching to OPDS tab
        if (tabName === 'opds') {
            this.renderSourcesList();
        }
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
        document.getElementById('cfg-max-file-size').value = s.max_file_size_mb || 250;
        
        // Performance tab - per-provider TTS workers
        const defaultWorkers = s.concurrent_tts_workers || 3;
        const providerWorkers = s.provider_tts_workers || {};
        document.getElementById('cfg-tts-workers-espeak').value = providerWorkers.espeak || defaultWorkers;
        document.getElementById('cfg-tts-workers-opentts').value = providerWorkers.opentts || defaultWorkers;
        document.getElementById('cfg-tts-workers-rhvoice').value = providerWorkers.rhvoice || defaultWorkers;
        document.getElementById('cfg-tts-workers-openai').value = providerWorkers.openai || defaultWorkers;
        document.getElementById('cfg-tts-workers-google').value = providerWorkers.google || defaultWorkers;
        document.getElementById('cfg-tts-workers-azure').value = providerWorkers.azure || defaultWorkers;
        document.getElementById('cfg-concurrent-encoders').value = s.concurrent_encoders || 2;
        
        // Cloud TTS tab
        document.getElementById('cfg-openai-api-key').value = s.openai_api_key || '';
        document.getElementById('cfg-google-api-key').value = s.google_api_key || '';
        document.getElementById('cfg-azure-tts-key').value = s.azure_tts_key || '';
        document.getElementById('cfg-azure-tts-region').value = s.azure_tts_region || '';
        
        // OpenTTS tab
        document.getElementById('cfg-opentts-url').value = s.opentts_url || '';
        
        // RHVoice tab
        document.getElementById('cfg-rhvoice-url').value = s.rhvoice_url || '';
        
        // Audiobookshelf tab
        document.getElementById('cfg-abs-url').value = s.audiobookshelf_url || '';
        document.getElementById('cfg-abs-user').value = s.audiobookshelf_user || 'admin';
        document.getElementById('cfg-abs-password').value = s.audiobookshelf_password || '';
        document.getElementById('cfg-abs-library').value = s.audiobookshelf_library || 'TTS books';
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
            
            // Performance - per-provider TTS workers
            provider_tts_workers: {
                espeak: parseInt(document.getElementById('cfg-tts-workers-espeak').value),
                opentts: parseInt(document.getElementById('cfg-tts-workers-opentts').value),
                rhvoice: parseInt(document.getElementById('cfg-tts-workers-rhvoice').value),
                openai: parseInt(document.getElementById('cfg-tts-workers-openai').value),
                google: parseInt(document.getElementById('cfg-tts-workers-google').value),
                azure: parseInt(document.getElementById('cfg-tts-workers-azure').value)
            },
            concurrent_encoders: parseInt(document.getElementById('cfg-concurrent-encoders').value),
            
            // Cloud TTS
            openai_api_key: document.getElementById('cfg-openai-api-key').value,
            google_api_key: document.getElementById('cfg-google-api-key').value,
            azure_tts_key: document.getElementById('cfg-azure-tts-key').value,
            azure_tts_region: document.getElementById('cfg-azure-tts-region').value,
            
            // OpenTTS
            opentts_url: document.getElementById('cfg-opentts-url').value,
            
            // RHVoice
            rhvoice_url: document.getElementById('cfg-rhvoice-url').value,
            
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

    async testOpenTTSConnection() {
        const url = document.getElementById('cfg-opentts-url').value;

        if (!url) {
            this.openTTSConnectionResult.textContent = '❌ Please enter a server URL';
            this.openTTSConnectionResult.className = 'connection-result error';
            return;
        }

        this.openTTSConnectionResult.textContent = '⏳ Testing...';
        this.openTTSConnectionResult.className = 'connection-result';

        try {
            const response = await fetch('/api/settings/test-opentts', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ url })
            });

            const data = await response.json();

            if (response.ok && data.success) {
                this.openTTSConnectionResult.textContent = `✅ Connected! ${data.voice_count} voices available`;
                this.openTTSConnectionResult.className = 'connection-result success';
            } else {
                this.openTTSConnectionResult.textContent = '❌ ' + (data.error || 'Connection failed');
                this.openTTSConnectionResult.className = 'connection-result error';
            }
        } catch (e) {
            this.openTTSConnectionResult.textContent = '❌ ' + e.message;
            this.openTTSConnectionResult.className = 'connection-result error';
        }
    }

    async testRHVoiceConnection() {
        const url = document.getElementById('cfg-rhvoice-url').value;

        if (!url) {
            this.rhvoiceConnectionResult.textContent = '❌ Please enter a server URL';
            this.rhvoiceConnectionResult.className = 'connection-result error';
            return;
        }

        this.rhvoiceConnectionResult.textContent = '⏳ Testing...';
        this.rhvoiceConnectionResult.className = 'connection-result';

        try {
            const response = await fetch('/api/settings/test-rhvoice', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ url })
            });

            const data = await response.json();

            if (response.ok && data.success) {
                this.rhvoiceConnectionResult.textContent = `✅ Connected! ${data.voice_count} voices available`;
                this.rhvoiceConnectionResult.className = 'connection-result success';
            } else {
                this.rhvoiceConnectionResult.textContent = '❌ ' + (data.error || 'Connection failed');
                this.rhvoiceConnectionResult.className = 'connection-result error';
            }
        } catch (e) {
            this.rhvoiceConnectionResult.textContent = '❌ ' + e.message;
            this.rhvoiceConnectionResult.className = 'connection-result error';
        }
    }

    // Test Voice Modal Methods
    openTestVoiceModal() {
        const provider = this.providerSelect.value;
        const voice = this.voiceSelect.value;
        const voiceName = this.voiceSelect.options[this.voiceSelect.selectedIndex]?.text || voice;
        const speed = this.speedInput.value;
        const pitch = this.pitchInput.value;

        if (!provider || !voice) {
            this.showToast('Please select a provider and voice first', 'error');
            return;
        }

        // Update display values
        this.testVoiceProviderDisplay.textContent = provider;
        this.testVoiceVoiceDisplay.textContent = voiceName;
        this.testVoiceSpeedDisplay.textContent = speed + 'x';
        this.testVoicePitchDisplay.textContent = pitch + 'x';

        // Reset audio player and status
        this.testVoiceAudio.src = '';
        this.testVoiceAudio.style.display = 'none';
        this.testVoiceStatus.textContent = '';
        this.testVoiceStatus.className = 'test-voice-status';

        // Show modal
        this.testVoiceModal.classList.add('active');
    }

    closeTestVoiceModal() {
        this.testVoiceModal.classList.remove('active');
        // Stop audio if playing
        if (this.testVoiceAudio) {
            this.testVoiceAudio.pause();
        }
    }

    async generateTestVoice() {
        const provider = this.providerSelect.value;
        const voice = this.voiceSelect.value;
        const text = this.testVoiceText?.value || 'Hello, this is a test of the text to speech voice.';
        const speed = parseFloat(this.speedInput.value || '1.0');
        const pitch = parseFloat(this.pitchInput.value || '1.0');

        if (!provider || !voice) {
            this.showToast('Please select a provider and voice first', 'error');
            return;
        }

        this.testVoiceGenerateBtn.disabled = true;
        this.testVoiceGenerateBtn.innerHTML = '<span class="spinner">⏳</span> Generating...';
        this.testVoiceStatus.textContent = '';
        this.testVoiceStatus.className = 'test-voice-status';
        this.testVoiceAudio.style.display = 'none';

        try {
            const response = await fetch('/api/test-voice', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ provider, voice, text, speed, pitch })
            });

            if (!response.ok) {
                const data = await response.json();
                throw new Error(data.error || 'Failed to generate audio');
            }

            const audioBlob = await response.blob();
            const audioUrl = URL.createObjectURL(audioBlob);
            
            this.testVoiceAudio.src = audioUrl;
            this.testVoiceAudio.style.display = 'block';
            this.testVoiceAudio.play();
            
            this.testVoiceStatus.textContent = '✅ Audio generated successfully';
            this.testVoiceStatus.className = 'test-voice-status success';
        } catch (e) {
            this.testVoiceStatus.textContent = '❌ ' + e.message;
            this.testVoiceStatus.className = 'test-voice-status error';
        } finally {
            this.testVoiceGenerateBtn.disabled = false;
            this.testVoiceGenerateBtn.innerHTML = '🔊 Generate Audio';
        }
    }

    // Main Tab Navigation
    switchMainTab(tabName) {
        this.mainTabs.forEach(tab => {
            tab.classList.toggle('active', tab.dataset.tab === tabName);
        });
        this.mainTabContents.forEach(content => {
            content.classList.toggle('active', content.id === `tab-${tabName}`);
        });
    }

    // OPDS Methods
    async loadOPDSSources() {
        try {
            const response = await fetch('/api/opds/sources');
            if (response.ok) {
                const data = await response.json();
                this.opdsSources = data.sources || [];
                this.populateSourceSelect();
            }
        } catch (e) {
            console.error('Failed to load OPDS sources:', e);
        }
    }

    populateSourceSelect() {
        if (!this.opdsSourceSelect) return;

        if (this.opdsSources.length === 0) {
            this.opdsSourceSelect.innerHTML = '<option value="">No sources configured</option>';
            return;
        }

        this.opdsSourceSelect.innerHTML = this.opdsSources
            .filter(s => s.enabled)
            .map(s => `<option value="${s.id}" data-url="${s.url}">${this.escapeHtml(s.name)}${s.username ? ' 🔒' : ''}</option>`)
            .join('');
    }

    getCurrentSourceId() {
        return this.opdsSourceSelect?.value || '';
    }

    getCurrentSourceUrl() {
        const option = this.opdsSourceSelect?.selectedOptions[0];
        return option?.dataset.url || '';
    }

    async browseOPDS() {
        const sourceId = this.getCurrentSourceId();
        const url = this.getCurrentSourceUrl();
        if (!sourceId || !url) {
            this.showToast('Please select a catalog source', 'error');
            return;
        }

        // Store current source ID for subsequent requests
        this.currentOPDSSourceId = sourceId;

        // Reset history and start fresh
        const sourceName = this.opdsSourceSelect.options[this.opdsSourceSelect.selectedIndex].text;
        this.opdsHistory = [{ url, title: sourceName }];
        await this.fetchOPDSCatalog(url);
    }

    async fetchOPDSCatalog(url) {
        this.showOPDSLoading(true);

        try {
            let apiUrl = `/api/opds/browse?url=${encodeURIComponent(url)}`;
            if (this.currentOPDSSourceId) {
                apiUrl += `&source_id=${encodeURIComponent(this.currentOPDSSourceId)}`;
            }
            const response = await fetch(apiUrl);
            if (!response.ok) {
                const data = await response.json();
                throw new Error(data.error || 'Failed to fetch catalog');
            }

            const catalog = await response.json();
            this.renderOPDSCatalog(catalog);
        } catch (e) {
            this.showToast('Failed to load catalog: ' + e.message, 'error');
            this.opdsContent.innerHTML = `
                <div class="opds-empty">
                    <span class="icon">❌</span>
                    <p>Failed to load catalog</p>
                    <p>${this.escapeHtml(e.message)}</p>
                </div>
            `;
        } finally {
            this.showOPDSLoading(false);
        }
    }

    renderOPDSCatalog(catalog) {
        // Update breadcrumb
        this.renderBreadcrumb();

        // Show search if available and configure search types
        if (catalog.search_info && catalog.search_info.supported) {
            this.currentSearchInfo = catalog.search_info;
            this.opdsSearchContainer.style.display = 'flex';
            
            // Configure search type options based on available search URLs
            this.updateSearchTypeOptions(catalog.search_info);
        } else {
            this.currentSearchInfo = null;
            this.opdsSearchContainer.style.display = 'none';
        }

        if (!catalog.entries || catalog.entries.length === 0) {
            this.opdsContent.innerHTML = `
                <div class="opds-empty">
                    <span class="icon">📭</span>
                    <p>No entries found</p>
                </div>
            `;
            return;
        }

        // Render entries as a grid
        let html = '<div class="opds-grid">';
        
        for (const entry of catalog.entries) {
            if (entry.is_navigation) {
                // Navigation entry (folder)
                html += this.renderNavigationEntry(entry);
            } else {
                // Book entry
                html += this.renderBookEntry(entry);
            }
        }

        html += '</div>';

        // Add pagination if available
        if (catalog.prev_page_url || catalog.next_page_url) {
            html += '<div class="opds-pagination">';
            
            if (catalog.prev_page_url) {
                html += `
                    <button class="btn btn-secondary" onclick="app.loadPage('${this.escapeHtml(catalog.prev_page_url)}')">
                        ← Previous
                    </button>
                `;
            }
            
            if (catalog.next_page_url) {
                html += `
                    <button class="btn btn-secondary" onclick="app.loadPage('${this.escapeHtml(catalog.next_page_url)}')">
                        Next →
                    </button>
                `;
            }
            
            html += '</div>';
        }

        this.opdsContent.innerHTML = html;
    }

    renderNavigationEntry(entry) {
        const navUrl = entry.navigation_link;
        return `
            <div class="opds-entry navigation" onclick="app.navigateOPDS('${this.escapeHtml(navUrl)}', '${this.escapeHtml(entry.title)}')">
                <div class="opds-entry-cover">
                    <span class="nav-icon">📁</span>
                </div>
                <div class="opds-entry-info">
                    <div class="opds-entry-title">${this.escapeHtml(entry.title)}</div>
                </div>
            </div>
        `;
    }

    renderBookEntry(entry) {
        const coverUrl = entry.cover_url || entry.thumbnail_url;
        const coverHtml = coverUrl 
            ? `<img src="/api/opds/proxy?url=${encodeURIComponent(coverUrl)}" alt="Cover" onerror="this.parentElement.innerHTML='<span class=\\'no-cover\\'>📖</span>'">`
            : '<span class="no-cover">📖</span>';

        const authors = entry.authors && entry.authors.length > 0 
            ? entry.authors.join(', ') 
            : '';

        const formats = entry.download_links
            .filter(dl => dl.format === 'epub' || dl.format === 'fb2')
            .map(dl => dl.format.toUpperCase())
            .filter((v, i, a) => a.indexOf(v) === i)
            .join(', ');

        return `
            <div class="opds-entry" onclick='app.showBookDetails(${JSON.stringify(entry).replace(/'/g, "\\'")})'>
                <div class="opds-entry-cover">${coverHtml}</div>
                <div class="opds-entry-info">
                    <div class="opds-entry-title">${this.escapeHtml(entry.title)}</div>
                    ${authors ? `<div class="opds-entry-author">${this.escapeHtml(authors)}</div>` : ''}
                    ${formats ? `<span class="opds-entry-format">${formats}</span>` : ''}
                </div>
            </div>
        `;
    }

    navigateOPDS(url, title) {
        this.opdsHistory.push({ url, title });
        this.fetchOPDSCatalog(url);
    }

    renderBreadcrumb() {
        if (this.opdsHistory.length <= 1) {
            this.opdsBreadcrumb.innerHTML = '';
            return;
        }

        let html = '';
        this.opdsHistory.forEach((item, index) => {
            if (index > 0) {
                html += '<span class="opds-breadcrumb-separator">›</span>';
            }
            if (index === this.opdsHistory.length - 1) {
                html += `<span class="opds-breadcrumb-current">${this.escapeHtml(item.title)}</span>`;
            } else {
                html += `<span class="opds-breadcrumb-item" onclick="app.goToHistoryIndex(${index})">${this.escapeHtml(item.title)}</span>`;
            }
        });

        this.opdsBreadcrumb.innerHTML = html;
    }

    goToHistoryIndex(index) {
        const item = this.opdsHistory[index];
        this.opdsHistory = this.opdsHistory.slice(0, index + 1);
        this.fetchOPDSCatalog(item.url);
    }

    loadPage(url) {
        // Load a page (next or previous) without adding to breadcrumb history
        this.fetchOPDSCatalog(url);
    }

    loadNextPage(url) {
        // Keep for backward compatibility
        this.loadPage(url);
    }

    async searchOPDS() {
        const query = this.opdsSearchInput.value.trim();
        if (!query) return;

        if (!this.currentSearchInfo || !this.currentSearchInfo.supported) {
            this.showToast('Search not available for this catalog', 'warning');
            return;
        }

        const sourceId = this.opdsSourceSelect.value;
        if (!sourceId) {
            this.showToast('Please select a catalog source', 'warning');
            return;
        }

        const searchType = this.opdsSearchType ? this.opdsSearchType.value : 'title';

        this.showOPDSLoading(true);

        try {
            const params = new URLSearchParams({
                source_id: sourceId,
                q: query,
                type: searchType
            });

            const response = await fetch(`/api/opds/search?${params}`);
            if (!response.ok) {
                const error = await response.json();
                throw new Error(error.error || 'Search failed');
            }

            const catalog = await response.json();
            
            // Add search results to breadcrumb
            this.opdsHistory.push({
                url: `search:${searchType}:${query}`,
                title: `Search: "${query}" (${searchType})`
            });
            
            this.renderOPDSCatalog(catalog);
            this.showToast(`Found ${catalog.entries ? catalog.entries.length : 0} results`, 'success');
        } catch (e) {
            console.error('OPDS search error:', e);
            this.showToast(`Search failed: ${e.message}`, 'error');
        } finally {
            this.showOPDSLoading(false);
        }
    }

    updateSearchTypeOptions(searchInfo) {
        if (!this.opdsSearchType) return;

        // Clear existing options
        this.opdsSearchType.innerHTML = '';

        // Add options based on available search URLs
        const hasTitleSearch = searchInfo.title_search_url || searchInfo.search_template_url || searchInfo.opensearch_url;
        const hasAuthorSearch = searchInfo.author_search_url;

        if (hasTitleSearch) {
            const option = document.createElement('option');
            option.value = 'title';
            option.textContent = '📖 By Title';
            this.opdsSearchType.appendChild(option);
        }

        if (hasAuthorSearch) {
            const option = document.createElement('option');
            option.value = 'author';
            option.textContent = '✍️ By Author';
            this.opdsSearchType.appendChild(option);
        }

        // If no specific search types, add a generic "All" option
        if (!hasTitleSearch && !hasAuthorSearch) {
            const option = document.createElement('option');
            option.value = '';
            option.textContent = '🔍 All';
            this.opdsSearchType.appendChild(option);
        }

        // Update placeholder based on selected type
        this.updateSearchPlaceholder();
        this.opdsSearchType.addEventListener('change', () => this.updateSearchPlaceholder());
    }

    updateSearchPlaceholder() {
        if (!this.opdsSearchInput || !this.opdsSearchType) return;
        
        const type = this.opdsSearchType.value;
        if (type === 'author') {
            this.opdsSearchInput.placeholder = 'Enter author name...';
        } else if (type === 'title') {
            this.opdsSearchInput.placeholder = 'Enter book title...';
        } else {
            this.opdsSearchInput.placeholder = 'Search books...';
        }
    }

    showOPDSLoading(show) {
        if (this.opdsLoading) {
            this.opdsLoading.style.display = show ? 'block' : 'none';
        }
        if (this.opdsContent) {
            this.opdsContent.style.display = show ? 'none' : 'block';
        }
    }

    // Book Details Modal
    showBookDetails(entry) {
        this.currentOPDSBook = entry;

        // Set title
        this.opdsBookTitle.textContent = entry.title || 'Unknown Title';

        // Set cover
        const coverUrl = entry.cover_url || entry.thumbnail_url;
        if (coverUrl) {
            this.opdsBookCover.innerHTML = `<img src="/api/opds/proxy?url=${encodeURIComponent(coverUrl)}" alt="Cover" onerror="this.parentElement.innerHTML='<span class=\\'no-cover\\'>No Cover</span>'">`;
        } else {
            this.opdsBookCover.innerHTML = '<span class="no-cover">No Cover</span>';
        }

        // Set author
        this.opdsBookAuthor.textContent = entry.authors && entry.authors.length > 0 
            ? 'by ' + entry.authors.join(', ')
            : '';

        // Set summary
        this.opdsBookSummary.textContent = entry.summary || 'No description available';

        // Set meta
        this.opdsBookLanguage.textContent = entry.language ? `Language: ${entry.language}` : '';
        this.opdsBookPublisher.textContent = entry.publisher ? `Publisher: ${entry.publisher}` : '';

        // Set categories
        if (entry.categories && entry.categories.length > 0) {
            this.opdsBookCategories.innerHTML = entry.categories
                .map(cat => `<span class="opds-book-category">${this.escapeHtml(cat)}</span>`)
                .join('');
        } else {
            this.opdsBookCategories.innerHTML = '';
        }

        // Set download options
        this.renderDownloadOptions(entry.download_links);

        // Show modal
        this.opdsBookModal.classList.add('active');
    }

    renderDownloadOptions(downloadLinks) {
        if (!downloadLinks || downloadLinks.length === 0) {
            this.opdsDownloadOptions.innerHTML = '<p>No download options available</p>';
            return;
        }

        const supportedFormats = ['epub', 'fb2'];
        
        this.opdsDownloadOptions.innerHTML = downloadLinks.map(dl => {
            const isSupported = supportedFormats.includes(dl.format);
            const formatClass = isSupported ? '' : 'unsupported';
            const buttonHtml = isSupported 
                ? `<button class="btn btn-primary" onclick="app.downloadAndConvert('${this.escapeHtml(dl.url)}', '${dl.format}')">📥 Convert to Audiobook</button>`
                : `<span class="btn btn-secondary" disabled>Not Supported</span>`;

            return `
                <div class="download-option">
                    <div class="download-option-info">
                        <span class="download-option-format ${formatClass}">${dl.format.toUpperCase()}</span>
                        <span class="download-option-title">${dl.title || dl.type || 'Download'}</span>
                    </div>
                    ${buttonHtml}
                </div>
            `;
        }).join('');
    }

    async downloadAndConvert(url, format) {
        this.closeBookModal();
        this.showToast('Downloading book...', 'info');

        try {
            const response = await fetch('/api/opds/download', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    url: url,
                    title: this.currentOPDSBook?.title || 'book',
                    format: format,
                    author: this.currentOPDSBook?.authors?.[0] || '',
                    source_id: this.currentOPDSSourceId || ''
                })
            });

            if (!response.ok) {
                const data = await response.json();
                throw new Error(data.error || 'Download failed');
            }

            const preview = await response.json();
            
            // Switch to upload tab and show preview
            this.switchMainTab('upload');
            this.currentPreview = preview;
            this.showPreview(preview);
            
            // Set the file info
            this.fileNameEl.textContent = preview.file_name;
            this.fileSizeEl.textContent = '';
            this.selectedFileEl.classList.add('visible');
            this.previewBtn.disabled = true; // Already have preview
            this.uploadBtn.disabled = true; // Use confirm button in preview

            this.showToast('Book downloaded! Review and start conversion.', 'success');
        } catch (e) {
            this.showToast('Download failed: ' + e.message, 'error');
        }
    }

    closeBookModal() {
        this.opdsBookModal.classList.remove('active');
        this.currentOPDSBook = null;
    }

    // OPDS Sources Management
    renderSourcesList() {
        if (!this.opdsSourcesList) return;
        if (this.opdsSources.length === 0) {
            this.opdsSourcesList.innerHTML = '<p style="padding: 1rem; color: var(--text-muted);">No sources configured</p>';
            return;
        }

        this.opdsSourcesList.innerHTML = this.opdsSources.map(source => `
            <div class="opds-source-item">
                <div class="opds-source-info">
                    <div class="opds-source-name">
                        ${this.escapeHtml(source.name)}
                        ${source.username ? '<span class="opds-source-badge">🔒 Auth</span>' : ''}
                        ${source.is_default ? '<span class="opds-source-badge">Default</span>' : ''}
                    </div>
                    <div class="opds-source-url">${this.escapeHtml(source.url)}</div>
                    ${source.description ? `<div class="opds-source-desc">${this.escapeHtml(source.description)}</div>` : ''}
                </div>
                <div class="opds-source-actions">
                    ${!source.is_default ? `<button class="btn btn-danger" onclick="app.deleteOPDSSource('${source.id}')">🗑️</button>` : ''}
                </div>
            </div>
        `).join('');
    }

    async addOPDSSource() {
        const name = this.newSourceName.value.trim();
        const url = this.newSourceUrl.value.trim();
        const description = this.newSourceDesc.value.trim();
        const username = this.newSourceUsername?.value.trim() || '';
        const password = this.newSourcePassword?.value || '';

        if (!name || !url) {
            this.showToast('Name and URL are required', 'error');
            return;
        }

        try {
            const response = await fetch('/api/opds/sources', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ name, url, description, username, password, enabled: true })
            });

            if (!response.ok) {
                const data = await response.json();
                throw new Error(data.error || 'Failed to add source');
            }

            const source = await response.json();
            this.opdsSources.push(source);
            this.renderSourcesList();
            this.populateSourceSelect();

            // Clear form
            this.newSourceName.value = '';
            this.newSourceUrl.value = '';
            this.newSourceDesc.value = '';
            if (this.newSourceUsername) this.newSourceUsername.value = '';
            if (this.newSourcePassword) this.newSourcePassword.value = '';

            this.showToast('Source added successfully', 'success');
        } catch (e) {
            this.showToast('Failed to add source: ' + e.message, 'error');
        }
    }

    async deleteOPDSSource(id) {
        if (!confirm('Are you sure you want to delete this source?')) return;

        try {
            const response = await fetch(`/api/opds/sources/${id}`, {
                method: 'DELETE'
            });

            if (!response.ok) {
                const data = await response.json();
                throw new Error(data.error || 'Failed to delete source');
            }

            this.opdsSources = this.opdsSources.filter(s => s.id !== id);
            this.renderSourcesList();
            this.populateSourceSelect();

            this.showToast('Source deleted', 'success');
        } catch (e) {
            this.showToast('Failed to delete source: ' + e.message, 'error');
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
