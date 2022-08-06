# 简悦 · 同步助手 · 命令行 · 伪

![Sync CLI](https://user-images.githubusercontent.com/81074/163512494-3c41c6c3-2158-49f8-9425-27637a3303c8.png)

模拟[简悦 · 同步助手](http://ksria.com/simpread/docs/#/Sync)的功能，需要高级账户。

## 详细说明

[简悦官方 Discussions](https://github.com/Kenshin/simpread/discussions/3704)

## 功能比较

| 功能                                        | 同步助手      | 命令行                | 备注           |
| ------------------------------------------- | ------------- | --------------------- | -------------- |
| 自动同步                                    | ●             | ●                     | -              |
| HTML、Markdown 导出                         | ●             | ●                     | -              |
| Email                                       | ●             | ●                     | -              |
| Epub                                        | ●             | ●                     | -              |
| Texbundle                                   | ●             | ●                     | -              |
| PDF                                         | ●             | ○                     | 客户端独有功能 |
| 内置解析                                    | ●             | ○                     | 客户端独有功能 |
| 小书签                                      | ●             | ○                     | 客户端独有功能 |
| 标注的自动同步（Hypothes.is / Readwise.io）| ●             | ○                     | 客户端独有功能 |
| 快照                                        | ●             | ●                     | -              |
| 客户端                                      | Mac / Windows | Mac / Windows / Linux | -              |

## 使用

### 配置

支持三种配置方式，参数名称参考下表。

| config.json    | 命令行参数         | 环境变量                | 默认值               |
| -------------- | ------------------ | ----------------------- | -------------------- |
| port           | -p/--port          | LISTEN_PORT             | 7026                 |
| syncPath       | --sync-path        | SYNC_PATH               | ""                   |
| outputPath     | --output-path      | OUTPUT_PATH             | ""                   |
| autoRemove     | --auto-remove      | AUTO_REMOVE             | False                |
| smtpHost       | --smtp-host        | SMTP_HOST               | ""                   |
| smtpPort       | --smtp-port        | SMTP_PORT               | 465                  |
| smtpUsername   | --smtp-username    | SMTP_USERNAME           | ""                   |
| smtpPassword   | --smtp-password    | SMTP_PASSWORD           | ""                   |
| mailTitle      | --mail-title       | MAIL_TITLE              | "[简悦] - {{title}}" |
| receiverMail   | --receiver-mail    | MAIL_RECEIVER           | ""                   |
| kindleMail     | --kindle-mail      | MAIL_KINDLE             | ""                   |
| enhancedOutput |                    |                         |                      |
|                | --{extension}-path | OUTPUT_PATH_{extension} |                      |

`syncPath` 必须填写，否则无法自动同步。

`outputPath` 如果不填写，默认为 `syncPath` 下的 output 文件夹。

如要使用 config.json 方式配置，可以通过 `-c`/`--config` 命令行参数指定配置文件路径，默认为当前工作目录下的 config.json 文件。

### 增强导出

在命令行参数和环境变量上的 `{extension}` 即为文件的扩展名，使用 config.json 则与其他两种配置方式有较大的不同。

假设在同步助手中的增强导出配置如下：

```jsonl
{"extension":"external", "path":"/Users/xxxx/xxxx/simpublish-demo/api/_output"}
{"extension":"pdf", "path":"/Users/xxxx/xxxx/Ebook"}
{"extension":"epub", "path":""}
{"extension":"docx", "path":""}
{"extension":"assets", "path":"/Users/xxxx/xxxx/Obsidian/SimpRead"}
{"extension":"textbundle", "path":""}
{"extension":"md", "path":"/Users/xxxx/xxxx/Obsidian/SimpRead"}
```

则命令行版的 config.json 中对应的配置应为：

```json
{
    "enhancedOutput": [
        {"extension":"external", "path":"/Users/xxxx/xxxx/simpublish-demo/api/_output"},
        {"extension":"pdf", "path":"/Users/xxxx/xxxx/Ebook"},
        {"extension":"epub", "path":""},
        {"extension":"docx", "path":""},
        {"extension":"assets", "path":"/Users/xxxx/xxxx/Obsidian/SimpRead"},
        {"extension":"textbundle", "path":""},
        {"extension":"md", "path":"/Users/xxxx/xxxx/Obsidian/SimpRead"}
    ]
}
```

**不支持特殊扩展名 `external`，但你可以通过配置两次 `html` 扩展名来实现相同的功能。**

更多配置请参考[如何配置增强导出](https://github.com/Kenshin/simpread/discussions/2958)。

### 部署

#### Linux

Linux 系统上推荐使用 systemd 进行部署，[AUR 软件包](https://aur.archlinux.org/packages/simpread-sync-git)已包含相关 service 文件 ，在其他 Linux 发行版上可能需要自行下载本仓库的 systemd 文件夹。

#### Windows

Windows 上建议使用计划任务，可参考[此文章](https://docs.syncthing.net/users/autostart.html#windows)。

#### Docker

参见[使用 Docker 部署简悦同步助手 · 命令行](https://github.com/Kenshin/simpread/discussions/4312)

## 如何更新

使用 `./simpread-sync -V` 来检查当前版本。（如有更新则会自动提示）

![image](https://user-images.githubusercontent.com/81074/162721397-ba796bc7-2d5a-4bd7-8472-f1aa7bfb3be0.png)

## 适用范围

1. Linux 用户
2. 因 AMD 显卡出现错误的用户

## 反馈渠道

[简悦官方反馈渠道](https://github.com/Kenshin/simpread/issues/3740)

## 其他

- [简悦官网](http://simpread.pro/)
- [简悦帮助](http://simpread.pro/help)
- [高级账户](http://ksria.com/simpread/docs/#/高级账户)
