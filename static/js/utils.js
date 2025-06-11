/**
 * Utility functions for the Airfocus API Tools application
 */

// Configuration
const CONFIG = {
    API_BASE_URL: '/api',
    MESSAGE_TIMEOUT: 5000,
    MIN_WORKSPACE_NAME_LENGTH: 2,
    MIN_FIELD_NAME_LENGTH: 2
};

/**
 * Displays a message in the message area with the specified type
 * @param {string} message - The message to display
 * @param {string} type - The type of message ('info', 'error', or 'success')
 */
function displayMessage(message, type = 'info') {
    const messageArea = document.getElementById('messageArea');
    if (!messageArea) return;
    
    messageArea.textContent = message;
    messageArea.classList.remove('hidden', 'bg-red-100', 'text-red-800', 'bg-green-100', 'text-green-800', 'bg-blue-100', 'text-blue-800');
    
    if (type === 'error') {
        messageArea.classList.add('bg-red-100', 'text-red-800');
    } else if (type === 'success') {
        messageArea.classList.add('bg-green-100', 'text-green-800');
    } else { // info
        messageArea.classList.add('bg-blue-100', 'text-blue-800');
    }
    
    // Hide after timeout
    setTimeout(() => {
        messageArea.classList.add('hidden');
    }, CONFIG.MESSAGE_TIMEOUT);
}

/**
 * Sets the disabled state of a button
 * @param {string} buttonId - The ID of the button
 * @param {boolean} disabled - Whether the button should be disabled
 */
function setButtonState(buttonId, disabled) {
    const button = document.getElementById(buttonId);
    if (!button) return;
    
    if (disabled) {
        button.disabled = true;
        button.classList.add('opacity-50', 'cursor-not-allowed');
    } else {
        button.disabled = false;
        button.classList.remove('opacity-50', 'cursor-not-allowed');
    }
}

/**
 * Validates API key
 * @param {string} apiKey - The API key to validate
 * @returns {boolean} - Whether the API key is valid
 */
function validateAPIKey(apiKey) {
    return apiKey && apiKey.trim().length > 0;
}

/**
 * Validates workspace name
 * @param {string} name - The workspace name to validate
 * @returns {boolean} - Whether the workspace name is valid
 */
function validateWorkspaceName(name) {
    return name && name.trim().length >= CONFIG.MIN_WORKSPACE_NAME_LENGTH;
}

/**
 * Validates field name
 * @param {string} name - The field name to validate
 * @returns {boolean} - Whether the field name is valid
 */
function validateFieldName(name) {
    return name && name.trim().length >= CONFIG.MIN_FIELD_NAME_LENGTH;
}

/**
 * Gets the current API key from the input field
 * @returns {string} - The API key or empty string
 */
function getAPIKey() {
    const apiKeyInput = document.getElementById('apiKey');
    return apiKeyInput ? apiKeyInput.value.trim() : '';
}

/**
 * Creates URLSearchParams for API requests
 * @param {Object} params - Parameters to include
 * @returns {URLSearchParams} - Formatted parameters
 */
function createFormData(params) {
    const formData = new URLSearchParams();
    formData.append('api_key', getAPIKey());
    
    Object.entries(params).forEach(([key, value]) => {
        if (value !== null && value !== undefined && value !== '') {
            formData.append(key, value);
        }
    });
    
    return formData;
}

/**
 * Makes an API request with standard error handling
 * @param {string} endpoint - The API endpoint
 * @param {Object} params - Request parameters
 * @returns {Promise<Object>} - The API response
 */
async function apiRequest(endpoint, params = {}) {
    const apiKey = getAPIKey();
    if (!validateAPIKey(apiKey)) {
        throw new Error('Please enter your API key');
    }
    
    const formData = createFormData(params);
    
    const response = await fetch(`${CONFIG.API_BASE_URL}${endpoint}`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/x-www-form-urlencoded',
        },
        body: formData
    });
    
    if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
    }
    
    return await response.json();
}

/**
 * Formats a date string for display
 * @param {string} dateStr - The date string to format
 * @returns {string} - Formatted date string
 */
function formatDate(dateStr) {
    if (!dateStr) return '-';
    const date = new Date(dateStr);
    return date.toLocaleDateString('en-US', {
        year: 'numeric',
        month: 'short',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit'
    });
}

/**
 * Gets color class based on permission level
 * @param {string} permission - The permission level
 * @returns {string} - CSS class for the permission
 */
function getPermissionColor(permission) {
    switch (permission.toLowerCase()) {
        case 'full':
            return 'bg-purple-100 text-purple-800';
        case 'write':
            return 'bg-blue-100 text-blue-800';
        case 'read':
            return 'bg-green-100 text-green-800';
        case 'comment':
            return 'bg-yellow-100 text-yellow-800';
        default:
            return 'bg-gray-100 text-gray-800';
    }
}

/**
 * Sanitizes HTML to prevent XSS
 * @param {string} str - The string to sanitize
 * @returns {string} - Sanitized string
 */
function sanitizeHTML(str) {
    const div = document.createElement('div');
    div.textContent = str;
    return div.innerHTML;
}

/**
 * Shows or hides a result div with appropriate styling
 * @param {string} divId - The ID of the div to update
 * @param {boolean} isSuccess - Whether the operation was successful
 * @param {string} content - The content to display
 */
function updateResultDiv(divId, isSuccess, content) {
    const div = document.getElementById(divId);
    if (!div) return;
    
    div.classList.remove('hidden');
    div.innerHTML = content;
    
    if (isSuccess) {
        div.classList.remove('bg-red-50');
        div.classList.add('bg-green-50');
    } else {
        div.classList.remove('bg-green-50');
        div.classList.add('bg-red-50');
    }
}

// Export functions for use in other modules
window.Utils = {
    displayMessage,
    setButtonState,
    validateAPIKey,
    validateWorkspaceName,
    validateFieldName,
    getAPIKey,
    createFormData,
    apiRequest,
    formatDate,
    getPermissionColor,
    sanitizeHTML,
    updateResultDiv,
    CONFIG
}; 