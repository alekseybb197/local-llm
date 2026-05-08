# Установка и использование локального LLM бэкэнда на MacBook Pro M4 24G

Далее будут описаны шаги, необходимые для установки локального бэкэнда LLM на macbook, после чего будет продемонстрирована настройка и использование на задаче кодирования набора доступных клиентов.

Предлагаемое решение рассчитано на использование архитектуры унифицированной памяти (Unified Memory) macbook, что хорошо работает на плотных (dense) моделях и именно такие далее рассмотриваются.

_Disclaimer:_

_Решение для архитектуры, требующей выгрузки слоев модели на GPU, должно строиться иначе и использовать возможности MoE архитектур LLM. Такой подход потребует пересчетов всех параметров и иных технических подходов._

_Ничего из далее сказанного не является истиной в последней инстанции, модели и программные компоненты развиваются постоянно, кроме того возможны иные инженерные компромиссы, определяемые размерностью модели и принятым решением по выбору квантизации._ 

## Загрузка и установка необходимых компонентов

__Для контроля загрузки памяти рекомендуется использовать Activity Monitor.__ 

Диаграмма __memory pressure__ будет показывать заполнение памяти в процессе работы решения. Если график будет оставаться зеленым, то беспокоиться не о чем. Но скорее всего диаграмма будет показывать оранжевые всплески. Лучше избегать появления насыщенных цветов и своевременно закрывать лишние приложения.

Macbook очень устойчиво работает и так организует приложения, чтобы они имели максимально отзывчивый интерфейс. Поэтому простое закрытие окон не гарантирует выгрузку приложений, а длительная работа без перезагрузки постепенно заполняет swap.

Рекомендуется перезагрузить macbook и пересмотреть работающие приложения чтобы избавиться от не самых необходимых, таких о которых просто забыли однажды.

---

__Устанавливаем бэкэнд llama.cpp__

```bash
❯ cd
❯ mkdir llm-server
❯ cd llm-server
❯ wget https://github.com/ggml-org/llama.cpp/releases/download/b8999/llama-b8999-bin-macos-arm64-kleidiai.tar.gz
❯ tar xzvf llama-b8999-bin-macos-arm64-kleidiai.tar.gz
❯ ln -s llama-b8999 app
```

Здесь указана ссылка, актуальная на день создания документа. Рекомендуется выбирать последнюю доступную версию из списка релизов.

---

__Проверяем доступную память и оборудование.__

```bash
❯ app/llama-server --help >/dev/null
ggml_metal_device_init: tensor API disabled for pre-M5 and pre-A19 devices
ggml_metal_library_init: using embedded metal library
ggml_metal_library_init: loaded in 7.138 sec
ggml_metal_rsets_init: creating a residency set collection (keep_alive = 180 s)
ggml_metal_device_init: GPU name:   MTL0 (Apple M4 Pro)
ggml_metal_device_init: GPU family: MTLGPUFamilyApple9  (1009)
ggml_metal_device_init: GPU family: MTLGPUFamilyCommon3 (3003)
ggml_metal_device_init: GPU family: MTLGPUFamilyMetal3  (5001)
ggml_metal_device_init: simdgroup reduction   = true
ggml_metal_device_init: simdgroup matrix mul. = true
ggml_metal_device_init: has unified memory    = true
ggml_metal_device_init: has bfloat            = true
ggml_metal_device_init: has tensor            = false
ggml_metal_device_init: use residency sets    = true
ggml_metal_device_init: use shared buffers    = true
ggml_metal_device_init: recommendedMaxWorkingSetSize  = 17179.89 MB
```

Такой запуск позволяет определить доступный объем рабочей памяти. И в нашем случае он __17179.89 MB__

---

__Выбираем модель LLM.__

На сайте Huggingface представлена огромная коллекция моделей. В данной статье не будем рассматривать как и почему выбрано именно семейство Qwen 3.5. Лучше конечно 3.6, но надо учитывать выше указанный предел __17179.89 MB__. И остается совсем не много доступных для работы вариантов моделей, которые могут уложиться в такой объем памяти. Фактически их три - 9B, 14B, 27B. Одного нет в линейке доступного семейства Qwen и выбор уже между Qwen3.5-9B и Qwen3.6-27B. 

Очевидно, что 9B модель можно взять с менее агрессивной квантизацией, а вот 27B придется выбрать совсем жестко пожатую. И даже в этих двух случаях кодирования далее весь trade-off будет строиться вокруг сжатия kv кэша и достижимого размера контекста.

Если бы здесь рассматривался macbook с памятью не менее 48G, то решение с Qwen3.6-27B смотрелось бы вполне привлекательно. Однако на macbook с 24G даже после самой агрессивной квантизации модель Qwen3.6-27B занимает критически много памяти, так что для других приложений практически ничего не остается.

Следующий выбор в том, брать ли оригинальную, но квантизированную модель, или fine-tuned/distilled, подготовленную для определенного класса задач.

Решение в пользу дообученных моделей должно приниматься с учетом характера использования. Например, если к бэкэнду будет обращаться только агент, настроенный на решение больших задач с построением длинных цепочек рассуждений, то применение модели дистиллированной под Opus 4.6 даст выигрыш.

