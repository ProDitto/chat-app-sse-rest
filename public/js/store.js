/**
 * @typedef {import('./types.js').User} User
 * @typedef {import('./types.js').Message} Message
 */

const DB_NAME = 'SSEChatDB';
const SESSION_USER_ID_KEY = 'sse_chat_userId';
const SESSION_USERNAME_KEY = 'sse_chat_username';

/** @type {Dexie} */
let db;

const store = {
    /**
     * Initializes the Dexie database.
     */
    async init() {
        db = new Dexie(DB_NAME);
        db.version(1).stores({
            messages: '++id, eventId, fromUserId, toUserId, timestamp',
            // We will use localStorage for simple session data for now.
        });
        await db.open();
    },

    /**
     * Saves the current user's session info.
     * @param {User} user
     */
    async saveCurrentUser(user) {
        localStorage.setItem(SESSION_USER_ID_KEY, user.userId);
        localStorage.setItem(SESSION_USERNAME_KEY, user.username);
    },

    /**
     * Retrieves the current user's session info.
     * @returns {Promise<User | null>}
     */
    async getCurrentUser() {
        const userId = localStorage.getItem(SESSION_USER_ID_KEY);
        const username = localStorage.getItem(SESSION_USERNAME_KEY);
        if (userId && username) {
            return { userId, username, isOnline: true };
        }
        return null;
    },

    /**
     * Clears the current user's session info.
     */
    async clearCurrentUser() {
        localStorage.removeItem(SESSION_USER_ID_KEY);
        localStorage.removeItem(SESSION_USERNAME_KEY);
    },

    // --- Placeholder methods for future steps ---

    /**
     * Gets the last saved event ID.
     * @returns {Promise<string | null>}
     */
    async getLastEventId() {
        // To be implemented
        return null;
    },

    /**
     * Saves the last received event ID.
     * @param {string} eventId
     */
    async saveLastEventId(eventId) {
        // To be implemented
    },

    /**
     * Adds a message to the database.
     * @param {Message} message
     */
    async addMessage(message) {
        // To be implemented
    },

    /**
     * Gets messages for a specific contact.
     * @param {string} contactId
     * @param {number} page
     * @returns {Promise<Message[]>}
     */
    async getMessages(contactId, page = 0) {
        // To be implemented
        return [];
    },
};

