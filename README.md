# godscan
<h4 align="center">你的下一台扫描器，又何必是扫描器</h4>
<h5 align="center">主动API/敏感信息识别与弱口令生成</h5>

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


## 1. 亮点
- 指纹 + API 抽取：GET/POST/404 探针、favicon hash（fofa/hunter）、SimHash/关键词。
- 爬虫：基于深度优先搜索，输出 `spider.db`/`report.xlsx`，提取敏感信息如API/CREDENTIALS/OSS等。
- SourceMap 发现：对 `.map` 自动探测
- 首页快照：保存首页 HTML/响应头，使用grep快速检索。
- 端口服务识别：nmap 样式探针 + 自定义 HTTP/JDWP 规则。
- 弱口令生成：基础/变异/阴历，前后缀/分隔符可自定义。

## 2. 快速上手
```bash
# 爬虫（指纹 + API + 敏感信息）
godscan sp -u https://example.com           # 简写 sp, ss
godscan sp -f urls.txt                      # 支持 -f/-uf 文件

# 爬虫跑完后：SourceMap/敏感数据/首页快照搜索
godscan grep "js.map"
godscan grep "elastic"   # body/header 查询

# 目录/端口/弱口令
godscan dir -u https://example.com
godscan port -i '1.2.3.4/28,example.com' -p 80,443
godscan weak -k "foo,bar" --full

# 导出离线 HTML 报表（大规模表格，自带分页/搜索）
godscan report --html report.html
# (optional) Generate LLM summary inside the HTML report（通过 profile 或 dry-run）
godscan llm -i                       # 交互式保存加密 profile（name 作为密钥）
godscan report --llm-profile prod    # 使用已保存的 profile 生成 LLM 摘要
godscan report --llm-dry-run         # 仅生成摘要预览（不调用模型）
```

## 3. 基础功能

### 批量目录扫描+指纹识别(基于golang协程)
```bash
./godscan dirbrute --url-file url.txt
```

### 根据图标地址计算图标hash
* fofa的hash 本质是murmur3哈希
* hunter的hash 本质是md5哈希
```bash
./godscan icon --url http://www.example.com/ico.ico
```
### 端口扫描
```bash
# 默认top 500
./godscan port -i '1.2.3.4/28'
# 自定义端口
./godscan port -i '1.2.3.4/28,example.com' -p '12312-12334,6379,22'
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
### 报表生成
```
./godscan report
./godscan report --html test.html
```

## 4. 增强功能
### 弱口令生成、离线爆破 ⭐️⭐️⭐️⭐️⭐️
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

### API资产推荐——使用爬虫，爬取各类地址并尝试获取重要信息 ⭐️⭐️⭐️⭐️⭐️
* 目前会寻找url地址、密码、各类token
* `spider`命令可简写为`sp`、`ss`
```bash
./godscan spider --url http://example.com
# -d 1 可以指定爬虫的深度 默认为2 可以适当提高深度至3 
# 从文件批量爬取 适合针对API型资产 url不建议太多
./godscan spider --url-file url.txt   # 简写：-f url.txt 或 -uf url.txt
# 首页内容会被写入数据库，可以直接搜索
./godscan grep 'xxxx' 
```
注意：godscan默认不扫内网IP，来减少卡顿，如果要扫内网IP,使用`--private-ip`参数即可

正则参考： https://gh0st.cn/HaE/
这里面有很多现成的规则 挑了一下重点
- [x] 国内手机号
- [x] OSS accessKey accessId
- [x] link 识别  
基于字符串混乱度（熵值）排序，可快速发现AK/SK等随机字符串

### 基于大模型进行文本总结 - Beta
gemini3每日有20次额度，可使用[API配置](https://aistudio.google.com/api-keys)获取key
```
# 配置ak，使用Profile Name为密钥进行加密存储
godscan llm 
# 基于已有报告或者爬虫现爬
godscan report
```
---
## 5. 输出
* `spider.db` / `report.xlsx`：指纹、API、敏感信息、CDN、SourceMap、首页快照。
* HTML 报表：离线可分页检索，可选 `--llm-key` 生成 “LLM Summary” 卡片，自动压缩爬虫长文本并输出潜在问题。
* LLM Profile：`output/config/llm_profiles.json.enc`，使用 `GODSCAN_LLM_SECRET` 加密存储 provider/model/key，命令 `godscan llm --name <n> --key ...`，生成报表时用 `--llm-profile <n>` 复用。
* `output/` 目录：`result.log`、`spider_summary.json`、`sourcemaps.txt` 等。

## 6. 跨平台编译 & 开发
```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w " -trimpath -o godscan_linux_amd64 
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-s -w " -trimpath -o godscan_win_amd64
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w " -trimpath -o godscan_darwin_amd64
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w " -trimpath -o godscan_darwin_arm64

# develop && auto release
git add . && git commit -m "fix bug" && git push -u origin main
git tag -a v1.xx
git push -u origin v1.xx

# delete
git tag -d v1.xx
git push origin :refs/tags/v1.xx
```


## 7. 功能截图
* icon_hash计算、关键字识别
![image](https://github.com/godspeedcurry/godscan/blob/main/images/img1.png)

* 弱口令生成
![image](https://github.com/godspeedcurry/godscan/blob/main/images/img2.png)

* 敏感信息识别
![image](https://github.com/godspeedcurry/godscan/blob/main/images/img3.png)

* 报表生成
![image](https://github.com/godspeedcurry/godscan/blob/main/images/img4.png)

## 8. 更新说明
* 2025-12-09 使用codex进行大面积重构和功能更新
* 2024-08-21 支持带cookie访问
* 2024-07-01 新增全球端口top20000列表，`-p`参数保持不变
  * `--top 500`
  * `--top 500-1000`
  * `--top 2000-3000`
  * `-p 1-65535`
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

## 9. 被淘汰的功能
1. 目录扫描根据域名生成对应的备份文件路径：实际出现的太少
2. 要对单一url进行大线程、多文件探测，请使用[dirsearch](https://github.com/maurosoria/dirsearch)
3. 正则提取注释 注释里往往有版本 github仓库等信息：误报太多
4. 版本识别并高亮：误报太多
5. 对注释里的内容匹配到关键字并高亮：误报太多

## 10. 专利功能
1. 基于相似哈希算法计算页面哈希，用于资产量较大的情况分辨重点资产
2. 基于高德猜想——四等分点算法提取`0,1/4,1/2,3/4,1`处的关键字 辅助识别页面内容 NLP暂时不考虑 还没找到golang下很成熟的主题识别框架

## 11. 致谢
* finger.txt来源
  * [Ehole](https://raw.githubusercontent.com/EdgeSecurityTeam/EHole/main/finger.json)
  * [nemasis](https://www.nemasisva.com/resource-library/Nemasis-Supported-Applications-Hardware-and-Platforms.pdf)
  * [chunsou](https://github.com/Funsiooo/chunsou)
  * [tide](https://github.com/TideSec/TideFinger)
## 12. 免责声明

本工具仅面向合法授权的企业安全建设行为，如您需要测试本工具的可用性，请自行搭建靶机环境。 为避免被恶意使用，本项目所有技术均为理论方法，不存在任何漏洞利用过程，不会对目标发起真实攻击和漏洞利用。 在使用本工具进行检测时，您应确保该行为符合当地的法律法规，并且已经取得了足够的授权。请勿对非授权目标进行扫描。 如您在使用本工具的过程中存在任何非法行为，您需自行承担相应后果，我们将不承担任何法律及连带责任。