Учитывая, что на используемом macbook нет технической возможности разместить более одной модели LLM, то очевидно выбираем base-line модель. Такие модели предоставляет Unsloth (https://unsloth.ai/). Спектр вариантов квантизации очень большой. Выбираем средний уровень квантизации из стремления получить модель в объеме примерно 5Gb. И здесь подходит только линейка 4-битных квантизаций, выбираем среднее качество, отдавая предпочтение приемлемой скорости -- Q4_K_S.

https://huggingface.co/unsloth/Qwen3.5-9B-GGUF?show_file_info=Qwen3.5-9B-Q4_K_S.gguf

## Выбор и оценка параметров настройки бэкэнда

__Настраиваем параметры запуска бэкэнда__

```bash
❯ cat .env
export LOCAL_LLM_MODEL="${LOCAL_LLM_MODEL:-Qwen3.5-9B-Q4_K_S.gguf}"
export LOCAL_LLM_PORT="${LOCAL_LLM_PORT:-8000}"
export LOCAL_LLM_HOST="${LOCAL_LLM_HOST:-127.0.0.1}"
export LOCAL_LLM_APIKEY="${LOCAL_LLM_APIKEY:-token_as_a_lot_of_symbols}"
export LOCAL_LLM_BATCH="${LOCAL_LLM_BATCH:-512}" # 512 256
export LOCAL_LLM_CONTEXT="${LOCAL_LLM_CONTEXT:-262144}" # 32768 65536 131072 262144
export LOCAL_LLM_NGLAYERS="${LOCAL_LLM_NGLAYERS:-999}"
export LOCAL_LLM_THREADS="${LOCAL_LLM_THREADS:-8}" # 8 6 4 3
export LOCAL_LLM_CACHE="${LOCAL_LLM_CACHE:-q8_0}" # q8_0 q5_1 q4_1
export LOCAL_LLM_TIMEOUT="${LOCAL_LLM_TIMEOUT:-900}" # 300 900
export LOCAL_LLM_SLOTS="${LOCAL_LLM_SLOTS:-1}"
export LOCAL_LLM_CACHERAM="${LOCAL_LLM_CACHERAM:-0}" # 0 2048 8096
export LOCAL_LLM_ALIAS="${LOCAL_LLM_ALIAS:-qwen35}"
export LOCAL_LLM_REASONING="${LOCAL_LLM_REASONING:-off}"

export GGML_METAL_EMBEDDING=1  # Принудительно вычислять эмбеддинги на GPU
```

_Значение токена надо установить одинаковым и для бэкэнда и для всех взаимодействющих клиентов._

---

### **Базовые параметры сервера**

| Ключ | Значение | Расшифровка |
|------|----------|-------------|
| `-m models/$LOCAL_LLM_MODEL` | `Qwen3.5-9B-Q4_K_S.gguf` | Путь к модели. **Qwen3.5-9B** — 9-миллиардная модель, **Q4_K_S** — квантизация 4-bit (K-quants, small), хороший баланс качество/память. На 24GB MacBook это ~5.5GB VRAM, остаётся запас для контекста. |
| `--port $LOCAL_LLM_PORT` | `8000` | Порт HTTP-сервера. Стандартный для локальных LLM. |
| `--host $LOCAL_LLM_HOST` | `127.0.0.1` | Бинд только на localhost. **Безопасно** — недоступен извне. Если нужен доступ с других устройств в сети, меняй на `0.0.0.0` (но тогда продумай firewall). |
| `--api-key $LOCAL_LLM_APIKEY` | `30cef16e...` | API-ключ для авторизации запросов. **Обязателен** для любого внешнего фронтенда (Open WebUI, ChatGPT-Next-Web и т.д.). Без него сервер отклонит запросы. |


### **Производительность и GPU**

| Ключ | Значение | Расшифровка |
|------|----------|-------------|
| `--n-gpu-layers $LOCAL_LLM_NGLAYERS` | `999` | **Все слои на GPU**. `999` — магическое число «загрузить сколько влезет». На M4 с 24GB Unified Memory модель полностью уйдёт на Metal. Это ключевой параметр для скорости. |
| `--flash-attn on` | `on` | **Flash Attention** — оптимизированный механизм внимания. Снижает потребление памяти и ускоряет инференс длинных контекстов. **Обязательно включать** на Metal/macOS, даёт ~20-40% прироста на длинных последовательностях. |
| `--threads $LOCAL_LLM_THREADS` | `8` | Количество CPU-тредов. M4 имеет 10 ядер (4 perf + 6 eff). `8` — разумный компромисс: оставляет ресурсы системе, но не слишком режет скорость CPU-операций (вроде токенизации). Можно попробовать `6` если система тормозит. |


### **Память и кэширование**

| Ключ | Значение | Расшифровка |
|------|----------|-------------|
| `--ctx-size $LOCAL_LLM_CONTEXT` | `262144` | **262K токенов контекста** (!). Это огромное значение. Qwen3.5 поддерживает до 128K нативно, но llama.cpp может «растянуть» через YaRN/NTK. **Предупреждение**: на 24GB с 262K контекстом и `--cache-type-k q8_0` кэш KV займёт ~**16GB** (2×262K×128×9B×1byte). Модель 5.5GB + кэш 16GB = 21.5GB. Это впритык к лимиту. Если будут OOM (out of memory), снижай до `131072` или `65536`. |
| `--batch-size $LOCAL_LLM_BATCH` | `512` | Размер батча для промпт-обработки. `512` — стандарт. Больше = быстрее обработка длинных промптов, но больше пиковая память. На 24GB можно попробовать `1024`, но с 262K контекстом это рискованно. |
| `--cache-type-k $LOCAL_LLM_CACHE` | `q8_0` | Квантизация **Key-кэша** до 8-bit. По умолчанию кэш KV хранится в fp16 (16-bit). `q8_0` срезает память кэша вдвое с минимальной потерей качества. |
| `--cache-type-v $LOCAL_LLM_CACHE` | `q8_0` | То же для **Value-кэша**. Параметр `--cache-type-k` и `--cache-type-v` в паре дают экономию ~50% памяти на KV-кэше. Альтернативы: `q5_1`, `q4_1` — ещё агрессивнее, но могут деградировать качество на очень длинных контекстах. |
| `--cache-ram $LOCAL_LLM_CACHERAM` | `0` | **RAM-кэш для KV**. `0` = выключен. Если включить (например, `8096`), llama.cpp будет скидывать часть KV-кэша в RAM при нехватке VRAM. На твоём MacBook с Unified Memory это бессмысленно — VRAM и RAM это одно и то же. Оставляй `0`. |


### **Функциональность API**

| Ключ | Значение | Расшифровка |
|------|----------|-------------|
| `--jinja` | — | Включает **Jinja2-шаблонизатор** для chat templates. Критично для Qwen3.5 — без этого модель не поймёт формат диалога (system/user/assistant) и будет генерировать мусор или игнорировать системный промпт. |
| `--parallel $LOCAL_LLM_SLOTS` | `1` | Количество параллельных слотов для обработки запросов. `1` = один запрос за раз. Увеличение до `2-3` позволит обрабатывать несколько диалогов одновременно, но KV-кэш делится между ними (память × слоты). На 24GB с 262K контекстом **строго 1**. |
| `--timeout $LOCAL_LLM_TIMEOUT` | `900` | Таймаут запроса в секундах (15 минут). Для длинных генераций с 262K контекстом это разумно. Стандартные 300 секунд могли бы обрывать длинные ответы. |
| `--alias $LOCAL_LLM_ALIAS` | `qwen35` | Имя модели в API-ответах. Open WebUI и другие фронтенды отображают это как название модели. |
| `--reasoning $LOCAL_LLM_REASONING` | `off` | Управление **reasoning content** (цепочка мыслей). `off` = скрывать `<think>...</think>` блоки из вывода. Qwen3.5 — reasoning-модель, она генерирует промежуточные рассуждения. `off` показывает только финальный ответ. Альтернативы: `on` (показывать), `plain` (показывать как обычный текст). |


### **Переменные окружения**

| Переменная | Значение | Расшифровка |
|------------|----------|-------------|
| `GGML_METAL_EMBEDDING=1` | `1` | **Принудительно вычислять эмбеддинги на GPU**. По умолчанию эмбеддинг-эндпоинт (`/v1/embeddings`) может работать на CPU. На M4 с Metal это ускоряет RAG-пайплайны и векторный поиск в разы. |


## Оценка производительности 

__Скрипт оценки скорости.__

```bash
❯ cat benchmark.sh
#!/bin/bash

. .env

echo -----
env | sort | grep '\(^GGML_\|^LOCAL_LLM_\)'
echo -----

app/llama-bench -m models/$LOCAL_LLM_MODEL \
  --n-gpu-layers $LOCAL_LLM_NGLAYERS \
  --n-gen 64 \
  --n-prompt 512 \
  --batch-size $LOCAL_LLM_BATCH \
  --threads $LOCAL_LLM_THREADS
```

---

__Оцениваем скорость работы.__

```bash
❯ ./benchmark.sh
-----
GGML_METAL_EMBEDDING=1
LOCAL_LLM_ALIAS=qwen35
LOCAL_LLM_APIKEY=token_as_a_lot_of_symbols
LOCAL_LLM_BATCH=512
LOCAL_LLM_CACHE=q8_0
LOCAL_LLM_CACHERAM=0
LOCAL_LLM_CONTEXT=262144
LOCAL_LLM_HOST=127.0.0.1
LOCAL_LLM_MODEL=Qwen3.5-9B-Q4_K_S.gguf
LOCAL_LLM_NGLAYERS=999
LOCAL_LLM_PORT=8000
LOCAL_LLM_REASONING=off
LOCAL_LLM_SLOTS=1
LOCAL_LLM_THREADS=8
LOCAL_LLM_TIMEOUT=900
-----
ggml_metal_device_init: tensor API disabled for pre-M5 and pre-A19 devices
ggml_metal_library_init: using embedded metal library
ggml_metal_library_init: loaded in 0.027 sec
ggml_metal_rsets_init: creating a residency set collection (keep_alive = 180 s)
ggml_metal_device_init: GPU name:   MTL0 (Apple M4 Pro)
ggml_metal_device_init: GPU family: MTLGPUFamilyApple9  (1009)
ggml_metal_device_init: GPU family: MTLGPUFamilyCommon3 (3003)
ggml_metal_device_init: GPU family: MTLGPUFamilyMetal3  (5001)
ggml_metal_device_init: simdgroup reduction   = true
ggml_metal_device_init: simdgroup matrix mul. = true
ggml_metal_device_init: has unified memory    = true
ggml_metal_device_init: has bfloat            = true
ggml_metal_device_init: has tensor            = false
ggml_metal_device_init: use residency sets    = true
ggml_metal_device_init: use shared buffers    = true
ggml_metal_device_init: recommendedMaxWorkingSetSize  = 17179.89 MB
| model                          |       size |     params | backend    | threads | n_batch |            test |                  t/s |
| ------------------------------ | ---------: | ---------: | ---------- | ------: | ------: | --------------: | -------------------: |
| qwen35 9B Q4_K - Small         |   5.01 GiB |     8.95 B | MTL,BLAS   |       8 |     512 |           pp512 |        347.60 ± 1.09 |
| qwen35 9B Q4_K - Small         |   5.01 GiB |     8.95 B | MTL,BLAS   |       8 |     512 |            tg64 |         30.44 ± 1.05 |

build: b97ebdc98 (8999)
```

Измеренная скорость:
- обработка промпта __374 токена/с__
- генерация __30 токенов/с__

Что такое генерация со скоростью __30 токенов/с__ -- это достаточно комфортная скорость, примерно немного выше скорости ручного ввода текста.

## Проверка работы в тестовом режиме

### Запуск бэкэнда

__Скрипт тестового запуска.__

```bash
❯ cat server.sh
#!/bin/bash

. .env

echo -----
env | sort | grep '\(^GGML_\|^LOCAL_LLM_\)'
echo -----

app/llama-server -m models/$LOCAL_LLM_MODEL \
  --port $LOCAL_LLM_PORT \
  --host $LOCAL_LLM_HOST \
  --api-key $LOCAL_LLM_APIKEY \
  --batch-size $LOCAL_LLM_BATCH \
  --ctx-size $LOCAL_LLM_CONTEXT \
  --n-gpu-layers $LOCAL_LLM_NGLAYERS \
  --flash-attn on \
  --threads $LOCAL_LLM_THREADS \
  --jinja \
  --cache-type-k $LOCAL_LLM_CACHE \
  --cache-type-v $LOCAL_LLM_CACHE \
  --timeout $LOCAL_LLM_TIMEOUT \
  --parallel $LOCAL_LLM_SLOTS \
  --alias $LOCAL_LLM_ALIAS \
  --cache-ram $LOCAL_LLM_CACHERAM \
  --reasoning $LOCAL_LLM_REASONING
```

---

__Запуск бэкэнда для тестирования.__

```bash
❯ ./server.sh
-----
GGML_METAL_EMBEDDING=1
LOCAL_LLM_ALIAS=qwen35
LOCAL_LLM_APIKEY=token_as_a_lot_of_symbols
LOCAL_LLM_BATCH=512
LOCAL_LLM_CACHE=q8_0
LOCAL_LLM_CACHERAM=0
LOCAL_LLM_CONTEXT=262144
LOCAL_LLM_HOST=127.0.0.1
LOCAL_LLM_MODEL=Qwen3.5-9B-Q4_K_S.gguf
LOCAL_LLM_NGLAYERS=999
LOCAL_LLM_PORT=8000
LOCAL_LLM_REASONING=off
LOCAL_LLM_SLOTS=1
LOCAL_LLM_THREADS=8
LOCAL_LLM_TIMEOUT=900
-----
ggml_metal_device_init: tensor API disabled for pre-M5 and pre-A19 devices
ggml_metal_library_init: using embedded metal library
ggml_metal_library_init: loaded in 0.011 sec
ggml_metal_rsets_init: creating a residency set collection (keep_alive = 180 s)
ggml_metal_device_init: GPU name:   MTL0 (Apple M4 Pro)
ggml_metal_device_init: GPU family: MTLGPUFamilyApple9  (1009)
ggml_metal_device_init: GPU family: MTLGPUFamilyCommon3 (3003)
ggml_metal_device_init: GPU family: MTLGPUFamilyMetal3  (5001)
ggml_metal_device_init: simdgroup reduction   = true
ggml_metal_device_init: simdgroup matrix mul. = true
ggml_metal_device_init: has unified memory    = true
ggml_metal_device_init: has bfloat            = true
ggml_metal_device_init: has tensor            = false
ggml_metal_device_init: use residency sets    = true
ggml_metal_device_init: use shared buffers    = true
ggml_metal_device_init: recommendedMaxWorkingSetSize  = 17179.89 MB
build_info: b8999-b97ebdc98
system_info: n_threads = 8 (n_threads_batch = 8) / 12 | MTL : EMBED_LIBRARY = 1 | CPU : NEON = 1 | ARM_FMA = 1 | FP16_VA = 1 | DOTPROD = 1 | LLAMAFILE = 1 | ACCELERATE = 1 | KLEIDIAI = 1 | REPACK = 1 | 
Running without SSL
init: api_keys: ****4c3e
init: using 11 threads for HTTP server
start: binding port with default address family
main: loading model
srv    load_model: loading model 'models/Qwen3.5-9B-Q4_K_S.gguf'
common_init_result: fitting params to device memory, for bugs during this step try to reproduce them with -fit off, or provide --verbose logs if the bug only occurs with -fit on
common_params_fit_impl: getting device memory data for initial parameters:
common_memory_breakdown_print: | memory breakdown [MiB]  | total    free    self   model   context   compute    unaccounted |
common_memory_breakdown_print: |   - MTL0 (Apple M4 Pro) | 16384 = 16333 + (9822 =  4588 +    4402 +     832) +       -9772 |
common_memory_breakdown_print: |   - Host                |                  1602 =   545 +       0 +    1056                |
common_params_fit_impl: projected to use 9822 MiB of device memory vs. 16333 MiB of free device memory
common_params_fit_impl: will leave 6510 >= 1024 MiB of free device memory, no changes needed
common_fit_params: successfully fit params to free device memory
common_fit_params: fitting params to free memory took 0.21 seconds
llama_model_load_from_file_impl: using device MTL0 (Apple M4 Pro) (unknown id) - 16383 MiB free
llama_model_loader: loaded meta data with 46 key-value pairs and 427 tensors from models/Qwen3.5-9B-Q4_K_S.gguf (version GGUF V3 (latest))
llama_model_loader: Dumping metadata keys/values. Note: KV overrides do not apply in this output.
llama_model_loader: - kv   0:                       general.architecture str              = qwen35
llama_model_loader: - kv   1:                               general.type str              = model
llama_model_loader: - kv   2:                               general.name str              = Qwen3.5-9B
llama_model_loader: - kv   3:                           general.basename str              = Qwen3.5-9B
llama_model_loader: - kv   4:                       general.quantized_by str              = Unsloth
llama_model_loader: - kv   5:                         general.size_label str              = 9B
llama_model_loader: - kv   6:                            general.license str              = apache-2.0
llama_model_loader: - kv   7:                       general.license.link str              = https://huggingface.co/Qwen/Qwen3.5-9...
llama_model_loader: - kv   8:                           general.repo_url str              = https://huggingface.co/unsloth
llama_model_loader: - kv   9:                   general.base_model.count u32              = 1
llama_model_loader: - kv  10:                  general.base_model.0.name str              = Qwen3.5 9B
llama_model_loader: - kv  11:          general.base_model.0.organization str              = Qwen
llama_model_loader: - kv  12:              general.base_model.0.repo_url str              = https://huggingface.co/Qwen/Qwen3.5-9B
llama_model_loader: - kv  13:                               general.tags arr[str,2]       = ["unsloth", "image-text-to-text"]
llama_model_loader: - kv  14:                         qwen35.block_count u32              = 32
llama_model_loader: - kv  15:                      qwen35.context_length u32              = 262144
llama_model_loader: - kv  16:                    qwen35.embedding_length u32              = 4096
llama_model_loader: - kv  17:                 qwen35.feed_forward_length u32              = 12288
llama_model_loader: - kv  18:                qwen35.attention.head_count u32              = 16
llama_model_loader: - kv  19:             qwen35.attention.head_count_kv u32              = 4
llama_model_loader: - kv  20:             qwen35.rope.dimension_sections arr[i32,4]       = [11, 11, 10, 0]
llama_model_loader: - kv  21:                      qwen35.rope.freq_base f32              = 10000000.000000
llama_model_loader: - kv  22:    qwen35.attention.layer_norm_rms_epsilon f32              = 0.000001
llama_model_loader: - kv  23:                qwen35.attention.key_length u32              = 256
llama_model_loader: - kv  24:              qwen35.attention.value_length u32              = 256
llama_model_loader: - kv  25:                     qwen35.ssm.conv_kernel u32              = 4
llama_model_loader: - kv  26:                      qwen35.ssm.state_size u32              = 128
llama_model_loader: - kv  27:                     qwen35.ssm.group_count u32              = 16
llama_model_loader: - kv  28:                  qwen35.ssm.time_step_rank u32              = 32
llama_model_loader: - kv  29:                      qwen35.ssm.inner_size u32              = 4096
llama_model_loader: - kv  30:             qwen35.full_attention_interval u32              = 4
llama_model_loader: - kv  31:                qwen35.rope.dimension_count u32              = 64
llama_model_loader: - kv  32:                       tokenizer.ggml.model str              = gpt2
llama_model_loader: - kv  33:                         tokenizer.ggml.pre str              = qwen35
llama_model_loader: - kv  34:                      tokenizer.ggml.tokens arr[str,248320]  = ["!", "\"", "#", "$", "%", "&", "'", ...
llama_model_loader: - kv  35:                  tokenizer.ggml.token_type arr[i32,248320]  = [1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, ...
llama_model_loader: - kv  36:                      tokenizer.ggml.merges arr[str,247587]  = ["Ġ Ġ", "ĠĠ ĠĠ", "i n", "Ġ t",...
llama_model_loader: - kv  37:                tokenizer.ggml.eos_token_id u32              = 248046
llama_model_loader: - kv  38:            tokenizer.ggml.padding_token_id u32              = 248055
llama_model_loader: - kv  39:                    tokenizer.chat_template str              = {%- set image_count = namespace(value...
llama_model_loader: - kv  40:               general.quantization_version u32              = 2
llama_model_loader: - kv  41:                          general.file_type u32              = 14
llama_model_loader: - kv  42:                      quantize.imatrix.file str              = Qwen3.5-9B-GGUF/imatrix_unsloth.gguf
llama_model_loader: - kv  43:                   quantize.imatrix.dataset str              = unsloth_calibration_Qwen3.5-9B.txt
llama_model_loader: - kv  44:             quantize.imatrix.entries_count u32              = 248
llama_model_loader: - kv  45:              quantize.imatrix.chunks_count u32              = 80
llama_model_loader: - type  f32:  177 tensors
llama_model_loader: - type q8_0:   48 tensors
llama_model_loader: - type q4_K:  169 tensors
llama_model_loader: - type q5_K:   32 tensors
llama_model_loader: - type q6_K:    1 tensors
print_info: file format = GGUF V3 (latest)
print_info: file type   = Q4_K - Small
print_info: file size   = 5.01 GiB (4.81 BPW) 
load: 0 unused tokens
load: printing all EOG tokens:
load:   - 248044 ('<|endoftext|>')
load:   - 248046 ('<|im_end|>')
load:   - 248063 ('<|fim_pad|>')
load:   - 248064 ('<|repo_name|>')
load:   - 248065 ('<|file_sep|>')
load: special tokens cache size = 33
load: token to piece cache size = 1.7581 MB
print_info: arch                  = qwen35
print_info: vocab_only            = 0
print_info: no_alloc              = 0
print_info: n_ctx_train           = 262144
print_info: n_embd                = 4096
print_info: n_embd_inp            = 4096
print_info: n_layer               = 32
print_info: n_head                = 16
print_info: n_head_kv             = 4
print_info: n_rot                 = 64
print_info: n_swa                 = 0
print_info: is_swa_any            = 0
print_info: n_embd_head_k         = 256
print_info: n_embd_head_v         = 256
print_info: n_gqa                 = 4
print_info: n_embd_k_gqa          = 1024
print_info: n_embd_v_gqa          = 1024
print_info: f_norm_eps            = 0.0e+00
print_info: f_norm_rms_eps        = 1.0e-06
print_info: f_clamp_kqv           = 0.0e+00
print_info: f_max_alibi_bias      = 0.0e+00
print_info: f_logit_scale         = 0.0e+00
print_info: f_attn_scale          = 0.0e+00
print_info: n_ff                  = 12288
print_info: n_expert              = 0
print_info: n_expert_used         = 0
print_info: n_expert_groups       = 0
print_info: n_group_used          = 0
print_info: causal attn           = 1
print_info: pooling type          = -1
print_info: rope type             = 40
print_info: rope scaling          = linear
print_info: freq_base_train       = 10000000.0
print_info: freq_scale_train      = 1
print_info: n_ctx_orig_yarn       = 262144
print_info: rope_yarn_log_mul     = 0.0000
print_info: rope_finetuned        = unknown
print_info: mrope sections        = [11, 11, 10, 0]
print_info: ssm_d_conv            = 4
print_info: ssm_d_inner           = 4096
print_info: ssm_d_state           = 128
print_info: ssm_dt_rank           = 32
print_info: ssm_n_group           = 16
print_info: ssm_dt_b_c_rms        = 0
print_info: model type            = 9B
print_info: model params          = 8.95 B
print_info: general.name          = Qwen3.5-9B
print_info: vocab type            = BPE
print_info: n_vocab               = 248320
print_info: n_merges              = 247587
print_info: BOS token             = 11 ','
print_info: EOS token             = 248046 '<|im_end|>'
print_info: EOT token             = 248046 '<|im_end|>'
print_info: PAD token             = 248055 '<|vision_pad|>'
print_info: LF token              = 198 'Ċ'
print_info: FIM PRE token         = 248060 '<|fim_prefix|>'
print_info: FIM SUF token         = 248062 '<|fim_suffix|>'
print_info: FIM MID token         = 248061 '<|fim_middle|>'
print_info: FIM PAD token         = 248063 '<|fim_pad|>'
print_info: FIM REP token         = 248064 '<|repo_name|>'
print_info: FIM SEP token         = 248065 '<|file_sep|>'
print_info: EOG token             = 248044 '<|endoftext|>'
print_info: EOG token             = 248046 '<|im_end|>'
print_info: EOG token             = 248063 '<|fim_pad|>'
print_info: EOG token             = 248064 '<|repo_name|>'
print_info: EOG token             = 248065 '<|file_sep|>'
print_info: max token length      = 256
load_tensors: loading model tensors, this can take a while... (mmap = true, direct_io = false)
load_tensors: offloading output layer to GPU
load_tensors: offloading 31 repeating layers to GPU
load_tensors: offloaded 33/33 layers to GPU
load_tensors:   CPU_Mapped model buffer size =   545.62 MiB
load_tensors:  MTL0_Mapped model buffer size =  5133.75 MiB
............................................................................
common_init_result: added <|endoftext|> logit bias = -inf
common_init_result: added <|im_end|> logit bias = -inf
common_init_result: added <|fim_pad|> logit bias = -inf
common_init_result: added <|repo_name|> logit bias = -inf
common_init_result: added <|file_sep|> logit bias = -inf
llama_context: constructing llama_context
llama_context: n_seq_max     = 1
llama_context: n_ctx         = 262144
llama_context: n_ctx_seq     = 262144
llama_context: n_batch       = 512
llama_context: n_ubatch      = 512
llama_context: causal_attn   = 1
llama_context: flash_attn    = enabled
llama_context: kv_unified    = false
llama_context: freq_base     = 10000000.0
llama_context: freq_scale    = 1
ggml_metal_init: allocating
ggml_metal_init: found device: Apple M4 Pro
ggml_metal_init: picking default device: Apple M4 Pro
ggml_metal_init: use fusion         = true
ggml_metal_init: use concurrency    = true
ggml_metal_init: use graph optimize = true
llama_context:        CPU  output buffer size =     0.95 MiB
llama_kv_cache:       MTL0 KV buffer size =  4352.00 MiB
llama_kv_cache: size = 4352.00 MiB (262144 cells,   8 layers,  1/1 seqs), K (q8_0): 2176.00 MiB, V (q8_0): 2176.00 MiB
llama_kv_cache: attn_rot_k = 1, n_embd_head_k_all = 256
llama_kv_cache: attn_rot_v = 1, n_embd_head_k_all = 256
llama_memory_recurrent:       MTL0 RS buffer size =    50.25 MiB
llama_memory_recurrent: size =   50.25 MiB (     1 cells,  32 layers,  1 seqs), R (f32):    2.25 MiB, S (f32):   48.00 MiB
sched_reserve: reserving ...
sched_reserve: resolving fused Gated Delta Net support:
sched_reserve: fused Gated Delta Net (autoregressive) enabled
sched_reserve: fused Gated Delta Net (chunked) enabled
sched_reserve:       MTL0 compute buffer size =   832.60 MiB
sched_reserve:        CPU compute buffer size =   528.29 MiB
sched_reserve: graph nodes  = 1929
sched_reserve: graph splits = 2
sched_reserve: reserve took 8.72 ms, sched copies = 1
common_init_from_params: warming up the model with an empty run - please wait ... (--no-warmup to disable)
srv    load_model: initializing slots, n_slots = 1
common_context_can_seq_rm: the target context does not support partial sequence removal
srv    load_model: speculative decoding will use checkpoints
no implementations specified for speculative decoding
slot   load_model: id  0 | task -1 | new slot, n_ctx = 262144
srv    load_model: prompt cache is disabled - use `--cache-ram N` to enable it
srv    load_model: for more info see https://github.com/ggml-org/llama.cpp/pull/16391
srv          init: init: --cache-idle-slots requires --kv-unified, disabling
init: chat template, example_format: '<|im_start|>system
You are a helpful assistant<|im_end|>
<|im_start|>user
Hello<|im_end|>
<|im_start|>assistant
Hi there<|im_end|>
<|im_start|>user
How are you?<|im_end|>
<|im_start|>assistant
<think>

</think>

'
srv          init: init: chat template, thinking = 0
main: model loaded
main: server is listening on http://127.0.0.1:8000
main: starting the main loop...
srv  update_slots: all slots are idle
```

Лог приведен полностью, чтобы любознательные могли провести оценку процесса с помощью облачных LLM чатов. Далее, собственно, эта оценка и приводится.

### Оценка занятой памяти по логу загрузки модели

#### **Сводная таблица памяти**

| Компонент | Размер (MiB) | Размер (GB) | Примечание |
|-----------|-------------|-------------|------------|
| **Модель (веса)** | `5133.75` | **5.01 GB** | Q4_K_S, 8.95B параметров |
| **CPU Mapped** | `545.62` | 0.53 GB | Часть модели на CPU |
| **KV-кэш** | `4352.00` | **4.25 GB** | 262k токенов, q8_0 квантование |
| **RS-буфер (Mamba?)** | `50.25` | 0.05 GB | Для recurrent state |
| **Compute buffer (GPU)** | `832.60` | 0.81 GB | Промежуточные вычисления |
| **Compute buffer (CPU)** | `528.29` | 0.52 GB | Промежуточные вычисления на CPU |
| **Output buffer** | `0.95` | ~0.001 GB | Выходные данные |
| **ИТОГО на GPU** | `~10,369 MiB` | **~10.13 GB** | Сумма всех GPU-буферов |
| **ИТОГО на CPU** | `~1,075 MiB` | **~1.05 GB** | CPU-буферы |


#### **Детальная расшифровка**

##### **1. Модель: 5133.75 MiB (≈5.01 GB)**
```
load_tensors:  MTL0_Mapped model buffer size =  5133.75 MiB
```
Это основные веса модели, загруженные в GPU (unified memory). Формат `Q4_K_S` даёт около 4.81 BPW (bits per weight).

Плюс `545.62 MiB` на CPU — вероятно, часть весов, не поместившаяся или служебная.

##### **2. KV-кэш: 4352 MiB (4.25 GB)**
```
llama_kv_cache: size = 4352.00 MiB (262144 cells, 8 layers, 1/1 seqs)
llama_kv_cache: K (q8_0): 2176.00 MiB, V (q8_0): 2176.00 MiB
```

**Расчёт для 8 слоёв:**
- Контекст: 262,144 токена
- head_dim = 256
- n_head_kv = 4
- Размер на токен для 1 слоя: `2 (K+V) × 256 (dim) × 4 (heads) × 1 байт (q8_0) = 2048 байт = 2 KB`
- На 8 слоёв: `2 KB × 8 = 16 KB/токен`
- На 262k токенов: `16 × 262,144 = 4,194,304 KB = 4096 MB`

Протокол показывает `4352 MiB` — чуть больше из-за служебных данных.

**Важно:** В модели **32 слоя**, но в текущей конфигурации разрежённые Mamba-слои (точнее Gated Delta Networks -- наследуемые от Mamba) не все используют стандартный KV-кэш. Полный KV-кэш для 32 слоёв занял бы **~17 GB** (не влезло бы).

##### **3. RS-буфер: 50.25 MiB**
```
llama_memory_recurrent: size = 50.25 MiB (1 cells, 32 layers, 1 seqs)
```
Для recurrent слоёв (Mamba-подобные):
- R (f32): 2.25 MB
- S (f32): 48 MB

##### **4. Compute buffers: 1360.89 MiB (1.33 GB)**
```
sched_reserve:       MTL0 compute buffer size =   832.60 MiB  # GPU
sched_reserve:        CPU compute buffer size =   528.29 MiB  # CPU
```
Используются для:
- Промежуточных активаций
- Буферов внимания
- Временных тензоров

---

#### **Итоговая память**

- **GPU (Unified Memory, M4 Pro):**
```
Модель передана в GPU:  5134 MB (80% от модели)
KV-кэш (8 слоёв):      4352 MB
RS-буфер:                50 MB
Compute buffer GPU:     833 MB
─────────────────────────────
Итого GPU:           ~10,369 MB ≈ 10.13 GB
```

- **CPU:**
```
Остаток модели:         546 MB
Compute buffer CPU:     528 MB
Output buffer:            1 MB
─────────────────────────────
Итого CPU:            ~1,075 MB ≈ 1.05 GB
```

- **Общая занятая память:** **≈11.18 GB**

#### **Проверка по логу**

Последние строки лога, которые выводятся после завершения процесса сервера:
```
common_memory_breakdown_print: | memory breakdown [MiB]  | total    free    self   model   context   compute    unaccounted |
common_memory_breakdown_print: |   - MTL0 (Apple M4 Pro) | 16384 = 16333 + (9822 =  4588 +    4402 +     832) +       -9772 |
```

- `model: 4588 MiB` — близко к 5134 (разница в учёте)
- `context: 4402 MiB` — это KV-кэш + RS-буфер
- `compute: 832 MiB` — совпадает
- **Сумма: 9822 MiB** занято на GPU из 16384 MiB

Остаётся свободно: **~6510 MiB (6.36 GB)**

### Проверка доступности сервиса

__Скрипт для проверки доступных моделей.__

```bash
#!/bin/bash

. .env

curl -s -k http://$LOCAL_LLM_HOST:$LOCAL_LLM_PORT/v1/models \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $LOCAL_LLM_APIKEY" | jq
```

---

__Проверка моделей на бэкэнде.__

```bash
❯ ./show_models.sh
{
  "models": [
    {
      "name": "qwen35",
      "model": "qwen35",
      "modified_at": "",
      "size": "",
      "digest": "",
      "type": "model",
      "description": "",
      "tags": [
        ""
      ],
      "capabilities": [
        "completion"
      ],
      "parameters": "",
      "details": {
        "parent_model": "",
        "format": "gguf",
        "family": "",
        "families": [
          ""
        ],
        "parameter_size": "",
        "quantization_level": ""
      }
    }
  ],
  "object": "list",
  "data": [
    {
      "id": "qwen35",
      "aliases": [
        "qwen35"
      ],
      "tags": [],
      "object": "model",
      "created": 1777894180,
      "owned_by": "llamacpp",
      "meta": {
        "vocab_type": 2,
        "n_vocab": 248320,
        "n_ctx_train": 262144,
        "n_embd": 4096,
        "n_params": 8953803264,
        "size": 5383129088
      }
    }
  ]
}
```

Использованный алиас __qwen35__ позволит кратко именовать модель в настройках клиентов.

### Тестирование на коротком запросе

__Скрипт короткого теста.__

```bash
#!/bin/bash

. .env

curl -s http://${LOCAL_LLM_HOST}:${LOCAL_LLM_PORT}/v1/chat/completions \
  -H "Authorization: Bearer $LOCAL_LLM_APIKEY" \
  -H "Content-Type: application/json" \
  -d "{\"model\": \"$LOCAL_LLM_ALIAS\", \"messages\": [{\"role\": \"user\", \"content\": \"def fib(n):\"}], \"max_tokens\": 2048}" | \
  jq
```

---

__Результат:__

```bash
❯ ./test1.sh
{
  "choices": [
    {
      "finish_reason": "stop",
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "Here’s a simple recursive implementation of the Fibonacci function in Python:\n\n```python\ndef fib(n):\n    if n <= 0:\n        return 0\n    if n == 1:\n        return 1\n    return fib(n - 1) + fib(n - 2)\n```\n\nFor better performance (especially for larger `n`), you can use a memoized or iterative version:\n\n```python\n# Iterative version (efficient)\ndef fib(n):\n    if n <= 0:\n        return 0\n    if n == 1:\n        return 1\n    a, b = 0, 1\n    for _ in range(2, n + 1):\n        a, b = b, a + b\n    return b\n```\n\nIf you tell me your use case (e.g., very large `n`, need for speed, or working in another language), I can tailor it further."
      }
    }
  ],
  "created": 1777894483,
  "model": "qwen35",
  "system_fingerprint": "b8999-b97ebdc98",
  "object": "chat.completion",
  "usage": {
    "completion_tokens": 205,
    "prompt_tokens": 16,
    "total_tokens": 221,
    "prompt_tokens_details": {
      "cached_tokens": 0
    }
  },
  "id": "chatcmpl-IIiDqdMGMgRthQg6ZjboZcSDhCkkYUQj",
  "timings": {
    "cache_n": 0,
    "prompt_n": 16,
    "prompt_ms": 2042.064,
    "prompt_per_token_ms": 127.629,
    "prompt_per_second": 7.835209866096263,
    "predicted_n": 205,
    "predicted_ms": 5923.114,
    "predicted_per_token_ms": 28.89323902439024,
    "predicted_per_second": 34.61017295969654
  }
}
```

---

__Лог сервера:__

```bash
srv  params_from_: Chat format: peg-native
slot get_availabl: id  0 | task -1 | selected slot by LRU, t_last = -1
slot launch_slot_: id  0 | task -1 | sampler chain: logits -> ?penalties -> ?dry -> ?top-n-sigma -> top-k -> ?typical -> top-p -> min-p -> ?xtc -> temp-ext -> dist 
slot launch_slot_: id  0 | task 0 | processing task, is_child = 0
slot update_slots: id  0 | task 0 | new prompt, n_ctx_slot = 262144, n_keep = 0, task.n_tokens = 16
slot update_slots: id  0 | task 0 | n_tokens = 0, memory_seq_rm [0, end)
slot update_slots: id  0 | task 0 | prompt processing progress, n_tokens = 12, batch.n_tokens = 12, progress = 0.750000
slot update_slots: id  0 | task 0 | n_tokens = 12, memory_seq_rm [12, end)
slot init_sampler: id  0 | task 0 | init sampler, took 0.00 ms, tokens: text = 16, total = 16
slot update_slots: id  0 | task 0 | prompt processing done, n_tokens = 16, batch.n_tokens = 4
slot print_timing: id  0 | task 0 | 
prompt eval time =    2042.06 ms /    16 tokens (  127.63 ms per token,     7.84 tokens per second)
       eval time =    5923.11 ms /   205 tokens (   28.89 ms per token,    34.61 tokens per second)
      total time =    7965.18 ms /   221 tokens
