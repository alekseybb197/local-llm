# Установка локального LLM бэкэнда Lama.cpp на MacBook Pro M4 24G

Далее будут описаны шаги, необходимые для установки локального бэкэнда LLM на macbook.

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

### Устанавливаем бэкэнд llama.cpp .

```bash
❯ cd
❯ mkdir llm-server
❯ cd llm-server
❯ wget https://github.com/ggml-org/llama.cpp/releases/download/b9222/llama-b9222-bin-macos-arm64-kleidiai.tar.gz
❯ tar xzvf llama-b9222-bin-macos-arm64-kleidiai.tar.gz
❯ ln -s llama-b9222 app
```

Здесь указана ссылка, актуальная на день создания документа. Рекомендуется выбирать последнюю доступную версию из списка релизов.

---

### Проверяем доступную память и оборудование.

```bash
❯ app/llama-bench --help >/dev/null
ggml_metal_device_init: tensor API disabled for pre-M5 and pre-A19 devices
ggml_metal_library_init: using embedded metal library
ggml_metal_library_init: loaded in 0.024 sec
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

На сайте Huggingface представлена огромная коллекция моделей. В данной статье не будем рассматривать как и почему выбрано именно семейство Qwen 3.5. Лучше конечно 3.6, но надо учитывать выше указанный предел __17179.89 MB__. И остается совсем не много доступных для работы вариантов моделей, которые могут уложиться в такой объем памяти даже с учетом доступной квантизации. Фактически их три - 9B, 14B, 27B. Одного нет в линейке доступного семейства Qwen и выбор уже между Qwen3.5-9B и Qwen3.6-27B. 

Очевидно, что 9B модель можно взять с менее агрессивной квантизацией, а вот 27B придется выбрать совсем жестко пожатую. И даже в этих двух случаях кодирования далее весь trade-off будет строиться вокруг сжатия kv кэша и достижимого размера контекста.

Если бы здесь рассматривался macbook с памятью не менее 48G, то решение с Qwen3.6-27B смотрелось бы вполне привлекательно. Однако на macbook с 24G даже после самой агрессивной квантизации модель Qwen3.6-27B занимает критически много памяти, так что для других приложений практически ничего не остается.

Следующий выбор в том, брать ли оригинальную, но квантизированную модель, или fine-tuned/distilled, подготовленную для определенного класса задач.

Решение в пользу дообученных моделей должно приниматься с учетом характера использования. Например, если к бэкэнду будет обращаться только агент, настроенный на решение больших задач с построением длинных цепочек рассуждений, то применение модели дистиллированной под Opus 4.6 даст выигрыш.

Учитывая, что на используемом macbook нет технической возможности разместить более одной модели LLM, то очевидно выбираем base-line модель. Такие модели предоставляет Unsloth (https://unsloth.ai/). Спектр вариантов квантизации очень большой. Выбираем средний уровень квантизации из стремления получить модель в объеме примерно 5Gb.

Дополнительным фактором выбора является предпочтительное использование технологии MTP (Multi-Token Prediction, многотокенное предсказание), которая совсем недавно была поддержана в Llama.cpp.

Итак, наш выбор https://huggingface.co/unsloth/Qwen3.5-9B-MTP-GGUF .

## Подбор параметров запуска.

Данный шаг необходим в процессе подбора параметров запуска модели. Весь trade-off происходит вокруг выделения памяти для работы сервиса. Критическая граница, в понимании llama.cpp это указанный выше llama-bench размер памяти 17179.89 MB. Однако далее сервер снижает её до примерно 16G. И подбор комфортных для работы параметров следует исходит из оценки сервера llama.cpp .

Побдор происходит путем настройки параметров и оценки степени загрузки, что получилась в итоге. На этапе подбора уровень логгирования держим 4 для уверенного анализа логов запуска.

### Способы экономии памяти.

1. Выбор модели с более сильной квантизацией. Для Qwen3.5-9B варианты можно подобрать на странице https://huggingface.co/unsloth/Qwen3.5-9B-MTP-GGUF. Рассматриваем именно MTP для ускорения инференса, что в dense модели должно работать особенно хорошо. Подбор проводится в 4-х битном квантовании:
- Q4_K_M базовый вариант 5.87Gb -- скушновато.
- Q4_K_S компромиссный выбор 5.58Gb -- проверено и надежно.
- UD-Q4_K_XL последний продвинутый квант с динамической настройкой 6.14Gb -- наш выбор

Здесь приводятся файловые объемы, в загрузке будет скорее всего немного меньше.

2. Размер контекста. Для данной модели максимум 262144. Надо понимать, что этот размер фактически задает мощность достижимых решений. Заданный размер кратно сказывается на потреблении памяти. В первую очередь он фактически определяет размер кэша, а в случае разрешения кеширования в RAM или параллельной обработки будет мультиплицироваться. Варианты:
- 262144 -- идеальный выбор.
- 131072 -- минимальное разумное значение, которое экономит память вдвое. 
- 65536 -- еще большая экономия, но модель становится практически игрушкой.

Обычно агенты по достижении 70% занятости контекста сжимают его. Значит перечисленные размеры контекста в реальной работе будут соответственно ниже.

3. Квантизация контекста. Варианты определяются возможностями бэкэнда llama.cpp . И скаждым релизом список квантов растет. Квантизацию в llama.cpp можно указывать раздельно для ключей и значений. Главное правило -- квант значений не может быть сильнее, чем квант ключей, так как это может привести к значительной деградации ответов в длинных рассуждениях. Варианты кватизации по убыванию точности:
- q8_0 -- безопасная замена FP16, которая еще и ускоряет инференс, можно сказать технологиченский идеал.
- q5_0/q5_1 -- компромиссный вариант, если никак не получается работать с q8_0 .
- iq4_nl -- Improved 4-bit Non-Linear. Появился в последних релизах llama.cpp .
- q4_1 -- более стабильный чем q4_0 вариант 4-х битного сжатия.
- q4_0 -- агрессивное сжатие, крайний вариант для очень ограниченной памяти.

Кроме очевидной экономии памяти, квантизация контекста добавляет накладные расходы декодирования. На контексте даже 65536 использование q4_0 может драматически замедлить обработку промпта!

Остальные параметры или сказываются не сильно или очевидны в выборе.

---

### Запуск бэкэнда

#### Скрипт тестового запуска.

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
  --kv-unified \
  --no-warmup \
  --threads $LOCAL_LLM_THREADS \
  --jinja \
  --cache-type-k $LOCAL_LLM_CACHEK \
  --cache-type-v $LOCAL_LLM_CACHEV \
  --spec-type $LOCAL_LLM_SPEC \
  --spec-draft-n-max $LOCAL_LLM_NMAX \
  --timeout $LOCAL_LLM_TIMEOUT \
  --parallel $LOCAL_LLM_SLOTS \
  --alias $LOCAL_LLM_ALIAS \
  --cache-ram $LOCAL_LLM_CACHERAM \
  --verbosity $LOCAL_LLM_VERBOSITY \
  --reasoning $LOCAL_LLM_REASONING
```

