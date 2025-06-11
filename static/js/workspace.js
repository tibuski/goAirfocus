/**
 * Workspace management functions for the Airfocus API Tools application
 */

/**
 * Retrieves and populates the list of workspaces
 * Updates the workspace select dropdown with the fetched workspaces
 */
async function getWorkspaces() {
    const { apiRequest, setButtonState, displayMessage } = window.Utils;
    
    const select = document.getElementById('workspaceSelect');
    if (!select) return;
    
    select.innerHTML = '<option value="">Loading workspaces...</option>';
    setButtonState('getWorkspacesBtn', true);

    try {
        const data = await apiRequest('/workspaces');
        
        if (data.status === 'success') {
            select.innerHTML = '<option value="">Select a workspace...</option>';
            if (data.data && data.data.length > 0) {
                data.data.forEach(workspace => {
                    const option = document.createElement('option');
                    option.value = workspace.id;
                    option.textContent = workspace.alias ? 
                        `${workspace.name} (${workspace.alias})` : 
                        workspace.name;
                    select.appendChild(option);
                });
                displayMessage('Workspaces loaded successfully.', 'success');
            } else {
                displayMessage('No workspaces found for this API key.', 'info');
            }
        } else {
            select.innerHTML = '<option value="">Error loading workspaces</option>';
            displayMessage('Failed to load workspaces: ' + data.error, 'error');
        }
    } catch (error) {
        select.innerHTML = '<option value="">Error loading workspaces</option>';
        displayMessage('Error fetching workspaces: ' + error.message, 'error');
    } finally {
        setButtonState('getWorkspacesBtn', false);
    }
}

/**
 * Retrieves a workspace ID by name
 * Updates the workspace ID input and triggers user list fetch if successful
 */
async function getWorkspaceID() {
    const { apiRequest, setButtonState, displayMessage, validateWorkspaceName } = window.Utils;
    
    const select = document.getElementById('workspaceSelect');
    const workspaceNameInput = document.getElementById('workspaceName');
    let workspaceIdentifier;

    if (select.value) {
        // Use the text content (name) for the backend endpoint that expects name
        workspaceIdentifier = select.options[select.selectedIndex].textContent.split(' (')[0]; 
    } else if (workspaceNameInput.value.trim()) {
        workspaceIdentifier = workspaceNameInput.value.trim();
    } else {
        displayMessage('Please select a workspace or enter a workspace name', 'error');
        return;
    }
    
    if (!validateWorkspaceName(workspaceIdentifier)) {
        displayMessage('Workspace name must be at least 2 characters long', 'error');
        return;
    }

    const resultDiv = document.getElementById('workspaceResult');
    const resultPre = resultDiv.querySelector('pre');
    resultDiv.classList.remove('hidden');
    setButtonState('getWorkspaceIDBtn', true);

    try {
        const data = await apiRequest('/workspace/id', {
            workspace_name: workspaceIdentifier
        });
        
        resultPre.textContent = JSON.stringify(data, null, 2);
        
        if (data.status === 'success') {
            document.getElementById('currentWorkspaceId').value = data.id;
            resultDiv.classList.remove('bg-red-50');
            resultDiv.classList.add('bg-green-50');
            displayMessage('Workspace ID retrieved successfully.', 'success');
        } else {
            document.getElementById('currentWorkspaceId').value = '';
            resultDiv.classList.remove('bg-green-50');
            resultDiv.classList.add('bg-red-50');
            displayMessage('Failed to get Workspace ID: ' + data.error, 'error');
        }
    } catch (error) {
        document.getElementById('currentWorkspaceId').value = '';
        resultPre.textContent = `Error: ${error.message}`;
        resultDiv.classList.remove('bg-green-50');
        resultDiv.classList.add('bg-red-50');
        displayMessage('Error fetching Workspace ID: ' + error.message, 'error');
    } finally {
        setButtonState('getWorkspaceIDBtn', false);
    }
}

