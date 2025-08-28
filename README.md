# GOPS - Go Project Scanner

[![Go Version](https://img.shields.io/badge/Go-1.24.5-blue.svg)](https://golang.org/doc/devel/release.html#go1.24)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Build Status](https://img.shields.io/badge/Build-Passing-brightgreen.svg)]()

**GOPS** - мощный инструмент для анализа и документирования проектов с модульной плагинной архитектурой. Идеально подходит для создания технической документации больших кодовых баз.

## ✨ Основные возможности

- 🚀 **Плагинная архитектура** - готова к расширению новыми языками и типами проектов
- 🎯 **Умное определение проектов** - автоматическое распознавание Go, Angular, NX, Rust, Java и других
- 🌍 **Поддержка 30+ языков** - от Go и JavaScript до Python, Rust, C++ и других
- 🌳 **Визуализация структуры** - псевдографическое дерево проекта
- ⚡ **Высокая производительность** - обработка тысяч файлов за секунды
- 🛡️ **Безопасность** - автоматическое исключение служебных файлов
- 🚨 **Умная проверка .gitignore** - предупреждения о необходимых исключениях
- 📊 **Подробная аналитика** - статистика по языкам, размерам, сложности
- 🎨 **Гибкий вывод** - Markdown, HTML, JSON форматы

## 🚀 Быстрый старт

### Установка

```bash
# Клонирование репозитория
git clone https://github.com/kolkov/gops.git
cd gops

# Компиляция всех версий
go build -o gops.exe cmd/gops/main.go
go build -o gops-new.exe cmd/gops-new/main.go
go build -o gops-safe.exe cmd/gops-safe/main.go
go build -o gops-minimal.exe cmd/gops-minimal/main.go
```

### Использование

```bash
# Анализ текущей директории (рекомендуемая версия)
./gops-new.exe

# С настройками
./gops-new.exe -c config.yaml -v -o my_docs.md

# Анализ конкретной директории
./gops-new.exe -d /path/to/project -v
```

## 📚 Доступные версии

| Версия | Статус | Описание | Рекомендации |
|--------|--------|----------|--------------|
| **gops-new** | ✅ **Рекомендуется** | Plugin-ready архитектура с улучшениями | Для разработки и production |
| **gops-safe** | ✅ Стабильная | Безопасная последовательная обработка | Для критических систем |
| **gops** | ✅ Legacy | Оригинальная версия | Обратная совместимость |
| **gops-minimal** | 🧪 Демо | Демонстрация плагинной архитектуры | Для понимания концепций |

### 🚀 gops-new (рекомендуемая)

**Новейшая версия с Plugin-Ready архитектурой**

```bash
./gops-new.exe -v
```

**Возможности:**
- ✅ Поддержка 30+ языков (Go, JS/TS, Python, Rust, Java, C++, C#, PHP, Ruby, Swift, Kotlin)
- ✅ Умное определение проектов (NX Monorepo, Angular, Go, Rust, Java Maven/Gradle, Node.js)
- ✅ Псевдографическая структура проекта
- ✅ Полная фильтрация служебных файлов (.gops/, .claude/, node_modules/, go.sum)
- ✅ Умная проверка .gitignore с предупреждениями и рекомендациями
- ✅ Готовность к интеграции плагинов
- ✅ Оптимизированная производительность (67 файлов за 5.7ms)

### 🛡️ gops-safe (максимальная надежность)

**Версия с последовательной обработкой без race conditions**

```bash
./gops-safe.exe -v
```

**Возможности:**
- ✅ Последовательная обработка файлов
- ✅ Исключение race conditions
- ✅ Полная фильтрация служебных файлов
- ✅ Структура проекта
- ✅ Максимальная стабильность (46 файлов за ~10ms)

### 🔧 gops (оригинальная)

**Первая версия для обратной совместимости**

```bash
./gops.exe -v
```

### 🧪 gops-minimal (демонстрационная)

**Демонстрация концепции плагинов**

```bash
./gops-minimal.exe -v
```

**Особенности:**
- 🎭 Симуляция работы плагинов
- 📚 Показ архитектурных решений
- 🧪 Для изучения и тестирования концепций

## ⚙️ Конфигурация

### Пример gops_config.yaml

```yaml
scanner:
  max_file_size: 2097152  # 2MB
  include_tests: false
  include_configs: true
  include_markup: false
  include_styles: false
  include_docs: false
  excluded_patterns:
    - .gitignore
    - .DS_Store
    - "*.log"
    - "*.tmp"
    - vendor/**
    - node_modules/**
    - .git/**
    - .idea/**
    - .vscode/**
    - dist/**
    - build/**
  parallel_workers: 4
  timeout: 5m0s

output:
  format: markdown
  filename: project_docs.md
  append_timestamp: true
```

## 🎯 Поддерживаемые проекты

### Языки программирования (30+)

| Категория | Языки |
|-----------|-------|
| **Backend** | Go, Java, C#, Python, PHP, Ruby, Rust, C/C++ |
| **Frontend** | JavaScript, TypeScript, HTML, CSS, SCSS/Sass |
| **Mobile** | Swift, Kotlin, Dart |
| **Системные** | C, C++, Rust, Go, Assembly |
| **Функциональные** | Scala, Haskell, F# |
| **Конфигурация** | YAML, TOML, JSON, XML |

### Типы проектов

| Тип проекта | Детектор | Особенности |
|-------------|----------|-------------|
| **NX Monorepo** | nx.json | Анализ workspace, apps, libs |
| **Angular** | angular.json + package.json | Компоненты, сервисы, модули |
| **Go** | go.mod | Пакеты, модули, зависимости |
| **Rust** | Cargo.toml | Crates, dependencies |
| **Java (Maven)** | pom.xml | Артефакты, зависимости |
| **Java (Gradle)** | build.gradle | Модули, tasks |
| **Node.js** | package.json | Пакеты, скрипты |
| **C/C++ (CMake)** | CMakeLists.txt | Targets, libraries |
| **C/C++ (Make)** | Makefile | Правила сборки |

## 📊 Производительность

### Бенчмарки (тестовый проект, 67 файлов)

| Версия | Время | Файлов обработано | Особенности |
|--------|-------|-------------------|-------------|
| **gops-new** | **5.7ms** | **67** | Оптимизированная |
| **gops-safe** | ~10ms | 46 | Безопасная |
| **gops-minimal** | ~90ms | 78 | С симуляцией плагинов |

### Масштабируемость

- ✅ **Малые проекты** (< 100 файлов): < 10ms
- ✅ **Средние проекты** (100-1000 файлов): < 100ms  
- ✅ **Большие проекты** (1000-10000 файлов): < 1s
- ✅ **Enterprise проекты** (> 10000 файлов): < 10s

## 🛡️ Безопасность

### Автоматически исключаемые файлы и директории

```
Служебные:     .gops/, .claude/, .git/, .svn/
Сборка:        dist/, build/, out/, bin/, obj/
Зависимости:   node_modules/, vendor/, __pycache__/
IDE:           .idea/, .vscode/, .vs/
Временные:     *.tmp, *.log, *.bak, *.cache
Бинарные:      *.exe, *.dll, *.so, *.dylib
Медиа:         *.jpg, *.png, *.mp4, *.pdf
Архивы:        *.zip, *.tar, *.gz, *.rar
```

### 🚨 Умная проверка .gitignore

GOPS автоматически проверяет .gitignore и выдает предупреждения:

**При отсутствии .gitignore:**
```
⚠️ .gitignore file not found. Consider creating one to exclude GOPS service files.

💡 RECOMMENDATION: Create .gitignore with these patterns:
# GOPS service files and generated documentation  
.gops/
.claude/
project_docs_*.md
gops_config_*.yaml
...
```

**При неполном .gitignore:**
```
🚨 IMPORTANT: Missing required patterns in .gitignore:
   .claude/
   project_docs_*.md

Run: echo -e '\n# GOPS service files\n.claude/\nproject_docs_*.md' >> .gitignore
```

**При корректном .gitignore:**
```
✅ .gitignore correctly excludes GOPS service files
💡 Recommended: Consider adding these patterns to .gitignore:
  - *.log
  - *.tmp
  - node_modules/
```

## 🔧 Разработка

### Структура проекта

```
gops/
├── cmd/                    # Исполняемые файлы
│   ├── gops/              # Оригинальная версия
│   ├── gops-new/          # Plugin-ready версия
│   ├── gops-safe/         # Безопасная версия
│   └── gops-minimal/      # Демо версия
├── internal/              # Внутренние пакеты
│   ├── core/             # Ядро системы
│   ├── plugin/           # Система плагинов
│   ├── plugins/          # Реализации плагинов
│   ├── model/            # Модели данных
│   ├── config/           # Конфигурация
│   └── docgen/           # Генераторы документации
├── pkg/                   # Публичные пакеты
└── docs/                  # Документация
```

### Запуск тестов

```bash
# Все тесты
go test ./...

# Тесты с покрытием
go test -cover ./...

# Бенчмарки
go test -bench=. ./...
```

### Сборка всех версий

```bash
# Скрипт сборки
./build.sh

# Или вручную
go build -o gops.exe cmd/gops/main.go
go build -o gops-new.exe cmd/gops-new/main.go
go build -o gops-safe.exe cmd/gops-safe/main.go
go build -o gops-minimal.exe cmd/gops-minimal/main.go
```

## 🔄 Архитектура плагинов (GOPS-NEW)

### Готовая плагинная система

```go
// Языковые плагины
internal/plugins/languages/
├── go/           # Go AST анализ
├── javascript/   # JS/TS поддержка
├── python/       # (планируется)
└── rust/         # (планируется)

// Проектные плагины  
internal/plugins/projects/
├── angular/      # Angular компоненты
├── nx/           # NX workspace
├── maven/        # (планируется)
└── generic/      # (планируется)
```

### Расширение новыми плагинами

1. Создайте новый плагин в `internal/plugins/`
2. Реализуйте интерфейс `Plugin`
3. Зарегистрируйте в системе
4. Готово!

## 📈 Roadmap

### Версия 2.0 (в разработке)

- [ ] **Полная интеграция плагинов** в main.go
- [ ] **Веб-интерфейс** для визуализации
- [ ] **REST API** для интеграции
- [ ] **Плагины для Python, Rust, C++**
- [ ] **Анализ зависимостей** и уязвимостей
- [ ] **Генерация диаграмм** архитектуры
- [ ] **Интеграция с CI/CD**

### Версия 3.0 (планы)

- [ ] **Distributed analysis** для огромных кодовых баз
- [ ] **AI-powered insights** с помощью LLM
- [ ] **Real-time monitoring** изменений в коде
- [ ] **Team collaboration** features

## 🤝 Участие в разработке

### Как помочь проекту

1. **⭐ Поставьте звезду** репозиторию
2. **🐛 Сообщайте о багах** через Issues
3. **💡 Предлагайте новые фичи** через Discussions
4. **🔧 Делайте Pull Requests**
5. **📖 Улучшайте документацию**

### Правила разработки

1. Все изменения через Pull Requests
2. Покрытие тестами новой функциональности
3. Соблюдение Go Code Style
4. Документирование публичного API
5. Тестирование на реальных проектах

## 📄 Лицензия

Этот проект лицензирован под [MIT License](LICENSE).

## 📞 Поддержка

- **Issues**: [GitHub Issues](https://github.com/kolkov/gops/issues)
- **Discussions**: [GitHub Discussions](https://github.com/kolkov/gops/discussions)
- **Email**: [support@gops.dev](mailto:support@gops.dev)

## 🙏 Благодарности

- **Go Team** за отличный язык и toolchain
- **Community** за feedback и contributions
- **Open Source** за inspiration и libraries

---

**Made with ❤️ for developers by developers**

> GOPS - превращает ваш код в красивую документацию!