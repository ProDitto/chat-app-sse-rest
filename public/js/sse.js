import { store } from './store.js';

const RECONNECT_DELAY_MS = [1000, 2000, 5000, 10000];

/** @type {EventSource | null} */
let eventSource = null;
let reconnectAttempt = 0;
let currentUserId = '';

/** @type {Map<string, Function[]>} */
const listeners = new Map();

function dispatch(event) {
    const eventData = JSON.parse(event.data);
    console.log('SSE Received:', event.type, eventData);

    const normalizedEvent = {
        eventId: event.lastEventId,
        type: event.type,
        timestamp: eventData.Timestamp,
        fromUserId: eventData.FromUserID,
        toUserId: eventData.ToUserID,
        payload: eventData.Payload,
    };

    // Persist all events
    store.addEvent(normalizedEvent);
    store.saveLastEventId(event.lastEventId);

    if (listeners.has(event.type)) {
        listeners.get(event.type).forEach(handler => handler(normalizedEvent));
    }
}

function connectInternal(userId, lastEventId) {
    if (eventSource) {
        eventSource.close();
    }

    const url = `/api/sse?userId=${userId}&lastEventId=${lastEventId}`;
    eventSource = new EventSource(url);

    eventSource.onopen = () => {
        console.log('SSE connection opened.');
        reconnectAttempt = 0;
        if (listeners.has('connect')) {
            listeners.get('connect').forEach(h => h());
        }
    };

    eventSource.onerror = (err) => {
        console.error('SSE connection error:', err);
        eventSource.close();
        const delay = RECONNECT_DELAY_MS[reconnectAttempt] || RECONNECT_DELAY_MS[RECONNECT_DELAY_MS.length - 1];
        console.log(`Reconnecting in ${delay}ms...`);
        setTimeout(async () => {
            const newLastEventId = await store.getLastEventId();
            connectInternal(currentUserId, newLastEventId);
        }, delay);
        reconnectAttempt++;
        if (listeners.has('disconnect')) {
            listeners.get('disconnect').forEach(h => h());
        }
    };

    eventSource.addEventListener('message', dispatch);
    eventSource.addEventListener('user_joined', dispatch);
    eventSource.addEventListener('user_left', dispatch);
    eventSource.addEventListener('username_changed', dispatch);
    eventSource.addEventListener('typing_start', dispatch);
    eventSource.addEventListener('typing_stop', dispatch);
    eventSource.addEventListener('heartbeat', (event) => {
        // console.log('SSE Heartbeat');
        store.saveLastEventId(event.lastEventId);
    });
}

export const sseClient = {
    /**
     * @param {string} userId
     * @param {string} lastEventId
     */
    connect(userId, lastEventId) {
        currentUserId = userId;
        connectInternal(userId, lastEventId);
    },

    close() {
        if (eventSource) {
            eventSource.close();
            eventSource = null;
            console.log('SSE connection closed by client.');
        }
    },

    /**
     * @param {string} type
     * @param {Function} handler
     */
    addEventListener(type, handler) {
        if (!listeners.has(type)) {
            listeners.set(type, []);
        }
        listeners.get(type).push(handler);
    },
};