slot      release: id  0 | task 0 | stop processing: n_tokens = 220, truncated = 0
srv  update_slots: all slots are idle
srv  log_server_r: done request: POST /v1/chat/completions 127.0.0.1 200
```

### Двухэтапный тест для компенсации разогрева

__Скрипт полного теста.__

```bash
#!/bin/bash

. .env

URL="http://${LOCAL_LLM_HOST}:${LOCAL_LLM_PORT}/v1/chat/completions"

echo "🔍 Тест 1: Короткий запрос (скорость ответа)"
time curl -s -X POST "$URL" \
  -H "Authorization: Bearer $LOCAL_LLM_APIKEY" \
  -H "Content-Type: application/json" \
  -d "{\"model\":\"$LOCAL_LLM_ALIAS\",\"messages\":[{\"role\":\"user\",\"content\":\"Напиши функцию Python для быстрой сортировки\"}],\"max_tokens\":2048}" | \
  tee test2.json | \
  jq -r '.choices[0].message.content'

echo -e "\n🔍 Тест 2: Длинный контекст (загрузка 8K токенов)"
# Генерируем промпт без переносов в bash-переменной
PROMPT=$(python3 -c "print('import numpy as np\n' * 200)")
# Безопасная сборка JSON через jq
PAYLOAD=$(jq -n \
  --arg model "$LOCAL_LLM_ALIAS" \
  --arg content "$PROMPT # Оптимизируй этот код, убери дубликаты и добавь типизацию" \
  --argjson max_tokens 2048 \
  '{
    model: $model,
    messages: [{role: "user", content: $content}],
    max_tokens: $max_tokens
  }')
