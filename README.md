# trojan
![](https://img.shields.io/github/v/release/Jrohy/trojan.svg) 
![](https://img.shields.io/docker/pulls/jrohy/trojan.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/Jrohy/trojan)](https://goreportcard.com/report/github.com/Jrohy/trojan)
[![Downloads](https://img.shields.io/github/downloads/Jrohy/trojan/total.svg)](https://img.shields.io/github/downloads/Jrohy/trojan/total.svg)
[![License](https://img.shields.io/badge/license-GPL%20V3-blue.svg?longCache=true)](https://www.gnu.org/licenses/gpl-3.0.en.html)


trojan多用户管理部署程序

## 功能
- 在线web页面和命令行两种方式管理trojan多用户
- 启动 / 停止 / 重启 trojan 服务端
- 支持流量统计和流量限制
- 命令行模式管理, 支持命令补全
- 集成acme.sh证书申请
- 生成客户端配置文件
- 在线实时查看trojan日志
- 在线trojan和trojan-go随时切换
- 支持trojan://分享链接和二维码分享(二维码仅限web页面)

## 安装方式
*trojan使用请提前准备好服务器可用的域名*  

###  a. 一键脚本安装
```
#安装/更新
source <(curl -sL https://git.io/trojan-install)

#卸载
source <(curl -sL https://git.io/trojan-install) --remove

```
安装完后输入'trojan'可进入管理程序   
浏览器访问 https://域名 可在线web页面管理trojan用户  
前端页面源码地址: [trojan-web](https://github.com/Jrohy/trojan-web)

### b. docker运行
1. 安装mysql  

因为mariadb内存使用比mysql至少减少一半, 所以推荐使用mariadb数据库
```
docker run --name trojan-mariadb --restart=always -p 3306:3306 -v /home/mariadb:/var/lib/mysql -e MYSQL_ROOT_PASSWORD=trojan -e MYSQL_ROOT_HOST=% -e MYSQL_DATABASE=trojan -d mariadb:10.2
```
端口和root密码以及持久化目录都可以改成其他的

2. 安装trojan
```
docker run -it -d --name trojan --net=host --restart=always --privileged jrohy/trojan init
```
运行完后进入容器 `docker exec -it trojan bash`, 然后输入'trojan'即可进行初始化安装   

启动web服务: `systemctl start trojan-web`   

设置自启动: `systemctl enable trojan-web`

更新管理程序: `source <(curl -sL https://git.io/trojan-install)`

## 运行截图
![avatar](asset/1.png)
![avatar](asset/2.png)

## 命令行
```
Usage:
  trojan [flags]
  trojan [command]

Available Commands:
  add         添加用户
  completion  自动命令补全(支持bash和zsh)
  del         删除用户
  help        Help about any command
  info        用户信息列表
  log         查看trojan日志
  restart     重启trojan
  start       启动trojan
  status      查看trojan状态
  stop        停止trojan
  tls         证书安装
  update      更新trojan
  updateWeb   更新trojan管理程序
  version     显示版本号
  web         以web方式启动

Flags:
  -h, --help   help for trojan
```

## 注意
安装完trojan后强烈建议开启BBR等加速: [Linux-NetSpeed](https://github.com/chiakge/Linux-NetSpeed)  

推荐的trojan客户端: 
   - pc: [Trojan-Qt5](https://github.com/TheWanderingCoel/Trojan-Qt5)
   - ios: [shadowrocket](https://apps.apple.com/us/app/shadowrocket/id932747118)
   - android: [igniter](https://github.com/trojan-gfw/igniter)