---

#### Запуск бэкэнда для оценки потребления памяти.

Очевидно, что это **НЕ** первый запуск, но в данном случае предпоследний. Приводится для демонстрации процесса подбора параметров.

Запускаем с параметрами:

```bash
❯ ./server.sh
-----
GGML_METAL_EMBEDDING=1
LOCAL_LLM_ALIAS=qwen35
LOCAL_LLM_APIKEY=token_as_a_lot_of_simbols
LOCAL_LLM_BATCH=512
LOCAL_LLM_CACHEK=q8_0
LOCAL_LLM_CACHERAM=0
LOCAL_LLM_CACHEV=q8_0
LOCAL_LLM_CONTEXT=131072
LOCAL_LLM_HOST=127.0.0.1
LOCAL_LLM_MODEL=Qwen3.5-9B-MTP-UD-Q4_K_XL.gguf
LOCAL_LLM_NGLAYERS=999
LOCAL_LLM_NMAX=6
LOCAL_LLM_PORT=8000
LOCAL_LLM_REASONING=off
LOCAL_LLM_SLOTS=1
LOCAL_LLM_SPEC=draft-mtp
LOCAL_LLM_THREADS=8
LOCAL_LLM_TIMEOUT=900
LOCAL_LLM_VERBOSITY=4
```

В логе ищем первые блок строк, посвященных выделению памяти:

```bash
0.00.228.719 I ggml_metal_init: allocating
0.00.228.752 I ggml_metal_init: found device: Apple M4 Pro
0.00.228.756 I ggml_metal_init: picking default device: Apple M4 Pro
0.00.228.790 I ggml_metal_init: use fusion         = true
0.00.228.791 I ggml_metal_init: use concurrency    = true
0.00.228.791 I ggml_metal_init: use graph optimize = true
0.00.228.843 I llama_context:        CPU  output buffer size =     0.95 MiB
0.00.229.440 I llama_kv_cache:       MTL0 KV buffer size =     0.00 MiB
0.00.229.443 I llama_kv_cache: size = 2176.00 MiB (131072 cells,   8 layers,  1/1 seqs), K (q8_0): 1088.00 MiB, V (q8_0): 1088.00 MiB
0.00.229.443 I llama_kv_cache: attn_rot_k = 1, n_embd_head_k_all = 256
0.00.229.444 I llama_kv_cache: attn_rot_v = 1, n_embd_head_k_all = 256
0.00.261.992 I llama_memory_recurrent:       MTL0 RS buffer size =   351.75 MiB
0.00.262.003 I llama_memory_recurrent: size =  351.75 MiB (     1 cells,  33 layers,  1 seqs  6 rs_seq), R (f32):   15.75 MiB, S (f32):  336.00 MiB
0.00.262.009 I sched_reserve: reserving ...
0.00.266.163 I sched_reserve: resolving fused Gated Delta Net support:
0.00.268.108 I sched_reserve: fused Gated Delta Net (autoregressive) enabled
0.00.269.037 I sched_reserve: fused Gated Delta Net (chunked) enabled
0.00.271.444 I sched_reserve:       MTL0 compute buffer size =   501.00 MiB
0.00.271.446 I sched_reserve:        CPU compute buffer size =   272.29 MiB
0.00.271.446 I sched_reserve:        CPU compute buffer size =   272.29 MiB
0.00.271.447 I sched_reserve: graph nodes  = 1952
0.00.271.447 I sched_reserve: graph splits = 2
0.00.271.447 I sched_reserve: reserve took 9.43 ms, sched copies = 1
0.00.271.560 I common_memory_breakdown_print: | memory breakdown [MiB]  | total    free    self   model   context   compute    unaccounted |
0.00.271.562 I common_memory_breakdown_print: |   - MTL0 (Apple M4 Pro) | 16384 = 16031 + (8323 =  5294 +    2527 +     501) +       -7971 |
0.00.271.562 I common_memory_breakdown_print: |   - Host                |                  1090 =   545 +       0 +     544                |
0.00.271.627 I ggml_metal_free: deallocating
0.00.296.090 I common_params_fit_impl: projected to use 8323 MiB of device memory vs. 16031 MiB of free device memory
0.00.296.092 I common_params_fit_impl: will leave 7708 >= 1024 MiB of free device memory, no changes needed
0.00.296.092 I common_fit_params: successfully fit params to free device memory
0.00.296.094 I common_fit_params: fitting params to free memory took 0.24 seconds
```

Важные для оценки строки в самом низу выделенного фрагмента. Прежде всего `projected to use 8323 MiB of device memory vs. 16031 MiB of free device memory` -- с такими настройками бэкэнд занял чуть более 8G из рабочего диапазона в 16G. Тут вспоминаем, что в компьютере всего 24G, и пробуем раскрутить бэкэнд на максимум возможностей.

#### Итоговый вариант настройки.

Меняем параметры, чтобы дабиться максимального размера контекста.

```bash
❯ ./server.sh
-----
GGML_METAL_EMBEDDING=1
LOCAL_LLM_ALIAS=qwen35
LOCAL_LLM_APIKEY=token_as_a_lot_of_simbols
LOCAL_LLM_BATCH=256
LOCAL_LLM_CACHEK=q5_1
LOCAL_LLM_CACHERAM=0
LOCAL_LLM_CACHEV=q8_0
LOCAL_LLM_CONTEXT=262144
LOCAL_LLM_HOST=127.0.0.1
LOCAL_LLM_MODEL=Qwen3.5-9B-MTP-UD-Q4_K_XL.gguf
LOCAL_LLM_NGLAYERS=999
LOCAL_LLM_NMAX=6
LOCAL_LLM_PORT=8000
LOCAL_LLM_REASONING=off
LOCAL_LLM_SLOTS=1
LOCAL_LLM_SPEC=draft-mtp
LOCAL_LLM_THREADS=8
LOCAL_LLM_TIMEOUT=900
LOCAL_LLM_VERBOSITY=4
```

Снова проверяем лог:

