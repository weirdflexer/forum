# Генерация документации кода

Из каталога backend:

```sh
go doc -all -u ./internal/forum > ../docs/generated/go-code.txt
```

Файл go-code.txt содержит структуры, функции и методы реального пакета forum, включая внутренние объявления. HTML — читаемое представление этого результата. HTTP-контракт находится рядом в docs/openapi.json.