# Отправка
time curl -s -X POST "$URL" \
  -H "Authorization: Bearer $LOCAL_LLM_APIKEY" \
  -H "Content-Type: application/json" \
  -d "$PAYLOAD" | \
  tee -a test2.json | \
  jq -r '.choices[0].message.content // "ОШИБКА: $(.error.message)"'

echo -e "\n💡 Проверьте Activity Monitor: Memory Pressure должен быть зеленым"
```

---

__Тестирование в два этапа:__

```bash
❯ ./test2.sh
🔍 Тест 1: Короткий запрос (скорость ответа)
Вот реализация алгоритма быстрой сортировки (QuickSort) на Python. Этот код использует метод «разделяй и властвуй»: выбор опорного элемента (pivot), разбиение массива на две части (меньшие и большие элементов) и рекурсивную сортировку этих частей.

```python
def quicksort(arr):
    # Базовый случай: если массив пуст или содержит один элемент, он уже отсортирован
    if len(arr) <= 1:
        return arr
    
    # Выбираем опорный элемент. Здесь используем последний элемент массива.
    # Для больших массивов лучше использовать медиану или случайный элемент.
    pivot = arr[-1]
    
    # Разделение массива на три части:
    # less — элементы меньше или равны пивоту
    # equal — элементы равные пивоту (оптимизация для многих одинаковых значений)
    # greater — элементы больше пивота
    less = [x for x in arr if x < pivot]
    equal = [x for x in arr if x == pivot]
    greater = [x for x in arr if x > pivot]
    
    # Рекурсивный вызов для подмассивов и их соединение
    return quicksort(less) + equal + quicksort(greater)

# Пример использования
if __name__ == "__main__":
    data = [3, 6, 8, 10, 1, 2, 1]
    print("Исходный массив:", data)
    sorted_data = quicksort(data)
    print("Отсортированный массив:", sorted_data)
```

