# godscan

## web应用指纹识别
- [x] HTTP响应 Server字段
- [x] 构造404 报错 得到中间件的详情
- POST请求构造报错 
* 解析html源代码 关键字匹配得到特征
* url特征 bing搜索引擎前十条
* 识别接口 从js里提取
* 词频统计
* 具体应用
  * 何种cms、组件
  * 版本识别 
    * v[\ \=]x.x.x
    * version=x.x.x
    * Vx.x.x
    * Ver x.x.x
* 图标哈希
* 版本识别
* 爬虫 递归访问

## 新增弱口令
- [x] 在fscan的基础上新增从若干个报告中获取到的弱口令