/**
 * @typedef {import('./types.js').User} User
 */

const ui = {
    // --- DOM Elements ---
    appContainer: document.getElementById('app'),
    modal: document.getElementById('username-modal'),
    modalError: document.getElementById('modal-error'),
    userList: document.getElementById('user-list'),
    currentUserPanel: document.getElementById('current-user-panel'),

    // --- Modal Management ---
    showModal() {
        this.modal.classList.remove('hidden');
    },

    hideModal() {
        this.modal.classList.add('hidden');
    },

    displayError(message) {
        this.modalError.textContent = message;
    },

    showApp() {
        this.appContainer.classList.remove('hidden');
    },

    // --- User List ---
    /**
     * Renders the list of active users in the sidebar.
     * @param {User[]} users
     * @param {string} currentUserId
     */
    renderUsers(users, currentUserId) {
        this.userList.innerHTML = '';
        const sortedUsers = [...users].sort((a, b) => {
            if (a.userId === currentUserId) return -1;
            if (b.userId === currentUserId) return 1;
            return a.username.localeCompare(b.username);
        });

        sortedUsers.forEach(user => {
            const isCurrentUser = user.userId === currentUserId;
            const userElement = document.createElement('div');
            userElement.className = `flex items-center p-2 rounded-md cursor-pointer mb-1 ${isCurrentUser ? 'bg-blue-900/50' : 'hover:bg-gray-700'}`;
            userElement.dataset.userId = user.userId;

            const avatar = createAvatar(user);
            
            const statusIndicator = document.createElement('div');
            statusIndicator.className = `absolute bottom-0 right-0 w-3 h-3 rounded-full border-2 border-gray-800 ${user.isOnline ? 'bg-green-500' : 'bg-gray-500'}`;
            
            const avatarContainer = document.createElement('div');
            avatarContainer.className = 'relative mr-3';
            avatarContainer.appendChild(avatar);
            avatarContainer.appendChild(statusIndicator);

            const usernameSpan = document.createElement('span');
            usernameSpan.className = 'font-medium truncate';
            usernameSpan.textContent = user.username + (isCurrentUser ? ' (You)' : '');

            userElement.appendChild(avatarContainer);
            userElement.appendChild(usernameSpan);
            this.userList.appendChild(userElement);
        });
    },

    /**
     * Renders the current user's information panel at the bottom of the sidebar.
     * @param {User} user
     */
    renderCurrentUserPanel(user) {
        this.currentUserPanel.innerHTML = `
            <div class="flex items-center">
                ${createAvatar(user, 'w-10 h-10').outerHTML}
                <div class="ml-3 flex-1 min-w-0">
                    <p class="font-semibold truncate">${user.username}</p>
                    <p class="text-sm text-gray-400">Online</p>
                </div>
                <button id="change-username-btn" title="Change username" class="p-2 rounded-full hover:bg-gray-700">
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
                        <path d="M13.586 3.586a2 2 0 112.828 2.828l-.793.793-2.828-2.828.793-.793zM11.379 5.793L3 14.172V17h2.828l8.38-8.379-2.83-2.828z" />
                    </svg>
                </button>
            </div>
        `;
    }
};
```