### Как это работает:
1.  **Выбор пивота**: В данной реализации в качестве опорного элемента берется последний элемент списка.
2.  **Разделение (Partitioning)**: Мы создаем три новых списка:
    *   `less`: содержит все элементы, строго меньшие пивота.
    *   `equal`: содержит все элементы, равные пивота (это важно для стабильности производительности на массивах с повторяющимися значениями).
    *   `greater`: содержит все элементы, строго большие пивота.
3.  **Рекурсия**: Функция вызывается сама для списков `less` и `greater`, пока они не станут достаточно маленькими, чтобы обработаться напрямую.
4.  **Сборка**: Результаты собираются обратно в один отсортированный список.

### Примечание по производительности
Хотя этот код лаконичен и легко читается, он использует много памяти для создания новых списков на каждом шаге. В продакшн-коде часто используют *in-place* сортировку (сортировку на месте), которая меняет элементы в исходном массиве, не создавая копии, что снижает потребление памяти, но делает код более сложным.

real	0m19.835s
user	0m0.010s
sys	0m0.024s

🔍 Тест 2: Длинный контекст (загрузка 8K токенов)
Этот код содержит **100** дублирующихся импортов одной и той же библиотеки, что делает его избыточным, нечитаемым и потенциально проблемным для парсеров.

Вот оптимизированная версия с использованием **типа `List`** (стандартная библиотека Python), отсутствием дубликатов и добавлением **типов**:

```python
from typing import List

# Оптимизированный список импортов: один раз, с указанием типа для переменной (если бы она использовалась)
# В данном случае мы просто импортируем модуль один раз.
import numpy as np

# Пример использования с типизацией для демонстрации:
def calculate_mean(data: List[np.ndarray]) -> float:
    """
    Вычисляет среднее арифметическое списка массивов NumPy.
    
    Args:
        data: Список двумерных массивов NumPy.
        
    Returns:
        Среднее значение всех элементов.
    """
    if not data:
        return 0.0
    
    # Объединяем все массивы в один
    combined = np.concatenate(data, axis=0)
    return float(np.mean(combined))

# Пример вызова
# if __name__ == "__main__":
#     a = np.array([1, 2, 3])
#     b = np.array([4, 5, 6])
#     print(calculate_mean([a, b]))
```

### Что было сделано:
1.  **Удалены дубликаты**: Оставлен только один `import numpy as np`.
2.  **Добавлена типизация**:
    *   Использован `from typing import List` для аннотации типов в функции.
    *   Аргументы и возвращаемые значения функции объявлены с типами (`data: List[np.ndarray]`, `-> float`).
3.  **Структура**: Код стал лаконичным и профессиональным.

Если вы планируете использовать этот код в проекте, рекомендуется также добавить проверку на наличие зависимости (`pip install numpy`), хотя для простого скрипта импорта достаточно одного `import`.

real	0m17.560s
user	0m0.007s
sys	0m0.010s

💡 Проверьте Activity Monitor: Memory Pressure должен быть зеленым
```
```

---

__Сообщения в логе сервера:__

```bash
srv  params_from_: Chat format: peg-native
slot get_availabl: id  0 | task -1 | selected slot by LCP similarity, sim_best = 0.130 (> 0.100 thold), f_keep = 0.014
slot launch_slot_: id  0 | task -1 | sampler chain: logits -> ?penalties -> ?dry -> ?top-n-sigma -> top-k -> ?typical -> top-p -> min-p -> ?xtc -> temp-ext -> dist 
slot launch_slot_: id  0 | task 207 | processing task, is_child = 0
slot update_slots: id  0 | task 207 | new prompt, n_ctx_slot = 262144, n_keep = 0, task.n_tokens = 23
slot update_slots: id  0 | task 207 | n_past = 3, slot.prompt.tokens.size() = 220, seq_id = 0, pos_min = 219, n_swa = 0
slot update_slots: id  0 | task 207 | forcing full prompt re-processing due to lack of cache data (likely due to SWA or hybrid/recurrent memory, see https://github.com/ggml-org/llama.cpp/pull/13194#issuecomment-2868343055)
slot update_slots: id  0 | task 207 | n_tokens = 0, memory_seq_rm [0, end)
slot update_slots: id  0 | task 207 | prompt processing progress, n_tokens = 19, batch.n_tokens = 19, progress = 0.826087
slot update_slots: id  0 | task 207 | n_tokens = 19, memory_seq_rm [19, end)
slot init_sampler: id  0 | task 207 | init sampler, took 0.00 ms, tokens: text = 23, total = 23
slot update_slots: id  0 | task 207 | prompt processing done, n_tokens = 23, batch.n_tokens = 4
slot print_timing: id  0 | task 207 | 
prompt eval time =    1291.05 ms /    23 tokens (   56.13 ms per token,    17.81 tokens per second)
       eval time =   18493.08 ms /   633 tokens (   29.21 ms per token,    34.23 tokens per second)
      total time =   19784.13 ms /   656 tokens
slot      release: id  0 | task 207 | stop processing: n_tokens = 655, truncated = 0
srv  update_slots: all slots are idle
srv  log_server_r: done request: POST /v1/chat/completions 127.0.0.1 200
srv  params_from_: Chat format: peg-native
slot get_availabl: id  0 | task -1 | selected slot by LRU, t_last = 2437495521545
slot launch_slot_: id  0 | task -1 | sampler chain: logits -> ?penalties -> ?dry -> ?top-n-sigma -> top-k -> ?typical -> top-p -> min-p -> ?xtc -> temp-ext -> dist 
slot launch_slot_: id  0 | task 842 | processing task, is_child = 0
slot update_slots: id  0 | task 842 | new prompt, n_ctx_slot = 262144, n_keep = 0, task.n_tokens = 1032
slot update_slots: id  0 | task 842 | n_past = 3, slot.prompt.tokens.size() = 655, seq_id = 0, pos_min = 654, n_swa = 0
slot update_slots: id  0 | task 842 | forcing full prompt re-processing due to lack of cache data (likely due to SWA or hybrid/recurrent memory, see https://github.com/ggml-org/llama.cpp/pull/13194#issuecomment-2868343055)
slot update_slots: id  0 | task 842 | n_tokens = 0, memory_seq_rm [0, end)
slot update_slots: id  0 | task 842 | prompt processing progress, n_tokens = 512, batch.n_tokens = 512, progress = 0.496124
slot update_slots: id  0 | task 842 | n_tokens = 512, memory_seq_rm [512, end)
slot update_slots: id  0 | task 842 | prompt processing progress, n_tokens = 520, batch.n_tokens = 8, progress = 0.503876
slot update_slots: id  0 | task 842 | n_tokens = 520, memory_seq_rm [520, end)
slot update_slots: id  0 | task 842 | prompt processing progress, n_tokens = 1028, batch.n_tokens = 508, progress = 0.996124
slot create_check: id  0 | task 842 | created context checkpoint 1 of 32 (pos_min = 519, pos_max = 519, n_tokens = 520, size = 50.251 MiB)
slot update_slots: id  0 | task 842 | n_tokens = 1028, memory_seq_rm [1028, end)
slot init_sampler: id  0 | task 842 | init sampler, took 0.09 ms, tokens: text = 1032, total = 1032
slot update_slots: id  0 | task 842 | prompt processing done, n_tokens = 1032, batch.n_tokens = 4
slot create_check: id  0 | task 842 | created context checkpoint 2 of 32 (pos_min = 1027, pos_max = 1027, n_tokens = 1028, size = 50.251 MiB)
slot print_timing: id  0 | task 842 | 
prompt eval time =    3228.74 ms /  1032 tokens (    3.13 ms per token,   319.63 tokens per second)
       eval time =   14298.99 ms /   483 tokens (   29.60 ms per token,    33.78 tokens per second)
      total time =   17527.73 ms /  1515 tokens
slot      release: id  0 | task 842 | stop processing: n_tokens = 1514, truncated = 0
srv  update_slots: all slots are idle
srv  log_server_r: done request: POST /v1/chat/completions 127.0.0.1 200
```

