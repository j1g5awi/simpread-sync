# 简悦 · 同步助手 · 命令行 · 伪

模拟「简悦 · 同步助手」的功能，需要高级账户。

## Features

- 自动同步
- 增强导出
  - [x] Markdown
  - [x] HTML
  - [ ] PDF
  - [x] epub
  - [x] 邮件
  - [x] Kindle

由于每次自动同步时都直接读取本地文件然后返回，文件大了可能会有性能问题。

## Usage

```sh
./simpread-sync <configPath>
```

启动时如果不带参数，配置文件默认为当前工作目录下的 config.json。

config.json 默认配置如下，`syncPath` 必须填写，否则无法自动同步。

`outputPath` 如果不填写，默认为 `syncPath` 下的 output 文件夹。

```json
{
    "port": 7026,
    "syncPath": "",
    "outputPath": "",
    "title": "[简悦] - %s",
    "smtpHost": "",
    "smtpPort": 465,
    "username": "",
    "password": "",
    "receiverMail": "",
    "kindleMail": ""
}
```
