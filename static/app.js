document.addEventListener('DOMContentLoaded', () => {
    // DOM Elements
    const searchForm = document.getElementById('search-form');
    const searchInput = document.getElementById('search-query');
    const sourceSelect = document.getElementById('source-select');
    const searchBtn = document.getElementById('search-btn');
    const resultsCount = document.getElementById('results-count');
    const resultsGrid = document.getElementById('results-grid');
    const noResults = document.getElementById('no-results');
    const searchLoading = document.getElementById('search-loading');
    
    const activeCount = document.getElementById('active-count');
    const emptyQueue = document.getElementById('empty-queue');
    const queueList = document.getElementById('queue-list');
    const serverConfig = document.getElementById('server-config');
    const organizeBtn = document.getElementById('organize-btn');

    // State Store
    let sources = [];
    let activeDownloads = {};

    // Initial Bootstrap
    fetchConfig();
    fetchSources();
    startPollingQueue();

    // Fetch Configurations (Port & Volume Path)
    async function fetchConfig() {
        try {
            const res = await fetch('/api/config');
            if (res.ok) {
                const config = await res.json();
                serverConfig.querySelector('.config-text').innerHTML = `Downloads folder: <strong>${config.downloadsDir}</strong>`;
            }
        } catch (err) {
            console.error('Error fetching config:', err);
            serverConfig.querySelector('.config-text').textContent = 'Server Unreachable';
            serverConfig.querySelector('.pulse-indicator').style.backgroundColor = 'var(--danger)';
            serverConfig.querySelector('.pulse-indicator').style.boxShadow = '0 0 10px rgba(239, 68, 68, 0.4)';
        }
    }

    // Fetch Scraper Sources
    async function fetchSources() {
        try {
            const res = await fetch('/api/sources');
            if (res.ok) {
                sources = await res.json();
                sourceSelect.innerHTML = sources.map(src => `<option value="${src}">${src}</option>`).join('');
            }
        } catch (err) {
            console.error('Error fetching sources:', err);
        }
    }

    // Handle ROM Search Submission
    searchForm.addEventListener('submit', async (e) => {
        e.preventDefault();
        const query = searchInput.value.trim();
        const source = sourceSelect.value;
        if (!query) return;

        // UI Feedback - Show loading state
        setSearchLoading(true);

        try {
            const res = await fetch(`/api/search?source=${encodeURIComponent(source)}&query=${encodeURIComponent(query)}`);
            if (res.ok) {
                const roms = await res.json();
                renderResults(roms);
            } else {
                showToast(`Search failed with status: ${res.status}`, 'danger');
                setSearchLoading(false);
            }
        } catch (err) {
            console.error('Error searching:', err);
            showToast('Network error during search', 'danger');
            setSearchLoading(false);
        }
    });

    function setSearchLoading(isLoading) {
        if (isLoading) {
            searchBtn.disabled = true;
            searchBtn.querySelector('span').textContent = 'Searching...';
            noResults.classList.add('hidden');
            resultsGrid.classList.add('hidden');
            searchLoading.classList.remove('hidden');
            resultsCount.textContent = 'Scanning...';
        } else {
            searchBtn.disabled = false;
            searchBtn.querySelector('span').textContent = 'Search ROMs';
            searchLoading.classList.add('hidden');
        }
    }

    // Render Scraped ROM results
    function renderResults(roms) {
        setSearchLoading(false);
        
        if (!roms || roms.length === 0) {
            noResults.classList.remove('hidden');
            noResults.querySelector('h3').textContent = 'No ROMs Found';
            noResults.querySelector('p').textContent = 'Try another name or search via a different source.';
            resultsGrid.classList.add('hidden');
            resultsCount.textContent = '0 items found';
            return;
        }

        resultsCount.textContent = `${roms.length} item${roms.length > 1 ? 's' : ''} found`;
        noResults.classList.add('hidden');
        resultsGrid.classList.remove('hidden');

        resultsGrid.innerHTML = roms.map((rom, index) => {
            // Escape values for safe attribute injecting
            const escName = escapeHtml(rom.Name);
            const escConsole = escapeHtml(rom.Console || 'Unknown Platform');
            const escUrl = escapeHtml(rom.URL);
            
            return `
                <div class="rom-item">
                    <div class="rom-info">
                        <span class="rom-name">${escName}</span>
                        <div class="rom-meta">
                            <span class="rom-console-badge">${escConsole}</span>
                        </div>
                    </div>
                    <button class="btn btn-secondary btn-download-action" 
                        data-source="${escapeHtml(sourceSelect.value)}" 
                        data-name="${escName}" 
                        data-console="${escConsole}" 
                        data-url="${escUrl}">
                        Download
                    </button>
                </div>
            `;
        }).join('');

        // Attach action handlers to download buttons
        document.querySelectorAll('.btn-download-action').forEach(btn => {
            btn.addEventListener('click', handleDownloadTrigger);
        });
    }

    // Trigger background download request
    async function handleDownloadTrigger(e) {
        const btn = e.currentTarget;
        const source = btn.dataset.source;
        const name = btn.dataset.name;
        const consoleName = btn.dataset.console;
        const url = btn.dataset.url;

        btn.disabled = true;
        btn.textContent = 'Queuing...';

        try {
            const res = await fetch('/api/download', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ source, name, console: consoleName, url })
            });

            if (res.ok) {
                const data = await res.json();
                showToast(`Queued download for "${name}"`, 'success');
                btn.textContent = 'Queued';
                // Trigger an immediate check on downloads list
                pollQueue();
            } else {
                const errData = await res.json().catch(() => ({}));
                showToast(`Queue failed: ${errData.error || res.statusText}`, 'danger');
                btn.disabled = false;
                btn.textContent = 'Download';
            }
        } catch (err) {
            console.error('Error triggering download:', err);
            showToast('Failed to queue download', 'danger');
            btn.disabled = false;
            btn.textContent = 'Download';
        }
    }

    // Live Queue Polling
    function startPollingQueue() {
        pollQueue();
        setInterval(pollQueue, 1000);
    }

    async function pollQueue() {
        try {
            const res = await fetch('/api/downloads');
            if (res.ok) {
                const downloads = await res.json();
                renderQueue(downloads);
            }
        } catch (err) {
            console.error('Error polling queue:', err);
        }
    }

    // Render Queue Items and Update Live Stats
    function renderQueue(downloads) {
        if (!downloads || downloads.length === 0) {
            emptyQueue.classList.remove('hidden');
            queueList.classList.add('hidden');
            activeCount.textContent = '0 active';
            return;
        }

        emptyQueue.classList.add('hidden');
        queueList.classList.remove('hidden');

        let activeCountVal = 0;

        downloads.forEach(task => {
            if (task.status === 'downloading' || task.status === 'queued' || task.status === 'decompressing') {
                activeCountVal++;
            }

            const existingCard = document.getElementById(`task-${task.id}`);
            const progressPct = task.progress ? task.progress.toFixed(1) : '0.0';
            const sizeStr = formatBytes(task.totalSize);
            const speedStr = task.status === 'downloading' ? `${formatBytes(task.speed)}/s` : '';
            const transStr = formatBytes(task.bytesTransferred);

            let statusLabel = task.status;
            let statusClass = `status-${task.status}`;

            if (task.status === 'downloading') {
                statusLabel = `${progressPct}%`;
            }

            if (existingCard) {
                // Task Card already exists - update it live
                const progressFill = existingCard.querySelector('.progress-bar-fill');
                const progressText = existingCard.querySelector('.progress-bar-fill-text');
                const badge = existingCard.querySelector('.task-status-badge');
                const metricLeft = existingCard.querySelector('.metric-left');

                // Update class depending on state changes
                existingCard.className = `download-task-card task-card-${task.status}`;
                badge.className = `task-status-badge ${statusClass}`;
                badge.textContent = statusLabel;

                progressFill.style.width = `${task.progress || 0}%`;
                
                if (task.status === 'downloading') {
                    metricLeft.innerHTML = `
                        <span>Transferred: <span class="metric-highlight">${transStr} / ${sizeStr}</span></span>
                        <span>Speed: <span class="metric-highlight">${speedStr}</span></span>
                    `;
                } else if (task.status === 'decompressing') {
                    metricLeft.innerHTML = `
                        <span class="metric-highlight">Decompressing archive...</span>
                    `;
                } else if (task.status === 'completed') {
                    metricLeft.innerHTML = `
                        <span class="metric-highlight">Saved as: ${escapeHtml(task.filename)}</span>
                    `;
                } else if (task.status === 'failed') {
                    metricLeft.innerHTML = `
                        <span class="metric-error" title="${escapeHtml(task.error)}">Error: ${escapeHtml(task.error)}</span>
                    `;
                } else {
                    metricLeft.innerHTML = `<span>Queued and awaiting slot...</span>`;
                }
            } else {
                // Create a new task card
                const card = document.createElement('div');
                card.id = `task-${task.id}`;
                card.className = `download-task-card task-card-${task.status}`;
                
                let metricHTML = '';
                if (task.status === 'downloading') {
                    metricHTML = `
                        <span>Transferred: <span class="metric-highlight">${transStr} / ${sizeStr}</span></span>
                        <span>Speed: <span class="metric-highlight">${speedStr}</span></span>
                    `;
                } else if (task.status === 'decompressing') {
                    metricHTML = `
                        <span class="metric-highlight">Decompressing archive...</span>
                    `;
                } else if (task.status === 'completed') {
                    metricHTML = `
                        <span class="metric-highlight">Saved as: ${escapeHtml(task.filename)}</span>
                    `;
                } else if (task.status === 'failed') {
                    metricHTML = `
                        <span class="metric-error" title="${escapeHtml(task.error)}">Error: ${escapeHtml(task.error)}</span>
                    `;
                } else {
                    metricHTML = `<span>Queued and awaiting slot...</span>`;
                }

                card.innerHTML = `
                    <div class="task-header">
                        <div>
                            <div class="task-title">${escapeHtml(task.name)}</div>
                            <div class="task-console">${escapeHtml(task.console)}</div>
                        </div>
                        <span class="task-status-badge ${statusClass}">${statusLabel}</span>
                    </div>
                    <div class="progress-container">
                        <div class="progress-bar-bg">
                            <div class="progress-bar-fill" style="width: ${task.progress || 0}%"></div>
                        </div>
                        <div class="progress-metrics">
                            <div class="metric-left">${metricHTML}</div>
                            <div>${sizeStr}</div>
                        </div>
                    </div>
                `;

                // Add to the top of the queue list
                queueList.insertBefore(card, queueList.firstChild);
            }
        });

        activeCount.textContent = `${activeCountVal} active`;
    }

    // Helper functions
    function formatBytes(bytes, decimals = 2) {
        if (!bytes || bytes === 0) return '0 Bytes';
        const k = 1024;
        const dm = decimals < 0 ? 0 : decimals;
        const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + ' ' + sizes[i];
    }

    function escapeHtml(str) {
        if (!str) return '';
        return str
            .replace(/&/g, "&amp;")
            .replace(/</g, "&lt;")
            .replace(/>/g, "&gt;")
            .replace(/"/g, "&quot;")
            .replace(/'/g, "&#039;");
    }

    // Super modern Custom Notification System (Toasts)
    function showToast(message, type = 'success') {
        const toast = document.createElement('div');
        toast.style.position = 'fixed';
        toast.style.bottom = '2rem';
        toast.style.right = '2rem';
        toast.style.padding = '1rem 1.5rem';
        toast.style.borderRadius = '12px';
        toast.style.zIndex = '9999';
        toast.style.color = '#fff';
        toast.style.boxShadow = '0 10px 25px rgba(0,0,0,0.3)';
        toast.style.backdropFilter = 'blur(16px)';
        toast.style.transition = 'all 0.3s cubic-bezier(0.4, 0, 0.2, 1)';
        toast.style.opacity = '0';
        toast.style.transform = 'translateY(20px)';
        toast.style.border = '1px solid rgba(255,255,255,0.08)';
        toast.style.fontFamily = 'var(--font-body)';
        toast.style.fontSize = '0.9rem';
        toast.style.fontWeight = '500';

        if (type === 'success') {
            toast.style.background = 'rgba(16, 185, 129, 0.85)';
            toast.style.borderColor = 'rgba(16, 185, 129, 0.3)';
        } else if (type === 'danger') {
            toast.style.background = 'rgba(239, 68, 68, 0.85)';
            toast.style.borderColor = 'rgba(239, 68, 68, 0.3)';
        }

        toast.textContent = message;
        document.body.appendChild(toast);

        // Animation in
        setTimeout(() => {
            toast.style.opacity = '1';
            toast.style.transform = 'translateY(0)';
        }, 50);

        // Clean up
        setTimeout(() => {
            toast.style.opacity = '0';
            toast.style.transform = 'translateY(20px)';
            setTimeout(() => {
                toast.remove();
            }, 300);
        }, 4000);
    }

    // Handle "Organize Loose ROMs" Button Click
    if (organizeBtn) {
        organizeBtn.addEventListener('click', async () => {
            organizeBtn.disabled = true;
            organizeBtn.textContent = 'Organizing...';
            try {
                const res = await fetch('/api/organize', { method: 'POST' });
                const data = await res.json();
                if (res.ok) {
                    showToast(data.message || 'Loose files organized successfully!', 'success');
                } else {
                    showToast(data.error || 'Failed to organize files', 'danger');
                }
            } catch (err) {
                console.error('Error organizing files:', err);
                showToast('Network error while organizing files', 'danger');
            } finally {
                organizeBtn.disabled = false;
                organizeBtn.textContent = 'Organize Loose ROMs';
            }
        });
    }
});