---

__Первый запрос:__
```
prompt eval time =    1291.05 ms /    23 tokens (   56.13 ms per token,    17.81 tokens per second)
       eval time =   18493.08 ms /   633 tokens (   29.21 ms per token,    34.23 tokens per second)
      total time =   19784.13 ms /   656 tokens
```

__Второй запрос:__
```
prompt eval time =    3228.74 ms /  1032 tokens (    3.13 ms per token,   319.63 tokens per second)
       eval time =   14298.99 ms /   483 tokens (   29.60 ms per token,    33.78 tokens per second)
      total time =   17527.73 ms /  1515 tokens
```

Первый запрос задержался из-за загрузки, или так называемого "разогрева" модели, а второй вышел на требуемую скорость обработки промпта. Генерация и там и там производилась с ожидаемой скоростью не менее 30 токенов/с.

### Отчет по памяти

Сообщение сопровождающее завершение работы бэкэнда LLM
```bash
common_memory_breakdown_print: | memory breakdown [MiB]  | total   free     self   model   context   compute    unaccounted |
common_memory_breakdown_print: |   - MTL0 (Apple M4 Pro) | 16384 = 6014 + (10368 =  5133 +    4402 +     832) +           0 |
common_memory_breakdown_print: |   - Host                |                  1073 =   545 +       0 +     528                |
```

## Настройка локального сервиса AI

Далее везде, где встречается идентификатор __username__ его следует заменить на имя реальной учетной записи.

### Скрипт запуска

```bash
#!/bin/bash

source ${HOME}/llm-server/.env

log_file="$HOME/Library/Logs/llm-server.log"
# Функция логирования с временной меткой
log_event() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] [LAUNCHER] $*" >> "$log_file"
}

log_event "Запуск llm-server (PID: $$)"

${HOME}/llm-server/app/llama-server -m ${HOME}/llm-server/models/$LOCAL_LLM_MODEL \
  --port $LOCAL_LLM_PORT \
  --host $LOCAL_LLM_HOST \
  --api-key $LOCAL_LLM_APIKEY \
  --batch-size $LOCAL_LLM_BATCH \
  --ctx-size $LOCAL_LLM_CONTEXT \
  --n-gpu-layers $LOCAL_LLM_NGLAYERS \
  --flash-attn on \
  --threads $LOCAL_LLM_THREADS \
  --jinja \
  --cache-type-k $LOCAL_LLM_CACHE \
  --cache-type-v $LOCAL_LLM_CACHE \
  --timeout $LOCAL_LLM_TIMEOUT \
  --parallel $LOCAL_LLM_SLOTS \
  --alias $LOCAL_LLM_ALIAS \
  --cache-ram $LOCAL_LLM_CACHERAM \
  --reasoning $LOCAL_LLM_REASONING \
  --log-file "$log_file" 2>&1

EXIT_CODE=$?

# Логирование завершения
if [ $EXIT_CODE -eq 0 ]; then
    log_event "Сервер завершил работу штатно (код: $EXIT_CODE)"
else
    log_event "⚠️ Сервер упал с ошибкой (код: $EXIT_CODE) — launchd попытается перезапустить"
fi

# Возвращаем код выхода для launchd
exit $EXIT_CODE
```

### Манифест сервиса

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.user.llm-server</string>
    
    <key>ProgramArguments</key>
    <array>
        <string>/bin/bash</string>
        <string>/Users/username/llm-server/launch.sh</string>
    </array>
    
    <key>WorkingDirectory</key>
    <string>/Users/username/llm-server</string>
    
    <!-- Автозапуск при входе в систему -->
    <key>RunAtLoad</key>
    <true/>
    
    <!-- 
      КЛЮЧЕВОЙ ПАРАМЕТР: перезапускать ТОЛЬКО при краше.
      Если сервер выходит с кодом 0 (штатный --timeout) — НЕ перезапускать.
      Если код ≠ 0 (segfault, OOM, panic) — перезапустить.
    -->
    <key>KeepAlive</key>
    <dict>
        <key>SuccessfulExit</key>
        <false/>
        <key>Crashed</key>
        <true/>
    </dict>
    
    <!-- Защита от циклических перезапусков: ждать 30 сек между попытками -->
    <key>ThrottleInterval</key>
    <integer>30</integer>
    
    <!-- Приоритет процесса: Background, чтобы не мешать офисной работе -->
    <key>ProcessType</key>
    <string>Background</string>
    
    <!-- Перенаправление логов -->
    <key>StandardOutPath</key>
    <string>/Users/username/Library/Logs/llm-server.log</string>
    <key>StandardErrorPath</key>
    <string>/Users/username/Library/Logs/llm-server.log</string>
    
    <!-- Минимальное окружение -->
    <key>EnvironmentVariables</key>
    <dict>
        <key>PATH</key>
        <string>/usr/local/bin:/opt/homebrew/bin:/usr/bin:/bin:/usr/sbin:/sbin</string>
        <key>HOME</key>
        <string>/Users/username</string>
        <key>GGML_METAL_EMBEDDING</key>
        <string>1</string>
        <key>GGML_NO_ACCELERATE</key>
        <string>1</string>
    </dict>
</dict>
</plist>
```

---

__Манифест следует скопировать в папку автозапуска.__

```bash
❯ cp com.user.llm-server.plist ~/Library/LaunchAgents/
```

### Управление сервисом

1. **Загрузите сервис:**
   ```bash
   ❯ launchctl load ~/Library/LaunchAgents/com.user.llm-server.plist
   ```
2. **Проверьте статус:**
   ```bash
   ❯ launchctl load ~/Library/LaunchAgents/com.user.llm-server.plist
   ❯ launchctl list | grep llm-server
   14878	0	com.user.llm-server
   ```
3. **Смотрите лог в реальном времени:**
   ```bash
   tail -f ~/Library/Logs/llm-server.log
   ```
4. **Перезапуск/остановка:**
   ```bash
   launchctl unload ~/Library/LaunchAgents/com.user.llm-server.plist  # стоп
   launchctl load   ~/Library/LaunchAgents/com.user.llm-server.plist  # старт
  ```

## Настройка клиентов

Настройки клиентов размещаются в папке __clients__ .

Проверка работы клиентов производится one-shot запросом:

__"создай внутри рабочей области прокси oauth2 для подключения к локальной llm . используй golang"__

Результаты проверки размещаются внутри в папках __test-*__ .

---

### Qwen-code

#### Установка или апгрейд:

```bash
❯ brew install qwen-code
. . . .
==> Upgrading qwen-code
  0.15.2 -> 0.15.6 
==> Pouring qwen-code--0.15.6.arm64_sequoia.bottle.tar.gz
🍺  /opt/homebrew/Cellar/qwen-code/0.15.6: 114 files, 30.5MB
==> Running `brew cleanup qwen-code`...
Disable this behaviour by setting `HOMEBREW_NO_INSTALL_CLEANUP=1`.
Hide these hints with `HOMEBREW_NO_ENV_HINTS=1` (see `man brew`).
Removing: /opt/homebrew/Cellar/qwen-code/0.15.2... (111 files, 29.7MB)
Removing: /Users/username/Library/Caches/Homebrew/qwen-code_bottle_manifest--0.15.2... (36.7KB)
Removing: /Users/username/Library/Caches/Homebrew/qwen-code--0.15.2... (6.1MB)
```

---

#### Настройка

Описание конфигурации агента расположено в ~/.qwen/settings.json

```json
{
  "modelProviders": {
    "openai": [
      {
        "id": "qwen35",
        "name": "Qwen3.5-9B-Q4_K_S",
        "baseUrl": "http://127.0.0.1:8000",
        "description": "Local Qwen3.5-9B",
        "envKey": "LOCAL_LLM_APIKEY",
        "generationConfig": {
          "timeout": 300000,
          "maxRetries": 1,
          "contextWindowSize": "262144",
          "samplingParams": {
            "temperature": 0.6,
            "top_p": 0.9,
            "max_tokens": 8192,
            "presence_penalty": 0,
            "frequency_penalty": 0
          },
          "extra_body": {
            "enable_thinking": true
          }
        }
      }
    ]
  },
  "env": {
    "LOCAL_LLM_APIKEY": "token_as_a_lot_of_symbols"
  },
  "security": {
    "auth": {
      "selectedType": "openai"
    }
  },
  "model": {
    "name": "${LOCAL_LLM_ALIAS}"
  },
  "tools": {
    "approvalMode": "default"
  },
  "$version": 3
}
```

---

#### Тестовое задание

Запускаем:

```bash
❯ mkdir -p tmp/test-qwen
❯ cd tmp/test-qwen
❯ qwen

   ▄▄▄▄▄▄  ▄▄     ▄▄ ▄▄▄▄▄▄▄ ▄▄▄    ▄▄   ┌──────────────────────────────────────────────────────────┐
  ██╔═══██╗██║    ██║██╔════╝████╗  ██║  │ >_ Qwen Code (v0.15.6)                                   │
  ██║   ██║██║ █╗ ██║█████╗  ██╔██╗ ██║  │                                                          │
  ██║▄▄ ██║██║███╗██║██╔══╝  ██║╚██╗██║  │ API Key | qwen35 (/model to change)                      │
  ╚██████╔╝╚███╔███╔╝███████╗██║ ╚████║  │ ~/tmp/test-qwen                                          │
   ╚══▀▀═╝  ╚══╝╚══╝ ╚══════╝╚═╝  ╚═══╝  └──────────────────────────────────────────────────────────┘

  Tips: You can run any shell commands from Qwen Code using ! (e.g. !ls).

─────────────────────────────────────────────────────────────────────────────────────────────────────
>   Type your message or @path/to/file
```

Вводим запрос:
- создай внутри рабочей области прокси oauth2 для подключения к локальной llm . используй golang

Отчет о выполненной работе:

```bash
✦ Прокси OAuth2 для локальной LLM создан. Вот структура проекта:

    Файлы:
     - main.go - основной сервер с OAuth2 потоком и обработчиками
     - oauth2_token.go - OAuth2 клиент для обмена токенов
     - llm_client.go - клиент для связи с LLM (Ollama API)
     - go.mod - зависимости (gorilla/mux, securecookie, prometheus)
     - Dockerfile - Docker образ
     - README.md - документация

    Запуск:

     1 go run *.go

    API эндпоинты:
     - /login - OAuth2 вход
     - /api/generate - генерация текста
     - /api/generate/stream - стриминг
     - /api/chat - чат
     - /api/embeddings - эмбеддинги

    Окружение:

     1 export LLM_HOST=localhost
     2 export LLM_PORT=11434
     3 export OLLAMA_CLIENT_ID=llm-proxy-client
     4 export OLLAMA_CLIENT_SECRET=your-secret
     5 export OLLAMA_TOKEN_URL=http://localhost:11434/oauth/token
     6 export ADMIN_USER=admin
     7 export ADMIN_PASS=password
     8 export PORT=8080
