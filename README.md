# gorm-v1to-v2

A tool that automatically migrates legacy GORM v1 code (`github.com/jinzhu/gorm`) to modern GORM v2 (`gorm.io/gorm`).

一款自动化迁移工具，用于将项目中的 GORM v1 使用方式平滑迁移到 GORM v2，同时保持业务逻辑与行为不变。

## Quick start

> 运行和编译本工具需要 Go **1.24.3** 工具链；go.mod 已锁定该版本以确保兼容性。

```bash
# Run the migrator against your codebase (defaults to current directory)
go run ./cmd/gormv1to2 --path /path/to/your/project
```

### 示例项目：Legacy Shop（迁移练手）

仓库内新增了一个较为复杂的 GORM v1 示例项目，涵盖交易、预加载、多对多关系、Hook 校验与库存扣减等场景，便于直接体验迁移效果：

```bash
# 1) 进入示例项目（仍为 GORM v1 代码）
cd examples/legacy_shop

# 2) 运行迁移工具，将代码从 v1 转为 v2（会就地改写）
go run ../../cmd/gormv1to2 --path .

# 3) 查看迁移后的代码，验证业务逻辑保持一致
git diff
```

## 业务功能记录（迁移前后保持一致）

| 功能/行为 | 迁移前 | 迁移后 |
| --- | --- | --- |
| GORM 版本依赖 | 使用 `github.com/jinzhu/gorm` | 使用 `gorm.io/gorm` 并自动补充对应驱动 | 
| 数据库连接写法 | `gorm.Open("mysql", dsn)` 等方言字符串 | `gorm.Open(mysql.Open(dsn), &gorm.Config{})` 等 v2 写法 |
| 日志配置 | 可调用 `db.LogMode(true)` | 自动移除 `LogMode`，使用 v2 默认 logger（业务逻辑不变） |
| 业务代码 | 保持原有业务逻辑与数据流 | 保持不变，仅更新 ORM API |

> 该表用于记录迁移前后的功能状态，确保每次修改都不会改变业务功能。

## System architecture

```mermaid
flowchart TD
    GoTool["Go 1.24.3 toolchain"] --> CLI["cmd/gormv1to2 (CLI)"]
    CLI --> Transformer["internal/transform.Transformer"]
    Transformer --> Walker["filesystem walker"]
    Walker --> Parser["go/parser AST loader"]
    Parser --> Rewriter["AST rewrites (Open, LogMode, imports)"]
    Rewriter --> Formatter["gofmt + write"]

    subgraph Sample["examples/legacy_shop (GORM v1 demo)"]
        SampleMain["main.go (legacy app entrypoint)"]
        SampleSvc["service/checkout.go"]
        SampleModel["models/* (hooks, relations)"]
    end

    SampleMain -. run migrator .-> CLI
```

## Data flow

```mermaid
sequenceDiagram
    participant U as User
    participant Demo as examples/legacy_shop
    participant CLI as CLI
    participant FS as Filesystem
    participant AST as AST Rewriter

    U->>Demo: review legacy GORM v1 code (hooks, relations, Preload)
    U->>CLI: run gormv1to2 --path ./examples/legacy_shop
    CLI->>FS: enumerate *.go files (skip vendor)
    CLI->>AST: parse file into AST
    AST-->>AST: rewrite imports, gorm.Open, LogMode
    AST->>FS: format and persist updated Go file
    CLI-->>U: exit status + diff for verification
```

## Call graph (core path)

```mermaid
flowchart LR
    main[cmd/gormv1to2/main.go] --> transform[Transformer.Transform]
    transform --> transformFile[Transformer.TransformFile]
    transformFile --> rewriteOpen[rewriteGormOpen]
    transformFile --> rewriteLog[rewriteLogMode]
    transformFile --> pruneLog[removeLogModeStatements]
    transformFile --> ensureImports
    ensureImports --> versionLock[go.mod go 1.24.3 requirement]

    subgraph LegacyShop[examples/legacy_shop]
        entry[main.go]
        checkout[service/checkout.go]
        models[models/*.go]
    end
    versionLock -. used to migrate -> entry
```

## User-visible use cases

- As a maintainer, I can point the CLI at a repository to rewrite GORM v1 usage to GORM v2 idioms without changing business logic while adhering to Go 1.24.3.
- 作为开发者，可以在 `gorm.Open` 使用 MySQL、Postgres、SQLite 或 SQL Server 方言时自动插入对应驱动 import。
- 作为工程师，可以自动移除 `LogMode` 调用并使用 GORM v2 默认日志行为。
- 作为体验者，可以直接在 `examples/legacy_shop` 运行迁移工具，验证事务、预加载、多对多与 Hook 等场景在迁移前后保持一致。

## Go 版本说明

- go.mod 已升级至 `go 1.24.3`，确保编译、gofmt 与工具链行为与当前迁移器一致。
- 如需在更低版本编译，请先升级本地 Go 版本或调整 go.mod，但需要重新验证迁移行为。

## Migration rules implemented

- Replace `github.com/jinzhu/gorm` imports with `gorm.io/gorm` and add the correct driver import inferred from `gorm.Open` dialect strings.
- Convert `gorm.Open("dialect", dsn)` to `gorm.Open(dialectDriver.Open(dsn), &gorm.Config{})`.
- Drop legacy `LogMode` calls while keeping surrounding logic intact.
- Preserve other business logic and only update GORM-specific code paths.
