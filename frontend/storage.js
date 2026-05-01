function getDeviceId() {
    let deviceId = localStorage.getItem('deviceId');
    if (!deviceId) {
        deviceId = crypto.randomUUID();
        localStorage.setItem('deviceId', deviceId)
    }
    return deviceId
}

function getNickName() {
    const input = document.getElementById('nickname-input').value.trim();
    const saved = localStorage.getItem('nickname');
    if (input) {
        localStorage.setItem('nickname', input)
        return input;
    }
    return saved || 'anonim'
}

// История сообщений (localStorage)
function loadHistory(roomId) {
    const raw = localStorage.getItem('messages_' + roomId);
    return raw ? JSON.parse(raw) : []
}

function saveMessage(roomId, message) {
    const history = loadHistory(roomId);
    history.push(message);
    localStorage.setItem('messages_' + roomId, JSON.stringify(history));
}