```bash
0.00.226.634 I ggml_metal_init: allocating
0.00.226.668 I ggml_metal_init: found device: Apple M4 Pro
0.00.226.672 I ggml_metal_init: picking default device: Apple M4 Pro
0.00.226.702 I ggml_metal_init: use fusion         = true
0.00.226.703 I ggml_metal_init: use concurrency    = true
0.00.226.703 I ggml_metal_init: use graph optimize = true
0.00.226.761 I llama_context:        CPU  output buffer size =     0.95 MiB
0.00.227.818 I llama_kv_cache:       MTL0 KV buffer size =     0.00 MiB
0.00.227.820 I llama_kv_cache: size = 3712.00 MiB (262144 cells,   8 layers,  1/1 seqs), K (q5_1): 1536.00 MiB, V (q8_0): 2176.00 MiB
0.00.227.821 I llama_kv_cache: attn_rot_k = 1, n_embd_head_k_all = 256
0.00.227.821 I llama_kv_cache: attn_rot_v = 1, n_embd_head_k_all = 256
0.00.297.005 I llama_memory_recurrent:       MTL0 RS buffer size =   351.75 MiB
0.00.297.022 I llama_memory_recurrent: size =  351.75 MiB (     1 cells,  33 layers,  1 seqs  6 rs_seq), R (f32):   15.75 MiB, S (f32):  336.00 MiB
0.00.297.037 I sched_reserve: reserving ...
0.00.302.103 I sched_reserve: resolving fused Gated Delta Net support:
0.00.303.097 I sched_reserve: fused Gated Delta Net (autoregressive) enabled
0.00.303.962 I sched_reserve: fused Gated Delta Net (chunked) enabled
0.00.306.407 I sched_reserve:       MTL0 compute buffer size =   416.52 MiB
0.00.306.410 I sched_reserve:        CPU compute buffer size =   660.28 MiB
0.00.306.410 I sched_reserve:        CPU compute buffer size =   660.28 MiB
0.00.306.410 I sched_reserve: graph nodes  = 1952
0.00.306.411 I sched_reserve: graph splits = 18
0.00.306.411 I sched_reserve: reserve took 9.37 ms, sched copies = 1
0.00.306.512 I common_memory_breakdown_print: | memory breakdown [MiB]  | total    free    self   model   context   compute    unaccounted |
0.00.306.515 I common_memory_breakdown_print: |   - MTL0 (Apple M4 Pro) | 16384 = 16031 + (9775 =  5294 +    4063 +     416) +       -9422 |
0.00.306.515 I common_memory_breakdown_print: |   - Host                |                  1866 =   545 +       0 +    1320                |
0.00.306.778 I ggml_metal_free: deallocating
0.00.332.072 I common_params_fit_impl: projected to use 9775 MiB of device memory vs. 16031 MiB of free device memory
0.00.332.075 I common_params_fit_impl: will leave 6256 >= 1024 MiB of free device memory, no changes needed
```

Видим `projected to use 9775 MiB of device memory vs. 16031 MiB` и далее `will leave 6256` , что оставляет не просто 6G+, а фактически 14G памяти для работы приложений. На этих настройках можно остановиться.

### Оценка и возможные пути дальнейшего тюнинга.

Конфигурация **отличная и сбалансированная**. Железо используется почти оптимально: ~9.8 GB из 16 GB Metal, запас **6.2 GB** — это очень комфортная зона, далеко от порога OOM.

---

#### Разбор по компонентам

| Компонент | Память | Комментарий |
|---|---|---|
| **Модель** (`Q4_K_XL`) | 5294 MiB | Максимально качественный 4-бит квант. Хороший выбор. |
| **KV-cache** (262K) | 3712 MiB | K=`q5_1` (1536) + V=`q8_0` (2176). Для 256K контекста это компактно. |
| **Recurrent state** | 352 MiB | SSM/Mamba-слои Qwen3.5. Фиксированная величина, не растёт с контекстом. |
| **Compute buffers** | 417 MiB | Для батча 256 на 256K контексте — норма. |
| **Draft-MTP** | вшито в модель | Спекулятивный декодинг работает, дополнительной аллокации в breakdown не видно (разделяет веса). |
| **Итого projected** | **9775 MiB** | |
| **Запас свободно** | **6256 MiB** | В 6 раз больше минимального порога llama.cpp (1024 MiB). |

---

#### Что можно улучшить (запасы)

У нас есть **три независимых рычага**, которые можно применить по отдельности или вместе:

1. Повысить качество KV-cache (самое простое)
Перевести K-cache с `q5_1` на `q8_0`:
- K-cache вырастет с 1536 MiB до **~2176 MiB** (+640 MiB)
- Общее использование станет **~10.4 GB**, запас останется **~5.6 GB**
- Это улучшит точность на длинных контекстах (256K), особенно при поздних токенах

2. Увеличить batch size
Сейчас `BATCH=256`. Можно смело ставить `512` или даже `1024`:
- Compute buffer вырастет с 417 MiB до ~600–800 MiB
- На фоне 6+ GB запаса это несущественно
- Ускорит обработку длинных промптов и параллельные запросы (если слотов станет больше)

3. Попробовать более тяжёлую модель
С таким запасом можно экспериментировать:
- `Q5_K_S` — модель станет примерно на **+1.5 GB** тяжелее (~6.8 GB). Итого ~11.3 GB, запас ~4.8 GB.
- `Q5_K_M` — ещё тяжелее на **~+2–2.5 GB**. Итого ~12 GB, запас ~4 GB. Это уже предел комфорта, но вполне безопасно для стабильной работы.

4. Мелочи для удобства
```bash
LOCAL_LLM_CACHERAM=1    # включить кэширование промптов в RAM (ускорит повторные запросы)
```

---

#### Что НЕ трогать

- `NGLAYERS=999` — правильно, весь offload на GPU.
- `SPEC=draft-mtp` — оставьте, спекулятивный декодинг даёт бесплатный прирост скорости (~10–30%).
- `CONTEXT=262144` — при текущих квантах это максимум комфорта.

---

#### Итог

Текущая конфигурация — **«золотая середина»** между качеством, длиной контекста и стабильностью. Если нужно выжать максимум качества — первым делом следует перевести `CACHEK` в `q8_0` (это +640 MB, запас позволяет). Если нужна скорость — увеличить `BATCH`. Если выбрать `Q5_K_M` — он влезет с запасом в ~4 GB.


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

#### Скрипт короткого теста.

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

#### Результат:

