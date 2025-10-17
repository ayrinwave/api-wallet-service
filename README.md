# Высокопроизводительный сервис "Кошелек"

Приложение, которое по REST принимает запрос вида

POST api/v1/wallet {

valletId: UUID,

operationType: DEPOSIT or WITHDRAW,

amount: 1000

}

После выполнять логику по изменению счета в базе данных
также есть возможность получить баланс кошелька

GET api/v1/wallets/{WALLET_UUID}

Обратите особое внимание проблемам при работе в конкурентной среде (1000 RPS по
одному кошельку). Ни один запрос не должен быть не обработан (50Х error)

## Технологии

- **Go**
- **PostgreSQL**
- **Docker, Docker Compose**
---

## Установка
```sh
git clone https://github.com/ayrinwave/api-wallet-service.git
```

### Перейти в каталог проекта через терминал
```sh
cd api-wallet-service
```
### Запуск приложения

```sh
docker-compose --env-file config.env up --build
```
где config.env - путь к файлу конфигурации

### Описание работы

Приложение по REST принимает: 
```sh
POST api/v1/wallet с телом запроса:
{

valletId: UUID,

operationType: DEPOSIT or WITHDRAW,

amount: 1000

}
```
Есть возможность возвращать балланс кошелька: 
```sh
GET api/v1/wallets/{WALLET_UUID}
```