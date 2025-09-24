/**
 * @typedef {import('./types.js').User} User
 */

const API_BASE = '/api';

/**
 * Helper function to handle API requests.
 * @param {string} endpoint
 * @param {RequestInit} options
 * @returns {Promise<any>}
 */
async function request(endpoint, options = {}) {
    try {
        const response = await fetch(`${API_BASE}${endpoint}`, {
            headers: {
                'Content-Type': 'application/json',
                ...options.headers,
            },
            ...options,
        });

        if (!response.ok) {
            const errorData = await response.json().catch(() => ({ message: response.statusText }));
            throw new Error(errorData.message || 'An unknown error occurred');
        }
        
        if (response.status === 204 || response.status === 202) {
            return null;
        }

        return response.json();
    } catch (error) {
        console.error(`API request to ${endpoint} failed:`, error);
        throw error;
    }
}

/**
 * Joins the chat with a desired username.
 * @param {string} desiredUsername
 * @returns {Promise<{userId: string, username: string, activeUsers: User[]}>}
 */
function join(desiredUsername) {
    return request('/join', {
        method: 'POST',
        body: JSON.stringify({ desiredUsername }),
    });
}

/**
 * Changes the username for a given user.
 * @param {string} userId
 * @param {string} newUsername
 * @returns {Promise<void>}
 */
function changeUsername(userId, newUsername) {
    return request('/change-username', {
        method: 'POST',
        body: JSON.stringify({ userId, newUsername }),
    });
}

