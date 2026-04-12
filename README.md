# Jevon CRM — Go Backend

## Стек
- **Go 1.22** + Gin
- **PostgreSQL** (удалённый сервер)
- **JWT** (access 15m + refresh 7d)
- **golang-migrate** — миграции БД
- **Air** — hot reload при разработке
- **Swagger** — автодокументация API

---

## Быстрый старт
Создание билда

Создаем билд на фронте через
npm run build
Далее создаем билд на стороне сервера 
go build -tags prod -ldflags="-s -w" -o jevon_prod.exe ./cmd/api/
Копируем exe на сервер и запускаем

## Сбрось всех данных
-- ============================================================
-- Jevon CRM — Скрипт полного сброса данных
-- Оставляет: пользователей, роли, каталоги, номенклатуру, поставщиков
-- Удаляет: заказы, проекты, склад, расходы, задачи, табели
-- ============================================================

BEGIN;

-- ── Расходные накладные ──────────────────────────────────
DELETE FROM outgoing_invoice_items;
DELETE FROM outgoing_invoices;

-- ── Приходные накладные ──────────────────────────────────
DELETE FROM warehouse_payments;
DELETE FROM warehouse_receipt_items;
DELETE FROM warehouse_receipts;

-- ── Остатки склада ───────────────────────────────────────
DELETE FROM warehouse_expenses;

-- ── Заказы и всё связанное ───────────────────────────────
DELETE FROM order_materials;
DELETE FROM order_stage_assignees;
DELETE FROM order_history;
DELETE FROM order_comments;
DELETE FROM order_payments;
DELETE FROM order_expenses;
DELETE FROM order_files;
DELETE FROM order_items;
DELETE FROM order_detail_estimates;
DELETE FROM order_estimate_materials;
DELETE FROM order_estimate_services;
DELETE FROM order_estimate_settings;
DELETE FROM order_calculations;
DELETE FROM order_stages;
DELETE FROM orders;

-- ── Проекты ──────────────────────────────────────────────
DELETE FROM project_members;
DELETE FROM project_stage_assignees;
DELETE FROM project_history;
DELETE FROM stage_files;
DELETE FROM stage_operations;
DELETE FROM operation_materials;
DELETE FROM project_stages;
DELETE FROM projects;

-- ── Задачи ───────────────────────────────────────────────
DELETE FROM tasks;

-- ── Расходы цеха ─────────────────────────────────────────
DELETE FROM workshop_expenses;

-- ── Клиенты ──────────────────────────────────────────────
DELETE FROM client_balance;
DELETE FROM clients;

-- ── Табели и зарплаты ────────────────────────────────────
DELETE FROM salary_payments;
DELETE FROM timesheets;
DELETE FROM master_wages;

-- ── Зарплаты сотрудников (ставки) ────────────────────────
UPDATE users SET salary = NULL, hourly_rate = NULL;

-- ── Сброс последовательностей (если есть) ────────────────
-- Сброс счётчика расходных накладных EXT-XXX
ALTER SEQUENCE IF EXISTS outgoing_invoice_ext_seq RESTART WITH 1;

COMMIT;

-- ============================================================
-- Проверка — должны вернуть 0
-- ============================================================
SELECT 'orders'                AS tbl, COUNT(*) FROM orders
UNION ALL SELECT 'outgoing_invoices',  COUNT(*) FROM outgoing_invoices
UNION ALL SELECT 'warehouse_receipts', COUNT(*) FROM warehouse_receipts
UNION ALL SELECT 'warehouse_expenses', COUNT(*) FROM warehouse_expenses
UNION ALL SELECT 'clients',            COUNT(*) FROM clients
UNION ALL SELECT 'projects',           COUNT(*) FROM projects
UNION ALL SELECT 'tasks',              COUNT(*) FROM tasks
UNION ALL SELECT 'workshop_expenses',  COUNT(*) FROM workshop_expenses
UNION ALL SELECT 'timesheets',         COUNT(*) FROM timesheets;

### 1. Установи инструменты

```bash
# Air — hot reload
go install github.com/air-verse/air@latest

# Swag — генерация Swagger документации
go install github.com/swaggo/swag/cmd/swag@latest
```

### 2. Настрой .env

```bash
cp .env.example .env
```

