#!/bin/bash

github_token=""

project="Jrohy/trojan"

#获取当前的这个脚本所在绝对路径
shell_path=$(cd `dirname $0`; pwd)

function uploadfile() {
	file=$1
	ctype=$(file -b --mime-type $file)

	curl -H "Authorization: token ${github_token}" -H "Content-Type: ${ctype}" --data-binary @$file "https://uploads.github.com/repos/$project/releases/${release_id}/assets?name=$(basename $file)"

	echo ""
}

function upload() {
	file=$1
	dgst=$1.dgst
	openssl dgst -md5 $file | sed 's/([^)]*)//g' >> $dgst
	openssl dgst -sha1 $file | sed 's/([^)]*)//g' >> $dgst
	openssl dgst -sha256 $file | sed 's/([^)]*)//g' >> $dgst
	openssl dgst -sha512 $file | sed 's/([^)]*)//g' >> $dgst
	uploadfile $file
	uploadfile $dgst
}

cd $shell_path

version=`git describe --tags $(git rev-list --tags --max-count=1)`
now=`TZ=Asia/Shanghai date "+%Y%m%d-%H%M"`
go_version=`go version|awk '{print $3,$4}'`
git_version=`git rev-parse HEAD`
ldflags="-w -s -X 'trojan/trojan.MVersion=$version' -X 'trojan/trojan.BuildDate=$now' -X 'trojan/trojan.GoVersion=$go_version' -X 'trojan/trojan.GitVersion=$git_version'"

GOOS=linux GOARCH=amd64 go build -ldflags "$ldflags" -o "result/trojan-linux-amd64" .
GOOS=linux GOARCH=arm64 go build -ldflags "$ldflags" -o "result/trojan-linux-arm64" .

if [[ $# == 0 ]];then
	cd result

	upload_item=($(ls -l|awk '{print $9}'|xargs -r))

	curl -X POST -H "Authorization: token ${github_token}" -H "Accept: application/vnd.github.v3+json" https://api.github.com/repos/$project/releases -d '{"tag_name":"'$version'", "name":"'$version'"}'

	sleep 2

	release_id=`curl -H 'Cache-Control: no-cache' -s https://api.github.com/repos/$project/releases/latest|grep id|awk 'NR==1{print $2}'|sed 's/,//'`

	for item in ${upload_item[@]}
	do
		upload $item
	done

	echo "upload completed!"

	cd $shell_path

	rm -rf result
fi
