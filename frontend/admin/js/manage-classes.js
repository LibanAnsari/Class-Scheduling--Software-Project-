// Backend API URL
const API_URL = 'http://localhost:3000/api';

// Class management functions
async function loadClasses() {
    try {
        const token = localStorage.getItem('token');
        if (!token) {
            throw new Error('Not authenticated');
        }
        const response = await fetch(`${API_URL}/classes`, {
            headers: {
                'Authorization': `Bearer ${token}`,
                'Content-Type': 'application/json'
            }
        });

        if (!response.ok) {
            if (response.status === 401) {
                throw new Error('Not authenticated');
            }
            throw new Error('Failed to load classes');
        }

        const classes = await response.json();
        displayClasses(classes);
    } catch (error) {
        showError('Error loading classes: ' + error.message);
        if (error.message === 'Not authenticated') {
            window.location.href = '../index.html';
        }
    }
}

function displayClasses(classes) {
    const container = document.getElementById('classesContainer');
    if (!classes.length) {
        container.innerHTML = '<p class="text-gray-500">No classes found.</p>';
        return;
    }

    container.innerHTML = classes.map(course => `
        <div class="p-4 bg-white rounded-lg shadow-sm hover:shadow-md transition-shadow">
            <div class="flex justify-between items-start">
                <div>
                    <h3 class="font-medium">${course.name}</h3>
                    <p class="text-sm text-gray-600">Code: ${course.code}</p>
                    <p class="text-sm text-gray-600">Faculty: ${course.facultyName}</p>
                </div>
                <span class="px-2 py-1 text-sm rounded ${getStatusClass(course.status)}">
                    ${course.status}
                </span>
            </div>
            <div class="mt-2">
                <p class="text-sm">${course.description}</p>
                <p class="text-sm mt-1">Capacity: ${course.enrolledStudents}/${course.capacity}</p>
            </div>
            <div class="flex gap-2 mt-4">
                <button onclick="editClass('${course._id}')" 
                        class="px-3 py-1 text-sm bg-blue-500 text-white rounded hover:bg-blue-600">
                    Edit
                </button>
                <button onclick="deleteClass('${course._id}')"
                        class="px-3 py-1 text-sm bg-red-500 text-white rounded hover:bg-red-600">
                    Delete
                </button>
            </div>
        </div>
    `).join('');
}

function getStatusClass(status) {
    const classes = {
        active: 'bg-green-100 text-green-800',
        inactive: 'bg-red-100 text-red-800',
        pending: 'bg-yellow-100 text-yellow-800'
    };
    return classes[status] || 'bg-gray-100 text-gray-800';
}

async function createClass(classData) {
    try {
        const token = localStorage.getItem('token');
        if (!token) {
            throw new Error('Not authenticated');
        }
        const response = await fetch(`${API_URL}/classes`, {
            method: 'POST',
            headers: {
                'Authorization': `Bearer ${token}`,
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(classData)
        });

        if (!response.ok) {
            if (response.status === 401) {
                throw new Error('Not authenticated');
            }
            throw new Error('Failed to create class');
        }

        showSuccess('Class created successfully');
        loadClasses();
        hideModal('createClassModal');
    } catch (error) {
        showError('Error creating class: ' + error.message);
        if (error.message === 'Not authenticated') {
            window.location.href = '../index.html';
        }
    }
}

async function editClass(classId) {
    try {
        const token = localStorage.getItem('token');
        if (!token) {
            throw new Error('Not authenticated');
        }
        const response = await fetch(`${API_URL}/classes/${classId}`, {
            headers: {
                'Authorization': `Bearer ${token}`,
                'Content-Type': 'application/json'
            }
        });

        if (!response.ok) {
            if (response.status === 401) {
                throw new Error('Not authenticated');
            }
            throw new Error('Failed to fetch class details');
        }

        const classData = await response.json();
        showEditClassModal(classData);
    } catch (error) {
        showError('Error fetching class details: ' + error.message);
        if (error.message === 'Not authenticated') {
            window.location.href = '../index.html';
        }
    }
}

async function updateClass(classId, classData) {
    try {
        const token = localStorage.getItem('token');
        if (!token) {
            throw new Error('Not authenticated');
        }
        const response = await fetch(`${API_URL}/classes/${classId}`, {
            method: 'PUT',
            headers: {
                'Authorization': `Bearer ${token}`,
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(classData)
        });

        if (!response.ok) {
            if (response.status === 401) {
                throw new Error('Not authenticated');
            }
            throw new Error('Failed to update class');
        }

        showSuccess('Class updated successfully');
        loadClasses();
        hideModal('editClassModal');
    } catch (error) {
        showError('Error updating class: ' + error.message);
        if (error.message === 'Not authenticated') {
            window.location.href = '../index.html';
        }
    }
}

async function deleteClass(classId) {
    try {
        if (!confirm('Are you sure you want to delete this class?')) {
            return;
        }

        const token = localStorage.getItem('token');
        if (!token) {
            throw new Error('Not authenticated');
        }
        const response = await fetch(`${API_URL}/classes/${classId}`, {
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
            throw new Error('Failed to delete class');
        }

        showSuccess('Class deleted successfully');
        loadClasses();
    } catch (error) {
        showError('Error deleting class: ' + error.message);
        if (error.message === 'Not authenticated') {
            window.location.href = '../index.html';
        }
    }
}

async function loadFaculties() {
    try {
        const token = localStorage.getItem('token');
        if (!token) {
            throw new Error('Not authenticated');
        }
        const response = await fetch(`${API_URL}/users?type=faculty`, {
            headers: {
                'Authorization': `Bearer ${token}`,
                'Content-Type': 'application/json'
            }
        });

        if (!response.ok) {
            if (response.status === 401) {
                throw new Error('Not authenticated');
            }
            throw new Error('Failed to load faculties');
        }

        const faculties = await response.json();
        populateFacultyDropdown(faculties);
    } catch (error) {
        showError('Error loading faculties: ' + error.message);
        if (error.message === 'Not authenticated') {
            window.location.href = '../index.html';
        }
    }
}

function populateFacultyDropdown(faculties) {
    const selects = document.querySelectorAll('.faculty-select');
    const options = faculties.map(faculty => 
        `<option value="${faculty._id}">${faculty.name}</option>`
    ).join('');
    
    selects.forEach(select => {
        select.innerHTML = '<option value="">Select Faculty</option>' + options;
    });
}

function showCreateClassModal() {
    document.getElementById('createClassForm').reset();
    showModal('createClassModal');
}

function showEditClassModal(classData) {
    const form = document.getElementById('editClassForm');
    form.elements.name.value = classData.name;
    form.elements.code.value = classData.code;
    form.elements.description.value = classData.description;
    form.elements.capacity.value = classData.capacity;
    form.elements.faculty.value = classData.facultyId;
    form.elements.status.value = classData.status;
    form.dataset.classId = classData._id;
    showModal('editClassModal');
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

    loadClasses();
    loadFaculties();

    // Setup form submissions
    document.getElementById('createClassForm').addEventListener('submit', (e) => {
        e.preventDefault();
        const formData = new FormData(e.target);
        createClass(Object.fromEntries(formData));
    });

    document.getElementById('editClassForm').addEventListener('submit', (e) => {
        e.preventDefault();
        const formData = new FormData(e.target);
        const classId = e.target.dataset.classId;
        updateClass(classId, Object.fromEntries(formData));
    });
});