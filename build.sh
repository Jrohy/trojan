#!/bin/bash

GITHUB_TOKEN=""

PROJECT="Jrohy/trojan"

#获取当前的这个脚本所在绝对路径
SHELL_PATH=$(cd `dirname $0`; pwd)

RELEASE_ID=`curl -H 'Cache-Control: no-cache' -s https://api.github.com/repos/$PROJECT/releases/latest|grep id|awk 'NR==1{print $2}'|sed 's/,//'`

function uploadfile() {
  FILE=$1
  CTYPE=$(file -b --mime-type $FILE)

  curl -H "Authorization: token ${GITHUB_TOKEN}" -H "Content-Type: ${CTYPE}" --data-binary @$FILE "https://uploads.github.com/repos/$PROJECT/releases/${RELEASE_ID}/assets?name=$(basename $FILE)"

  echo ""
}

function upload() {
  FILE=$1
  DGST=$1.dgst
  openssl dgst -md5 $FILE | sed 's/([^)]*)//g' >> $DGST
  openssl dgst -sha1 $FILE | sed 's/([^)]*)//g' >> $DGST
  openssl dgst -sha256 $FILE | sed 's/([^)]*)//g' >> $DGST
  openssl dgst -sha512 $FILE | sed 's/([^)]*)//g' >> $DGST
  uploadfile $FILE
  uploadfile $DGST
}

cd $SHELL_PATH

packr2

go build -ldflags "-s -w -X 'trojan/trojan.MVersion=`git describe --tags $(git rev-list --tags --max-count=1)`' -X 'trojan/trojan.BuildDate=`TZ=Asia/Shanghai date "+%Y%m%d-%H%M"`' -X 'trojan/trojan.GoVersion=`go version|awk '{print $3,$4}'`' -X 'trojan/trojan.GitVersion=`git rev-parse HEAD`'" -o "result/trojan" .

cd result

UPLOAD_ITEM=($(ls -l|awk '{print $9}'|xargs -r))

for ITEM in ${UPLOAD_ITEM[@]}
do
   upload $ITEM
done

echo "upload completed!"

cd $SHELL_PATH

packr2 clean

rm -rf result