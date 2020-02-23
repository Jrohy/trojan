# trojan

trojan多用户管理部署脚本

## 功能
- 启动 / 停止 / 重启 trojan 服务端
- 默认安装mysql, 以实现流量统计和流量限制
- 多用户管理
- 命令行模式管理trojan
- 集成acme.sh证书申请

## 安装方式
###  a. 一键脚本安装
```
#安装
source <(curl -sL https://git.io/trojan-install)

#卸载
source <(curl -sL https://git.io/trojan-install) --remove

```

### b. 二进制文件运行
到release页面直接下载二进制文件后, 放到linux服务器上  
**因为trojan本身仅支持x86_64, 所以只编译了x86_64版本的管理程序**
```
chomd +x trojan
./trojan
```

### c. docker运行