```json
{
  "choices": [
    {
      "finish_reason": "stop",
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "Here’s a common recursive implementation of the Fibonacci function, plus a more efficient iterative version:\n\n```python\n# Recursive version (simple but exponential time)\ndef fib(n: int) -> int:\n    if n < 0:\n        raise ValueError(\"n must be non-negative\")\n    if n == 0:\n        return 0\n    if n == 1:\n        return 1\n    return fib(n - 1) + fib(n - 2)\n```\n\nEfficient iterative version (O(n) time, O(1) space):\n\n```python\ndef fib(n: int) -> int:\n    if n < 0:\n        raise ValueError(\"n must be non-negative\")\n    if n == 0:\n        return 0\n    if n == 1:\n        return 1\n\n    a, b = 0, 1  # fib(0), fib(1)\n    for _ in range(2, n + 1):\n        a, b = b, a + b\n    return b\n```\n\nIf you tell me what you need (e.g., memoized recursive, matrix exponentiation, generator for a sequence), I can tailor it further."
      }
    }
  ],
  "created": 1779290052,
  "model": "qwen35",
  "system_fingerprint": "b9222-9a532ae4b",
  "object": "chat.completion",
  "usage": {
    "completion_tokens": 263,
    "prompt_tokens": 16,
    "total_tokens": 279,
    "prompt_tokens_details": {
      "cached_tokens": 0
    }
  },
  "id": "chatcmpl-DQIofMZ5LoCi2YtpAep2S2byYsxILhcl",
  "timings": {
    "cache_n": 0,
    "prompt_n": 16,
    "prompt_ms": 2132.893,
    "prompt_per_token_ms": 133.3058125,
    "prompt_per_second": 7.501548366467516,
    "predicted_n": 263,
    "predicted_ms": 18165.398,
    "predicted_per_token_ms": 69.06995437262358,
    "predicted_per_second": 14.478075294579286,
    "draft_n": 372,
    "draft_n_accepted": 204
  }
}
```

---

#### Лог сервера:

```bash
27.18.777.913 I srv  params_from_: Chat format: peg-native
27.18.778.252 I slot get_availabl: id  0 | task -1 | selected slot by LRU, t_last = -1
27.18.778.465 I slot launch_slot_: id  0 | task -1 | sampler chain: logits -> ?penalties -> ?dry -> ?top-n-sigma -> top-k -> ?typical -> top-p -> min-p -> ?xtc -> temp-ext -> dist 
27.18.778.475 I slot launch_slot_: id  0 | task -1 | sampler params: 
	repeat_last_n = 64, repeat_penalty = 1.000, frequency_penalty = 0.000, presence_penalty = 0.000
	dry_multiplier = 0.000, dry_base = 1.750, dry_allowed_length = 2, dry_penalty_last_n = 262144
	top_k = 40, top_p = 0.950, min_p = 0.050, xtc_probability = 0.000, xtc_threshold = 0.100, typical_p = 1.000, top_n_sigma = -1.000, temp = 0.800
	mirostat = 0, mirostat_lr = 0.100, mirostat_ent = 5.000, adaptive_target = -1.000, adaptive_decay = 0.900
27.18.778.478 I slot launch_slot_: id  0 | task 0 | processing task, is_child = 0
27.18.778.633 I slot update_slots: id  0 | task 0 | new prompt, n_ctx_slot = 262144, n_keep = 0, task.n_tokens = 16
27.18.778.642 I slot update_slots: id  0 | task 0 | cached n_tokens = 0, memory_seq_rm [0, end)
27.20.702.306 I slot update_slots: id  0 | task 0 | cached n_tokens = 12, memory_seq_rm [12, end)
27.20.702.675 I slot init_sampler: id  0 | task 0 | init sampler, took 0.00 ms, tokens: text = 16, total = 16
27.28.461.935 I slot print_timing: id  0 | task 0 | n_decoded =    103, tg =  13.64 t/s
27.31.696.199 I slot print_timing: id  0 | task 0 | n_decoded =    162, tg =  15.02 t/s
27.34.943.388 I slot print_timing: id  0 | task 0 | n_decoded =    222, tg =  15.82 t/s
27.38.199.943 I slot print_timing: id  0 | task 0 | n_decoded =    254, tg =  14.69 t/s
27.39.077.123 I slot print_timing: id  0 | task 0 | 
prompt eval time =    2132.89 ms /    16 tokens (  133.31 ms per token,     7.50 tokens per second)
       eval time =   18165.40 ms /   263 tokens (   69.07 ms per token,    14.48 tokens per second)
      total time =   20298.29 ms /   279 tokens
