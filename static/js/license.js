/**
 * License management functions for the Airfocus API Tools application
 */

/**
 * Retrieves and displays team license information
 * Updates the license statistics and role counts in the UI
 */
async function getLicenseInfo() {
    const { apiRequest, setButtonState, displayMessage } = window.Utils;
    
    setButtonState('getLicenseInfoBtn', true);

    try {
        // Get license information
        const licenseData = await apiRequest('/team/license');
        
        if (licenseData.status === 'success') {
            const seats = licenseData.data.state.seats.any;
            const total = seats.total;
            const used = seats.used;
            const free = seats.free;

            // Update the license cards
            document.getElementById('totalLicenses').textContent = total;
            document.getElementById('usedLicenses').textContent = used;
            document.getElementById('freeLicenses').textContent = free;
        } else {
            // Reset license cards on error
            document.getElementById('totalLicenses').textContent = '-';
            document.getElementById('usedLicenses').textContent = '-';
            document.getElementById('freeLicenses').textContent = '-';
            displayMessage('Failed to get license information: ' + (licenseData.error || 'Unknown error'), 'error');
            return;
        }

        // Get role statistics
        const rolesData = await apiRequest('/users/roles');
        
        if (rolesData.status === 'success') {
            if (rolesData.data && rolesData.data.length > 0) {
                // Count users by role
                const stats = {
                    total: rolesData.data.length,
                    admin: 0,
                    editor: 0,
                    contributor: 0
                };

                rolesData.data.forEach(user => {
                    const role = user.role.toLowerCase();
                    if (role === 'admin') stats.admin++;
                    else if (role === 'editor') stats.editor++;
                    else if (role === 'contributor') stats.contributor++;
                });

                // Update the statistics cards
                document.getElementById('totalRoleUsers').textContent = stats.total;
                document.getElementById('totalAdmins').textContent = stats.admin;
                document.getElementById('totalEditors').textContent = stats.editor;
                document.getElementById('totalContributors').textContent = stats.contributor;
            } else {
                // Reset statistics cards
                document.getElementById('totalRoleUsers').textContent = '0';
                document.getElementById('totalAdmins').textContent = '0';
                document.getElementById('totalEditors').textContent = '0';
                document.getElementById('totalContributors').textContent = '0';
            }
        } else {
            // Reset statistics cards on error
            document.getElementById('totalRoleUsers').textContent = '-';
            document.getElementById('totalAdmins').textContent = '-';
            document.getElementById('totalEditors').textContent = '-';
            document.getElementById('totalContributors').textContent = '-';
            displayMessage('Failed to get role statistics: ' + rolesData.error, 'error');
        }

        displayMessage('Information updated successfully.', 'success');
    } catch (error) {
        // Reset statistics cards on error
        document.getElementById('totalRoleUsers').textContent = '-';
        document.getElementById('totalAdmins').textContent = '-';
        document.getElementById('totalEditors').textContent = '-';
        document.getElementById('totalContributors').textContent = '-';
        displayMessage('Error fetching information: ' + error.message, 'error');
    } finally {
        setButtonState('getLicenseInfoBtn', false);
    }
}

// Export functions for use in other modules
window.LicenseManager = {
    getLicenseInfo
}; 