# CG Proto

Централизованное хранилище Proto контрактов для всех микросервисов CG Platform.

Go 1.24.5, gRPC v1.78.0.

## Структура

```
cg-proto/
├── users/                    # Домен пользователей
│   ├── auth/auth.proto       # Авторизация
│   ├── user/user.proto       # Профили
│   ├── garage/garage.proto   # Гараж (автомобили)
│   └── organization/organization.proto  # Организации
│
├── services/                 # Домен сервисов
│   ├── request/request.proto # Заявки
│   ├── bid/bid.proto         # Отклики
│   └── search/search.proto   # Поиск
│
├── communication/            # Домен коммуникаций
│   ├── chat/chat.proto       # Чаты
│   └── notification/notification.proto  # Уведомления
│
├── platform/                 # Инфраструктура
│   ├── counter/counter.proto # Счётчики
│   └── nsi/nsi.proto         # Справочники
│
├── jobs/                     # Домен вакансий
│   └── job/job.proto         # Вакансии и резюме
│
├── workshop/                 # Домен мастерских
│   └── workshop.proto        # Ремонтные заказы, мастера, зарплата
│
├── orders/                   # Домен заказов
│   └── order/v1/order.proto  # Заказы запчастей
│
├── payments/                 # Домен платежей
│   └── payment/v1/payment.proto  # Платежи
│
├── billing/                  # Домен биллинга
│   └── billing/v1/billing.proto  # Биллинг
│
├── booking/                  # Домен бронирования
│   └── booking/v1/booking.proto  # Бронирование
│
├── ai/                       # Домен AI
│   └── chatbot.proto         # AI чатбот
│
├── gen/go/                   # Сгенерированный Go код
│   ├── users/
│   │   ├── auth.pb.go        # Protobuf типы
│   │   ├── auth_grpc.pb.go   # gRPC сервисы
│   │   └── ...
├── buf.yaml                  # Buf конфигурация
├── buf.gen.yaml              # Генерация кода (стандартный gRPC)
└── go.mod
```

## Установка

```bash
go get gitlab.com/xakpro/cg-proto@latest
```

## buf.gen.yaml

Генерирует только стандартный gRPC код (protobuf типы + gRPC сервисы):

```yaml
version: v1
plugins:
  # Standard protobuf Go code (типы)
  - plugin: buf.build/protocolbuffers/go
    out: gen/go
    opt:
      - paths=source_relative

  # Standard gRPC сервисы (*_grpc.pb.go)
  - plugin: buf.build/grpc/go
    out: gen/go
    opt:
      - paths=source_relative
```

**Генерируемые файлы:**
- `.pb.go` -- protobuf типы
- `*_grpc.pb.go` -- стандартный gRPC сервисы

## Использование

### Server (gRPC Server)

```go
import (
    pb "gitlab.com/xakpro/cg-proto/gen/go/services/search"
    sharedGRPC "gitlab.com/xakpro/cg-shared-libs/grpc"
)

// Implement the gRPC service interface
type SearchHandler struct {
    pb.UnimplementedSearchServiceServer
    searchService SearchServiceI
}

func (h *SearchHandler) Search(
    ctx context.Context,
    req *pb.SearchRequest,
) (*pb.SearchResponse, error) {
    // Your logic here
    return &pb.SearchResponse{
        Results: []*pb.SearchResult{},
    }, nil
}

// Register handler
func main() {
    grpcServer, err := sharedGRPC.NewServer(cfg.GRPC)
    if err != nil {
        log.Fatal(err)
    }

    pb.RegisterSearchServiceServer(grpcServer.Server(), &SearchHandler{})

    if err := grpcServer.Start(); err != nil {
        log.Fatal(err)
    }
}
```

### Client (gRPC Client)

```go
import (
    pb "gitlab.com/xakpro/cg-proto/gen/go/services/search"
    sharedGRPC "gitlab.com/xakpro/cg-shared-libs/grpc"
)

// Create client
conn, err := sharedGRPC.NewClient(ctx, cfg)
if err != nil {
    log.Fatal(err)
}
defer conn.Close()

client := pb.NewSearchServiceClient(conn.Conn())

// Call method
resp, err := client.Search(ctx, &pb.SearchRequest{
    Query: "test",
})
if err != nil {
    // Handle error (status.Code(err) returns error code)
}
```

