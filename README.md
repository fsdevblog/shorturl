# go-musthave-shortener-tpl

Шаблон репозитория для трека «Сервис сокращения URL».

## Начало работы

1. Склонируйте репозиторий в любую подходящую директорию на вашем компьютере.
2. В корне репозитория выполните команду `go mod init <name>` (где `<name>` — адрес вашего репозитория на GitHub без префикса `https://`) для создания модуля.

## Обновление шаблона

Чтобы иметь возможность получать обновления автотестов и других частей шаблона, выполните команду:

```
git remote add -m main template https://github.com/Yandex-Practicum/go-musthave-shortener-tpl.git
```

Для обновления кода автотестов выполните команду:

```
git fetch template && git checkout template/main .github
```

Затем добавьте полученные изменения в свой репозиторий.

## Запуск автотестов

Для успешного запуска автотестов называйте ветки `iter<number>`, где `<number>` — порядковый номер инкремента. Например, в ветке с названием `iter4` запустятся автотесты для инкрементов с первого по четвёртый.

При мёрже ветки с инкрементом в основную ветку `main` будут запускаться все автотесты.

Подробнее про локальный и автоматический запуск читайте в [README автотестов](https://github.com/Yandex-Practicum/go-autotests).

## Оптимизация
### Бенчмарк
Реализован бенчмарк тест на пакетную вставку для анализа производительности при разных размерах пакетов:

```shell
$ go test -bench=. ./internal/services/ -benchmem
goos: linux
goarch: amd64
pkg: github.com/fsdevblog/shorturl/internal/services
cpu: 12th Gen Intel(R) Core(TM) i9-12900HX
BenchmarkURLService_BatchCreate_Different_Sizes/size_1-24                1563705               748.7 ns/op           536 B/op         12 allocs/op
BenchmarkURLService_BatchCreate_Different_Sizes/size_10-24                379792              3153 ns/op            4616 B/op         47 allocs/op
BenchmarkURLService_BatchCreate_Different_Sizes/size_100-24                44804             26622 ns/op           42456 B/op        408 allocs/op
BenchmarkURLService_BatchCreate_Different_Sizes/size_1000-24                4060            285901 ns/op          421331 B/op       4007 allocs/op
PASS
ok      github.com/fsdevblog/shorturl/internal/services 5.849s
```


## Анализ профилирования
Профилирование показало высокое потребление памяти `compress/flate.NewWriter` и `compress/flate.(*compressor).initDeflate`. 
В рефакторинге был добавлен `sync.Pool` чтоб избегать постоянной аллокации памяти при создании новых `gzip.Writer` и `gzip.Reader`

```shell
$ go tool pprof -top -base ./profiles/base.pprof ./profiles/result.pprof 
File: main
Build ID: 569e00d39f1891af2665cf1cdf21c727c2cf4279
Type: inuse_space
Time: 2025-07-10 12:11:36 MSK
Showing nodes accounting for -34.58MB, 59.30% of 58.32MB total
      flat  flat%   sum%        cum   cum%
  -28.21MB 48.36% 48.36%   -34.61MB 59.34%  compress/flate.NewWriter (inline)
   -6.40MB 10.98% 59.34%    -6.40MB 10.98%  compress/flate.(*compressor).initDeflate (inline)
    1.03MB  1.76% 57.58%     1.03MB  1.76%  github.com/go-playground/validator/v10.map.init.7
   -0.51MB  0.87% 58.46%    -0.51MB  0.87%  encoding/xml.map.init.0
    0.51MB  0.87% 57.59%     0.51MB  0.87%  github.com/go-playground/validator/v10.map.init.1
    0.50MB  0.86% 56.72%     0.50MB  0.86%  io.init.func1
   -0.50MB  0.86% 57.58%    -0.50MB  0.86%  compress/flate.(*huffmanEncoder).generate
    0.50MB  0.86% 56.73%     0.50MB  0.86%  regexp/syntax.(*compiler).inst (inline)
   -0.50MB  0.86% 57.58%    -0.50MB  0.86%  github.com/gin-gonic/gin.(*Context).Set
   -0.50MB  0.86% 58.44%    -0.50MB  0.86%  net/textproto.(*Reader).ReadLine (inline)
   -0.50MB  0.86% 59.30%    -0.50MB  0.86%  hash/crc32.init
```

Также, без всякого профилирования, было ясно что использовавшийся ранее логгер `logrus` следует заменить на `zap`, что и было сделано. 

## Сборка

Передача информации версии даты и коммита используется с помощью флагов линкера `-ldflags`:

```
   go build -ldflags "-X main.buildVersion=1.0.0 -X main.buildDate=$(date +%Y-%m-%d) -X main.buildCommit=$(git rev-parse --short HEAD)" -o shortener ./cmd/shortener
```