```

Отчет об использовании контекста

```bash
  ╭────────────────────────────────────────────────────────────╮
  │                                                            │
  │  Context Usage                                             │
  │                                                            │
  │  Model: qwen35              Context window: 262.1k tokens  │
  │                                                            │
  │  ████████░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒  │
  │                                                            │
  │  █ Used                              38.1k tokens (14.6%)  │
  │  ░ Free                             145.4k tokens (55.4%)  │
  │  ▒ Autocompact buffer                78.6k tokens (30.0%)  │
  │                                                            │
  │  Usage by category                                         │
  │  █ System prompt                       5.7k tokens (2.2%)  │
  │  █ Built-in tools                      9.2k tokens (3.5%)  │
  │  █ Memory files                         169 tokens (0.1%)  │
  │  █ Skills                               802 tokens (0.3%)  │
  │  █ Messages                             346 tokens (0.1%)  │
  │                                                            │
  │  Built-in tools                                            │
  │    └ todo_write                               2.5k tokens  │
  │    └ run_shell_command                        1.4k tokens  │
  │    └ agent                                    1.4k tokens  │
  │    └ ask_user_question                         751 tokens  │
  │    └ edit                                      612 tokens  │
  │    └ web_fetch                                 511 tokens  │
  │    └ exit_plan_mode                            466 tokens  │
  │    └ read_file                                 415 tokens  │
  │    └ grep_search                               291 tokens  │
  │    └ glob                                      266 tokens  │
  │    └ list_directory                            240 tokens  │
  │    └ send_message                              134 tokens  │
  │    └ write_file                                133 tokens  │
  │    └ task_stop                                  92 tokens  │
  │                                                            │
  │  Memory files                                              │
  │    └ ../../.qwen/output-language.md            169 tokens  │
  │                                                            │
  │  Skills                                                    │
  │    └ qc-helper                                 112 tokens  │
  │    └ review                                    100 tokens  │
  │    └ loop                                       79 tokens  │
  │    └ batch                                      78 tokens  │
  │                                                            │
  ╰────────────────────────────────────────────────────────────╯
```

---

Кроме cli агента есть еще расширение для Vscode, называется Qwen Code Companion. В работе использует настройки cli агента

---

Результат представлен в папке __test-qwen__ .

Следует отметить сильную проработку кода, что обусловлено максимальной настройкой qwen-code на архитектуру моделей Qwen.

### Continue.dev

Поставляется в формате расширения для Vscode https://www.continue.dev/

---

Для continue.dev конфиг живёт в `~/.continue/config.json`. Далее файл настроек под установленный бэк:

```json
{
  "models": [
    {
      "title": "Qwen3.5-9B (local)",
      "provider": "openai",
      "model": "qwen35",
      "apiBase": "http://127.0.0.1:8000/v1",
      "apiKey": "token_as_a_lot_of_symbols",
      "contextLength": 262144,
      "completionOptions": {
        "temperature": 0.6,
        "topP": 0.95,
        "maxTokens": 8192,
        "presencePenalty": 0,
        "frequencyPenalty": 0
      },
      "requestOptions": {
        "timeout": 300000,
        "extraBodyProperties": {
          "enable_thinking": true,
          "thinking": {
            "budget_tokens": 4096
          }
        }
      }
    }
  ],

  "tabAutocompleteModel": {
    "title": "Qwen3.5-9B (autocomplete)",
    "provider": "openai",
    "model": "qwen35",
    "apiBase": "http://127.0.0.1:8000/v1",
    "apiKey": "token_as_a_lot_of_symbols",
    "contextLength": 65536,
    "completionOptions": {
      "temperature": 0.2,
      "maxTokens": 512
    },
    "requestOptions": {
      "timeout": 30000,
      "extraBodyProperties": {
        "enable_thinking": false
      }
    }
  },

  "embeddingsProvider": {
    "provider": "transformers.js"
  },

  "slashCommands": [
    { "name": "edit",    "description": "Edit selected code" },
    { "name": "comment", "description": "Add comments to code" },
    { "name": "share",   "description": "Export chat as markdown" },
    { "name": "cmd",     "description": "Generate a shell command" }
  ],

  "contextProviders": [
    { "name": "code" },
    { "name": "docs" },
    { "name": "diff" },
    { "name": "terminal" },
    { "name": "problems" },
    { "name": "folder" },
    { "name": "codebase" }
  ],

  "allowAnonymousTelemetry": false
}
```

---

**Что важно знать по каждому блоку:**

**`models`** — основная чат-модель. Thinking включён, параметры те же, что в qwen-code.

**`tabAutocompleteModel`** — отдельная конфигурация для инлайн-автодополнения. Здесь `enable_thinking: false` принципиально — thinking mode для автокомплита бесполезен (медленно, большой overhead), а `maxTokens: 512` отражает реальные нужды. Temperature 0.2 — детерминированнее для кода.

**`embeddingsProvider: transformers.js`** — работает локально в самом VS Code без дополнительного сервера, не нагружает llama-server. Используется для `@codebase` контекста.

**`contextProviders`** — включи `codebase` только если проект небольшой, он индексирует весь репо через embeddings. Для больших монорепо лучше убрать или настроить `.continueignore`.

---

**Важные моменты:**

1. **`tabAutocompleteModel`** — если `qwen35` не обучена на FIM (fill-in-the-middle) токены, автодополнение будет работать плохо. Для autocomplete обычно используют специализированные модели (Qwen2.5-Coder, DeepSeek-Coder, Codellama и т.д.).
2. **`embeddingsProvider: transformers.js`** — в Continue.dev этот провайдер работает только если расширение запущено в режиме с доступом к WASM/браузерному движку. Если индексация codebase не работает, замени на `"openai"` с тем же `apiBase` (если сервер поддерживает эмбеддинги) или используй отдельную модель эмбеддингов.

---

```bash
❯ cd
❯ mkdir -p tmp/test-continue
❯ cd tmp/test-continue
❯ code .
```

Поручаем то же самое задание

```bash
создай внутри рабочей области прокси oauth2 для подключения к локальной llm . используй golang


Agent

Qwen3.5-9B (local)
⏎
Для реализации OAuth2 прокси для локального LLM на Go, нам нужно создать сервер, который:

Принимает запросы от клиента.
Проверяет/получает токен доступа.
Перепроксирует запрос к локальной модели (например, Ollama или локальный API).
Возвращает ответ обратно клиенту.
Я создам структуру проекта с основным файлом main.go и конфигурацией.

Continue will create proxy/oauth2/main.go
```

---

И вот результат в окне расширения:

```bash
Код уже написан с использованием os.Getenv(), поэтому он готов к работе. Вам нужно будет только:

Установить зависимости: go mod tidy
Запустить проект: go run main.go
Установить переменные окружения перед запуском (например, через export в терминале или в .env файле, который вы создадите вручную в своей ОС).
Вот как установить переменные окружения для запуска:

Bash
Run
export OAUTH_CLIENT_ID="your_client_id"
export OAUTH_CLIENT_SECRET="your_client_secret"
export OAUTH_TOKEN_URL="http://localhost:8080/oauth/token"
export LLM_API_URL="http://localhost:11434/api"
export PORT="8080"

cd proxy/oauth2
go run main.go
Проект готов к использованию!
```

Результат в коде представлен в папке __test-continue__ .

И надо заметить, это самый слабый результат!

### Claude Code

Устанавливаем https://code.claude.com/docs/en/quickstart

```bash
❯ brew install --cask claude-code
```

---

Файл с настройками в ~/.claude/settings.json

```json
{
  "env": {
    "ANTHROPIC_BASE_URL": "http://127.0.0.1:8000",
    "ANTHROPIC_API_KEY": "token_as_a_lot_of_symbols",
    "CLAUDE_CODE_ATTRIBUTION_HEADER": "0"
  }
}
```

---

Создаем тестовое окружение:

```bash
❯ mkdir -p tmp/test-claude
❯ cd tmp/test-claude
```

---

Запуск без VPN не работает:

```bash
❯ claude --model qwen35
Welcome to Claude Code v2.1.118
…………………………………………………………………………………………………………………………………………………………

     *                                       █████▓▓░
                                 *         ███▓░     ░░
            ░░░░░░                        ███▓░
    ░░░   ░░░░░░░░░░                      ███▓░
   ░░░░░░░░░░░░░░░░░░░    *                ██▓░░      ▓
                                             ░▓▓███▓▓░
 *                                 ░░░░
                                 ░░░░░░░░
                               ░░░░░░░░░░░░░░░░
       █████████                                        *
      ██▄█████▄██                        *
       █████████      *
…………………█ █   █ █………………………………………………………………………………………………………………

 Unable to connect to Anthropic services

 Failed to connect to api.anthropic.com: ERR_BAD_REQUEST

 Please check your internet connection and network settings.

 Note: Claude Code might not be available in your country. Check supported countries at https://anthropic.com/supported-countries
 ```

 ---

После включения VPN , можно уже залогиниться и приступить к работе.

```bash
❯ source ~/.claude/.env
❯ claude --model qwen35
╭─── Claude Code v2.1.118 ───────────────────────────────────────────────────────────────────────────╮
│                                │ Tips for getting started                                          │
│          Welcome back!         │ Run /init to create a CLAUDE.md file with instructions for Claude │
│                                │ ───────────────────────────────────────────────────────────────── │
│             ▐▛███▜▌            │ Recent activity                                                   │
│            ▝▜█████▛▘           │ No recent activity                                                │
│              ▘▘ ▝▝             │                                                                   │
│                                │                                                                   │
│   qwen35 · API Usage Billing   │                                                                   │
│       ~/tmp/test-claude        │                                                                   │
╰────────────────────────────────────────────────────────────────────────────────────────────────────╯
```

После авторизации можно отключить VPN и далее использовать агента claude code с локальной моделью и без необходимого VPN.

Поручаем то же самое задание:
- создай внутри рабочей области прокси oauth2 для подключения к локальной llm . используй golang


```bash
⏺ Go не установлен в этой среде. Вот файлы которые я создал:
 
  Созданные файлы:

  1. main.go - OAuth2 прокси сервер:
    - Поддержка OAuth2 client credentials flow 
    - Reverse proxy к локальной LLM (по умолчанию localhost:11434 - Ollama)
    - Проверка токенов и добавление Authorization header 
    - Генерация self-signed TLS сертификатов для локальной разработки
  2. go.mod - зависимости (golang.org/x/oauth2)
  3. README.md - документация 
 
  Как использовать:
  # Установка зависимостей 
  go mod tidy

  # Запуск прокси 
  go run main.go 
 
  # Подключение к LLM через прокси
  curl http://localhost:8080/api/generate \ 
    -H "Authorization: Bearer YOUR_OAUTH_TOKEN" 
  
  1 tasks (0 done, 1 open)
  ◻ Create OAuth2 proxy for local LLM
