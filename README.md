![Sync CLI](https://user-images.githubusercontent.com/81074/163512494-3c41c6c3-2158-49f8-9425-27637a3303c8.png)

# 简悦 · 同步助手 · 命令行 · 伪

模拟 [简悦 · 同步助手](http://ksria.com/simpread/docs/#/Sync) 的功能，需要高级账户。

## 详细说明

[简悦官方 Discussions](https://github.com/Kenshin/simpread/discussions/3704)

## 功能比较

| 功能                                 | 同步助手          | 命令行                   | 备注      |
|------------------------------------|---------------|-----------------------|---------|
| 自动同步                               | ●             | ●                     | -       |
| HTML、Markdown 导出                   | ●             | ●                     | -       |
| Email                              | ●             | ●                     | -       |
| Epub                               | ●             | ●                     | -       |
| Texbundle                          | ●             | ●                     | -       |
| PDF                                | ●             | ○                     | 客户端独有功能 |
| 内置解析                               | ●             | ○                     | 客户端独有功能 |
| 小书签                                | ●             | ○                     | 客户端独有功能 |
| 标注的自动同步（Hypothes.is / Readwise.io） | ●             | ○                     | 客户端独有功能 |
| 快照                                 | ●             | ●                     | -       |
| 客户端                                | Mac / Windows | Mac / Windows / Linux | -       |


## 使用

```sh
./simpread-sync -c <configPath>
```

config.json 默认配置如下：

```json
{
    "port": 7026,
    "syncPath": "",
    "outputPath": "",
    "smtpHost": "",
    "smtpPort": 465,
    "smtpUsername": "",
    "smtpPassword": "",
    "mailTitle": "[简悦] - {{title}}",
    "receiverMail": "",
    "kindleMail": ""
}
```

`syncPath` 必须填写，否则无法自动同步。

`outputPath` 如果不填写，默认为 `syncPath` 下的 output 文件夹。

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