/**
 * Retrieves and displays users for the current workspace
 * Groups users by their permission level and updates the UI
 */
async function getWorkspaceUsers() {
    const { apiRequest, setButtonState, displayMessage, getPermissionColor, sanitizeHTML } = window.Utils;
    
    const workspaceSelect = document.getElementById('workspaceSelect');
    const workspaceNameInput = document.getElementById('workspaceName');
    const currentWorkspaceId = document.getElementById('currentWorkspaceId').value;

    let params = {};
    
    // Determine which identifier to send to the backend
    if (currentWorkspaceId) {
        params.workspace_id = currentWorkspaceId;
    } else if (workspaceSelect.value) {
        params.workspace_id = workspaceSelect.value;
    } else if (workspaceNameInput.value.trim()) {
        params.workspace_name = workspaceNameInput.value.trim();
    } else {
        displayMessage('Please select a workspace or enter a workspace name first.', 'error');
        return;
    }

    const resultDiv = document.getElementById('usersResult');
    const treeViewDiv = document.getElementById('usersTreeView');
    resultDiv.classList.remove('hidden');
    setButtonState('getWorkspaceUsersBtn', true);

    try {
        const data = await apiRequest('/workspace/users', params);
        
        if (data.status === 'success') {
            // Group users by permission level
            const usersByPermission = {
                'full': [],
                'write': [],
                'comment': [],
                'read': []
            };

            // Sort users into their respective permission groups
            if (data.data.users && data.data.users.length > 0) {
                data.data.users.forEach(user => {
                    const permission = user.permission.toLowerCase();
                    if (usersByPermission[permission]) {
                        usersByPermission[permission].push(user);
                    }
                });
            }

            // Generate tree view HTML
            let treeViewHtml = '';
            const permissionOrder = ['full', 'write', 'comment', 'read'];
            
            permissionOrder.forEach(permission => {
                const users = usersByPermission[permission];
                if (users && users.length > 0) {
                    const permissionDisplay = permission.charAt(0).toUpperCase() + permission.slice(1);
                    const displayPermissionName = (permission === 'read') ? 'Read-Only' : permissionDisplay;
                    treeViewHtml += `
                        <div class="mb-4">
                            <h4 class="font-semibold text-gray-700 mb-2">${displayPermissionName} (${users.length})</h4>
                            <ul class="list-disc pl-5 space-y-1">
                                ${users.sort((a, b) => a.fullName.localeCompare(b.fullName)).map(user => `
                                    <li class="flex items-center gap-2">
                                        <span class="font-medium">${sanitizeHTML(user.fullName)}</span>
                                        ${user.email ? `<span class="text-sm text-gray-500">(${sanitizeHTML(user.email)})</span>` : ''}
                                    </li>
                                `).join('')}
                            </ul>
                        </div>
                    `;
                }
            });

            treeViewDiv.innerHTML = treeViewHtml || '<p class="text-gray-500 text-center py-4">No users found in this workspace.</p>';
            displayMessage('User statistics retrieved successfully.', 'success');
            resultDiv.classList.remove('bg-red-50');
            resultDiv.classList.add('bg-green-50');
        } else {
            treeViewDiv.innerHTML = '';
            resultDiv.classList.remove('bg-green-50');
            resultDiv.classList.add('bg-red-50');
            displayMessage('Failed to get user statistics: ' + data.error, 'error');
        }
    } catch (error) {
        treeViewDiv.innerHTML = '';
        resultDiv.classList.remove('bg-green-50');
        resultDiv.classList.add('bg-red-50');
        displayMessage('Error fetching user statistics: ' + error.message, 'error');
    } finally {
        setButtonState('getWorkspaceUsersBtn', false);
    }
}

// Export functions for use in other modules
window.WorkspaceManager = {
    getWorkspaces,
    getWorkspaceID,
    getWorkspaceUsers
}; 