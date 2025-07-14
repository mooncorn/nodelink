const ws = new WebSocket('wss://localhost:8443');
const nodeSelect = document.getElementById('nodeSelect');
const commandInput = document.getElementById('commandInput');
const output = document.getElementById('output');
const executeBtn = document.getElementById('executeBtn');
const connectionStatus = document.getElementById('connectionStatus');

let isConnected = false;

// WebSocket connection handlers
ws.onopen = () => {
    console.log('Connected to server');
    isConnected = true;
    updateConnectionStatus(true);
    output.textContent = 'Connected to server. Waiting for nodes...\n';
};

ws.onclose = () => {
    console.log('Disconnected from server');
    isConnected = false;
    updateConnectionStatus(false);
    output.textContent += 'Disconnected from server.\n';
    executeBtn.disabled = true;
};

ws.onerror = (error) => {
    console.error('WebSocket error:', error);
    output.textContent += 'Connection error. Please check if the server is running.\n';
};

ws.onmessage = (event) => {
    const data = JSON.parse(event.data);

    if (data.type === 'node_list') {
        updateNodeList(data.nodes);
    }

    if (data.type === 'command_output') {
        output.textContent += data.output;
        output.scrollTop = output.scrollHeight;
    }

    if (data.type === 'command_finished') {
        output.textContent += `\n--- Command finished with exit code ${data.exitCode} ---\n\n`;
        output.scrollTop = output.scrollHeight;
        executeBtn.disabled = false;
        executeBtn.textContent = 'Execute Command';
    }
};

// Update connection status indicator
function updateConnectionStatus(connected) {
    if (connected) {
        connectionStatus.textContent = 'Connected';
        connectionStatus.className = 'status connected';
    } else {
        connectionStatus.textContent = 'Disconnected';
        connectionStatus.className = 'status disconnected';
    }
}

// Update the node dropdown list
function updateNodeList(nodes) {
    nodeSelect.innerHTML = '';
    
    if (nodes.length === 0) {
        const option = document.createElement('option');
        option.value = '';
        option.textContent = 'No nodes available';
        nodeSelect.appendChild(option);
        executeBtn.disabled = true;
    } else {
        nodes.forEach(node => {
            const option = document.createElement('option');
            option.value = node.id;
            option.textContent = `${node.id} (${node.status})`;
            nodeSelect.appendChild(option);
        });
        executeBtn.disabled = false;
    }
}

// Execute command on selected node
function executeCommand() {
    const nodeId = nodeSelect.value;
    const command = commandInput.value.trim();
    
    if (!isConnected) {
        output.textContent += 'Not connected to server.\n';
        return;
    }
    
    if (!nodeId) {
        output.textContent += 'Please select a node.\n';
        return;
    }
    
    if (!command) {
        output.textContent += 'Please enter a command.\n';
        return;
    }

    const requestId = `req-${Date.now()}`;
    ws.send(JSON.stringify({ 
        type: 'execute_command', 
        requestId, 
        nodeId, 
        command 
    }));
    
    output.textContent += `\n=== Executing "${command}" on ${nodeId} ===\n`;
    output.scrollTop = output.scrollHeight;
    commandInput.value = '';
    executeBtn.disabled = true;
    executeBtn.textContent = 'Executing...';
}

// Allow Enter key to execute command
commandInput.addEventListener('keypress', (e) => {
    if (e.key === 'Enter') {
        executeCommand();
    }
});

// Enable/disable execute button based on node selection
nodeSelect.addEventListener('change', () => {
    executeBtn.disabled = !nodeSelect.value || !isConnected;
}); 