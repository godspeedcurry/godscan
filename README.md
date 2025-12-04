# godscan
<h4 align="center">你的下一台扫描器，又何必是扫描器</h4>

<p align="center">
  <a href="https://goreportcard.com/report/github.com/godspeedcurry/godscan">
    <img src="https://goreportcard.com/badge/github.com/godspeedcurry/godscan">	
  </a>
  <a href="https://opensource.org/licenses/MIT">
    <img src="https://img.shields.io/badge/license-MIT-_red.svg">
  </a>
  <a href="https://github.com/godspeedcurry/godscan/releases">
  	<img src="https://img.shields.io/github/downloads/godspeedcurry/godscan/total">
  </a>
</p>

## 亮点
- Web 指纹 + API 抽取：GET/POST/404 多探针、favicon hash（fofa/hunter）、SimHash/关键词辅助识别。
- 爬虫情报：DFS 深度可调，自动存档 `spider.db`/`report.xlsx`，敏感信息 + 熵值高亮，CDN/OSS 提取。
- SourceMap 发现：从 JS/脚本标签自动探测同源 `.map`，记录到 `spider.db`/`sourcemaps.txt`，少量请求高命中。
- 端口服务识别：nmap 样式探针 + JDWP/HTTP 自定义规则，域名/IP/CIDR/范围混合输入。
- 弱口令生成：基础/变异/阴历模式，支持前后缀、分隔符；可直接喂在线爆破或 hashcat。
- 纯 Go，默认 `CGO_ENABLED=0`，提供 linux/windows/macos/freebsd 多平台包。

## 快速上手
```bash
# 单域名爬虫（指纹+API）
godscan sp -u https://example.com           # 简写：sp, ss
godscan sp -f urls.txt                      # 简写：--url-file/-f/-uf

# 目录扫描
godscan dir -u https://example.com          # 简写：dir, dirb, dd

# 端口扫描（支持域名/IP/CIDR/范围）
godscan port -i '1.2.3.4/28,example.com' -p 80,443

# 图标 hash
godscan icon -u https://example.com/favicon.ico

# 弱口令生成
godscan weak -k "foo,bar" --full

# 数据检索（正则）
godscan search --pattern "/api/v1" --db spider.db   # 去重搜索 api_paths/sensitive_hits，结果保存到 output/search_results.json

# 进度与输出
- 爬虫默认打印进度（10% 阶梯）和每个 URL 的开始日志，可用 `--progress-log=false` 关闭。
- 所有日志/产物写入 `--output-dir`（默认 output），包括 result.log、service.txt、spider_summary.json、search_results.json。
```

### 主要命令

```
Usage:
  godscan [flags]
  godscan [command]

Available Commands:
  clean       clean logs (Aliases: cc)
  completion  Generate the autocompletion script for the specified shell
  dirbrute    Dirbrute on sensitive file (Aliases: dir, dirb, dd)
  help        Help about any command
  icon        Calculate hash of an icon, eg: godscan icon -u http://example.com/favicon.ico (Aliases: ico)
  port        port scanner (Aliases: pp)
  spider      Analyze website using DFS, quick usage: -u (Aliases: sp, ss)
  weakpass    Start the application (Aliases: weak, wp, wk, ww)

Flags:
  -e, --filter stringArray    Filter url, eg: -e 'abc.com'
  -H, --headers stringArray   Custom headers, eg: -H 'Cookie: 123'
  -h, --help                  help for godscan
      --host string           singel host
      --host-file string      host file
-v, --loglevel int          level of your log (default 2)
-o, --output string         output file to write log and results (default "result.log")
-O, --output-dir string     directory to write logs/results/json (default "output")
      --private-ip            scan private ip
      --proxy string          proxy
      --ua string             set user agent (default "user agent")
  -u, --url string            singel url
  -f, --url-file string       url file (也接受 -uf file)

Use "godscan [command] --help" for more information about a command.
```



### 命令自动补全
```
./godscan completion zsh > /tmp/x
source /tmp/x
```

### 发布与版本
- 只要打新 tag，GitHub Actions + GoReleaser 会自动打包发布，覆盖 linux/windows/macos/freebsd (amd64/arm64/386/armv7)。
- 归档命名：`godscan_<version>_<os>_<arch>`，macOS 显示为 `macos`，windows 用 zip。
- 版本号从 tag 注入，手动本地构建默认显示 `dev`，如需本地指定版本：`go build -ldflags="-X github.com/godspeedcurry/godscan/cmd.version=vX.Y.Z"`。

