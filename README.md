# Gophermart - Накопительная система лояльности «Гофермарт»
## Запуск проект локально

### Клонировать проект
```bash
mkdir myproject && cd myproject
git clone https://github.com/playmixer/gophermart.git .
```
### Запустить контейнеры
```bash
cd deploy
docker-compose up
```
После запуска для тестирования перейти на
http://localhost:8080/swagger/index.html



# Генерация swagger документации
выполнить из корня проекта команду:
```sh
swag init -o ./docs -g ./internal/adapters/api/rest/rest.go
```
подробнее в https://github.com/swaggo/swag

#
# go-musthave-diploma-tpl

Шаблон репозитория для индивидуального дипломного проекта курса «Go-разработчик»

# Начало работы

1. Склонируйте репозиторий в любую подходящую директорию на вашем компьютере.
2. В корне репозитория выполните команду `go mod init <name>` (где `<name>` — адрес вашего репозитория на GitHub без
   префикса `https://`) для создания модуля

# Обновление шаблона

Чтобы иметь возможность получать обновления автотестов и других частей шаблона, выполните команду:

```
git remote add -m master template https://github.com/yandex-praktikum/go-musthave-diploma-tpl.git
```

Для обновления кода автотестов выполните команду:

```
git fetch template && git checkout template/master .github
```

Затем добавьте полученные изменения в свой репозиторий.