draft acceptance rate = 0.54839 (  204 accepted /   372 generated)
27.39.077.138 I statistics draft-mtp: #calls(b,g,a) = 1 62 62, #gen drafts = 62, #acc drafts = 55, #gen tokens = 372, #acc tokens = 204, dur(b,g,a) = 0.001, 3935.948, 0.063 ms
27.39.077.178 I slot      release: id  0 | task 0 | stop processing: n_tokens = 282, truncated = 0
27.39.077.188 I srv  update_slots: all slots are idle
27.39.078.672 I srv  log_server_r: done request: POST /v1/chat/completions 127.0.0.1 200
```

#### Комментарий.

Результат **качественный** (код корректный, с комментариями, сразу предложены две реализации), но с точки зрения инференса есть важная особенность, которую нужно понимать.

---

1. Технический разбор таймингов

| Метрика | Значение | Оценка |
|---|---|---|
| **Prompt (prefill)** | 16 токенов за **2.13 с** → **7.5 t/s** | ⚠️ Аномально медленно |
| **Generation (decode)** | 263 токена за **18.17 с** → **14.5 t/s** | ✅ Норма для 9B на M4 Pro |
| **Draft acceptance** | **54.8%** (204/372) | ✅ Хорошо |
| **Общее время** | **20.3 с** | — |

---

2. Почему prefill такой медленный (7.5 t/s)

Это **не баг конфигурации**, а особенность архитектуры вашей модели. В логе явно видно:

```
fused Gated Delta Net (autoregressive) enabled
fused Gated Delta Net (chunked) enabled
llama_memory_recurrent: size = 351.75 MiB (1 cells, 33 layers...)
```

Qwen3.5 — это **гибридная модель** (не чистый Transformer, а с Mamba/SSM-слоями). Для таких моделей prefill требует не просто параллельного прогона через attention, а **последовательной инициализации рекуррентных скрытых состояний** (R- и S-буферы в логе). На коротком промпте этот overhead доминирует: 16 токенов обрабатываются за 2 секунды не из-за вычислительной сложности, а из-за «прогрева» 33 recurrent-слоёв.

**Важно:** на длинных промптах (1K–10K токенов) prefill-per-token будет **ниже**, а абсолютная скорость **выше**, т.к. инициализация амортизируется на большую длину. Но на коротких запросах (как этот, 16 токенов) — всегда будет «тормозить».

---

3. Speculative decoding (draft-mtp)

Работает эффективно:
- Сгенерировано **372 draft-токена**, принято **204** → acceptance rate **54.8%**
- Это даёт прирост скорости примерно **1.4–1.6×** по сравнению с обычной генерацией
- Без draft-MTP decode скорость была бы ближе к **9–11 t/s** вместо текущих 14.5

---

4. Что можно улучшить

- 1. **Включить prompt cache (`CACHERAM`)** — критично для этой модели
   ```bash
   LOCAL_LLM_CACHERAM=1   # или больше, если системный промпт постоянный
   ```
   Т.к. prefill дорогой из-за recurrent-слоёв, повторное использование закэшированного префикса даст огромный выигрыш на типовых диалогах.

- 2. **Batch size можно поднять** до `512` или `1024`
   Сейчас `BATCH=256`, но при одиночных запросах compute buffer всё равно аллоцируется под максимум. На M4 Pro с вашим запасом памяти это безопасно и немного снизит latency на длинных промптах.

- 3. **Не гнаться за скоростью prefill на коротких промптах**
   Для recurrent-моделей это физическое ограничение архитектуры, а не настройки. Если нужен интерактивный чат с мгновенным откликом — только кэширование поможет.

---

5. Итог

Конфигурация работает **корректно и близко к оптимуму** для данной архитектуры. 14.5 t/s на генерации — хороший результат для 9B hybrid-модели в Q4_K_XL на MacBook. Единственное узкое место — медленный prefill, но это лечится исключительно кэшированием (`CACHERAM`), а не тюнингом Metal/GPU.


### Двухэтапный тест для компенсации разогрева

#### Скрипт полного теста.

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

#### Тестирование в два этапа:

```bash
❯ ./test2.sh
❯ ./test2.sh
🔍 Тест 1: Короткий запрос (скорость ответа)
Вот реализация алгоритма быстрой сортировки (Quicksort) на Python. Этот вариант использует метод «разделяй и властвуй» и оптимизирован для работы с массивами.

### Реализация функции

```python
def quicksort(arr):
    # Базовый случай: если массив имеет 0 или 1 элемент, он уже отсортирован
    if len(arr) <= 1:
        return arr
    
    # Выбираем опорный элемент (pivot). 
    # Для лучшей производительности часто берут средний или последний элемент,
    # но для простоты здесь берем последний.
    pivot = arr[-1]
    
    # Разделяем массив на три части:
    # 1. Элементы меньше опорного
    # 2. Элементы, равные опорному (опционально, можно включить в одну из групп)
    # 3. Элементы больше опорного
    less = [x for x in arr if x < pivot]
    equal = [x for x in arr if x == pivot]
    greater = [x for x in arr if x > pivot]
    
    # Рекурсивно сортируем левую и правую части и объединяем результаты
    return quicksort(less) + equal + quicksort(greater)
```

### Как это работает:
1.  **Базовый случай**: Если массив пуст или содержит один элемент, функция сразу возвращает его.
2.  **Выбор опорного элемента (`pivot`)**: В коде выбран последний элемент массива.
3.  **Разделение**: Используется список-включение (list comprehension) для создания трех новых списков:
    *   `less`: элементы строго меньше `pivot`.
    *   `equal`: элементы равные `pivot`.
    *   `greater`: элементы строго больше `pivot`.
4.  **Рекурсия**: Функция вызывает сама себя для подмассивов `less` и `greater`.
5.  **Сборка**: Результаты рекурсии склеиваются оператором `+` в итоговый отсортированный массив.

### Пример использования

```python
if __name__ == "__main__":
    data = [3, 6, 8, 10, 1, 2, 1]
    print("Оригинал:", data)
    sorted_data = quicksort(data)
    print("Отсортировано:", sorted_data)
    # Вывод: [1, 1, 2, 3, 6, 8, 10]
```

### Важное примечание по производительности
Хотя этот код понятен и компактен, он создает множество новых списков на каждом шаге рекурсии. Для очень больших массивов это может быть менее эффективно по памяти, чем классическая реализация с индексами. Однако для большинства учебных целей и средних объемов данных этот вариант отлично подходит. Если вам нужна версия с минимальным использованием памяти, можно реализовать её, используя индексы вместо создания новых списков.

real	1m1.457s
user	0m0.008s
sys	0m0.020s

🔍 Тест 2: Длинный контекст (загрузка 8K токенов)
В вашем коде есть две основные проблемы:
1. **Огромное количество дубликатов**: Импорт `numpy` повторяется сотни раз, что делает код неэффективным и «грязным».
2. **Отсутствие логики**: Сам по себе набор импортов не делает ничего, кроме загрузки библиотеки в память много раз (хотя Python кэширует импорты, это всё равно плохая практика).

Ниже представлен **оптимизированный, типизированный и чистый** вариант.

### Оптимизированный код

```python
from typing import Any, Tuple

import numpy as np  # Импорт делается только один раз

def process_data(x: np.ndarray, y: np.ndarray) -> Tuple[np.ndarray, np.ndarray]:
    """
    Пример функции для демонстрации типизации.
    Принимает два массива, проверяет их размерность и возвращает обработанные данные.
    
    Args:
        x: Входной массив данных.
        y: Целевой массив данных.
    
    Returns:
        Кортеж из двух обработанных массивов.
    """
    if x.shape != y.shape:
        raise ValueError("Массивы x и y должны иметь одинаковую размерность.")
    
    # Пример простой обработки: нормализация (условно)
    # В реальном коде здесь была бы сложная логика
    x_normalized = x / np.max(np.abs(x))
    y_scaled = y * 2.0
    
    return x_normalized, y_scaled

if __name__ == "__main__":
    # Пример использования
    data = np.random.rand(100, 5)
    labels = np.random.rand(100)
    
    processed_x, processed_y = process_data(data, labels)
    print("Обработка завершена.")
```

### Что было сделано:

1.  **Убраны дубликаты**: Вместо 100+ строк `import numpy as np` осталось **одна** строка. Это стандарт Python.
2.  **Добавлена типизация**:
    *   Использован модуль `typing`.
    *   Аргументы функции помечены типами (`np.ndarray`).
    *   Возвращаемое значение типизировано как `Tuple`.
    *   Добавлены аннотации для лучшей читаемости и автодополнения в IDE.
