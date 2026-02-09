document.addEventListener("DOMContentLoaded", function() {
    const messageContainer = document.createElement('div');
    document.body.appendChild(messageContainer);

    const loadingText = document.querySelector('p');
    if (loadingText) {
        loadingText.textContent = '';
    }

    fetch("/api/messages")
        .then(response => response.json())
        .then(messages => {
            if (messages.length === 0) {
                messageContainer.innerHTML = '<p>Сообщений пока нет.</p>';
                return;
            }

            const messageList = document.createElement('ul');
            messages.forEach(msg => {
                const listItem = document.createElement('li');
                listItem.textContent = `${msg.user}: ${msg.text}`;
                messageList.appendChild(listItem);
            });
            messageContainer.appendChild(messageList);
        })
        .catch(error => {
            console.error('Ошбика при загрузке сообщений:', error);
            messageContainer.innerHTML = '<p>Не удалось загрузить сообщения</p>';
        });
});