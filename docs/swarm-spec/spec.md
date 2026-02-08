# HTTP-спецификация клиента Swarm

Документ описывает структуру HTTP-запросов, формируемых клиентом [`HttpClient`](http.go:22) при выполнении методов `Get`, `GetK1`, `Set`, `Delete` из [`Client`](client.go:35).

## Общие сведения

* Базовый URL собирается из схемы и хоста DSN и сохраняется в поле `baseAddr`, см. [`newClient`](client.go:76). Все запросы направляются на `baseAddr + путь`.
* Ключ `K1` обязан содержать двоеточие. Функции [`singleV1`](http.go:34) и [`k1V1`](http.go:42) делят `K1` на две части: `K1_PREFIX = K1[:idx(':')]`, `K1_SUFFIX = K1[idx(':')+1:]`. Эти две части используются в URL как отдельные сегменты пути.
* Путь всегда заканчивается символом `/`.
* Клиент автоматически выставляет заголовки в [`Do`](http.go:77):
  * `X-Request-ID` — при наличии значения в контексте, см. [`GetContextRequestID`](context.go:?).
  * `Authorization: Basic …` — когда DSN включает учетные данные (`user:pass@`), см. [`newClient`](client.go:164).
  * `x-envoy-upstream-rq-timeout-ms` — строковое значение времени `QueryTimeout`, см. [`newClient`](client.go:164).
  * `x-envoy-retry-on: gateway-error,connect-failure` — всегда присутствует.
  * `x-envoy-max-retries` — при `MaxRetries > 0`, см. [`newClient`](client.go:136).
  * `X-Swarm-Consensus` — при указании `Consensus` в DSN.

## Формат бинарных тел

Функция [`PackTuple`](raw.go:116) сериализует последовательность `[]Field` (поля `Tuple`) в формат:

1. `uint32` (big-endian) — количество полей.
2. Для каждого поля: `uint32` (big-endian) длины, затем содержимое байт.

Функция [`PackTuplesList`](raw.go:139) формирует список кортежей: сначала `uint32` количества кортежей, далее для каждого кортежа структура `PackTuple`.

Ответы методов `Get`/`Set` используют `Tuple`, `GetK1` — список `[]Tuple`. Расшифровка выполняется [`UnpackTuple`](raw.go:86) и [`UnpackTuplesList`](raw.go:42).

## Метод Get

* **HTTP-метод:** `GET`
* **Путь:** `/swarm/{K1_PREFIX}/{K1_SUFFIX}/{K2}/{K3}/`
  * Если `K3` пуст, путь: `/swarm/{K1_PREFIX}/{K1_SUFFIX}/{K2}/`.
* **Тело запроса:** отсутствует.
* **Ответы:**
  * `200 OK` — тело содержит бинарный `Tuple`, см. [`Get`](http.go:112).
  * `404 Not Found` — ключ отсутствует, тело пустое.
* **Пример curl:**
  ```bash
  curl -v -X GET \
    'http://{HOST}/swarm/{K1_PREFIX}/{K1_SUFFIX}/{K2}/{K3}/' \
    -H 'X-Request-ID: {REQUEST_ID}' \
    -H 'Authorization: Basic {BASE64_CREDENTIALS}' \
    -H 'x-envoy-upstream-rq-timeout-ms: {TIMEOUT_MS}' \
    -H 'x-envoy-retry-on: gateway-error,connect-failure' \
    -H 'x-envoy-max-retries: {MAX_RETRIES}'
  ```

## Метод GetK1

* **HTTP-метод:** `GET`
* **Путь:** `/swarm/{K1_PREFIX}/{K1_SUFFIX}/`
* **Параметры:** для каждого `K2` добавляется `t={K2}`. Пример: `?t={K2_1}&t={K2_2}`. См. [`GetK1`](http.go:132).
* **Тело запроса:** отсутствует.
* **Ответы:**
  * `200 OK` — тело содержит бинарный список `[]Tuple`.
  * `404 Not Found` — данных нет, тело пустое.
* **Пример curl:**
  ```bash
  curl -v -X GET \
    'http://{HOST}/swarm/{K1_PREFIX}/{K1_SUFFIX}/?t={K2_1}&t={K2_2}' \
    -H 'Authorization: Basic {BASE64_CREDENTIALS}' \
    -H 'x-envoy-upstream-rq-timeout-ms: {TIMEOUT_MS}' \
    -H 'x-envoy-retry-on: gateway-error,connect-failure' \
    -H 'x-envoy-max-retries: {MAX_RETRIES}'
  ```

## Метод Set

* **HTTP-метод:** `PUT`
* **Путь:** `/swarm/{K1_PREFIX}/{K1_SUFFIX}/{K2}/{K3}/` (или без `{K3}`).
* **Тело:** `PackTuple(value...)`. Порядок полей следует формату `Tuple` (`K1`, `K2`, `K3`, `V1`, `V2`, ...), см. [`Set`](http.go:152).
* **Ответы:**
  * `200 OK` — запись успешна.
  * Иной код => ошибка согласно [`statusError`](http.go:264).
* **Пример curl:**
  ```bash
  python3 - <<'PY' > payload.bin
  import struct
  fields = [b'{K1}', b'{K2}', b'{K3}', b'{VALUE1}', b'{VALUE2}']
  payload = struct.pack('>I', len(fields))
  for f in fields:
      payload += struct.pack('>I', len(f)) + f
  open('payload.bin', 'wb').write(payload)
  PY
  curl -v -X PUT \
    'http://{HOST}/swarm/{K1_PREFIX}/{K1_SUFFIX}/{K2}/{K3}/' \
    -H 'Authorization: Basic {BASE64_CREDENTIALS}' \
    -H 'x-envoy-upstream-rq-timeout-ms: {TIMEOUT_MS}' \
    -H 'x-envoy-retry-on: gateway-error,connect-failure' \
    -H 'x-envoy-max-retries: {MAX_RETRIES}' \
    --data-binary '@payload.bin'
  ```

## Метод Delete

* **HTTP-метод:** `DELETE`
* **Путь:** `/swarm/{K1_PREFIX}/{K1_SUFFIX}/{K2}/{K3}/` (или без `{K3}`), см. [`Delete`](http.go:169).
* **Тело запроса:** отсутствует.
* **Ответы:**
  * `200 OK` — ключ существовал и удален (возвращается `true`).
  * `404 Not Found` — ключ отсутствовал (`false`).
* **Пример curl:**
  ```bash
  curl -v -X DELETE \
    'http://{HOST}/swarm/{K1_PREFIX}/{K1_SUFFIX}/{K2}/{K3}/' \
    -H 'Authorization: Basic {BASE64_CREDENTIALS}' \
    -H 'x-envoy-upstream-rq-timeout-ms: {TIMEOUT_MS}' \
    -H 'x-envoy-retry-on: gateway-error,connect-failure' \
    -H 'x-envoy-max-retries: {MAX_RETRIES}'
  ```

## Обработка ошибок

Ошибки HTTP-статусов транслируются через [`statusError`](http.go:264). Карта [`commonErrors`](http.go:272) задает соответствия статус-кодов (`400`, `401`, `403`, `404`, `405`, `408`, `429`, `500`, `501`, `502`, `503`, `504`) доменным ошибкам.
