/**
 * Field management functions for the Airfocus API Tools application
 */

/**
 * Retrieves and populates the list of fields
 * Updates the field select dropdown with the fetched fields
 */
async function getFields() {
    const { apiRequest, setButtonState, displayMessage } = window.Utils;
    
    console.log('getFields function called');
    
    const select = document.getElementById('fieldSelect');
    if (!select) {
        console.error('fieldSelect element not found');
        return;
    }
    
    select.innerHTML = '<option value="">Loading fields...</option>';
    setButtonState('getFieldsBtn', true);

    try {
        console.log('Making API request to /fields');
        const data = await apiRequest('/fields');
        console.log('API response:', data);
        
        if (data.status === 'success') {
            // Filter out fields created on 2025-03-20 with empty updatedAt
            // This is a specific filter; remove if not generally needed.
            data.data = data.data.filter(field => {
                const shouldFilter = field.createdAt && 
                    field.createdAt.startsWith('2025-03-20') && 
                    (!field.updatedAt || field.updatedAt === '');
                return !shouldFilter;
            });

            console.log('Filtered fields:', data.data);

            select.innerHTML = '<option value="">Select a field...</option>';
            if (data.data && data.data.length > 0) {
                data.data.forEach(field => {
                    const option = document.createElement('option');
                    option.value = field.name;
                    let displayText = field.name;
                    
                    if (field.isTeamField) {
                        displayText += ' (Team Field)';
                        if (field.workspaceNames && field.workspaceNames.length > 0) {
                            displayText += ` - Used in ${field.workspaceNames.length} workspaces`;
                        }
                    } else {
                        // For non-team fields, show workspace names in parentheses
                        if (field.workspaceNames && field.workspaceNames.length > 0) {
                            displayText += ` (${field.workspaceNames.join(', ')})`;
                        }
                    }
                    
                    option.textContent = displayText;
                    select.appendChild(option);
                });
                // Ensure the dropdown container is visible
                const fieldsResultDiv = document.getElementById('fieldsResult');
                if (fieldsResultDiv) {
                    fieldsResultDiv.classList.remove('hidden');
                }
                console.log('Added', data.data.length, 'fields to dropdown');
                displayMessage('Fields loaded successfully.', 'success');
            } else {
                console.log('No fields found in response');
                displayMessage('No fields found for this API key.', 'info');
            }
        } else {
            console.error('API returned error status:', data.error);
            select.innerHTML = '<option value="">Error loading fields</option>';
            displayMessage('Failed to load fields: ' + data.error, 'error');
        }
    } catch (error) {
        console.error('Exception in getFields:', error);
        select.innerHTML = '<option value="">Error loading fields</option>';
        displayMessage('Error fetching fields: ' + error.message, 'error');
    } finally {
        setButtonState('getFieldsBtn', false);
    }
}

/**
 * Retrieves a field ID by name
 * Updates the field result display with the field details
 */
async function getFieldID() {
    const { apiRequest, setButtonState, displayMessage, validateFieldName } = window.Utils;
    
    const select = document.getElementById('fieldSelect');
    const fieldNameInput = document.getElementById('fieldName');
    const fieldName = select.value || fieldNameInput.value.trim();
    
    if (!fieldName) {
        displayMessage('Please select a field or enter a field name', 'error');
        return;
    }
    if (!validateFieldName(fieldName)) {
        displayMessage('Field name must be at least 2 characters long', 'error');
        return;
    }

    const resultDiv = document.getElementById('fieldResult');
    const resultPre = resultDiv.querySelector('pre');
    resultDiv.classList.remove('hidden');
    setButtonState('getFieldIDBtn', true);

    try {
        const data = await apiRequest('/field/id', {
            field_name: fieldName
        });
        
        let resultText = JSON.stringify(data, null, 2);
        
        if (data.status === 'success' && data.field && !data.field.isTeamField) {
            const field = data.field;
            if (field.workspaceNames && field.workspaceNames.length > 0) {
                resultText += '\n\nWorkspaces: ' + field.workspaceNames.join(', ');
            }
        }
        
        resultPre.textContent = resultText;
        
        if (data.status === 'success') {
            resultDiv.classList.remove('bg-red-50');
            resultDiv.classList.add('bg-green-50');
            displayMessage('Field ID retrieved successfully.', 'success');
        } else {
            resultDiv.classList.remove('bg-green-50');
            resultDiv.classList.add('bg-red-50');
            displayMessage('Failed to get Field ID: ' + data.error, 'error');
        }
    } catch (error) {
        resultPre.textContent = `Error: ${error.message}`;
        resultDiv.classList.remove('bg-green-50');
        resultDiv.classList.add('bg-red-50');
        displayMessage('Error fetching Field ID: ' + error.message, 'error');
    } finally {
        setButtonState('getFieldIDBtn', false);
    }
}

// Export functions for use in other modules
window.FieldManager = {
    getFields,
    getFieldID
}; 