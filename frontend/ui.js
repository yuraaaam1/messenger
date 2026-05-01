function showError(message) {
    document.getElementById('auth-error').textContent = message;
}

function clearError() {
    document.getElementById('auth-error').textContent = '';
}

// Переключение на экран чата
function showChatScreen(roomName) {
    currentRoomName = roomName;
    document.getElementById('room-name').textContent = roomName;
    document.getElementById('auth-screen').style.display = 'none';
    document.getElementById('chat-screen').style.display = 'flex';
}


// Переключение на экран входа
function showAuthScreen() {
    document.getElementById('auth-screen').style.display = 'block';
    document.getElementById('chat-screen').style.display = 'none';
    document.getElementById('messages-list').innerHTML = '';
}

// Отображение сообщений
function renderMessage(msg) {
    const list = document.getElementById('messages-list');
    const div = document.createElement('div');
    div.className = 'message ' + (msg.own ? 'own' : 'other');

    if (msg.type === 'file') {
        // Создаём ссылку для скачивания
        const url = `data:${msg.mime};base64,${msg.data}`;
        div.innerHTML = `
            <span class="time">${msg.time}</span>
            <strong>${msg.sender}</strong> прислал файл:<br>
            <a class="file-link" href="${url}" download="${msg.filename}">⬇ ${msg.filename}</a>
        `;
    } else {
        div.innerHTML = `<span class="time">${msg.time}</span> <strong>${msg.sender}</strong>: ${msg.text}`;
    }

    list.appendChild(div);
    list.scrollTop = list.scrollHeight;
}

function renderHistory(roomId) {
    const history = loadHistory(roomId);
    history.forEach(renderMessage);
}
