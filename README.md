# godscan

## web应用指纹识别 [DONE]
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
- [x] 对注释里的内容匹配到关键字并高亮
- [x] 识别接口 从js里提取
- [x] url特征 人工看吧 有些组件的url是很有特征的 google: `inurl:/wh/servlet`
- [x] vue.js 前端 识别app.xxxx.js 并使用正则提取里面的path
- [x] finger.txt来源
  * Ehole https://raw.githubusercontent.com/EdgeSecurityTeam/EHole/main/finger.json
  * https://www.nemasisva.com/resource-library/Nemasis-Supported-Applications-Hardware-and-Platforms.pdf
  
- [x] 图标哈希

## 新增弱口令 [DONE]
- [x] 在fscan的基础上新增从若干个报告中获取到的弱口令

## 敏感信息搜集 [DONE]
https://gh0st.cn/HaE/
这里面有很多现成的规则 挑了一下重点
- [x] JSON-WEB-Token
- [x] 国内手机号
- [x] 邮箱
- [x] hmtl注释
- [x] ueditor swagger 等
- [x] OSS accessKey accessId
- [x] link 识别  


## 弱口令生成模块 [DONE]
* 人工筛选 筛选可关注
  * 域名 
    * 自身域名
    * 非自身域名 如开发商 很多默认密码和域名有千丝万缕的联系
  * html注释
* 根据给定的keyword list 生成 逗号分隔

```
go run main.go -k "张三,110101199003070759,18288888888"                    
```



## 端口扫描+协议识别 TODO
https://github.com/4dogs-cn/TXPortMap
https://github.com/redtoolskobe/scaninfo

## web poc
todo

## 各端口弱口令爆破
todo

## log4j 扫描模块
todo
要能指定listener
好用的dnslog



## 跨平台编译
```
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w " -trimpath -o godscan_linux_amd64 
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-s -w " -trimpath -o godscan_win_amd64
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w " -trimpath -o godscan_darwin_amd64
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w " -trimpath -o godscan_darwin_arm64
```


## 更新说明
* 2022-07-04 修复没有子路径的bug, 移除packr 改用原生的embed库进行静态资源的打包
* 2022-06-10 更新了正则 对输出的表格进行了优化
* 2022-06-09 修复了大小写导致不高亮的问题
* 2022-06-08 修复了os.Open导致找不到文件的错误，改用packr库

## TODO
弱口令生成有点问题
尽快支持--only-url功能

![image](https://github.com/godspeedcurry/godscan/blob/master/images/img1.png)


![image](https://github.com/godspeedcurry/godscan/blob/master/images/img2.png)


![image](https://github.com/godspeedcurry/godscan/blob/master/images/img3.png)