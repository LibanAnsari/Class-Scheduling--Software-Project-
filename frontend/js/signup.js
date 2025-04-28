// Input validation functions
function validateUsername(username) {
    return username.length >= 3 && /^[a-zA-Z0-9]+$/.test(username);
}

function validatePassword(password) {
    return password.length >= 6;
}

// Backend API URL
const API_URL = 'http://localhost:3000/api';

// Handle signup form submission
document.getElementById('signupForm')?.addEventListener('submit', handleSignup);

async function handleSignup(event) {
    event.preventDefault();

    const username = document.getElementById('username').value.trim();
    const password = document.getElementById('password').value;
    const confirmPassword = document.getElementById('confirmPassword').value;
    const email = document.getElementById('email').value.trim();
    const userType = document.getElementById('userType').value;
    const rollNumber = document.getElementById('rollNumber')?.value.trim();

    const errorElement = document.getElementById('error-message');
    const successElement = document.getElementById('success-message');

    try {
        // Clear previous messages
        errorElement.classList.add('hidden');
        successElement.classList.add('hidden');

        // Client-side validation
        if (!validateUsername(username)) {
            throw new Error('Username must be at least 3 characters long and contain only letters and numbers.');
        }

        if (!validatePassword(password)) {
            throw new Error('Password must be at least 6 characters.');
        }

        if (password !== confirmPassword) {
            throw new Error('Passwords do not match.');
        }

        if (!email.includes('@')) {
            throw new Error('Invalid email address.');
        }

        if (userType === 'student' && !rollNumber) {
            throw new Error('Roll Number is required for students.');
        }

        try {
            const response = await fetch(`${API_URL}/auth/signup`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    username,
                    password,
                    email,
                    userType,
                    rollNumber: userType === 'student' ? rollNumber : undefined,
                }),
            });

            let data;
            try {
                data = await response.json();
            } catch (e) {
                throw new Error('Server error: Unable to process response');
            }

            if (!response.ok) {
                throw new Error(data.error || 'Signup failed. Please try again.');
            }

            // Store token and user info
            localStorage.setItem('token', data.token);
            localStorage.setItem('currentUser', JSON.stringify(data.user));

            successElement.textContent = 'Signup successful! Redirecting...';
            successElement.classList.remove('hidden');

            // Redirect to appropriate dashboard based on user type
            setTimeout(() => {
                switch(data.user.userType) {
                    case 'admin':
                        window.location.href = 'admin-dashboard.html';
                        break;
                    case 'faculty':
                        window.location.href = 'faculty-dashboard.html';
                        break;
                    case 'student':
                        window.location.href = 'student-dashboard.html';
                        break;
                    default:
                        window.location.href = 'index.html';
                }
            }, 1500);
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

        // If it's a server connection error, add retry button
        if (error.message.includes('Unable to connect to server')) {
            // Add retry button if it doesn't exist
            if (!document.getElementById('retryButton')) {
                const retryButton = document.createElement('button');
                retryButton.id = 'retryButton';
                retryButton.className = 'mt-2 px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600 transition-colors';
                retryButton.textContent = 'Retry Connection';
                retryButton.onclick = () => document.getElementById('signupForm').dispatchEvent(new Event('submit'));
                errorElement.appendChild(document.createElement('br'));
                errorElement.appendChild(retryButton);
            }
        }
    }
}

// Show/hide Roll Number field based on user type selection
document.getElementById('userType')?.addEventListener('change', function () {
    const rollField = document.getElementById('rollNumberField');
    if (this.value === 'student') {
        rollField.style.display = 'block';
    } else {
        rollField.style.display = 'none';
    }

    // Clear error on type switch
    document.getElementById('error-message').classList.add('hidden');
});

// Show roll number field on page load if default is student
window.addEventListener('DOMContentLoaded', () => {
    const userType = document.getElementById('userType');
    const rollField = document.getElementById('rollNumberField');
    if (userType.value === 'student') {
        rollField.style.display = 'block';
    }
});
