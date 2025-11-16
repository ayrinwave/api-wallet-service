# Тестирование Wallet API через cURL

## 1. Регистрация + Логин → получение TOKEN

    TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/register -H "Content-Type: application/json" -d '{"username":"testuser2","password":"pass123","email":"test2@example.com"}' && curl -s -X POST http://localhost:8080/api/v1/login -H "Content-Type: application/json" -d '{"username":"testuser2","password":"pass123"}' | jq -r '.token')

## 2. Проверка баланса

    curl -X GET http://localhost:8080/api/v1/balance -H "Authorization: Bearer $TOKEN"

## 3. Пополнение USD

    curl -X POST http://localhost:8080/api/v1/wallet/deposit -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d '{"amount":1000.00,"currency":"USD","requestID":"dep-1"}'

## 4. Пополнение EUR

    curl -X POST http://localhost:8080/api/v1/wallet/deposit -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d '{"amount":500.00,"currency":"EUR","requestID":"dep-2"}'

## 5. Проверка баланса

    curl -X GET http://localhost:8080/api/v1/balance -H "Authorization: Bearer $TOKEN"

## 6. Вывод USD

    curl -X POST http://localhost:8080/api/v1/wallet/withdraw -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d '{"amount":200.00,"currency":"USD","requestID":"with-1"}'

## 7. Итоговый баланс

    curl -X GET http://localhost:8080/api/v1/balance -H "Authorization: Bearer $TOKEN"
