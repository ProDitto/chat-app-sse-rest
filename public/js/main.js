/**
 * @typedef {import('./types.js').User} User
 */

document.addEventListener('DOMContentLoaded', () => {
    const state = {
        /** @type {User | null} */
        currentUser: null,
        /** @type {User[]} */
        activeUsers: [],
    };

    const joinForm = document.getElementById('join-form');
    const usernameInput = document.getElementById('username-input');
    const modalError = document.getElementById('modal-error');
    const joinButton = document.getElementById('join-button');

    /**
     * Handles the user join process.
     * @param {Event} e
     */
    async function handleJoin(e) {
        e.preventDefault();
        const desiredUsername = usernameInput.value.trim();
        if (!desiredUsername) return;

        joinButton.disabled = true;
        joinButton.textContent = 'Joining...';
        modalError.textContent = '';

        try {
            const data = await join(desiredUsername);
            state.currentUser = { userId: data.userId, username: data.username, isOnline: true };
            state.activeUsers = data.activeUsers;

            await store.saveCurrentUser(state.currentUser);
            
            ui.renderUsers(state.activeUsers, state.currentUser.userId);
            ui.renderCurrentUserPanel(state.currentUser);
            ui.hideModal();
            ui.showApp();

            // Add event listener for username change after user has joined
            document.getElementById('change-username-btn').addEventListener('click', handleChangeUsername);

        } catch (error) {
            modalError.textContent = error.message;
        } finally {
            joinButton.disabled = false;
            joinButton.textContent = 'Join';
        }
    }

    /**
     * Handles the username change process.
     */
    async function handleChangeUsername() {
        const newUsername = prompt('Enter your new username:', state.currentUser.username);
        if (!newUsername || newUsername.trim() === state.currentUser.username) {
            return;
        }

        try {
            await changeUsername(state.currentUser.userId, newUsername.trim());
            
            // Update local state immediately
            state.currentUser.username = newUsername.trim();
            await store.saveCurrentUser(state.currentUser);
            
            // The SSE event will update the list for everyone, but we update our own panel immediately.
            ui.renderCurrentUserPanel(state.currentUser);
            
            // Find and update the user in the active users list
            const userInList = state.activeUsers.find(u => u.userId === state.currentUser.userId);
            if (userInList) {
                userInList.username = newUsername.trim();
                ui.renderUsers(state.activeUsers, state.currentUser.userId);
            }

        } catch (error) {
            alert(`Failed to change username: ${error.message}`);
        }
    }

    /**
     * Initializes the application.
     */
    async function init() {
        await store.init();
        const user = await store.getCurrentUser();

        if (user && user.userId) {
            // In a real scenario with reconnection, we'd try to reconnect here.
            // For this step, we just clear the old session and force a new join.
            console.log('Found previous session, clearing for new join.');
            await store.clearCurrentUser();
            ui.showModal();
        } else {
            ui.showModal();
        }

        joinForm.addEventListener('submit', handleJoin);
    }

    init();
});

