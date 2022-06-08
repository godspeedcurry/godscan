# godscan

## web应用指纹识别
- [x] HTTP响应 Server字段
- [x] 构造404 报错 得到中间件的详情
- [x] POST请求构造报错 
- [x] 解析html源代码 关键字匹配得到特征, 根据指纹特征进行词频统计, 并表格化输出
- [x] 爬虫 递归访问
- [x] 正则提取注释 注释里往往有版本 github仓库等信息
- [x] 版本识别 一般会有多个 正则实现 如下均可识别
```
版本 4.x
v6
v1.11.3
version 2.1
version: 4.2.2
v1.7.2
v2.1.1
版本 5.x
```
- [x] 识别接口 从js里提取
- [x] url特征 人工看吧 有些组件的url是很有特征的 google: `inurl:/wh/servlet`
- [x] finger.txt来源
  * Ehole https://raw.githubusercontent.com/EdgeSecurityTeam/EHole/main/finger.json
  * https://www.nemasisva.com/resource-library/Nemasis-Supported-Applications-Hardware-and-Platforms.pdf
  
* 图标哈希 todo

## 新增弱口令
- [x] 在fscan的基础上新增从若干个报告中获取到的弱口令

## 弱口令自动生成
todo

## log4j 扫描 
todo
要能指定listener
好用的dnslog


## 
```
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(go env GOPATH)/bin/packr build -ldflags="-s -w " -trimpath -o godscan_linux_amd64 
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(go env GOPATH)/bin/packr build -ldflags="-s -w " -trimpath -o godscan_win_amd64
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(go env GOPATH)/bin/packr build -ldflags="-s -w " -trimpath -o godscan_darwin_amd64
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(go env GOPATH)/bin/packr build -ldflags="-s -w " -trimpath -o godscan_darwin_arm64
```


## 更新说明
* 修复了os.Open导致找不到文件的错误，改用packr库


![image](https://github.com/godspeedcurry/godscan/blob/master/images/img1.png)