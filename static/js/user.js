/**
 * User management functions for the Airfocus API Tools application
 */

/**
 * Retrieves and displays all users with their roles
 * Updates the user select dropdown with the fetched users
 */
async function getUsersWithRoles() {
    const { apiRequest, setButtonState, displayMessage } = window.Utils;
    
    const resultDiv = document.getElementById('usersWithRolesResult');
    const userSelect = document.getElementById('userSelect');
    resultDiv.classList.remove('hidden');
    setButtonState('getUsersWithRolesBtn', true);

    try {
        const data = await apiRequest('/users/roles');
        
        if (data.status === 'success') {
            if (data.data && data.data.length > 0) {
                // Clear and populate the user select dropdown
                userSelect.innerHTML = '<option value="">Select a user to view their workspaces...</option>';
                data.data.forEach(user => {
                    const option = document.createElement('option');
                    option.value = user.userId;
                    option.textContent = `${user.fullName} (${user.role})`;
                    userSelect.appendChild(option);
                });
                displayMessage('Users loaded successfully.', 'success');
                resultDiv.classList.remove('bg-red-50');
                resultDiv.classList.add('bg-green-50');
            } else {
                userSelect.innerHTML = '<option value="">No users found</option>';
                displayMessage('No users found.', 'info');
                resultDiv.classList.remove('bg-red-50');
                resultDiv.classList.add('bg-green-50');
            }
        } else {
            userSelect.innerHTML = '<option value="">Error loading users</option>';
            resultDiv.classList.remove('bg-green-50');
            resultDiv.classList.add('bg-red-50');
            displayMessage('Failed to get users: ' + data.error, 'error');
        }
    } catch (error) {
        userSelect.innerHTML = '<option value="">Error loading users</option>';
        resultDiv.classList.remove('bg-green-50');
        resultDiv.classList.add('bg-red-50');
        displayMessage('Error fetching users: ' + error.message, 'error');
    } finally {
        setButtonState('getUsersWithRolesBtn', false);
    }
}

/**
 * Retrieves and displays all workspaces a selected user has access to
 * Shows the workspaces in a formatted list with permission levels
 */
async function getUserWorkspaces() {
    const { apiRequest, setButtonState, displayMessage, sanitizeHTML } = window.Utils;
    
    const userSelect = document.getElementById('userSelect');
    const selectedUserId = userSelect.value;
    if (!selectedUserId) {
        displayMessage('Please select a user first', 'error');
        return;
    }

    const resultDiv = document.getElementById('userWorkspacesResult');
    resultDiv.classList.remove('hidden');
    setButtonState('getUserWorkspacesBtn', true);

    try {
        const data = await apiRequest('/user/workspaces', {
            user_id: selectedUserId
        });
        
        if (data.status === 'success') {
            if (data.data && data.data.length > 0) {
                // Group workspaces by their group path
                const groupedWorkspaces = {};
                data.data.forEach(access => {
                    const groupPath = access.groupPath || 'Ungrouped Workspaces';
                    if (!groupedWorkspaces[groupPath]) {
                        groupedWorkspaces[groupPath] = [];
                    }
                    groupedWorkspaces[groupPath].push(access);
                });

                // Sort group paths alphabetically
                const sortedGroupPaths = Object.keys(groupedWorkspaces).sort((a, b) => {
                    if (a === 'Ungrouped Workspaces') return 1;
                    if (b === 'Ungrouped Workspaces') return -1;
                    return a.localeCompare(b);
                });

                let workspaceListHtml = '';
                sortedGroupPaths.forEach(groupPath => {
                    const workspaces = groupedWorkspaces[groupPath];
                    // Sort workspaces within each group by name
                    workspaces.sort((a, b) => a.workspaceName.localeCompare(b.workspaceName));

                    workspaceListHtml += `<div class="mb-4">`;
                    workspaceListHtml += `<h4 class="font-semibold text-gray-700 mb-2">${sanitizeHTML(groupPath)}</h4>`;
                    workspaceListHtml += `<ul class="list-disc pl-5 space-y-1">`;
                    workspaces.forEach(access => {
                        workspaceListHtml += `<li class="flex items-center gap-2">`;
                        workspaceListHtml += `<span class="font-medium">${sanitizeHTML(access.workspaceName)}</span>`;
                        workspaceListHtml += `<span class="text-sm px-2 py-0.5 rounded-full 
                            ${access.permission === 'full' ? 'bg-green-100 text-green-800' : 
                              access.permission === 'write' ? 'bg-blue-100 text-blue-800' : 
                              access.permission === 'read' ? 'bg-yellow-100 text-yellow-800' : 
                              'bg-gray-100 text-gray-800'}">
                            ${access.permission}
                        </span>`;
                        workspaceListHtml += `</li>`;
                    });
                    workspaceListHtml += `</ul></div>`;
                });

                resultDiv.innerHTML = workspaceListHtml;
                displayMessage('User workspaces retrieved successfully.', 'success');
                resultDiv.classList.remove('bg-red-50');
                resultDiv.classList.add('bg-green-50');
            } else {
                resultDiv.innerHTML = '<div class="text-center text-gray-500">No workspaces found for this user</div>';
                displayMessage('No workspaces found for this user.', 'info');
                resultDiv.classList.remove('bg-red-50');
                resultDiv.classList.add('bg-green-50');
            }
        } else {
            resultDiv.innerHTML = '<div class="text-center text-red-500">Error loading user workspaces</div>';
            resultDiv.classList.remove('bg-green-50');
            resultDiv.classList.add('bg-red-50');
            displayMessage('Failed to get user workspaces: ' + data.error, 'error');
        }
    } catch (error) {
        resultDiv.innerHTML = '<div class="text-center text-red-500">Error loading user workspaces</div>';
        resultDiv.classList.remove('bg-green-50');
        resultDiv.classList.add('bg-red-50');
        displayMessage('Error fetching user workspaces: ' + error.message, 'error');
    } finally {
        setButtonState('getUserWorkspacesBtn', false);
    }
}

