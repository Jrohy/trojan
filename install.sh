#!/bin/bash
# Author: Jrohy
# github: https://github.com/Jrohy/trojan

#定义操作变量, 0为否, 1为是
help=0

remove=0

update=0

download_url="https://github.com/Jrohy/trojan/releases/download/"

version_check="https://api.github.com/repos/Jrohy/trojan/releases/latest"

service_url="https://raw.githubusercontent.com/Jrohy/trojan/master/asset/trojan-web.service"

[[ -e /var/lib/trojan-manager ]] && update=1

#Centos 临时取消别名
[[ -f /etc/redhat-release && -z $(echo $SHELL|grep zsh) ]] && unalias -a

[[ -z $(echo $SHELL|grep zsh) ]] && shell_way="bash" || shell_way="zsh"

#######color code########
red="31m"
green="32m"
yellow="33m"
blue="36m"
fuchsia="35m"

colorEcho(){
    color=$1
    echo -e "\033[${color}${@:2}\033[0m"
}

#######get params#########
while [[ $# > 0 ]];do
    key="$1"
    case $key in
        --remove)
        remove=1
        ;;
        -h|--help)
        help=1
        ;;
        *)
                # unknown option
        ;;
    esac
    shift # past argument or value
done
#############################

help(){
    echo "bash $0 [-h|--help] [--remove]"
    echo "  -h, --help           Show help"
    echo "      --remove         remove trojan"
    return 0
}

removeTrojan() {
    #移除trojan
    rm -rf /usr/bin/trojan >/dev/null 2>&1
    rm -rf /usr/local/etc/trojan >/dev/null 2>&1
    rm -f /etc/systemd/system/trojan.service >/dev/null 2>&1

    #移除trojan管理程序
    rm -f /usr/local/bin/trojan >/dev/null 2>&1
    rm -rf /var/lib/trojan-manager >/dev/null 2>&1
    rm -f /etc/systemd/system/trojan-web.service >/dev/null 2>&1

    systemctl daemon-reload

    #移除trojan的专用db
    docker rm -f trojan-mysql trojan-mariadb >/dev/null 2>&1
    rm -rf /home/mysql /home/mariadb >/dev/null 2>&1
    
    #移除环境变量
    sed -i '/trojan/d' ~/.${shell_way}rc
    source ~/.${shell_way}rc

    colorEcho ${green} "uninstall success!"
}

checkSys() {
    #检查是否为Root
    [ $(id -u) != "0" ] && { colorEcho ${red} "Error: You must be root to run this script"; exit 1; }

    arch=$(uname -m 2> /dev/null)
    if [[ $arch != x86_64 && $arch != aarch64 ]];then
        colorEcho $yellow "not support $arch machine".
        exit 1
    fi

    if [[ `command -v apt-get` ]];then
        package_manager='apt-get'
    elif [[ `command -v dnf` ]];then
        package_manager='dnf'
    elif [[ `command -v yum` ]];then
        package_manager='yum'
    else
        colorEcho $red "Not support OS!"
        exit 1
    fi

    # 缺失/usr/local/bin路径时自动添加
    [[ -z `echo $PATH|grep /usr/local/bin` ]] && { echo 'export PATH=$PATH:/usr/local/bin' >> /etc/bashrc; source /etc/bashrc; }
}

#安装依赖
installDependent(){
    if [[ ${package_manager} == 'dnf' || ${package_manager} == 'yum' ]];then
        ${package_manager} install socat crontabs bash-completion -y
    else
        ${package_manager} update
        ${package_manager} install socat cron bash-completion xz-utils -y
    fi
}

setupCron() {
    if [[ `crontab -l 2>/dev/null|grep acme` ]]; then
        if [[ -z `crontab -l 2>/dev/null|grep trojan-web` || `crontab -l 2>/dev/null|grep trojan-web|grep "&"` ]]; then
            #计算北京时间早上3点时VPS的实际时间
            origin_time_zone=$(date -R|awk '{printf"%d",$6}')
            local_time_zone=${origin_time_zone%00}
            beijing_zone=8
            beijing_update_time=3
            diff_zone=$[$beijing_zone-$local_time_zone]
            local_time=$[$beijing_update_time-$diff_zone]
            if [ $local_time -lt 0 ];then
                local_time=$[24+$local_time]
            elif [ $local_time -ge 24 ];then
                local_time=$[$local_time-24]
            fi
            crontab -l 2>/dev/null|sed '/acme.sh/d' > crontab.txt
            echo "0 ${local_time}"' * * * systemctl stop trojan-web; "/root/.acme.sh"/acme.sh --cron --home "/root/.acme.sh" > /dev/null; systemctl start trojan-web' >> crontab.txt
            crontab crontab.txt
            rm -f crontab.txt
        fi
    fi
}

installTrojan(){
    local show_tip=0
    if [[ $update == 1 ]];then
        systemctl stop trojan-web >/dev/null 2>&1
        rm -f /usr/local/bin/trojan
    fi
    lastest_version=$(curl -H 'Cache-Control: no-cache' -s "$version_check" | grep 'tag_name' | cut -d\" -f4)
    echo "正在下载管理程序`colorEcho $blue $lastest_version`版本..."
    [[ $arch == x86_64 ]] && bin="trojan-linux-amd64" || bin="trojan-linux-arm64" 
    curl -L "$download_url/$lastest_version/$bin" -o /usr/local/bin/trojan
    chmod +x /usr/local/bin/trojan
    if [[ ! -e /etc/systemd/system/trojan-web.service ]];then
        show_tip=1
        curl -L $service_url -o /etc/systemd/system/trojan-web.service
        systemctl daemon-reload
        systemctl enable trojan-web
    fi
    #命令补全环境变量
    [[ -z $(grep trojan ~/.${shell_way}rc) ]] && echo "source <(trojan completion ${shell_way})" >> ~/.${shell_way}rc
    source ~/.${shell_way}rc
    if [[ $update == 0 ]];then
        colorEcho $green "安装trojan管理程序成功!\n"
        echo -e "运行命令`colorEcho $blue trojan`可进行trojan管理\n"
        /usr/local/bin/trojan
    else
        if [[ `cat /usr/local/etc/trojan/config.json|grep -w "\"db\""` ]];then
            sed -i "s/\"db\"/\"database\"/g" /usr/local/etc/trojan/config.json
            systemctl restart trojan
        fi
        /usr/local/bin/trojan upgrade db
        if [[ -z `cat /usr/local/etc/trojan/config.json|grep sni` ]];then
            /usr/local/bin/trojan upgrade config
        fi
        systemctl restart trojan-web
        colorEcho $green "更新trojan管理程序成功!\n"
    fi
    setupCron
    [[ $show_tip == 1 ]] && echo "浏览器访问'`colorEcho $blue https://域名`'可在线trojan多用户管理"
}

main(){
    [[ ${help} == 1 ]] && help && return
    [[ ${remove} == 1 ]] && removeTrojan && return
    [[ $update == 0 ]] && echo "正在安装trojan管理程序.." || echo "正在更新trojan管理程序.."
    checkSys
    [[ $update == 0 ]] && installDependent
    installTrojan
}

main
