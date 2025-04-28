// Backend API URL
const API_URL = 'http://localhost:3000/api';

// User management functions
async function loadUsers() {
    try {
        const token = localStorage.getItem('token');
        if (!token) {
            throw new Error('Not authenticated');
        }
        const response = await fetch(`${API_URL}/users`, {
            headers: {
                'Authorization': `Bearer ${token}`,
                'Content-Type': 'application/json'
            }
        });

        if (!response.ok) {
            if (response.status === 401) {
                throw new Error('Not authenticated');
            }
            throw new Error('Failed to load users');
        }

        const users = await response.json();
        displayUsers(users);
    } catch (error) {
        showError('Error loading users: ' + error.message);
        if (error.message === 'Not authenticated') {
            window.location.href = '../index.html';
        }
    }
}

function displayUsers(users) {
    const container = document.getElementById('usersContainer');
    if (!users.length) {
        container.innerHTML = '<p class="text-gray-500">No users found.</p>';
        return;
    }

    container.innerHTML = users.map(user => `
        <div class="p-4 bg-white rounded-lg shadow-sm hover:shadow-md transition-shadow">
            <div class="flex justify-between items-start mb-2">
                <div>
                    <h3 class="font-medium">${user.username}</h3>
                    <p class="text-sm text-gray-600">${user.email}</p>
                </div>
                <span class="px-2 py-1 text-sm rounded ${getUserTypeClass(user.userType)}">
                    ${user.userType}
                </span>
            </div>
            <div class="flex gap-2 mt-4">
                <button onclick="editUser('${user._id}')" 
                        class="px-3 py-1 text-sm bg-blue-500 text-white rounded hover:bg-blue-600">
                    Edit
                </button>
                <button onclick="deleteUser('${user._id}')"
                        class="px-3 py-1 text-sm bg-red-500 text-white rounded hover:bg-red-600">
                    Delete
                </button>
            </div>
        </div>
    `).join('');
}

function getUserTypeClass(userType) {
    const classes = {
        admin: 'bg-purple-100 text-purple-800',
        faculty: 'bg-blue-100 text-blue-800',
        student: 'bg-green-100 text-green-800'
    };
    return classes[userType] || 'bg-gray-100 text-gray-800';
}

async function createUser(userData) {
    try {
        const token = localStorage.getItem('token');
        if (!token) {
            throw new Error('Not authenticated');
        }
        const response = await fetch(`${API_URL}/users`, {
            method: 'POST',
            headers: {
                'Authorization': `Bearer ${token}`,
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(userData)
        });

        if (!response.ok) {
            if (response.status === 401) {
                throw new Error('Not authenticated');
            }
            const data = await response.json();
            throw new Error(data.error || 'Failed to create user');
        }

        showSuccess('User created successfully');
        loadUsers();
        hideModal('createUserModal');
    } catch (error) {
        showError('Error creating user: ' + error.message);
        if (error.message === 'Not authenticated') {
            window.location.href = '../index.html';
        }
    }
}

async function editUser(userId) {
    try {
        const token = localStorage.getItem('token');
        if (!token) {
            throw new Error('Not authenticated');
        }
        const response = await fetch(`${API_URL}/users/${userId}`, {
            headers: {
                'Authorization': `Bearer ${token}`,
                'Content-Type': 'application/json'
            }
        });

        if (!response.ok) {
            if (response.status === 401) {
                throw new Error('Not authenticated');
            }
            throw new Error('Failed to fetch user details');
        }

        const user = await response.json();
        showEditUserModal(user);
    } catch (error) {
        showError('Error fetching user details: ' + error.message);
        if (error.message === 'Not authenticated') {
            window.location.href = '../index.html';
        }
    }
}

async function updateUser(userId, userData) {
    try {
        const token = localStorage.getItem('token');
        if (!token) {
            throw new Error('Not authenticated');
        }
        const response = await fetch(`${API_URL}/users/${userId}`, {
            method: 'PUT',
            headers: {
                'Authorization': `Bearer ${token}`,
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(userData)
        });

        if (!response.ok) {
            if (response.status === 401) {
                throw new Error('Not authenticated');
            }
            const data = await response.json();
            throw new Error(data.error || 'Failed to update user');
        }

        showSuccess('User updated successfully');
        loadUsers();
        hideModal('editUserModal');
    } catch (error) {
        showError('Error updating user: ' + error.message);
        if (error.message === 'Not authenticated') {
            window.location.href = '../index.html';
        }
    }
}

async function deleteUser(userId) {
    try {
        if (!confirm('Are you sure you want to delete this user?')) {
            return;
        }

        const token = localStorage.getItem('token');
        if (!token) {
            throw new Error('Not authenticated');
        }
        const response = await fetch(`${API_URL}/users/${userId}`, {
            method: 'DELETE',
            headers: {
                'Authorization': `Bearer ${token}`,
                'Content-Type': 'application/json'
            }
        });

        if (!response.ok) {
            if (response.status === 401) {
                throw new Error('Not authenticated');
            }
            const data = await response.json();
            throw new Error(data.error || 'Failed to delete user');
        }

        showSuccess('User deleted successfully');
        loadUsers();
    } catch (error) {
        showError('Error deleting user: ' + error.message);
        if (error.message === 'Not authenticated') {
            window.location.href = '../index.html';
        }
    }
}

function showCreateUserModal() {
    document.getElementById('createUserForm').reset();
    showModal('createUserModal');
}

function showEditUserModal(user) {
    const form = document.getElementById('editUserForm');
    form.elements.username.value = user.username;
    form.elements.email.value = user.email;
    form.elements.userType.value = user.userType;
    form.dataset.userId = user._id;
    showModal('editUserModal');
}

// UI Helper functions
function showModal(modalId) {
    document.getElementById(modalId).classList.remove('hidden');
}

function hideModal(modalId) {
    document.getElementById(modalId).classList.add('hidden');
}

function showError(message) {
    const alert = document.getElementById('errorAlert');
    alert.textContent = message;
    alert.classList.remove('hidden');
    setTimeout(() => alert.classList.add('hidden'), 5000);
}

function showSuccess(message) {
    const alert = document.getElementById('successAlert');
    alert.textContent = message;
    alert.classList.remove('hidden');
    setTimeout(() => alert.classList.add('hidden'), 5000);
}

// Event Listeners
document.addEventListener('DOMContentLoaded', () => {
    const currentUser = checkAuth();
    if (!currentUser || currentUser.userType !== 'admin') {
        window.location.href = '../index.html';
        return;
    }

    loadUsers();

    // Setup form submissions
    document.getElementById('createUserForm').addEventListener('submit', (e) => {
        e.preventDefault();
        const formData = new FormData(e.target);
        createUser(Object.fromEntries(formData));
    });

    document.getElementById('editUserForm').addEventListener('submit', (e) => {
        e.preventDefault();
        const formData = new FormData(e.target);
        const userId = e.target.dataset.userId;
        updateUser(userId, Object.fromEntries(formData));
    });
});