/**
 * Retrieves and displays all contributors in a table
 */
async function getContributors() {
    const { apiRequest, setButtonState, displayMessage, formatDate, sanitizeHTML } = window.Utils;
    
    const resultDiv = document.getElementById('contributorsResult');
    const tableBody = document.getElementById('contributorsTableBody');
    resultDiv.classList.remove('hidden');
    setButtonState('getContributorsBtn', true);

    try {
        const data = await apiRequest('/contributors');
        
        if (data.status === 'success') {
            if (data.data && data.data.length > 0) {
                // Generate table rows
                const tableRows = data.data.map(user => `
                    <tr>
                        <td class="px-6 py-4 whitespace-nowrap">
                            <div class="text-sm font-medium text-gray-900">${sanitizeHTML(user.fullName)}</div>
                        </td>
                        <td class="px-6 py-4 whitespace-nowrap">
                            <div class="text-sm text-gray-500">${formatDate(user.createdAt)}</div>
                        </td>
                        <td class="px-6 py-4 whitespace-nowrap">
                            <div class="text-sm text-gray-500">${formatDate(user.updatedAt)}</div>
                        </td>
                    </tr>
                `).join('');

                tableBody.innerHTML = tableRows;
                
                // Apply the current sort
                if (window.TableManager) {
                    window.TableManager.sortTable(window.TableManager.currentSort.column);
                }
                
                displayMessage('Contributors loaded successfully.', 'success');
                resultDiv.classList.remove('bg-red-50');
                resultDiv.classList.add('bg-green-50');
            } else {
                tableBody.innerHTML = '<tr><td colspan="3" class="px-6 py-4 text-center text-gray-500">No contributors found</td></tr>';
                displayMessage('No contributors found.', 'info');
                resultDiv.classList.remove('bg-red-50');
                resultDiv.classList.add('bg-green-50');
            }
        } else {
            tableBody.innerHTML = '<tr><td colspan="3" class="px-6 py-4 text-center text-red-500">Error loading contributors</td></tr>';
            resultDiv.classList.remove('bg-green-50');
            resultDiv.classList.add('bg-red-50');
            displayMessage('Failed to get contributors: ' + data.error, 'error');
        }
    } catch (error) {
        tableBody.innerHTML = '<tr><td colspan="3" class="px-6 py-4 text-center text-red-500">Error loading contributors</td></tr>';
        resultDiv.classList.remove('bg-green-50');
        resultDiv.classList.add('bg-red-50');
        displayMessage('Error fetching contributors: ' + error.message, 'error');
    } finally {
        setButtonState('getContributorsBtn', false);
    }
}

// Export functions for use in other modules
window.UserManager = {
    getUsersWithRoles,
    getUserWorkspaces,
    getContributors
}; 