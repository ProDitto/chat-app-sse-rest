import * as api from './api.js';
import { store } from './store.js';
import { ui } from './ui.js';
import { sseClient } from './sse.js';

/**
 * @typedef {import('./types.js').User} User
 * @typedef {import('./types.js').Message} Message
 */

const state = {
    /** @type {User | null} */
    currentUser: null,
    /** @type {Map<string, User>} */
    activeUsers: new Map(),
    /** @type {string | null} */
    selectedUserId: null,
    typingTimeout: null,
};

async function handleChangeUsername() {
    const newUsername = prompt('Enter new username:', state.currentUser.username);
    if (newUsername && newUsername.trim() && newUsername !== state.currentUser.username) {
        try {
            await api.changeUsername(state.currentUser.userId, newUsername.trim());
            // The change will be confirmed via SSE, but we can update optimistically
            state.currentUser.username = newUsername.trim();
            await store.saveCurrentUser(state.currentUser);
            ui.renderCurrentUserPanel(state.currentUser, handleChangeUsername);
        } catch (error) {
            alert(`Failed to change username: ${error.message}`);
        }
    }
}

async function handleUserClick(user) {
    state.selectedUserId = user.userId;
    ui.renderUsers(Array.from(state.activeUsers.values()), state.currentUser.userId, state.selectedUserId);
    ui.renderChatWindow(user);
    ui.hideTypingIndicator();

    const messages = await store.getMessages(state.currentUser.userId, state.selectedUserId, 0, 50);
    messages.forEach(msg => {
        const fromUser = state.activeUsers.get(msg.fromUserId) || (msg.fromUserId === state.currentUser.userId ? state.currentUser : null);
        if (fromUser) {
            ui.addMessage(msg, msg.fromUserId === state.currentUser.userId, fromUser);
        }
    });
}

function updateUserList(users) {
    state.activeUsers.clear();
    users.forEach(u => state.activeUsers.set(u.userId, u));
    ui.renderUsers(Array.from(state.activeUsers.values()), state.currentUser.userId, state.selectedUserId);
}

function processEvent(event) {
    const { type, payload, fromUserId, toUserId, eventId, timestamp } = event;

    switch (type) {
        case 'user_joined':
            if (payload.userId !== state.currentUser.userId) {
                state.activeUsers.set(payload.userId, payload);
                ui.renderUsers(Array.from(state.activeUsers.values()), state.currentUser.userId, state.selectedUserId);
            }
            break;
        case 'user_left':
            state.activeUsers.delete(payload.userId);
            if (state.selectedUserId === payload.userId) {
                state.selectedUserId = null;
                ui.renderChatWindow(null);
            }
            ui.renderUsers(Array.from(state.activeUsers.values()), state.currentUser.userId, state.selectedUserId);
            break;
        case 'username_changed':
            if (state.activeUsers.has(payload.userId)) {
                const user = state.activeUsers.get(payload.userId);
                user.username = payload.newUsername;
                ui.renderUsers(Array.from(state.activeUsers.values()), state.currentUser.userId, state.selectedUserId);
                if (state.selectedUserId === payload.userId) {
                    ui.renderChatWindow(user);
                }
            }
            if (payload.userId === state.currentUser.userId) {
                state.currentUser.username = payload.newUsername;
                store.saveCurrentUser(state.currentUser);
                ui.renderCurrentUserPanel(state.currentUser, handleChangeUsername);
            }
            break;
        case 'message':
            const message = { eventId, fromUserId, toUserId, text: payload.text, timestamp };
            store.addMessage(message);
            if (
                (message.fromUserId === state.selectedUserId && message.toUserId === state.currentUser.userId) ||
                (message.fromUserId === state.currentUser.userId && message.toUserId === state.selectedUserId)
            ) {
                const fromUser = state.activeUsers.get(message.fromUserId) || (message.fromUserId === state.currentUser.userId ? state.currentUser : null);
                if (fromUser) {
                    ui.addMessage(message, message.fromUserId === state.currentUser.userId, fromUser);
                }
            }
            break;
        case 'typing_start':
            if (fromUserId === state.selectedUserId && toUserId === state.currentUser.userId) {
                const fromUser = state.activeUsers.get(fromUserId);
                if (fromUser) ui.showTypingIndicator(fromUser);
            }
            break;
        case 'typing_stop':
            if (fromUserId === state.selectedUserId && toUserId === state.currentUser.userId) {
                ui.hideTypingIndicator();
            }
            break;
    }
}

