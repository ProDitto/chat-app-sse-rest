/**
 * @typedef {import('./types.js').User} User
 */

const API_BASE = '/api';

async function request(endpoint, options = {}) {
    const defaultHeaders = {
        'Content-Type': 'application/json',
    };

    const config = {
        ...options,
        headers: {
            ...defaultHeaders,
            ...options.headers,
        },
    };

    try {
        const response = await fetch(`${API_BASE}${endpoint}`, config);
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

/** @returns {Promise<{userId: string, username: string, activeUsers: User[]}>} */
export async function join(desiredUsername) {
    return request('/join', {
        method: 'POST',
        body: JSON.stringify({ desiredUsername }),
    });
}

/** @returns {Promise<void>} */
export async function changeUsername(userId, newUsername) {
    return request('/change-username', {
        method: 'POST',
        body: JSON.stringify({ userId, newUsername }),
    });
}

/** @returns {Promise<void>} */
export async function sendMessage(fromUserId, toUserId, text) {
    return request('/message', {
        method: 'POST',
        body: JSON.stringify({ fromUserId, toUserId, text }),
    });
}

/** @returns {Promise<void>} */
export async function sendTyping(userId, toUserId, typing) {
    return request('/typing', {
        method: 'POST',
        body: JSON.stringify({ userId, toUserId, typing }),
    });
}

/** @returns {Promise<{events: any[], activeUsers: User[], lastEventId: string}>} */
export async function getSnapshot(userId, lastEventId) {
    const params = new URLSearchParams({ userId });
    if (lastEventId) {
        params.set('lastEventId', lastEventId);
    }
    return request(`/snapshot?${params.toString()}`);
}