3.  **Добавлена логика**: Поскольку исходный код не выполнял никаких действий, я добавил простую функцию-пример, чтобы показать, как правильно структурировать код.
4.  **Добавлен `if __name__ == "__main__":`**: Это лучший способ тестирования кода без глобальных переменных, которые могут конфликтовать при импорте модуля.

real	1m6.943s
user	0m0.007s
sys	0m0.012s

💡 Проверьте Activity Monitor: Memory Pressure должен быть зеленым
```
```

---

#### Сообщения в логе сервера:

```bash
36.28.927.200 I srv  params_from_: Chat format: peg-native
36.28.927.441 I slot get_availabl: id  0 | task -1 | selected slot by LCP similarity, sim_best = 0.130 (> 0.100 thold), f_keep = 0.011
36.28.927.531 I slot launch_slot_: id  0 | task -1 | sampler chain: logits -> ?penalties -> ?dry -> ?top-n-sigma -> top-k -> ?typical -> top-p -> min-p -> ?xtc -> temp-ext -> dist 
36.28.927.537 I slot launch_slot_: id  0 | task -1 | sampler params: 
	repeat_last_n = 64, repeat_penalty = 1.000, frequency_penalty = 0.000, presence_penalty = 0.000
	dry_multiplier = 0.000, dry_base = 1.750, dry_allowed_length = 2, dry_penalty_last_n = 262144
	top_k = 40, top_p = 0.950, min_p = 0.050, xtc_probability = 0.000, xtc_threshold = 0.100, typical_p = 1.000, top_n_sigma = -1.000, temp = 0.800
	mirostat = 0, mirostat_lr = 0.100, mirostat_ent = 5.000, adaptive_target = -1.000, adaptive_decay = 0.900
36.28.927.538 I slot launch_slot_: id  0 | task 65 | processing task, is_child = 0
36.28.927.541 I slot update_slots: id  0 | task 65 | new prompt, n_ctx_slot = 262144, n_keep = 0, task.n_tokens = 23
36.28.927.544 W slot update_slots: id  0 | task 65 | n_past = 3, slot.prompt.tokens.size() = 282, seq_id = 0, pos_min = 281, n_swa = 0
36.28.927.545 W slot update_slots: id  0 | task 65 | forcing full prompt re-processing due to lack of cache data (likely due to SWA or hybrid/recurrent memory, see https://github.com/ggml-org/llama.cpp/pull/13194#issuecomment-2868343055)
36.28.927.545 I slot update_slots: id  0 | task 65 | cached n_tokens = 0, memory_seq_rm [0, end)
36.30.621.815 I slot update_slots: id  0 | task 65 | cached n_tokens = 19, memory_seq_rm [19, end)
36.30.622.260 I slot init_sampler: id  0 | task 65 | init sampler, took 0.00 ms, tokens: text = 23, total = 23
36.39.980.727 I slot print_timing: id  0 | task 65 | n_decoded =    102, tg =  11.07 t/s
36.43.131.788 I slot print_timing: id  0 | task 65 | n_decoded =    135, tg =  10.92 t/s
36.46.275.851 I slot print_timing: id  0 | task 65 | n_decoded =    175, tg =  11.29 t/s
36.49.431.536 I slot print_timing: id  0 | task 65 | n_decoded =    216, tg =  11.57 t/s
36.52.724.821 I slot print_timing: id  0 | task 65 | n_decoded =    258, tg =  11.75 t/s
36.55.914.443 I slot print_timing: id  0 | task 65 | n_decoded =    299, tg =  11.89 t/s
36.59.100.434 I slot print_timing: id  0 | task 65 | n_decoded =    336, tg =  11.86 t/s
37.02.298.056 I slot print_timing: id  0 | task 65 | n_decoded =    368, tg =  11.67 t/s
37.05.527.375 I slot print_timing: id  0 | task 65 | n_decoded =    419, tg =  12.05 t/s
37.08.723.067 I slot print_timing: id  0 | task 65 | n_decoded =    451, tg =  11.88 t/s
37.11.922.114 I slot print_timing: id  0 | task 65 | n_decoded =    487, tg =  11.83 t/s
37.15.127.472 I slot print_timing: id  0 | task 65 | n_decoded =    526, tg =  11.86 t/s
37.18.331.956 I slot print_timing: id  0 | task 65 | n_decoded =    579, tg =  12.17 t/s
37.21.546.243 I slot print_timing: id  0 | task 65 | n_decoded =    611, tg =  12.03 t/s
37.24.769.468 I slot print_timing: id  0 | task 65 | n_decoded =    636, tg =  11.78 t/s
37.27.994.435 I slot print_timing: id  0 | task 65 | n_decoded =    656, tg =  11.46 t/s
37.30.342.831 I slot print_timing: id  0 | task 65 | 
prompt eval time =    1841.83 ms /    23 tokens (   80.08 ms per token,    12.49 tokens per second)
       eval time =   59573.28 ms /   676 tokens (   88.13 ms per token,    11.35 tokens per second)
      total time =   61415.10 ms /   699 tokens
draft acceptance rate = 0.38374 (  472 accepted /  1230 generated)
37.30.342.847 I statistics draft-mtp: #calls(b,g,a) = 2 267 267, #gen drafts = 267, #acc drafts = 225, #gen tokens = 1602, #acc tokens = 676, dur(b,g,a) = 0.001, 16778.491, 0.220 ms
37.30.342.899 I slot      release: id  0 | task 65 | stop processing: n_tokens = 700, truncated = 0
37.30.342.914 I srv  update_slots: all slots are idle
37.30.344.910 I srv  log_server_r: done request: POST /v1/chat/completions 127.0.0.1 200
37.30.654.042 I srv  params_from_: Chat format: peg-native
37.30.654.467 I slot get_availabl: id  0 | task -1 | selected slot by LRU, t_last = 1301179696193
37.30.654.509 I slot launch_slot_: id  0 | task -1 | sampler chain: logits -> ?penalties -> ?dry -> ?top-n-sigma -> top-k -> ?typical -> top-p -> min-p -> ?xtc -> temp-ext -> dist 
37.30.654.518 I slot launch_slot_: id  0 | task -1 | sampler params: 
	repeat_last_n = 64, repeat_penalty = 1.000, frequency_penalty = 0.000, presence_penalty = 0.000
	dry_multiplier = 0.000, dry_base = 1.750, dry_allowed_length = 2, dry_penalty_last_n = 262144
	top_k = 40, top_p = 0.950, min_p = 0.050, xtc_probability = 0.000, xtc_threshold = 0.100, typical_p = 1.000, top_n_sigma = -1.000, temp = 0.800
	mirostat = 0, mirostat_lr = 0.100, mirostat_ent = 5.000, adaptive_target = -1.000, adaptive_decay = 0.900
37.30.654.519 I slot launch_slot_: id  0 | task 273 | processing task, is_child = 0
37.30.654.521 I slot update_slots: id  0 | task 273 | new prompt, n_ctx_slot = 262144, n_keep = 0, task.n_tokens = 1032
37.30.654.524 W slot update_slots: id  0 | task 273 | n_past = 3, slot.prompt.tokens.size() = 700, seq_id = 0, pos_min = 699, n_swa = 0
37.30.654.525 W slot update_slots: id  0 | task 273 | forcing full prompt re-processing due to lack of cache data (likely due to SWA or hybrid/recurrent memory, see https://github.com/ggml-org/llama.cpp/pull/13194#issuecomment-2868343055)
37.30.654.541 I slot update_slots: id  0 | task 273 | cached n_tokens = 0, memory_seq_rm [0, end)
37.31.588.499 I slot update_slots: id  0 | task 273 | cached n_tokens = 256, memory_seq_rm [256, end)
37.32.594.880 I slot update_slots: id  0 | task 273 | cached n_tokens = 512, memory_seq_rm [512, end)
37.33.677.931 I slot print_timing: id  0 | task 273 | prompt processing, n_tokens =    768, progress = 0.74, t =   3.02 s / 254.02 tokens per second
37.33.677.934 I slot update_slots: id  0 | task 273 | cached n_tokens = 768, memory_seq_rm [768, end)
37.33.995.932 I slot print_timing: id  0 | task 273 | prompt processing, n_tokens =    776, progress = 0.75, t =   3.34 s / 232.24 tokens per second
37.33.995.933 I slot update_slots: id  0 | task 273 | cached n_tokens = 776, memory_seq_rm [776, end)
37.34.032.020 I slot create_check: id  0 | task 273 | created context checkpoint 1 of 32 (pos_min = 775, pos_max = 775, n_tokens = 776, size = 51.639 MiB)
37.35.083.812 I slot print_timing: id  0 | task 273 | prompt processing, n_tokens =   1028, progress = 1.00, t =   4.43 s / 232.09 tokens per second
37.35.083.816 I slot update_slots: id  0 | task 273 | cached n_tokens = 1028, memory_seq_rm [1028, end)
37.35.084.116 I slot init_sampler: id  0 | task 273 | init sampler, took 0.08 ms, tokens: text = 1032, total = 1032
37.35.203.506 I slot create_check: id  0 | task 273 | created context checkpoint 2 of 32 (pos_min = 1027, pos_max = 1027, n_tokens = 1028, size = 52.090 MiB)
37.47.085.439 I slot print_timing: id  0 | task 273 | n_decoded =    102, tg =   8.68 t/s
37.50.104.999 I slot print_timing: id  0 | task 273 | n_decoded =    139, tg =   9.41 t/s
37.53.137.265 I slot print_timing: id  0 | task 273 | n_decoded =    163, tg =   9.15 t/s
37.56.156.582 I slot print_timing: id  0 | task 273 | n_decoded =    194, tg =   9.32 t/s
37.59.189.402 I slot print_timing: id  0 | task 273 | n_decoded =    222, tg =   9.31 t/s
38.02.234.414 I slot print_timing: id  0 | task 273 | n_decoded =    268, tg =   9.96 t/s
38.05.261.668 I slot print_timing: id  0 | task 273 | n_decoded =    297, tg =   9.92 t/s
38.08.309.442 I slot print_timing: id  0 | task 273 | n_decoded =    321, tg =   9.73 t/s
38.11.342.932 I slot print_timing: id  0 | task 273 | n_decoded =    351, tg =   9.75 t/s
38.14.388.885 I slot print_timing: id  0 | task 273 | n_decoded =    383, tg =   9.81 t/s
38.17.418.673 I slot print_timing: id  0 | task 273 | n_decoded =    418, tg =   9.93 t/s
38.20.481.132 I slot print_timing: id  0 | task 273 | n_decoded =    449, tg =   9.94 t/s
38.23.521.580 I slot print_timing: id  0 | task 273 | n_decoded =    484, tg =  10.04 t/s
38.26.572.409 I slot print_timing: id  0 | task 273 | n_decoded =    520, tg =  10.15 t/s
38.29.621.111 I slot print_timing: id  0 | task 273 | n_decoded =    549, tg =  10.11 t/s
38.32.668.893 I slot print_timing: id  0 | task 273 | n_decoded =    579, tg =  10.10 t/s
38.35.724.268 I slot print_timing: id  0 | task 273 | n_decoded =    605, tg =  10.02 t/s
38.37.558.784 I slot print_timing: id  0 | task 273 | 
prompt eval time =    4676.89 ms /  1032 tokens (    4.53 ms per token,   220.66 tokens per second)
       eval time =   62227.18 ms /   621 tokens (  100.20 ms per token,     9.98 tokens per second)
      total time =   66904.07 ms /  1653 tokens
draft acceptance rate = 0.33740 (  415 accepted /  1230 generated)
38.37.558.797 I statistics draft-mtp: #calls(b,g,a) = 3 472 472, #gen drafts = 472, #acc drafts = 378, #gen tokens = 2832, #acc tokens = 1091, dur(b,g,a) = 0.001, 29912.826, 0.402 ms
38.37.558.922 I slot      release: id  0 | task 273 | stop processing: n_tokens = 1652, truncated = 0
38.37.558.942 I srv  update_slots: all slots are idle
38.37.559.016 I srv  log_server_r: done request: POST /v1/chat/completions 127.0.0.1 200
```

