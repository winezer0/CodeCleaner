# CodeCleaner - Go 代码文件清理工具

一个基于 Go 语言开发的命令行程序，用于清理指定目录及其子目录中的非代码文件。

## 功能特性

- 支持文件数量和类型统计模式
- 根据预设规则或自定义规则删除非代码文件
- 支持空文件检测、空目录检测, 删除无用路径
- 提供预览模式（Dry Run）
- 默认黑名单模式(仅移除指定后缀或预设后缀)
- 支持白名单模式(仅保留指定后缀或预设后缀) 
- 后缀配置时, 支持使用 none 表示 无后缀文件
- 使用 YAML 配置文件 实现预设的清理和存储后缀

## 安装

```bash
go build -o codecleaner main.go
```

## 使用方法

### 基本用法

```bash
# 使用 Go 预设清理当前目录
./codecleaner -p . -P go

# 预览模式（不实际删除）
./codecleaner -p . -P go -d

# 统计模式 统计后缀文件数量
./codecleaner --path . -s

# 统计模式 统计每个目录下文件数量
./codecleaner --path . -S

# 白名单模式（仅保留预设中指定的文件类型）
./codecleaner -p . -P go --white
```
## 日志配置

```bash
# 设置日志级别
./codecleaner -p . -P go --ll debug

# 设置日志文件
./codecleaner -p . -P go --lf cleanup.log

# 设置控制台日志格式
./codecleaner -p . -P go --cf "T L M"
```

## 注意事项

- 建议先使用统计模式确定需要删除的文件
- 统计模式优先级最高，启用后不执行删除操作
