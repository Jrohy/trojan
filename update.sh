#!/bin/bash

#获取当前的这个脚本所在绝对路径
SHELL_PATH=$(cd `dirname $0`; pwd)

cd $SHELL_PATH

mkdir -p web/templates

touch web/templates/test

go get -u

rm -rf web/templates