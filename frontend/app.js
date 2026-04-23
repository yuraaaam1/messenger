function strToBytes(str) {
    return new TextEncoder().encode(str);
}

// Превращает байты в hex-строку вида "a3f9..."
// Используется чтобы отправить хеш на сервер как обычный текст
function bytesToHex(buffer) {
    return Array.from(new Uint8Array(buffer))
        .map(b => b.toString(16).padStart(2, '0'))
        .join('');
}


// Считает SHA-256 от строки и возвращает hex
// Используется для key_hash и device_key_hash которые хранятся на сервере
async function sha256(str) {
    const hashBuffer = await crypto.subtle.digest('SHA-256',strToBytes(str))
    return bytesToHex(hashBuffer);
}

// deriveAesKey вызывается при вводе ключа и входе в комнату, дял создания aesKey
// Из текстового ключа делает настоящий AES-ключ через PBKDF2
// PBKDF2 — это алгоритм "усиления" ключа: даже короткий пароль превращается
// в криптографически стойкий 256-битный ключ
async function deriveAesKey(roomKey) {
    const baseKey = await crypto.subtle.importKey(
        'raw',                    // формат входных данных (простой массив байтов)
        strToBytes(roomKey),      // сам пароль в виде байтов
        {name: 'PBKDF2'},         // алгоритм, для которого импортируем
        false,                    // можно ли экспортировать ключ (нет)
        ['deriveKey']             // что можно делать с ключом (только порождать другие ключи)
    );

        return crypto.subtle.deriveKey(
        {
            name: 'PBKDF2',           // алгоритм растяжения ключа
            salt: strToBytes('backrooms-salt'),  // случайная соль (здесь фиксированная!)
            iterations: 100000,       // 100 тысяч раундов (чем больше, тем медленнее для взлома)
            hash: 'SHA-256'           // хеш-функция внутри PBKDF2
        },
        baseKey,                      // ключ из шага 1
        {name: 'AES-GCM', length: 256 },  // какой ключ хотим получить (AES 256 бит)
        false,                        // экспортировать нельзя
        ['encrypt', 'decrypt']        // можно шифровать и расшифровывать
    );
}

// Шифрует текст, возвращает base64-строку вида "iv:ciphertext"
// IV (вектор инициализации) — случайные 12 байт, уникальные для каждого сообщения
// Без IV одинаковые сообщения давали бы одинаковый шифротекст — это уязвимость

async function encryptMessage(aesKey, plainText) {
    const iv = crypto.getRandomValues(new Uint8Array(12));
    const ciphertext = await crypto.subtle.encrypt(
        {name: 'AES-GCM', iv},
        aesKey,
        strToBytes(plainText)
    );

    // Упаковываем iv и ciphertext в одну строку через разделитель ":"
    const ivB64 = btoa(String.fromCharCode(...iv));
    const ctB64 = btoa(String.fromCharCode(...new Uint8Array(ciphertext)));
    return ivB64 + ':' + ctB64;    
}

async function decryptMessage(aesKey, payload){
    const [ivB64, ctB64] = payload.split(':');
    const iv = Uint8Array.from(atob(ivB64), c => c.charCodeAt(0));
    const ciphertext = Uint8Array.from(atob(ctB64), c => c.charCodeAt(0));

    const plaintext = await crypto.subtle.decrypt(
        {name: 'AES-GCM', iv},
        aesKey,
        ciphertext
    );

    return new TextDecoder().decode(plaintext);
}

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

// Эти переменные хранят текущее состояние — они живут пока открыта вкладка
let aesKey = null;      // AES-ключ для шифрования/расшифровки сообщений
let socket = null;      // WebSocket соединение
let currentRoomName = ''; // Название текущей комнаты
let currentRoomId = null;

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

// Отоюражение сообщений
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

function connectWebSocket(device_key_hash) {
    const wsUrl = `ws://${window.location.host}/ws?device_key_hash=${device_key_hash}`;
    socket = new WebSocket(wsUrl);

    socket.onopen = () => {
        console.log('WebSocket подключён');
    };

    socket.onmessage = async (event) => {
        try {
            const payload = event.data;
            const decrypted = await decryptMessage(aesKey, payload);
            const parsed = JSON.parse(decrypted);

            const msg = {
                type: parsed.type || 'text',
                sender: parsed.sender,
                text: parsed.text,
                filename: parsed.filename,
                mime: parsed.mime,
                data: parsed.data,
                time: new Date().toLocaleTimeString()
            };

            saveMessage(currentRoomId, msg);
            renderMessage(msg);
        } catch (e) {
            console.error('Ошибка расшифровки:', e)
        }
    };

    socket.onclose = () => {
        console.log('WebSocket отключён');
    };

    socket.onerror = (e) => {
        console.error('WebSocket ошибка:', e);
    };
}

// Отправка сообщений
document.getElementById('message-form').addEventListener('submit', async (e) => {
    e.preventDefault();

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