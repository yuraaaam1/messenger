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
