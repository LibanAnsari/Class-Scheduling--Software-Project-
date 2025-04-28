// Input validation functions
function validateUsername(username) {
    return username.length >= 3 && /^[a-zA-Z0-9]+$/.test(username);
}

function validatePassword(password) {
    return password.length >= 6;
}

// Backend API URL
const API_URL = 'http://localhost:3000/api';

// Get base path for redirects
function getBasePath() {
    const path = window.location.pathname;
    // If in admin subfolder, need to go up one level
    if (path.includes('/admin/')) {
        return '../';
    }
    return './';
}

// Handle form submission
document.getElementById('loginForm')?.addEventListener('submit', handleLogin);

async function handleLogin(event) {
    event.preventDefault();
    const username = document.getElementById('username').value;
    const password = document.getElementById('password').value;
    const userType = document.getElementById('userType').value;
    const errorElement = document.getElementById('error-message');
    
    try {
        // Clear previous error
        errorElement.classList.add('hidden');
        errorElement.style.display = 'none';

        // Client-side validation
        if (!validateUsername(username)) {
            throw new Error('Invalid username format');
        }
        if (!validatePassword(password)) {
            throw new Error('Password must be at least 6 characters');
        }

        // Check if server is running
        try {
            const response = await fetch(`${API_URL}/auth/login`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    username,
                    password,
                    userType
                })
            });

            let data;
            try {
                data = await response.json();
            } catch (e) {
                throw new Error('Server error: Unable to process response');
            }

            if (!response.ok) {
                throw new Error(data.error || 'Invalid credentials');
            }

            // Store token and user info
            localStorage.setItem('token', data.token);
            localStorage.setItem('currentUser', JSON.stringify(data.user));

            const basePath = getBasePath();
            // Redirect based on user type
            switch(data.user.userType) {
                case 'admin':
                    window.location.href = basePath + 'admin-dashboard.html';
                    break;
                case 'faculty':
                    window.location.href = basePath + 'faculty-dashboard.html';
                    break;
                case 'student':
                    window.location.href = basePath + 'student-dashboard.html';
                    break;
                default:
                    throw new Error('Invalid user type');
            }
        } catch (networkError) {
            // Check if it's a network/server connection error
            if (!navigator.onLine) {
                throw new Error('No internet connection. Please check your network.');
            }
            if (networkError.message.includes('Failed to fetch') || networkError.message.includes('NetworkError')) {
                throw new Error('Unable to connect to server. Please ensure the server is running.');
            }
            throw networkError; // Re-throw other errors
        }
    } catch (error) {
        errorElement.textContent = error.message;
        errorElement.classList.remove('hidden');
        errorElement.style.display = 'block';

        // If it's a server connection error, add retry button
        if (error.message.includes('Unable to connect to server')) {
            // Add retry button if it doesn't exist
            if (!document.getElementById('retryButton')) {
                const retryButton = document.createElement('button');
                retryButton.id = 'retryButton';
                retryButton.className = 'mt-2 px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600 transition-colors';
                retryButton.textContent = 'Retry Connection';
                retryButton.onclick = () => document.getElementById('loginForm').dispatchEvent(new Event('submit'));
                errorElement.appendChild(document.createElement('br'));
                errorElement.appendChild(retryButton);
            }
        }
    }
}

// Check authentication status
function checkAuth() {
    const token = localStorage.getItem('token');
    const currentUser = JSON.parse(localStorage.getItem('currentUser') || 'null');
    
    if (!token || !currentUser) {
        window.location.href = getBasePath() + 'index.html';
        return null;
    }
    
    return currentUser;
}

// Logout function
function logout() {
    localStorage.removeItem('token');
    localStorage.removeItem('currentUser');
    window.location.href = getBasePath() + 'index.html';
}

// Add input event listeners for real-time validation
document.getElementById('username')?.addEventListener('input', function(e) {
    const username = e.target.value.trim();
    const errorMessage = document.getElementById('error-message');
    
    if (username && !validateUsername(username)) {
        errorMessage.textContent = 'Username must be at least 3 characters long and contain only letters and numbers';
        errorMessage.classList.remove('hidden');
    } else {
        errorMessage.classList.add('hidden');
    }
});

// Clear error message when switching user types
document.getElementById('userType')?.addEventListener('change', function() {
    document.getElementById('error-message').classList.add('hidden');
});

// Check authentication on page load if not on login page
if (!window.location.pathname.includes('index.html') && !window.location.pathname.includes('signup.html')) {
    document.addEventListener('DOMContentLoaded', function() {
        const currentUser = checkAuth();
        if (currentUser) {
            const currentPage = window.location.pathname.split('/').pop();
            // Only force redirect if user is on a dashboard page of the wrong type
            if (currentPage === 'admin-dashboard.html' && currentUser.userType !== 'admin') {
                window.location.href = `${currentUser.userType}-dashboard.html`;
            } else if (currentPage === 'faculty-dashboard.html' && currentUser.userType !== 'faculty') {
                window.location.href = `${currentUser.userType}-dashboard.html`;
            } else if (currentPage === 'student-dashboard.html' && currentUser.userType !== 'student') {
                window.location.href = `${currentUser.userType}-dashboard.html`;
            }
        }
    });
}