```

---

Отчет об использовании модели:

```bash
❯ /context
  ⎿  Context Usage
     ⛁ ⛁ ⛁ ⛁ ⛁ ⛁ ⛁ ⛁ ⛁ ⛁   qwen35
     ⛁ ⛁ ⛀ ⛀ ⛶ ⛶ ⛶ ⛶ ⛶ ⛶   25.3k/200k tokens (13%)
     ⛶ ⛶ ⛶ ⛶ ⛶ ⛶ ⛶ ⛶ ⛶ ⛶                                                                                                      
     ⛶ ⛶ ⛶ ⛶ ⛶ ⛶ ⛶ ⛶ ⛶ ⛶   Estimated usage by category
     ⛶ ⛶ ⛶ ⛶ ⛶ ⛶ ⛶ ⛶ ⛶ ⛶   ⛁ System prompt: 5.8k tokens (2.9%)
     ⛶ ⛶ ⛶ ⛶ ⛶ ⛶ ⛶ ⛶ ⛶ ⛶   ⛁ System tools: 18.1k tokens (9.1%)
     ⛶ ⛶ ⛶ ⛶ ⛶ ⛶ ⛶ ⛶ ⛶ ⛶   ⛁ Skills: 724 tokens (0.4%)                                                                       
     ⛶ ⛶ ⛶ ⛶ ⛶ ⛶ ⛶ ⛶ ⛶ ⛶   ⛁ Messages: 1k tokens (0.5%)
     ⛶ ⛶ ⛶ ⛝ ⛝ ⛝ ⛝ ⛝ ⛝ ⛝   ⛶ Free space: 141.3k (70.7%)
     ⛝ ⛝ ⛝ ⛝ ⛝ ⛝ ⛝ ⛝ ⛝ ⛝   ⛝ Autocompact buffer: 33k tokens (16.5%)

     Skills · /skills

❯ /status
    
───────────────────────────────────────────────────────────────────────────────────
   Status   Config   Usage   Stats
                                                                                                                              
  Session
  Total cost:            $0.1532 (costs may be inaccurate due to usage of unknown models)
  Total duration (API):  3m 23s 
  Total duration (wall): 7m 22s
  Total code changes:    0 lines added, 0 lines removed   
  Usage by model: 
      claude-haiku-4-5:  205 input, 12 output, 0 cache read, 0 cache write ($0.0003)
                qwen35:  50.4k input, 108 output, 0 cache read, 0 cache write ($0.1529)
 
  Esc to cancel
```

Результат представлен в папке __test-claude__ .

---

### Kilocode

#### Установка

1. Консольная утилита

Исходный код https://github.com/kilo-org/kilocode

Пакет установки https://github.com/Kilo-Org/homebrew-tap

```bash
❯ brew install Kilo-Org/homebrew-tap/kilo
```

2. Расширение Vscode

Ставится через поиск в библиотеке расширений.

#### Настройка

1. С помощью файла конфигурации

Конфиг лежит в корне проекта или глобально, формат `opencode.json`. Для custom/local моделей обязательно указывать `limit.context` и `limit.output` — без них автоматическое управление контекстом отключается и разговор будет расти неограниченно до отказа провайдера.

`~/.kilo/opencode.json` (или в корне проекта `.kilo/opencode.json`):

```json
{
  "$schema": "https://app.kilo.ai/config.json",
  "model": "llama-local/qwen35",
  "provider": {
    "llama-local": {
      "name": "llama-server local",
      "npm": "@ai-sdk/openai-compatible",
      "options": {
        "baseURL": "http://127.0.0.1:8000/v1",
        "apiKey": "token_as_a_lot_of_symbols"
      },
      "models": {
        "qwen35": {
          "id": "qwen35",
          "name": "Qwen3.5-9B (local)",
          "tool_call": true,
          "limit": {
            "context": 262144,
            "output": 8192
          }
        }
      }
    }
  }
}
```

---

**Важный момент про timeout:**

Для локальных моделей дефолтный таймаут 10 минут. Если модель медленная — увеличь: VS Code Extensions panel → Kilo Code gear → Settings → **API Request Timeout**.

После настройки проверь что llama-server запущен и модель доступна — в Kilo Code panel должен появиться провайдер `llama-server local` с моделью `Qwen3.5-9B (local)` в пикере.

2. Через UI

В VS Code открой Kilo Code panel → шестерёнка → **Providers** → листай вниз → **Custom provider**:

- **Provider ID:** `llama-local`
- **Name:** `llama-server local`
- **Base URL:** `http://127.0.0.1:8000/v1`
- **API Key:** `token_as_a_lot_of_symbols`

Затем в поле модели введи `qwen35`.

#### Запуск в консоли

Настраиваем тестовое окружение.

```bash
❯ mkdir -p tmp/test-kilo
❯ cd tmp/test-kilo
❯ kilo
```

Поручаем то же самое
- создай внутри рабочей области прокси oauth2 для подключения к локальной llm . используй golang

---

Результат в консоли агента:

```bash
Создан OAuth2 прокси для подключения к локальной LLM в /Users/username/tmp/test-kilo/proxy-oauth2/:
Файлы:
- proxy.go — основной код прокси
- config.json — конфигурация
- oauth2-proxy — скомпилированный бинарный файл
- start.sh — скрипт запуска
Запуск:
cd /Users/username/tmp/test-kilo/proxy-oauth2
./start.sh
Или вручную:
./oauth2-proxy
Назначение:
- Прокси перенаправляет запросы на localhost:11434 (ollama) через OAuth2 аутентификацию
- Генерирует клиентские сертификаты для мTLS
- Валидирует токены через JWKS
- Поддерживает мTLS аутентификацию
```

---

Отчет о расходовании токенов.

```bash
OAuth2 прокси для локальной LLM на Go

Context
17,540 tokens
7% used
$0.00 spent

Token Usage
Input.     24,794
Output.     4,759
Cached.   255,144

LSP
LSPs will activate as files are read
```

---

Результат в коде представлен в папке __test-kilo__ .

Надо отметить, что это единственный агент, который не только написал код, но и собрал его, тем самым проверил как минимум непротиворечивость синтаксиса.

### Opencode

https://github.com/anomalyco/opencode

#### Установка

```bash
❯ brew install --cask opencode-desktop
```

Это десктопное приложение и установка cli происходит из меню после старта.

#### Настройка

OpenCode использует тот же формат что и Kilo CLI (Kilo — форк OpenCode).

Конфиг `~/.config/opencode/opencode.json` (или в корне проекта `opencode.json`):

```json
{
  "$schema": "https://opencode.ai/config.json",
  "model": "llama-local/qwen35",
  "provider": {
    "llama-local": {
      "name": "llama-server local",
      "npm": "@ai-sdk/openai-compatible",
      "options": {
        "baseURL": "http://127.0.0.1:8000/v1",
        "apiKey": "token_as_a_lot_of_symbols"
      },
      "models": {
        "qwen35": {
          "id": "qwen35",
          "name": "Qwen3.5-9B (local)",
          "tool_call": true,
          "limit": {
            "context": 262144,
            "output": 8192
          }
        }
      }
    }
  }
}
```

---

#### Запуск

```bash
❯ mkdir -p tmp/test-opencode
❯ cd tmp/test-opencode
❯ opencode
```

Поручаем то же самое
- создай внутри рабочей области прокси oauth2 для подключения к локальной llm . используй golang

---

Результат в консоли агента:

```bash
Готово! OAuth2 прокси для локальной LLM создан:
Структура:
oauth2-proxy-llm/
├── main.go      # основной код
└── go.mod       # зависимости
Эндпоинты:
- http://localhost:3000/proxy - прокси запросы к LLM (требует Bearer token)
- http://localhost:3001/oauth2/authorize - авторизация OAuth2 (авторизационный сервер)
- http://localhost:3001/oauth2/token - получение токена
Использование:
1. Авторизуйтесь: http://localhost:3001/oauth2/authorize
2. Получите токен
3. Используйте в запросах: Authorization: Bearer <token>
Прокси переадресовывает запросы на http://localhost:11434 (LLM).
```

Отчет о расходовании токенов.
```bash
OAuth2 прокси для локальной LLM на Golang

Context
13,984 tokens
5% used
$0.00 spent

LSP
LSPs will activate as files are read
```

Результат в коде представлен в папке __test-opencode__ .

### А почему не проверили codex ?

Проверили! Однако продукты OpenAI перестали использовать эндпойнт __v1/chat/comletitions__ и вместо этого перешли на эндпойниг __v1/responses__ , который не полностью поддерживается в доступной сейчас версии __llama.cpp__. 

## Заключение.

Оказывается, локально развернутый бэкэнд даже с моделью размером 9B параметров является уже не забавной игрушкой, а вполне состоятельным инструментом.

С его помощью легко решаются вопросы анализа кода, оценка качества PR, разработка планов и поэтапное их исполнение, что за счет контекста в 256K позволяет решать много разнообразных задач. Конечно, надо быть готовым к неспешному общению. Например оценка PR занимает от 3 минут, разработка плана или анализ кода может длиться до 10 - 15 минут. Время затраченное на перевод текста, или контроль грамматики, или синтаксиса сильно зависит от размера текста. Надо учитывать то, что обычно агенты не допускают занятие контекста более 70%, потом происходит сжатие, что рано или поздно приведет к потере точности ответов агента. 

Одинаковые задания, порученные разным агентам и выполненные с разным качеством, доказывают, что сложные агентские приложения могут сильно улучшать качество генерации за счет неочевидных внутренних алгоритмов. Такие приложения помимо обслуживания цепочки обсуждений (Chain of Thought) и предоставления инструментов (tools) в ReAct взаимодействие, могут также выступать инициаторами дополнительно размышления над задачей. И в таком случае размер контекста становится ключевым параметром. Именно это дает возможность вести длинное обсуждение без потери качества.

Все агенты, использованные в обзоре, работали в настройке по умолчанию, без привлечения дополнительных навыков (skill) или специально настроенных субагентов (subagent) и при этом справились с тестовым заданием, хотя и с разным успехом. 

---

Сравнение результатов тестовых заданий дает основания для ряда оценок:
- расширение __Continue.dev__ практически устаревший инструмент и не стоит его использовать.
- агент __claude code__, предполагаю, раскрывается во всей мощности только на больших проприетарных моделях, на Qwen3.5 он не смог продемонстрировать ничего достойного.
- очень привлекательно выглядит __Kilocode__, который позволяет достойно работать как внутри расширения Vscode, так и в качестве cli агента.
- агент __opencode__, который является прорадителем Kilocode, мало отличается от последнего, и так как не имеет опции компаньена Vscode, то можно его далее и не рассматривать.
- особняком стоит проект __qwen-code__, который показал самый развитый проект в тестовом задании. Этот тоже имеет компаньен для Vscode, но по удобству немного хуже, чем Kilocode. Однако как cli агент он обладает множеством преимуществ и это единственный проект, который внутри своего репозитория содержит очень подробную документацию и ADR внутри.

---

Выводы:
- самым подходящим для работы в UI окружени Vscode стоит признать __Kilocode__.
- работа в cli режиме будет комфортной как в __Kilocode__ так и в __qwen-code__.
- для использования агента в headless режиме, в том числе в CI/CD пайплайнах, самым подходящим стоит признать __qwen-code__.
  

