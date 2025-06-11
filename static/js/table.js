/**
 * Table management functions for the Airfocus API Tools application
 */

// Current sort state
let currentSort = {
    column: 'name',
    direction: 'asc'
};

/**
 * Updates sort icons to reflect current sort state
 * @param {string} column - The column being sorted
 */
function updateSortIcons(column) {
    // Reset all icons
    document.getElementById('nameSortIcon').textContent = '↕';
    document.getElementById('dateJoinedSortIcon').textContent = '↕';
    document.getElementById('lastUpdatedSortIcon').textContent = '↕';

    // Set the active sort icon
    const icon = document.getElementById(column + 'SortIcon');
    if (icon) {
        icon.textContent = currentSort.direction === 'asc' ? '↑' : '↓';
    }
}

/**
 * Sorts a table by the specified column
 * @param {string} column - The column to sort by
 */
function sortTable(column) {
    const tableBody = document.getElementById('contributorsTableBody');
    if (!tableBody) return;
    
    const rows = Array.from(tableBody.getElementsByTagName('tr'));

    // Toggle sort direction if clicking the same column
    if (currentSort.column === column) {
        currentSort.direction = currentSort.direction === 'asc' ? 'desc' : 'asc';
    } else {
        currentSort.column = column;
        currentSort.direction = 'asc';
    }

    // Update sort icons
    updateSortIcons(column);

    // Sort the rows
    rows.sort((a, b) => {
        let aValue, bValue;

        switch (column) {
            case 'name':
                aValue = a.cells[0].textContent.trim();
                bValue = b.cells[0].textContent.trim();
                break;
            case 'dateJoined':
                aValue = new Date(a.cells[1].textContent.trim());
                bValue = new Date(b.cells[1].textContent.trim());
                break;
            case 'lastUpdated':
                aValue = new Date(a.cells[2].textContent.trim());
                bValue = new Date(b.cells[2].textContent.trim());
                break;
            default:
                aValue = a.cells[0].textContent.trim();
                bValue = b.cells[0].textContent.trim();
        }

        if (currentSort.direction === 'asc') {
            return aValue > bValue ? 1 : -1;
        } else {
            return aValue < bValue ? 1 : -1;
        }
    });

    // Reorder the rows in the table
    rows.forEach(row => tableBody.appendChild(row));
}

// Export functions for use in other modules
window.TableManager = {
    currentSort,
    updateSortIcons,
    sortTable
}; 