## Генерация кода

```bash
# Установить buf
brew install bufbuild/buf/buf

# Сгенерировать Go код
buf generate

# Проверить линтером
buf lint

# Проверить breaking changes
buf breaking --against '.git#branch=main'
```

## Правила именования

### Пакеты

```protobuf
package {domain}.{service}.v1;

// Примеры:
package users.auth.v1;
package communication.chat.v1;
```

### Go Package

```protobuf
// Формат: путь;alias
option go_package = "gitlab.com/xakpro/cg-proto/gen/go/users/auth/v1;authv1";
```

### Сервисы

```protobuf
// Имя сервиса = Имя файла + Service
service AuthService { ... }
service RequestService { ... }
service ChatService { ... }
```

### Методы

```protobuf
// CRUD
rpc Create{Entity}(...) returns (...);
rpc Get{Entity}(...) returns (...);
rpc Update{Entity}(...) returns (...);
rpc Delete{Entity}(...) returns (...);
rpc List{Entities}(...) returns (...);

// Бизнес-действия
rpc SendCode(...) returns (...);
rpc VerifyCode(...) returns (...);
```

### Сообщения

```protobuf
// Request/Response
message Get{Entity}Request { ... }
message Get{Entity}Response { ... }

// Entity
message User { ... }
message Message { ... }
```

## gRPC Interceptors

```go
import (
    "google.golang.org/grpc"
    sharedGRPC "gitlab.com/xakpro/cg-shared-libs/grpc"
)

// Используйте interceptors из cg-shared-libs/grpc
grpcServer, err := sharedGRPC.NewServer(cfg.GRPC,
    grpc.ChainUnaryInterceptor(
        customInterceptor1,
        customInterceptor2,
    ),
)

// Встроенные interceptors:
// - recoveryInterceptor (автоматически)
// - loggingInterceptor (автоматически)
// - timeoutInterceptor (автоматически)
// - AuthInterceptor (через sharedGRPC.AuthInterceptor)
```

## Error Handling

```go
import (
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
)

// Return error
return nil, status.Error(codes.NotFound, "user not found")

// Check error code
if status.Code(err) == codes.NotFound {
    // Handle not found
}

// Common codes:
// - codes.InvalidArgument
// - codes.NotFound
// - codes.Unauthenticated
// - codes.PermissionDenied
// - codes.Internal
```

## Версионирование

### Semver

```
v1.0.0 - первый стабильный релиз
v1.1.0 - добавлены новые методы (backward compatible)
v1.1.1 - исправления
v2.0.0 - breaking changes
```

### Breaking Changes

Что считается breaking change:
- Удаление поля
- Изменение типа поля
- Изменение номера поля
- Удаление метода
- Изменение сигнатуры метода

Что НЕ breaking change:
- Добавление нового поля
- Добавление нового метода
- Добавление нового enum value

## Проверка перед релизом

```bash
# Lint
buf lint

# Breaking changes против main
buf breaking --against '.git#branch=main'

# Генерация
buf generate

# Commit и создание тега
git add .
git commit -m "feat: add new method to AuthService"
git tag -a v1.1.0 -m "Release v1.1.0: описание изменений"
git push origin main
git push origin v1.1.0
git push github v1.1.0  # если используете GitHub mirror
```

## Обновление зависимостей в других проектах

После создания нового тега версии, обновите зависимость в проектах:

### Для проектов с GitHub replace

```bash
# В каждом сервисе обновите версию в go.mod (require + replace)
# Затем:
go mod tidy
go build ./...
go vet ./...
```

### Для проектов с локальным replace

Проекты с `replace gitlab.com/xakpro/cg-proto => ../cg-proto` автоматически используют последнюю версию из локальной директории. Просто выполните:

```bash
go mod tidy
```
