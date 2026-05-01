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
            salt: strToBytes('rooms-salt:' + roomKey),  // фиксированная соль для того чтобы другие участники комнаты имели такойже AES-ключ
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