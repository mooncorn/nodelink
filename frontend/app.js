class NodeLinkClient {
    constructor() {
        this.socket = null;
        this.nodes = [];
        this.tasks = [];
        this.connected = false;
        
        this.initializeSocket();
        this.setupEventHandlers();
        this.refreshData();
    }
    
    initializeSocket() {
        this.socket = io('https://localhost:8443', {
            rejectUnauthorized: false
        });
        
        this.socket.on('connect', () => {
            this.connected = true;
            this.updateConnectionStatus();
            this.log('Connected to server');
            this.refreshData();
        });
        
        this.socket.on('disconnect', () => {
            this.connected = false;
            this.updateConnectionStatus();
            this.log('Disconnected from server');
        });
        
        this.socket.on('connect_error', (error) => {
            this.log(`Connection error: ${error.message}`);
        });
        
        // Real-time event handlers
        this.socket.on('task.created', (data) => {
            this.log(`Task created: ${data.task.id} (${data.task.type})`);
            this.refreshTasks();
        });
        
        this.socket.on('task.updated', (data) => {
            this.log(`Task updated: ${data.task.id} -> ${data.task.status}`);
            this.refreshTasks();
        });
        
        this.socket.on('task.completed', (data) => {
            this.log(`Task completed: ${data.task.id}`);
            this.refreshTasks();
        });
        
        this.socket.on('task.output', (data) => {
            this.log(`[${data.taskId}] ${data.output}`);
        });
        
        this.socket.on('node.list', (data) => {
            this.nodes = data.nodes;
            this.updateNodesDisplay();
            this.updateNodeSelects();
        });
        
        this.socket.on('error', (error) => {
            this.log(`Error: ${error.message}`);
        });
    }
    
    setupEventHandlers() {
        // Tab switching
        window.showTab = (tabName) => {
            document.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
            document.querySelectorAll('.tab-content').forEach(t => t.classList.remove('active'));
            
            document.querySelector(`[onclick="showTab('${tabName}')"]`).classList.add('active');
            document.getElementById(`${tabName}-tab`).classList.add('active');
        };
        
        // Command execution functions
        window.executeShellCommand = () => this.executeShellCommand();
        window.executeDockerAction = () => this.executeDockerAction();
        window.executeSystemAction = () => this.executeSystemAction();
        window.refreshData = () => this.refreshData();
        window.updateDockerForm = () => this.updateDockerForm();
        
        // Auto-refresh tasks
        setInterval(() => {
            if (this.connected) {
                this.refreshTasks();
            }
        }, 5000);
    }
    
    async apiCall(method, endpoint, data = null) {
        const config = {
            method,
            headers: {
                'Content-Type': 'application/json',
            },
        };
        
        if (data) {
            config.body = JSON.stringify(data);
        }
        
        try {
            const response = await fetch(`https://localhost:8443${endpoint}`, config);
            const result = await response.json();
            
            if (!response.ok) {
                throw new Error(result.error || 'Request failed');
            }
            
            return result;
        } catch (error) {
            this.log(`API Error: ${error.message}`);
            throw error;
        }
    }
    
    async refreshData() {
        try {
            await Promise.all([
                this.refreshNodes(),
                this.refreshTasks(),
                this.refreshStats()
            ]);
        } catch (error) {
            this.log(`Failed to refresh data: ${error.message}`);
        }
    }
    
    async refreshNodes() {
        try {
            const result = await this.apiCall('GET', '/api/nodes');
            this.nodes = result.data;
            this.updateNodesDisplay();
            this.updateNodeSelects();
        } catch (error) {
            this.log(`Failed to refresh nodes: ${error.message}`);
        }
    }
    
    async refreshTasks() {
        try {
            const nodeFilter = document.getElementById('taskNodeFilter')?.value || '';
            const statusFilter = document.getElementById('taskStatusFilter')?.value || '';
            
            let endpoint = '/api/tasks';
            const params = new URLSearchParams();
            
            if (nodeFilter) params.append('nodeId', nodeFilter);
            if (statusFilter) params.append('status', statusFilter);
            
            if (params.toString()) {
                endpoint += `?${params.toString()}`;
            }
            
            const result = await this.apiCall('GET', endpoint);
            this.tasks = result.data;
            this.updateTasksDisplay();
        } catch (error) {
            this.log(`Failed to refresh tasks: ${error.message}`);
        }
    }
    
    async refreshStats() {
        try {
            const result = await this.apiCall('GET', '/api/stats');
            const stats = result.data;
            
            document.getElementById('nodeCount').textContent = stats.nodes;
            document.getElementById('taskCount').textContent = stats.totalTasks;
        } catch (error) {
            this.log(`Failed to refresh stats: ${error.message}`);
        }
    }
    
    async executeShellCommand() {
        const nodeId = document.getElementById('shellNodeSelect').value;
        const command = document.getElementById('shellCommand').value;
        const cwd = document.getElementById('shellCwd').value;
        const timeout = parseInt(document.getElementById('shellTimeout').value);
        
        if (!nodeId || !command) {
            this.log('Please select a node and enter a command');
            return;
        }
        
        const payload = {
            command,
            timeout,
        };
        
        if (cwd) payload.cwd = cwd;
        
        try {
            const result = await this.apiCall('POST', '/api/tasks', {
                nodeId,
                type: 'shell.execute',
                payload
            });
            
            this.log(`Shell command task created: ${result.data.id}`);
            this.displayJson('Task Created', result.data);
        } catch (error) {
            this.log(`Failed to execute shell command: ${error.message}`);
        }
    }
    
    async executeDockerAction() {
        const nodeId = document.getElementById('dockerNodeSelect').value;
        const action = document.getElementById('dockerAction').value;
        
        if (!nodeId) {
            this.log('Please select a node');
            return;
        }
        
        let payload = {};
        
        if (action === 'docker.run') {
            const image = document.getElementById('dockerImage').value;
            const containerName = document.getElementById('dockerContainerName').value;
            const portHost = document.getElementById('dockerPortHost').value;
            const portContainer = document.getElementById('dockerPortContainer').value;
            
            if (!image) {
                this.log('Please enter a Docker image');
                return;
            }
            
            payload = { image };
            if (containerName) payload.containerName = containerName;
            
            // Add port mapping if specified
            if (portHost && portContainer) {
                payload.ports = [{
                    host: parseInt(portHost),
                    container: parseInt(portContainer),
                    protocol: 'tcp'
                }];
            }
        } else if (action === 'docker.start' || action === 'docker.stop' || action === 'docker.delete') {
            const containerId = document.getElementById('dockerContainerId').value;
            
            if (!containerId) {
                this.log('Please enter a container ID');
                return;
            }
            
            payload = { containerId };
            
            // Add additional options for delete action
            if (action === 'docker.delete') {
                payload.force = document.getElementById('dockerForce')?.checked || false;
                payload.removeVolumes = document.getElementById('dockerRemoveVolumes')?.checked || false;
            }
        } else if (action === 'docker.list') {
            payload = {
                all: document.getElementById('dockerShowAll')?.checked || false
            };
        }
        
        try {
            const result = await this.apiCall('POST', '/api/tasks', {
                nodeId,
                type: action,
                payload
            });
            
            this.log(`Docker action task created: ${result.data.id}`);
            this.displayJson('Task Created', result.data);
        } catch (error) {
            this.log(`Failed to execute Docker action: ${error.message}`);
        }
    }
    
    async executeSystemAction() {
        const nodeId = document.getElementById('systemNodeSelect').value;
        const action = document.getElementById('systemAction').value;
        
        if (!nodeId) {
            this.log('Please select a node');
            return;
        }
        
        const payload = {
            includeMetrics: document.getElementById('includeMetrics').checked,
            includeProcesses: document.getElementById('includeProcesses').checked,
            includeNetwork: document.getElementById('includeNetwork').checked,
        };
        
        if (action === 'system.health') {
            payload.checkDisk = true;
            payload.checkMemory = true;
            payload.checkCpu = true;
        }
        
        try {
            const result = await this.apiCall('POST', '/api/tasks', {
                nodeId,
                type: action,
                payload
            });
            
            this.log(`System action task created: ${result.data.id}`);
            this.displayJson('Task Created', result.data);
        } catch (error) {
            this.log(`Failed to execute system action: ${error.message}`);
        }
    }
    
    updateDockerForm() {
        const action = document.getElementById('dockerAction').value;
        const formDiv = document.getElementById('dockerForm');
        
        let html = '';
        
        if (action === 'docker.run') {
            html = `
                <div class="form-group">
                    <label>Docker Image:</label>
                    <input type="text" id="dockerImage" placeholder="nginx:latest">
                </div>
                <div class="form-group">
                    <label>Container Name (optional):</label>
                    <input type="text" id="dockerContainerName" placeholder="my-container">
                </div>
                <div class="form-group">
                    <label>Port Mapping (optional):</label>
                    <div style="display: flex; gap: 10px;">
                        <input type="number" id="dockerPortHost" placeholder="Host Port (e.g. 8080)" style="flex: 1;">
                        <span style="align-self: center;">→</span>
                        <input type="number" id="dockerPortContainer" placeholder="Container Port (e.g. 80)" style="flex: 1;">
                    </div>
                </div>
            `;
        } else if (action === 'docker.start' || action === 'docker.stop') {
            html = `
                <div class="form-group">
                    <label>Container ID:</label>
                    <input type="text" id="dockerContainerId" placeholder="container-id">
                </div>
            `;
        } else if (action === 'docker.delete') {
            html = `
                <div class="form-group">
                    <label>Container ID:</label>
                    <input type="text" id="dockerContainerId" placeholder="container-id">
                </div>
                <div class="form-group">
                    <label>
                        <input type="checkbox" id="dockerForce"> Force delete (stop running container)
                    </label>
                </div>
                <div class="form-group">
                    <label>
                        <input type="checkbox" id="dockerRemoveVolumes"> Remove associated volumes
                    </label>
                </div>
            `;
        } else if (action === 'docker.list') {
            html = `
                <div class="form-group">
                    <label>
                        <input type="checkbox" id="dockerShowAll"> Show all containers (including stopped)
                    </label>
                </div>
            `;
        }
        
        formDiv.innerHTML = html;
    }
    
    updateConnectionStatus() {
        const statusDot = document.getElementById('connectionStatus');
        const statusText = document.getElementById('connectionText');
        
        if (this.connected) {
            statusDot.classList.add('connected');
            statusText.textContent = 'Connected';
        } else {
            statusDot.classList.remove('connected');
            statusText.textContent = 'Disconnected';
        }
    }
    
    updateNodesDisplay() {
        const container = document.getElementById('nodesGrid');
        
        if (this.nodes.length === 0) {
            container.innerHTML = '<p>No nodes connected</p>';
            return;
        }
        
        const html = this.nodes.map(node => `
            <div class="node-card ${node.status}">
                <div class="node-status">
                    <span class="node-name">${node.id}</span>
                    <span class="task-status ${node.status}">${node.status}</span>
                </div>
                <div class="node-info">
                    <p><strong>Platform:</strong> ${node.systemInfo.platform}</p>
                    <p><strong>Architecture:</strong> ${node.systemInfo.arch}</p>
                    <p><strong>Hostname:</strong> ${node.systemInfo.hostname}</p>
                    <p><strong>Capabilities:</strong> ${node.capabilities.join(', ')}</p>
                    <p><strong>Running Tasks:</strong> ${node.runningTasks}</p>
                </div>
            </div>
        `).join('');
        
        container.innerHTML = html;
    }
    
    updateTasksDisplay() {
        const container = document.getElementById('tasksList');
        
        if (this.tasks.length === 0) {
            container.innerHTML = '<p>No tasks found</p>';
            return;
        }
        
        const html = this.tasks.map(task => `
            <div class="task-item">
                <div class="task-info">
                    <div class="task-id">${task.id}</div>
                    <div class="task-type">${task.type}</div>
                    <div>Node: ${task.nodeId}</div>
                    <div>Created: ${new Date(task.createdAt).toLocaleString()}</div>
                </div>
                <div class="task-status ${task.status}">${task.status}</div>
            </div>
        `).join('');
        
        container.innerHTML = html;
    }
    
    updateNodeSelects() {
        const selects = [
            'shellNodeSelect',
            'dockerNodeSelect',
            'systemNodeSelect',
            'taskNodeFilter'
        ];
        
        selects.forEach(selectId => {
            const select = document.getElementById(selectId);
            if (!select) return;
            
            // Keep current value if possible
            const currentValue = select.value;
            
            // Clear and add default option
            select.innerHTML = selectId === 'taskNodeFilter' 
                ? '<option value="">All Nodes</option>' 
                : '<option value="">Select a node...</option>';
            
            // Add node options
            this.nodes.forEach(node => {
                const option = document.createElement('option');
                option.value = node.id;
                option.textContent = `${node.id} (${node.status})`;
                select.appendChild(option);
            });
            
            // Restore value if it still exists
            if (currentValue && Array.from(select.options).some(opt => opt.value === currentValue)) {
                select.value = currentValue;
            }
        });
    }
    
    log(message) {
        const output = document.getElementById('output');
        const timestamp = new Date().toLocaleTimeString();
        output.textContent += `[${timestamp}] ${message}\n`;
        output.scrollTop = output.scrollHeight;
    }
    
    displayJson(title, data) {
        const output = document.getElementById('output');
        const timestamp = new Date().toLocaleTimeString();
        output.textContent += `[${timestamp}] ${title}:\n${JSON.stringify(data, null, 2)}\n\n`;
        output.scrollTop = output.scrollHeight;
    }
}

// Initialize the client when the page loads
document.addEventListener('DOMContentLoaded', () => {
    new NodeLinkClient();
});

// Initialize Docker form on page load
document.addEventListener('DOMContentLoaded', () => {
    setTimeout(() => {
        if (window.updateDockerForm) {
            window.updateDockerForm();
        }
    }, 100);
}); 