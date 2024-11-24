# Используем базовый образ Node.js
FROM node:18

# Устанавливаем рабочую директорию внутри контейнера
WORKDIR /usr/src/app

RUN npm init -y && npm install express axios cors body-parser

# Копируем файл server.js и папку public в контейнер
COPY server.js ./
COPY public ./public

# Открываем порт, используемый приложением
EXPOSE 3000

# Команда для запуска сервера
CMD ["node", "server.js"]
