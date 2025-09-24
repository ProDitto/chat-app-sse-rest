/**
 * @typedef {import('../types.js').User} User
 */

/**
 * Simple string hashing function.
 * @param {string} str
 * @returns {number}
 */
function simpleHash(str) {
    let hash = 0;
    for (let i = 0; i < str.length; i++) {
        const char = str.charCodeAt(i);
        hash = (hash << 5) - hash + char;
        hash |= 0; // Convert to 32bit integer
    }
    return hash;
}

/**
 * Generates a color from a string hash.
 * @param {string} str
 * @returns {string} HSL color string
 */
function stringToColor(str) {
    const hash = simpleHash(str);
    const h = Math.abs(hash) % 360;
    return `hsl(${h}, 60%, 50%)`;
}

/**
 * Creates an avatar HTML element for a user.
 * @param {User} user
 * @param {string} [extraClasses='']
 * @returns {HTMLElement}
 */
function createAvatar(user, extraClasses = 'w-10 h-10') {
    const avatar = document.createElement('div');
    const initial = user.username ? user.username.charAt(0).toUpperCase() : '?';
    const color = stringToColor(user.userId);

    avatar.className = `flex items-center justify-center rounded-full font-bold text-white ${extraClasses}`;
    avatar.style.backgroundColor = color;
    avatar.textContent = initial;
    avatar.title = user.username;

    return avatar;
}