## 开发
```bash
# 提交改动
git add . && git commit -m "fix bug" && git push -u origin main

# 发布（打 tag 即触发 GitHub Actions + GoReleaser 构建发布包，版本号自动注入二进制）
git tag -a v1.xx -m "v1.xx"
git push -u origin v1.xx

# 删除 tag（如需撤回发布）
git tag -d v1.xx
git push origin :refs/tags/v1.xx
```
### 日志清理
```
./godscan clean
```
### 基础功能一：目录扫描
#### 单一目录扫描
* `dirbrute`可简写为`dir`,`dirb`,`dd`
```bash
./godscan dirbrute --url http://www.example.com
```
1. 目录扫描数量较少，约50个，只针对渗透测试中容易造成数据泄漏、命令执行的几个点进行了探测
2. 目录扫描会根据域名生成对应的备份文件路径，说不定会有意外之喜(删掉了，实际出现的太少)
3. 要对单一url进行大线程、多文件探测，请使用[dirsearch](https://github.com/maurosoria/dirsearch)
4. 基于相似哈希算法计算页面哈希，用于资产量较大的情况分辨重点资产
5. 基于高德猜想——四等分点算法提取`0,1/4,1/2,3/4,1`处的关键字 辅助识别页面内容 NLP暂时不考虑 还没找到golang下很成熟的主题识别框架
6. 默认不跟随重定向，可使用`-L`跟随重定向

#### 批量目录扫描+指纹识别(基于golang协程)
```bash
./godscan dirbrute --url-file url.txt
```

### 基础功能二：根据图标地址计算图标hash
* 该hash为fofa的hash 本质是murmur3哈希
* 新增了hunter的hash 本质是md5
```bash
./godscan icon --url http://www.example.com/ico.ico
```
### 基础功能三：端口扫描
```bash
# 默认top 500
./godscan port -i '1.2.3.4/28'
# 自定义端口
./godscan port -i '1.2.3.4/28' -p '12312-12334,6379,22'
# 也支持域名列表，自动解析
# ./godscan port -i 'example.com,foo.bar'
# 指定top端口 数字范围在20000以内
./godscan port -i '1.2.3.4/28' --top 1000
# 第top100~第top200端口
./godscan port -i '1.2.3.4/28' --top 100-200
# 第top1000~第top2000端口
./godscan port -i '1.2.3.4/28' --top 1000-2000
```
* 使用nmap的探针规则进行探测,基于golang版本的nmap探针工具[vscan_go](https://github.com/RickGray/vscan-go)修改而来
* 加入了自定义规则，针对性识别JDWP和HTTP协议，目前市面上的扫描器不针对性实现的话无法扫到JDWP
```
Probe TCP myhttp q|GET / HTTP/1.1\r\n\r\n|
match http m|^HTTP| p/HTTP Protocol/
```

### 增强功能一：弱口令生成、离线爆破 ⭐️⭐️⭐️⭐️⭐️
* 默认基础模式，基础数量在数千级，可配合网络端在线使用或配合本地`hashcat`离线使用
* `--full`模式，生成数量在万级，建议只配合本地`hashcat`离线使用
* 会自动识别输入的身份证、电话号码，并根据常见的弱口令规则生成对应的弱口令
* `weakpass`可简写为`weak`, `wp`, `wk`, `ww`
* 新加入生日对应的阴历
```bash
./godscan weakpass -k "张三,110101199003070759,18288888888"
# 中文会被转成英文，以一定格式生成弱口令，如干饭集团，需要自己去找一下他在网站中经常提到的一些叫法
./godscan weakpass -k "干饭,干饭集团,干饭有限公司"

# 自定义前缀
./godscan weakpass -k "张三,110101199003070759,18288888888" --prefix '_'

# 自定义分隔符
./godscan weakpass -k "张三,110101199003070759,18288888888" --sep '@,_'

# 自定义后缀
./godscan weakpass -k "张三,110101199003070759,18288888888" --suffix '123,qwe,123456'

# 连起来
./godscan weakpass -k '百度,baidu.com,password,pass,root,server,qwer,admin' --prefix '@,!,",123' --suffix '!,1234,123,321' --sep '_,!,.,/,&,+' > 1.txt


# 查看工具默认的后缀
./godscan weakpass --show
# 更为复杂的前后缀，适合本地跑hashcat爆破,本方法还会对字符串作变异，如o->0,i->1,a->4等等
./godscan weakpass -k '百度' --full > 1.txt  

# -l 获取python格式的list 如["11","222"]
# mac下拷贝至剪贴板，其余系统可自行探索
./godscan weakpass -k "张三,110101199003070759,18288888888" | pbcopy
```


### 增强功能二：使用爬虫，爬取各类地址并尝试获取重要信息 ⭐️⭐️⭐️⭐️
* 目前会寻找url地址、密码、各类token
* `spider`命令可简写为`sp`、`ss`
```bash
./godscan spider --url http://example.com
# -d 1 可以指定爬虫的深度 默认为2 可以适当提高深度至3 
# 从文件批量爬取 适合针对API型资产 url不建议太多
./godscan spider --url-file url.txt   # 简写：-f url.txt 或 -uf url.txt
```
注意：godscan默认不扫内网IP，来减少卡顿，如果要扫内网IP,使用`--private-ip`参数即可

## 功能详细介绍
### web应用指纹识别 
- [x] HTTP响应 Server字段
- [x] 构造404 报错 得到中间件的详情
- [x] POST请求构造报错 
- [x] 爬虫 递归访问
- [x] 正则提取注释 注释里往往有版本 github仓库等信息(删了 误报太多)
- [x] 版本识别并高亮（删了 误报太多）
- [x] 对注释里的内容匹配到关键字并高亮(删了 误报太多)
- [x] 识别接口 主要从js里提取
- [x] url特征 人工看吧 有些组件的url是很有特征的 google: `inurl:/wh/servlet`
- [x] vue.js 前端 识别`(app|index|main|config).xxxx.js` 并使用正则提取里面的path
- [x] finger.txt来源
  * [Ehole](https://raw.githubusercontent.com/EdgeSecurityTeam/EHole/main/finger.json)
  * [nemasis](https://www.nemasisva.com/resource-library/Nemasis-Supported-Applications-Hardware-and-Platforms.pdf)
  * [chunsou](https://github.com/Funsiooo/chunsou)
  * [tide](https://github.com/TideSec/TideFinger)
- [x] 图标哈希
  - [x] 新增可直接根据图标地址计算hash的功能

### 功能二：弱口令生成器
- [x] 在fscan的基础上新增从实际渗透测试中获取到的弱口令
- [x] 根据给定的关键词生成大量弱口令


人工筛选时，可关注
* 域名 
  * 自身域名
  * 非自身域名 如开发商 很多默认密码和域名有千丝万缕的联系
* html注释


## 功能三：敏感信息搜集
参考： https://gh0st.cn/HaE/
这里面有很多现成的规则 挑了一下重点
- [x] 国内手机号
- [x] OSS accessKey accessId
- [x] link 识别  


---


## 跨平台编译
```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w " -trimpath -o godscan_linux_amd64 
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-s -w " -trimpath -o godscan_win_amd64
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w " -trimpath -o godscan_darwin_amd64
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w " -trimpath -o godscan_darwin_arm64
```


## 功能截图
* icon_hash计算、关键字识别
![image](https://github.com/godspeedcurry/godscan/blob/main/images/img1.png)

* 弱口令生成
![image](https://github.com/godspeedcurry/godscan/blob/main/images/img2.png)

* 敏感信息识别
![image](https://github.com/godspeedcurry/godscan/blob/main/images/img3.png)


## 开发
```
# develop && auto release
git add . && git commit -m "fix bug" && git push -u origin main
git tag -a v1.xx
git push -u origin v1.xx

# delete
git tag -d v1.xx
git push origin :refs/tags/v1.xx
```



## 更新说明
* 2024-08-21 支持带cookie访问
* 2024-07-01 新增全球端口top20000列表，`-p`参数保持不变
  * `--top 500`
  * `--top 500-1000`
  * `--top 2000-3000`
  * `-p 1-65535`
* 2024-06-29 
  * 支持从环境变量(http_proxy,https_proxy,all_proxy)读代理，支持http、socks5
  * 新增单目标api去重结果(`_api_unique.txt`)
  * 新增单目标api完整路径结果(`_api_unique_path.txt`)此时可使用dirbrute模块再递归扫一遍
  * dirbrute模块新增多个actuator目录遍历绕过，`..;/`的数量为1-5和8，共六个，用于暴力遍历寻找actuator接口,可与`_api_unique.path`结合使用
* 2024-04-15 修复map并发状态下的读写条件竞争，换成sync.Map
* 2024-04-09 优化日志输出
* 2024-01-11 修改从JS中寻找弱口令的正则，使用香农熵算法计算密码复杂度,使用表格显示
* 2023-09-13 新增js中提取api路径的功能
* 2023-08-11 新增目录单线程爆破的功能，并会根据域名爆破一些备份文件
* 2023-07-03 新增直接对icon计算hash的功能
* 2023-07-02 新增批量url关键路径扫描的功能
* 2022-08-01 新增部分真实场景中得到的弱口令 新增弱口令后缀，如123,qwe等，丰富生成后的弱口令
* 2022-07-04 修复没有子路径的bug, 移除packr 改用原生的embed库进行静态资源的打包
* 2022-06-10 更新了正则 对输出的表格进行了优化
* 2022-06-09 修复了大小写导致不高亮的问题
* 2022-06-08 修复了os.Open导致找不到文件的错误，改用packr库


## 免责声明

本工具仅面向合法授权的企业安全建设行为，如您需要测试本工具的可用性，请自行搭建靶机环境。 为避免被恶意使用，本项目所有技术均为理论方法，不存在任何漏洞利用过程，不会对目标发起真实攻击和漏洞利用。 在使用本工具进行检测时，您应确保该行为符合当地的法律法规，并且已经取得了足够的授权。请勿对非授权目标进行扫描。 如您在使用本工具的过程中存在任何非法行为，您需自行承担相应后果，我们将不承担任何法律及连带责任。
