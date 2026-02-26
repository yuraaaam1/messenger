document.addEventListener("DOMContentLoaded", function() {
    const messageList = document.getElementById("messages-list");
    const messageForm = document.getElementById("message-form");
    const usernameInput = document.getElementById("username-input");
    const messageInput = document.getElementById("message-input");

    function addMessage(msg) {
        const listItem = document.createElement('div');

        let formattedDate = '';
        if (msg.sent_at) {
            formattedDate = new Date(msg.sent_at).toLocaleDateString();
        }

        listItem.innerHTML = `<strong>${msg.user}:</strong> ${msg.text} <span class="timestamp">${formattedDate}</span>`;
        messageList.appendChild(listItem);
        messageList.scrollTop = messageList.scrollHeight;
    }

    fetch("/api/messages")
        .then(response => {
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }
            return response.json();
        })
        .then(messages => {
            if (messages && messages.length > 0) {
                messages.forEach(addMessage);
            }
        })
        .catch(error => {
            console.error('Ошбика при загрузке истории сообщений:', error);
            messageList.innerHTML = '<p>Не удалось загрузить историю сообщения</p>';
        });

        const socket = new WebSocket(`ws://${window.location.host}/ws`);

        socket.onopen = () => {
            console.log("Websocket соединение успешно установлено.");
        };

        socket.onmessage = (event) => {
            try {
                const msg = JSON.parse(event.data);
                addMessage(msg);
            } catch (e) {
                console.error("Ошибка парсинга входящего сообщения:", e);
            }
        };

        socket.onclose = () => {
            console.log("Websocket соединение закрыто.");
            const notice = document.createElement('div');
            notice.textContent = "Соединение с сервером разорвано.";
            notice.style.color = "red";
            notice.style.textAlign = "center";
            messageList.appendChild(notice);
        };

        socket.onerror = (error) => {
            console.error("Websocket ошибка:", error);
        };

        messageForm.onsubmit = (event) => {
            event.preventDefault();

            const user = usernameInput.value;
            const text = messageInput.value;

            if (!text) return;
            if (!user) {
                alert("Пожалуйста, введите ваше имя.");
                return;
            }

            const message = {
                user: user,
                text: text
            };

            socket.send(JSON.stringify(message));

            messageInput.value = '';
        };

});