---

#### Оценка работы.

Результаты теста показывают **два фундаментально разных режима** работы данной модели, и главный ворнинг объясняет, почему диалоговые сценарии будут иметь неожиданную задержку.

---

1. Главный ворнинг: `forcing full prompt re-processing`

**Что значит:**  
Каждый новый запрос **перепроцессирует весь предыдущий контекст с нуля**, даже если добавляется всего 1–2 новых токена к диалогу.

**Почему:**  
```
likely due to SWA or hybrid/recurrent memory
```
Qwen3.5 — это **гибридная модель** (Transformer + Mamba/SSM recurrent-слои). Recurrent-состояния (R/S-буферы, 351 MiB) нельзя «докинуть» к существующему кэшу, как в чистом Transformer. Их нужно вычислять последовательно от начала контекста. Поэтому llama.cpp сбрасывает кэш и гонит полный пересчёт.

**Последствия для практики:**
- **Task 65** (короткий запрос): предыдущий слот содержал 282 токена, но кэш = 0, пересчёт с нуля.
- **Task 273** (длинный запрос): предыдущий слот содержал 700 токенов, кэш = 0, пересчёт с нуля.
- Если ведётся диалог на 4000 токенов и добавляется 20 токенов нового сообщения — llama.cpp пережуёт все 4020 токенов заново. Диалоги будут **линейно замедляться** по мере роста истории.

