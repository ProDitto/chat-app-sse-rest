import { createAvatar } from './components/avatar.js';

/**
 * @typedef {import('./types.js').User} User
 * @typedef {import('./types.js').Message} Message
 */

export const ui = {
    appContainer: document.getElementById('app'),
    modal: document.getElementById('username-modal'),
    modalError: document.getElementById('modal-error'),
    userList: document.getElementById('user-list'),
    currentUserPanel: document.getElementById('current-user-panel'),
    messageList: document.getElementById('message-list'),
    messageInput: document.getElementById('message-input'),
    sendButton: document.getElementById('send-button'),
    chatHeader: document.getElementById('chat-header'),
    typingIndicator: document.getElementById('typing-indicator'),

    showModal() {
        this.modal.classList.remove('hidden');
    },

    hideModal() {
        this.modal.classList.add('hidden');
    },

    /** @param {string} message */
    displayError(message) {
        this.modalError.textContent = message;
    },

    showApp() {
        this.appContainer.classList.remove('hidden');
    },

    /**
     * @param {User[]} users
     * @param {string} currentUserId
     * @param {string | null} selectedUserId
     */
    renderUsers(users, currentUserId, selectedUserId) {
        this.userList.innerHTML = '';
        const sortedUsers = users
            .filter(u => u.userId !== currentUserId)
            .sort((a, b) => a.username.localeCompare(b.username));

        sortedUsers.forEach(user => {
            const userItem = document.createElement('div');
            userItem.className = `flex items-center gap-3 p-2 rounded-lg cursor-pointer hover:bg-gray-700 ${user.userId === selectedUserId ? 'bg-gray-600' : ''}`;
            userItem.dataset.userId = user.userId;

            const avatar = createAvatar(user, 'w-10 h-10');
            const username = document.createElement('span');
            username.textContent = user.username;

            userItem.append(avatar, username);
            this.userList.appendChild(userItem);
        });
    },

    /**
     * @param {User} user
     * @param {() => void} onChangeUsername
     */
    renderCurrentUserPanel(user, onChangeUsername) {
        this.currentUserPanel.innerHTML = '';
        const avatar = createAvatar(user, 'w-10 h-10');
        
        const userInfo = document.createElement('div');
        userInfo.className = 'flex-1';
        const usernameEl = document.createElement('div');
        usernameEl.className = 'font-bold';
        usernameEl.textContent = user.username;
        userInfo.appendChild(usernameEl);

        const changeBtn = document.createElement('button');
        changeBtn.textContent = 'Change';
        changeBtn.className = 'text-sm text-blue-400 hover:underline';
        changeBtn.onclick = onChangeUsername;
        userInfo.appendChild(changeBtn);

        this.currentUserPanel.append(avatar, userInfo);
    },

    /**
     * @param {User} contactUser
     */
    renderChatWindow(contactUser) {
        this.chatHeader.innerHTML = '';
        this.messageList.innerHTML = '';
        if (contactUser) {
            const avatar = createAvatar(contactUser, 'w-10 h-10');
            const name = document.createElement('span');
            name.className = 'font-bold';
            name.textContent = contactUser.username;
            this.chatHeader.append(avatar, name);
            this.messageInput.disabled = false;
            this.sendButton.disabled = false;
        } else {
            this.chatHeader.textContent = 'Select a user to start chatting';
            this.messageInput.disabled = true;
            this.sendButton.disabled = true;
        }
    },

    /**
     * @param {Message} message
     * @param {boolean} isOwnMessage
     * @param {User} fromUser
     */
    addMessage(message, isOwnMessage, fromUser) {
        const messageWrapper = document.createElement('div');
        messageWrapper.className = `flex items-end gap-2 my-2 ${isOwnMessage ? 'justify-end' : 'justify-start'}`;

        const avatar = createAvatar(fromUser, 'w-8 h-8');

        const bubble = document.createElement('div');
        bubble.className = `max-w-xs md:max-w-md p-3 rounded-lg break-words ${isOwnMessage ? 'bg-blue-600 rounded-br-none' : 'bg-gray-700 rounded-bl-none'}`;
        bubble.textContent = message.text;

        if (isOwnMessage) {
            messageWrapper.append(bubble, avatar);
        } else {
            messageWrapper.append(avatar, bubble);
        }

        this.messageList.appendChild(messageWrapper);
        this.messageList.scrollTop = this.messageList.scrollHeight;
    },

    /**
     * @param {User} user
     */
    showTypingIndicator(user) {
        this.typingIndicator.textContent = `${user.username} is typing...`;
        this.typingIndicator.classList.remove('hidden');
    },

    hideTypingIndicator() {
        this.typingIndicator.classList.add('hidden');
    },
};
