# APIGateway
Skillfactory

### запустить приложение:
#### Соединение со свей базой данных PostgreSql можно отредактировать в файле "Skillfactory-APIGateway/cmd/gonews/sqlPostgres.json"


#### gateway запускается на localhost:8000

### Доступные API , примеры:

постраничная навигация
* http://localhost:80/news?page=2&s=

вывод последних новостей
* http://localhost:80/news/latest

вывод по номеру страннице
* http://localhost:80/news/latest?page=2

поиск по заголовкам
* http://localhost:80/news?s=gRPC

детальная информация о посте с комментарием
* http://localhost:80/news/detailed?id=1

добавление комментария методом post в формате JSON , 
с проверкой на слова из стоп листа (qwerty , йцукен , zxvbnm)
* http://localhost:80/comments/add

Удаляет комментарий по id методом delete в формате JSON
* http://localhost:80/comments/del