# CodeCleaner - Go 代码文件清理工具

一个基于 Go 语言开发的命令行程序，用于清理指定目录及其子目录中的非代码文件。

## 功能特性

- 递归扫描指定目录下的所有文件
- 根据预设规则或自定义规则删除非代码文件
- 支持文件类型统计模式
- 提供预览模式（Dry Run）
- 支持白名单模式
- 灵活的 YAML 配置文件

## 安装

```bash
go build -o codecleaner main.go
```

## 使用方法

### 基本用法

```bash
# 使用 Go 预设清理当前目录
./codecleaner -p . -r go

# 预览模式（不实际删除）
./codecleaner -p . -r go -t

# 统计模式
./codecleaner -p . -s

# 自定义删除扩展名
./codecleaner -p . -D "jpg,png,pdf,zip"

# 白名单模式（仅保留预设中指定的文件类型）
./codecleaner -p . -r go --white
```

### 命令行参数

| 参数 | 短标志 | 说明 |
|------|--------|------|
| `--path` | `-p` | 扫描起始目录路径（必需） |
| `--preset` | `-r` | 使用预设清理规则 |
| `--config` | `-c` | 自定义 YAML 配置文件路径 |
| `--delete` | `-D` | 强制指定要删除的文件扩展名 |
| `--stats` | `-s` | 启用统计模式 |
| `--try` | `-t` | 预览模式（不实际删除） |
| `--white` | | 白名单模式 |

## 配置文件

默认配置文件 `.cleaner.yaml`：

```yaml
presets:
  go:
    description: "Golang 项目清理"
    stored:
      - go
      - mod
      - sum
    remove:
      - exe
      - dll
      - jpg
      - png
      - pdf
      - zip
```

## 预设规则

工具内置了多种预设规则：

- `go`: Golang 项目清理
- `python`: Python 项目清理  
- `web`: Web 项目清理

## 日志配置

```bash
# 设置日志级别
./codecleaner -p . -r go --ll debug

# 设置日志文件
./codecleaner -p . -r go --lf cleanup.log

# 设置控制台日志格式
./codecleaner -p . -r go --cf "T L M"
```

## 注意事项

- 使用预览模式（`-t`）先检查将要删除的文件
- 白名单模式需要指定预设规则
- 统计模式优先级最高，启用后不执行删除操作