async function startChatSession() {
    const lastEventId = await store.getLastEventId();
    try {
        const snapshot = await api.getSnapshot(state.currentUser.userId, lastEventId);
        updateUserList(snapshot.activeUsers);

        snapshot.events.forEach(event => {
            const parsedEvent = {
                eventId: event.EventID,
                type: event.Type,
                timestamp: event.Timestamp,
                fromUserId: event.FromUserID,
                toUserId: event.ToUserID,
                payload: event.Payload,
            };
            processEvent(parsedEvent);
        });

        const finalEventId = snapshot.lastEventId || lastEventId;
        await store.saveLastEventId(finalEventId);

        sseClient.connect(state.currentUser.userId, finalEventId);
        sseClient.addEventListener('message', processEvent);
        sseClient.addEventListener('user_joined', processEvent);
        sseClient.addEventListener('user_left', processEvent);
        sseClient.addEventListener('username_changed', processEvent);
        sseClient.addEventListener('typing_start', processEvent);
        sseClient.addEventListener('typing_stop', processEvent);

    } catch (error) {
        console.error('Failed to get snapshot and start session:', error);
        alert('Could not connect to the chat server. Please try again.');
    }
}

async function handleSendMessage(e) {
    e.preventDefault();
    const input = ui.messageInput;
    const text = input.value.trim();

    if (text && state.selectedUserId) {
        try {
            input.value = '';
            input.focus();
            // Stop sending typing indicator
            if (state.typingTimeout) {
                clearTimeout(state.typingTimeout);
                api.sendTyping(state.currentUser.userId, state.selectedUserId, false);
                state.typingTimeout = null;
            }
            await api.sendMessage(state.currentUser.userId, state.selectedUserId, text);
        } catch (error) {
            console.error('Failed to send message:', error);
            alert('Message could not be sent.');
            input.value = text;
        }
    }
}

function handleTyping() {
    if (!state.selectedUserId) return;

    if (!state.typingTimeout) {
        api.sendTyping(state.currentUser.userId, state.selectedUserId, true);
    } else {
        clearTimeout(state.typingTimeout);
    }

    state.typingTimeout = setTimeout(() => {
        api.sendTyping(state.currentUser.userId, state.selectedUserId, false);
        state.typingTimeout = null;
    }, 2000);
}

async function handleJoin(e) {
    e.preventDefault();
    const joinButton = document.getElementById('join-button');
    const usernameInput = document.getElementById('username-input');
    const modalError = document.getElementById('modal-error');

    if (!usernameInput.checkValidity()) {
        modalError.textContent = 'Username must be 3-20 characters.';
        return;
    }

    joinButton.disabled = true;
    joinButton.textContent = 'Joining...';
    modalError.textContent = '';

    try {
        const { userId, username, activeUsers } = await api.join(usernameInput.value.trim());
        state.currentUser = { userId, username, isOnline: true };
        
        await store.saveCurrentUser(state.currentUser);
        
        ui.renderCurrentUserPanel(state.currentUser, handleChangeUsername);
        ui.hideModal();
        ui.showApp();

        document.getElementById('message-form').addEventListener('submit', handleSendMessage);
        ui.messageInput.addEventListener('input', handleTyping);
        
        ui.userList.addEventListener('click', (e) => {
            const userItem = e.target.closest('[data-user-id]');
            if (userItem) {
                const clickedUserId = userItem.dataset.userId;
                if (clickedUserId !== state.currentUser.userId) {
                    const user = state.activeUsers.get(clickedUserId);
                    if (user) handleUserClick(user);
                }
            }
        });

        await startChatSession();

    } catch (error) {
        modalError.textContent = error.message;
    } finally {
        joinButton.disabled = false;
        joinButton.textContent = 'Join';
    }
}

async function init() {
    await store.init();
    // For this iteration, we always start fresh.
    // In a future step, we might try to restore a session.
    await store.clearCurrentUser();

    ui.showModal();
    document.getElementById('join-form').addEventListener('submit', handleJoin);
}

document.addEventListener('DOMContentLoaded', init);
