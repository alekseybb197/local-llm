# Local LLM Backend for MacBook Pro M4 (24GB)

Локальный LLM-бэкенд на базе llama.cpp, оптимизированный для работы с плотными моделями через Metal на Apple Silicon.

[Инструкция по установке и настройке бэкенда](./backend.md)
[Настройка агентов для работы с LLM](./agents/README.md)

## Файлы

- [README.md](README.md) — краткое описание проекта
- [backend.md](backend.md) — детальная инструкция по установке и конфигурации бэкенда
- [agents/README.md](agents/README.md) — настройка агентов (Aider, Claude, Continue.dev и др.)
- [LICENSE](LICENSE) — лицензия Apache 2.0

## Скрипты

- [start.sh](start.sh) — запуск сервиса через launchd
- [stop.sh](stop.sh) — остановка сервиса
- [launch.sh](launch.sh) — запуск сервера с логированием
- [server.sh](server.sh) — тестовый запуск сервера
- [benchmark.sh](benchmark.sh) — бенчмарк скорости генерации (tokens/s)
- [show_models.sh](show_models.sh) — список доступных моделей на бэкенде
- [test1.sh](test1.sh) — тестовый запрос к API
- [test2.sh](test2.sh) — дополнительный тест

## Конфигурация

- [com.user.llm-server.plist](com.user.llm-server.plist) — файл launchd для автозапуска и перезапуска при крахе
- [sample.env-...](sample.env-*) — примеры файлов окружения для разных моделей

## Директории

- `models/` — загруженные модели в GGUF формате
- `tests/` — тесты для клиентов и бэкенда
- `agents/` — настройки и примеры конфигурации агентов

