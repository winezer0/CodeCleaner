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


- 首先使用 -s/-S 统计目录下的文件后缀数量 确定需要删除的文件后缀
- 然后使用 -P/-w 指定需要保留或需要移除的后缀类型

### 命令行支持
```
Usage:
  CodeCleaner [OPTIONS]

代码文件清理工具, 清理指定目录中的非代码文件

Application Options:
  -p, -path:           扫描起始目录路径
  -P, -preset:         使用预设清理规则(默认common) 或 ext-dir:逗号分割的后缀列表 (如ext: exe,txt)
  -c, -preset_config:  自定义 YAML 配置文件路径 (default: cleaner.yaml)
  -d, -dry_run         预览尝试模式：显示将删除的文件，不执行删除
  -w, -en_white        白名单模式：仅保留预设中 stored 指定的文件后缀类型
  -e, -rm_empty        移除空文件：启用时移除空目录和空文件路径
  -s, -stats_ext       启用统计模式：显示目录下(后缀类型) 数量分布
  -S, -stats_dir       启用统计模式：显示目录下(目录文件) 数量分布
  -v, -version         输出版本信息
      -lf:             Log file path (default: null)
      -ll:             Log level (debug-info-warn-error) (default: info)
      -cf:             Console log format (T L C M F combination or off|null to disable) (default: M)

Help Options:
  -h, -help            Show this help message
```
### 常用命令

```bash
# 统计模式 统计目录下各种后缀频率 常用
./codecleaner -p path/to/src/dir -s

# 统计模式 统计每个目录下文件数量 极少使用
./codecleaner -p path/to/src/dir -S

# 使用 Go 预设清理源码目录 黑名单模式（仅移除预设中 remove 键中指定的文件类型）
./codecleaner -p path/to/src/dir -P go 

# 使用 Go 预设清理源码目录 白名单模式（仅保留预设中 stored 键 指定的文件类型）
./codecleaner -p path/to/src/dir -P go -w

# 预览模式，仅显示操作,但是不实际进行删除
./codecleaner -p path/to/src/dir -P go -d

# 使用 自定义后缀清理目录 黑名单模式（移除ext:指定的 js html后缀文件类型, 移除 dir:指定的目录名）
./codecleaner -p path/to/src/dir -P ext:js,html,dir:temp

# 使用 自定义后缀清理目录 白名单模式（保留ext:指定的go后缀文件类型, 移除dir:指定的目录名）
./codecleaner -p path/to/src/dir -P ext:go,env,dir:temp -w


注意: dir:关键字当前只支持声明需要移除的目录, 和配置文件中的rmdirs键相同
```

### 配置文件格式
```
presets:
  java:
    description: "常见Java文件清理文档"
    stored:
      - java
      - class
      - jsp
      - jar
      - groovy
      - ini
      - prop
      - properties
      - xml
      - yml
      - yaml
    remove:
    rmdirs:
```
