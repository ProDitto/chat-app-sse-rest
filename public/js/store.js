import './lib/dexie.min.js';

/**
 * @typedef {import('./types.js').User} User
 * @typedef {import('./types.js').Message} Message
 */

const DB_NAME = 'SSEChatDB';
const SESSION_USER_ID_KEY = 'sse_chat_user_id';
const SESSION_USERNAME_KEY = 'sse_chat_username';
const LAST_EVENT_ID_KEY = 'lastEventId';

/** @type {import('./lib/dexie.min.js').Dexie} */
let db;

export const store = {
    async init() {
        db = new Dexie(DB_NAME);
        db.version(1).stores({
            messages: '++id, &eventId, conversationId, timestamp',
            events: '++id, &eventId, type, timestamp',
            kvstore: 'key',
        });
        await db.open();
    },

    /**
     * @param {User} user
     * @returns {Promise<void>}
     */
    async saveCurrentUser(user) {
        localStorage.setItem(SESSION_USER_ID_KEY, user.userId);
        localStorage.setItem(SESSION_USERNAME_KEY, user.username);
    },

    /** @returns {Promise<User | null>} */
    async getCurrentUser() {
        const userId = localStorage.getItem(SESSION_USER_ID_KEY);
        const username = localStorage.getItem(SESSION_USERNAME_KEY);
        if (userId && username) {
            return { userId, username, isOnline: true };
        }
        return null;
    },

    async clearCurrentUser() {
        localStorage.removeItem(SESSION_USER_ID_KEY);
        localStorage.removeItem(SESSION_USERNAME_KEY);
        if (db) {
            await db.delete();
            await db.open();
        }
    },

    /** @returns {Promise<string | null>} */
    async getLastEventId() {
        const item = await db.kvstore.get(LAST_EVENT_ID_KEY);
        return item ? item.value : '0';
    },

    /** @param {string} eventId */
    async saveLastEventId(eventId) {
        if (!eventId) return;
        await db.kvstore.put({ key: LAST_EVENT_ID_KEY, value: eventId });
    },

    /** @param {any} event */
    async addEvent(event) {
        try {
            await db.events.put({
                eventId: event.eventId,
                type: event.type,
                timestamp: event.timestamp,
                payload: event.payload,
            });
        } catch (e) {
            if (e.name !== 'ConstraintError') {
                console.error('Failed to add event to store:', e);
            }
        }
    },

    /** @param {Message} message */
    async addMessage(message) {
        const conversationId = [message.fromUserId, message.toUserId].sort().join(':');
        try {
            await db.messages.put({ ...message, conversationId });
        } catch (e) {
            if (e.name !== 'ConstraintError') {
                console.error('Failed to add message to store:', e);
            }
        }
    },

    /**
     * @param {string} userId1
     * @param {string} userId2
     * @param {number} page
     * @param {number} pageSize
     * @returns {Promise<Message[]>}
     */
    async getMessages(userId1, userId2, page = 0, pageSize = 50) {
        const conversationId = [userId1, userId2].sort().join(':');
        return db.messages
            .where({ conversationId })
            .orderBy('timestamp')
            .reverse()
            .offset(page * pageSize)
            .limit(pageSize)
            .toArray();
    },
};
