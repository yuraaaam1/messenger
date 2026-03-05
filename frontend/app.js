document.addEventListener("DOMContentLoaded", function() {
    const authContainer = document.getElementById('auth-container');
    const chatContainer = document.getElementById('chat-container');

    const loginFormContainer = document.getElementById('login-form-container');
    const registerFormContainer = document.getElementById('register-form-container');

    const loginForm = document.getElementById('login-form');
    const registerForm = document.getElementById('register-form')


    const showRegisterLink = document.getElementById('show-register');
    const showLoginLink = document.getElementById('show-login');

    const authError = document.getElementById('auth-error');


    const messageList = document.getElementById("messages-list");
    const messageForm = document.getElementById("message-form");
    const messageInput = document.getElementById("message-input");
    const logoutButton = document.getElementById('logout-button');

    let socket;
    let token = localStorage.getItem('authToken');

    // Переключение между формами login и register;
    showRegisterLink.addEventListener('click', (e) => {
        e.preventDefault();
        loginFormContainer.style.display = 'none';
        registerFormContainer.style.display = 'block';
        if (authError) authError.textContent = '';
    });
    showLoginLink.addEventListener('click', (e) => {
        e.preventDefault();
        registerFormContainer.style.display = 'none';
        loginFormContainer.style.display = 'block';
        if (authError) authError.textContent = '';
    });

    function showAuthError(message) {
        if (authError) authError.textContent = message;
    }

    function addMessage(msg) {
        const listItem = document.createElement('div');
        const formattedDate = new Date(msg.sent_at).toLocaleString();
        
        listItem.innerHTML = `<strong>${msg.user}:</strong> ${msg.text} <span class="timestamp">${formattedDate}</span>`;
        messageList.appendChild(listItem);
        messageList.scrollTop = messageList.scrollHeight;
    }

    // Auth logic;

    loginForm.addEventListener('submit', async (e) => {
        e.preventDefault();
        const email = document.getElementById('login-email').value;
        const password = document.getElementById('login-password').value;

        try {
            const response = await fetch('/api/auth/login', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ email, password })
            });

            const data = await response.json();

            if (!response.ok){
                throw new Error(data.error || 'Ошибка ввода');
            }

            token = data.token;
            localStorage.setItem('authToken', token);
            initializeApp();
        
        } catch (error) {
            showAuthError(error.message);
            console.error('Ошибка ввода:', error);
        } 
    });

    registerForm.addEventListener('submit', async(e) => {
        e.preventDefault();
        const username = document.getElementById('register-username').value;
        const email = document.getElementById('register-email').value;
        const password = document.getElementById('register-password').value;

        try {
            const response = await fetch('/api/auth/register', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ username, email, password })
            });

            const data = await response.json();

            if (!response.ok) {
                throw new Error(data.error || 'Ошибка регистрации');
            }

            token = data.token;
            localStorage.setItem('authToken', token);
            initializeApp();
        
        } catch (error) {
            showAuthError(error.message);
            console.error('Ошибка регистрации', error);
        }
    });

    logoutButton.addEventListener('click', () => {
        token = null;
        localStorage.removeItem('authToken');
        if (socket) {
            socket.close();
        }
        authContainer.style.display = 'block';
        chatContainer.style.display = 'none';
        messageList.innerHTML = '';
        console.log("Выход из системы.");
    });

    // Chat logic;

    function connectWebSocket() {
        if (!token) return;

        // Устанавливаем соединение добавляя токен в URL;
        const wsUrl = `ws://${window.location.host}/ws?token=${token}`;
        socket = new WebSocket(wsUrl);

        socket.onopen = () => {
            console.log("Websocket соединение успешно установлено.");
            loadMessageHistory();
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
            notice.textContent = "Соединение с сервером разорвано. Пожалуйста, обновите страницу.";
            notice.className = "error-message"
            messageList.appendChild(notice);
        };

        socket.onerror = (error) => {
            console.error("Websocket ошибка:", error);
        };
    }

    function loadMessageHistory() {
        if (!token) return;

        fetch("/api/messages", {
            headers: { 'Authorization': `Bearer ${token}` }
        })
        .then(response => {
            if (!response.ok) {
                throw new Error('Не удалось получить историю сообщений: ' + response.statusText);
            }
            return response.json();
        })
        .then(messages => {
            messageList.innerHTML = "";
            if (messages && messages.length > 0) {
                messages.forEach(addMessage);
            }
        })
        .catch(error => {
            console.error('Ошибка при загрузке истории сообщений:', error);
            messageList.innerHTML = '<p>Не удалось загрузить историю сообщений</p>';
        });
    }

    messageForm.onsubmit = (event) => {
        event.preventDefault();
        const text = messageInput.value;

        if (!text || !socket || socket.readyState !== WebSocket.OPEN) return;

        const message = {
            text: text
        };

        socket.send(JSON.stringify(message));
        messageInput.value = '';
    };

    function initializeApp() {
        if (token) {
            authContainer.style.display = 'none';
            chatContainer.style.display = 'block';
            if (authError) authError.textContainer = '';

            loadMessageHistory();
            connectWebSocket();
        } else {
            authContainer.style.display = 'block';
            chatContainer.style.display = 'none';
        }
    }

    initializeApp();
});    