**Это не баг конфигурации**, это архитектурное ограничение Qwen3.5 в текущей версии llama.cpp. Для stateless API (одиночные запросы без истории) — не критично.

---

2. Контраст скоростей: короткий vs длинный prefill

| Метрика | Task 65 (короткий) | Task 273 (длинный, ~8K) |
|---|---|---|
| **Prefill speed** | **12.49 t/s** (23 токена за 1.84 с) | **220.66 t/s** (1032 токена за 4.43 с) |
| **Generation** | 11.35 t/s | 9.98 t/s |

**Почему длинный prefill в 17 раз быстрее (per token):**  
Recurrent-инициализация — это фиксированный overhead. На 23 токенах он «съедает» почти всё время (1.84 с), а на 1032 токенах амортизируется (4.43 с на весь промпт). Это подтверждает закономерность из первого теста: **чем длиннее промпт, тем эффективнее prefill**.

**Вывод:** модель отлично подходит для **batch-обработки длинных документов** (один тяжёлый prefill, потом generation), но плохо для **интерактивных чатов с короткими репликами** (каждый раз полный пересчёт).

---

3. Draft-MTP: acceptance rate упал

| Запрос | Acceptance | Комментарий |
|---|---|---|
| Первый тест (Fibonacci) | **54.8%** | Общий текст, предсказуемый |
| Task 65 (быстрая сортировка) | **38.4%** | Код, менее предсказуем для draft |
| Task 273 (оптимизация numpy) | **33.7%** | Длинный контекст + код |

Draft-модель (MTP) путается на задачах с **кодом и техническими деталями**. 33–38% — это всё ещё положительный прирост скорости (без draft было бы ~7–8 t/s вместо 10–11), но эффективность ниже, чем на «разговорных» задачах.

---

4. Context checkpoints (нормально)

```
created context checkpoint 1 of 32 ... size = 51.639 MiB
created context checkpoint 2 of 32 ... size = 52.090 MiB
```
Это штатный механизм llama.cpp для **длинных контекстов** — периодическое сохранение recurrent/Mamba-состояний, чтобы не пересчитывать всё при откатах или truncation. Работает корректно, память не течёт.

---

5. Итоговая оценка конфигурации

| Параметр | Оценка |
|---|---|
| Память | ✅ Запас ~6 GB, чекпоинты не давят |
| Длинный prefill (~1K) | ✅ Отлично (220 t/s) |
| Generation | ✅ Стабильно ~10 t/s |
| Короткий prefill | ⚠️ Медленно из-за recurrent overhead |
| Диалоги/чат | ❌ **Полный пересчёт контекста каждый раз** |
| Draft-MTP | ⚠️ Работает, но acceptance 33–38% на коде |

---

#### Итоговый вариант настроек.

Учитывая ограниченность памяти придется мириться с длительной задержкой в процессе обработки промпта. Спектр предполагаемых задач для данной модели - кодирование, и вся работа будет строиться с использованием кодовых агентов.

Поэтому все что можно в данном случае позволить, это вернуть квантизацию ключей в q8_0, что не слишком увеличит нагрузку на память, то улучшит и качество и скорость обработки.


#### Оценка предполагаемой скорости работы бэкэнда.

```bash
❯ ./benchmark.sh
-----
GGML_METAL_EMBEDDING=1
LOCAL_LLM_ALIAS=qwen35
LOCAL_LLM_APIKEY=token_as_a_lot_of_simbols
LOCAL_LLM_BATCH=256
LOCAL_LLM_CACHEK=q8_0
LOCAL_LLM_CACHERAM=0
LOCAL_LLM_CACHEV=q8_0
LOCAL_LLM_CONTEXT=262144
LOCAL_LLM_HOST=127.0.0.1
LOCAL_LLM_MODEL=Qwen3.5-9B-MTP-UD-Q4_K_XL.gguf
LOCAL_LLM_NGLAYERS=999
LOCAL_LLM_NMAX=6
LOCAL_LLM_PORT=8000
LOCAL_LLM_REASONING=off
LOCAL_LLM_SLOTS=1
LOCAL_LLM_SPEC=draft-mtp
LOCAL_LLM_THREADS=8
LOCAL_LLM_TIMEOUT=900
LOCAL_LLM_VERBOSITY=4
-----
ggml_metal_device_init: tensor API disabled for pre-M5 and pre-A19 devices
ggml_metal_library_init: using embedded metal library
ggml_metal_library_init: loaded in 0.025 sec
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
| qwen35 9B Q4_K - Medium        |   5.70 GiB |     9.20 B | MTL,BLAS   |       8 |     256 |           pp512 |        334.41 ± 0.19 |
| qwen35 9B Q4_K - Medium        |   5.70 GiB |     9.20 B | MTL,BLAS   |       8 |     256 |            tg64 |         31.84 ± 0.56 |

build: 9a532ae4b (9222)
```

Таким образом, ожидаемо на "разогретой" и загруженной в память модели можно получить до 334 t/s в процессе обработки промпта, и до 31 t/s генерации. 

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
  --kv-unified \
  --no-warmup \
  --threads $LOCAL_LLM_THREADS \
  --jinja \
  --cache-type-k $LOCAL_LLM_CACHEK \
  --cache-type-v $LOCAL_LLM_CACHEV \
  --spec-type $LOCAL_LLM_SPEC \
  --spec-draft-n-max $LOCAL_LLM_NMAX \
  --timeout $LOCAL_LLM_TIMEOUT \
  --parallel $LOCAL_LLM_SLOTS \
  --alias $LOCAL_LLM_ALIAS \
  --cache-ram $LOCAL_LLM_CACHERAM \
  --verbosity $LOCAL_LLM_VERBOSITY \
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

## 


log show --predicate 'process == "launchd"' --last 5m | grep com.user.llm-server

