// Эти переменные хранят текущее состояние — они живут пока открыта вкладка
let aesKey = null;      // AES-ключ для шифрования/расшифровки сообщений
let socket = null;      // WebSocket соединение
let currentRoomName = ''; // Название текущей комнаты
let currentRoomId = null;

// Логика входа в комнату
async function enterRoom(roomKey, roomId, roomName) {
    aesKey = await deriveAesKey(roomKey);

    const deviceId = getDeviceId();

    const deviceKeyHash = await sha256(roomKey + ':' + deviceId);

    const joinRes = await fetch('/api/rooms/join', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ key_hash: await sha256(roomKey), device_key_hash: deviceKeyHash })
    });

    if (!joinRes.ok) {
        throw new Error('Не удалось войти в комнату');
    }

    currentRoomId = roomId;
    showChatScreen(roomName);
    renderHistory(roomId);
    connectWebSocket(deviceKeyHash);
}

// Обработчик кнопки создать комнату
document.getElementById('create-btn').addEventListener('click', async () => {
    const name = document.getElementById('create-room-name').value.trim();
    const key = document.getElementById('create-room-key').value;

    if (!name || !key) {
        showError('Введите название и ключ')
        return;
    }

    clearError();

    try{
        const keyHash = await sha256(key);

        const res = await fetch('/api/rooms', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ name, key_hash: keyHash })
        });

        const data = await res.json();

        if (!res.ok) {
            showError(data.error || 'Ошибка создания комнаты');
            return
        }

        await enterRoom(key, data.id, data.name);
    } catch (e) {
        showError('Ошибка: '+ e.message)
    }
});

// Обработчик кнопки "Войти в комнату"
document.getElementById('join-btn').addEventListener('click', async () => {
    const key = document.getElementById('join-room-key').value;

    if (!key) {
        showError('Введите ключ')
        return
    }

    clearError()

    try {
        const keyHash = await sha256(key);

        // Сначала проверяем что комната с таким ключом существует
        const res = await fetch('/api/rooms/join', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                key_hash: keyHash,
                device_key_hash: await sha256(key + ':' + getDeviceId())
            })
        });

        const data = await res.json();

        if (!res.ok) {
            showError('Комната не найдена — проверьте ключ');
            return;
        }

        aesKey = await deriveAesKey(key);
        currentRoomId = data.room_id;
        showChatScreen(data.name);
        renderHistory(data.room_id);
        connectWebSocket(await sha256(key + ':' + getDeviceId()));

    } catch (e) {
        showError('Ошибка: ' + e.message);
    }
});

// Отправка сообщений
document.getElementById('message-form').addEventListener('submit', async (e) => {
    e.preventDefault(); // отключение перезагрузок страницы

    const input = document.getElementById('message-input');
    const text = input.value.trim();
    if (!text || !socket || socket.readyState !== WebSocket.OPEN) return;

    const payload = JSON.stringify({sender: getNickName(), text});
    const encrypted = await encryptMessage(aesKey, payload);
    
    socket.send(encrypted);

    const msg = {
        type: 'text',
        own: true,
        text,
        sender: getNickName(),
        time: new Date().toLocaleTimeString()
    };

    saveMessage(currentRoomId, msg);
    renderMessage(msg);

    input.value = '';
});

document.getElementById('leave-btn').addEventListener('click', () => {
    if (socket) socket.close();
    aesKey = null;
    currentRoomId = null;
    showAuthScreen();
})

// Отправка файла
document.getElementById('file-input').addEventListener('change', async(e) => {
    const file = e.target.files[0];
    if (!file) return;

    if (file.size > 10 * 1024 * 1024) {
        alert('Файл слишком большой. Максимум 10MB.');
        e.target.value = '';
        return;
    }

    if (!socket || socket.readyState !== WebSocket.OPEN) {
        alert('Нет соединения с сервером');
        return;
    }

// FileReader читает файл как ArrayBuffer(массив байтов) 
    const reader = new FileReader();
    reader.onload = async (event) => {
        // Ковертируем в байты в base64 чтобы положить в JSON
        const bytes = new Uint8Array(event.target.result);
        const base64 = btoa(String.fromCharCode(...bytes));

        // Упаковываем в JSON - имя файла, тип и данные
        const payload = JSON.stringify({
            type: 'file',
            sender: getNickName(),
            filename: file.name,
            mime: file.type,
            data: base64
        });

        // Шифруем и отправляем
        const encrypted = await encryptMessage(aesKey, payload);
        socket.send(encrypted);

        //Показываем у себя сразу
        const msg = {
            type: 'file',
            own: true,
            sender: getNickName(),
            filename: file.name,
            mime: file.type,
            data: base64,
            time: new Date().toLocaleTimeString()
        };
        saveMessage(currentRoomId, msg)
        renderMessage(msg);
    };

    reader.readAsArrayBuffer(file);
    e.target.value = ''

});