Заполни `.env`:
```env
PORT=8181
DB_HOST=your-remote-host
DB_PORT=5432
DB_USER=your_db_user
DB_PASSWORD=your_db_password
DB_NAME=jevon_crm
DB_SSLMODE=disable
JWT_ACCESS_SECRET=your_secret_min_32_chars_here!!
JWT_REFRESH_SECRET=your_refresh_secret_32_chars!!
CORS_ALLOWED_ORIGINS=http://localhost:3000
```

### 3. Установи зависимости

```bash
go mod tidy
```

### 4. Сгенерируй Swagger документацию

```bash
swag init -g cmd/api/main.go -o docs
```

### 5. Запуск

**Разработка (с hot reload):**
```bash
air
```

**Продакшн:**
```bash
go build -o ./bin/server ./cmd/api
./bin/server
```

---

## Миграции

Миграции запускаются **автоматически** при старте сервера.

Если нужно запустить вручную:
```bash
# Установи migrate CLI
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Применить миграции
migrate -path ./migrations -database "postgres://user:pass@host:5432/jevon_crm?sslmode=disable" up

# Откатить последнюю
migrate -path ./migrations -database "..." down 1
```

**Файлы миграций:**
```
migrations/
├── 000001_init_schema.up.sql    ← создание таблиц
├── 000001_init_schema.down.sql  ← удаление таблиц
├── 000002_seed_admin.up.sql     ← дефолтный admin
└── 000002_seed_admin.down.sql   ← удаление admin
```

**Дефолтный admin после миграции:**
```
Email:    admin@jevon.uz
Password: Admin@1234
```
⚠️ Смени пароль сразу после первого входа!

---

## API эндпоинты

| Метод  | URL                          | Роль               | Описание              |
|--------|------------------------------|--------------------|-----------------------|
| POST   | /api/auth/login              | публичный          | Вход                  |
| POST   | /api/auth/refresh            | публичный          | Обновить токен        |
| POST   | /api/auth/logout             | публичный          | Выход                 |
| GET    | /api/dashboard/stats         | все                | Статистика            |
| GET    | /api/users                   | admin, supervisor  | Список сотрудников    |
| POST   | /api/users                   | admin              | Создать сотрудника    |
| GET    | /api/users/:id               | admin, supervisor  | Карточка сотрудника   |
| PATCH  | /api/users/:id/toggle-active | admin              | Активировать/блокировать |
| GET    | /api/projects                | все                | Список проектов       |
| POST   | /api/projects                | admin, supervisor  | Создать проект        |
| PATCH  | /api/projects/:id            | admin, supervisor  | Обновить проект       |
| DELETE | /api/projects/:id            | admin              | Отменить проект       |
| GET    | /api/tasks                   | все                | Список задач          |
| POST   | /api/tasks                   | admin, supervisor  | Создать задачу        |
| PATCH  | /api/tasks/:id               | все                | Обновить задачу       |
| PATCH  | /api/tasks/:id/status        | все                | Сменить статус        |
| DELETE | /api/tasks/:id               | admin, supervisor  | Удалить задачу        |

**Swagger UI:** http://localhost:8181/swagger/index.html

---

## Структура проекта

```
jevon-backend/
├── cmd/api/
│   └── main.go                  ← точка входа, роутер
├── internal/
│   ├── config/config.go         ← загрузка .env
│   ├── db/
│   │   ├── db.go                ← подключение к PostgreSQL
│   │   └── migrate.go           ← запуск миграций
│   ├── auth/jwt.go              ← генерация и парсинг JWT
│   ├── middleware/auth.go       ← RequireAuth, RequireRole
│   ├── models/models.go         ← структуры + DTO
│   ├── repository/
│   │   ├── users.go             ← SQL запросы для users
│   │   ├── projects.go          ← SQL запросы для projects
│   │   └── tasks.go             ← SQL запросы для tasks + dashboard
│   └── handlers/handlers.go    ← HTTP handlers (все в одном)
├── migrations/
│   ├── 000001_init_schema.up.sql
│   ├── 000001_init_schema.down.sql
│   ├── 000002_seed_admin.up.sql
│   └── 000002_seed_admin.down.sql
├── docs/                        ← генерируется командой swag init
├── .air.toml                    ← конфиг hot reload
├── .env.example                 ← шаблон переменных окружения
├── .gitignore
└── go.mod
```

---

## Добавление новой миграции

```bash
# Создай файлы вручную со следующим номером
# Например: 000003_add_materials_table.up.sql
```

Формат имени: `{номер}_{описание}.up.sql` и `{номер}_{описание}.down.sql`
"# jevon-backend" 
