/**
 * Main application initialization for the Airfocus API Tools application
 */

/**
 * Initialize the application when DOM is loaded
 */
function initializeApp() {
    // Restore saved API key
    const storedApiKey = localStorage.getItem('airfocus_api_key');
    if (storedApiKey) {
        const apiKeyInput = document.getElementById('apiKey');
        if (apiKeyInput) {
            apiKeyInput.value = storedApiKey;
        }
    }

    // Set up API key input event listener
    const apiKeyInput = document.getElementById('apiKey');
    if (apiKeyInput) {
        apiKeyInput.addEventListener('input', (event) => {
            localStorage.setItem('airfocus_api_key', event.target.value);
        });
    }

    // Set up workspace select change event
    const workspaceSelect = document.getElementById('workspaceSelect');
    if (workspaceSelect) {
        workspaceSelect.addEventListener('change', () => {
            document.getElementById('workspaceName').value = '';
            document.getElementById('currentWorkspaceId').value = '';
            if (workspaceSelect.value) {
                if (window.WorkspaceManager) {
                    window.WorkspaceManager.getWorkspaceID();
                    window.WorkspaceManager.getWorkspaceUsers();
                }
            }
        });
    }

    // Set up field select change event
    const fieldSelect = document.getElementById('fieldSelect');
    if (fieldSelect) {
        fieldSelect.addEventListener('change', () => {
            document.getElementById('fieldName').value = '';
        });
    }

    // Set up workspace name input event
    const workspaceNameInput = document.getElementById('workspaceName');
    if (workspaceNameInput) {
        workspaceNameInput.addEventListener('input', () => {
            document.getElementById('workspaceSelect').value = '';
            document.getElementById('currentWorkspaceId').value = '';
        });
    }

    // Set up field name input event
    const fieldNameInput = document.getElementById('fieldName');
    if (fieldNameInput) {
        fieldNameInput.addEventListener('input', () => {
            document.getElementById('fieldSelect').value = '';
        });
    }

    console.log('Airfocus API Tools application initialized');
}

// Initialize when DOM is loaded
document.addEventListener('DOMContentLoaded', initializeApp);

// Export for potential use in other modules
window.App = {
    initializeApp
}; 