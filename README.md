# trojan
![](https://img.shields.io/github/stars/Jrohy/trojan.svg) 
![](https://img.shields.io/github/forks/Jrohy/trojan.svg) 
[![Go Report Card](https://goreportcard.com/badge/github.com/Jrohy/trojan)](https://goreportcard.com/report/github.com/Jrohy/sshcopy)
[![Downloads](https://img.shields.io/github/downloads/Jrohy/trojan/total.svg)](https://img.shields.io/github/downloads/Jrohy/sshcopy/total.svg)


trojan多用户管理部署程序

## 功能
- 启动 / 停止 / 重启 trojan 服务端
- 支持流量统计和流量限制
- 命令行模式管理, 支持命令补全
- 多用户管理
- 集成acme.sh证书申请
- 生成客户端配置文件
- 支持trojan://分享链接

## 安装方式
*trojan使用请提前准备好服务器可用的域名*  

###  a. 一键脚本安装
```
#安装/更新
source <(curl -sL https://git.io/trojan-install)

#卸载
source <(curl -sL https://git.io/trojan-install) --remove

```
安装完后输入'trojan'即可进入管理程序

### b. 二进制文件运行
到release页面直接下载二进制文件后, 放到linux服务器上  
**因为trojan本身仅支持x86_64, 所以只编译了x86_64版本的管理程序**
```
chomd +x trojan
./trojan
```
若需要命令补全, 则需将trojan文件放置在/usr/local/bin/目录下, 然后运行
```
#bash环境
echo "source <(trojan completion bash)" >> ~/.bashrc
source ~/.bashrc

#zsh环境
echo "source <(trojan completion zsh)" >> ~/.zshrc
source ~/.zshrc
```
如果还是无法补全, 可能系统缺少bash-completion依赖, 需手动安装

### c. docker运行
1. 安装mysql
```
docker run --name trojan-mysql --restart=always -p 3306:3306 -v /home/mysql:/var/lib/mysql -e MYSQL_ROOT_PASSWORD=trojan -e MYSQL_ROOT_HOST=% -e MYSQL_DATABASE=trojan -d mysql/mysql-server:5.7
```
端口和root密码以及持久化目录都可以改成其他的

2. 安装trojan
```
docker run -it -d --name trojan --net=host --restart=always --privileged jrohy/trojan init
```
运行完后进入容器 `docker exec -it trojan bash`, 然后输入'trojan'即可进行初始化安装

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
  restart     重启trojan
  start       启动trojan
  status      查看trojan状态
  stop        停止trojan
  tls         证书安装
  update      更新trojan
  version     显示版本号

Flags:
  -h, --help   help for trojan
```

## 注意
安装完trojan后建议开启BBR等加速: [Linux-NetSpeed](https://github.com/chiakge/Linux-NetSpeed)
