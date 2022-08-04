# errorformator

format go err with httpcode、businesscode、message struct


## 业务架构图
```plantuml
@startuml
!define onlineBusiness rectangle #lightgreen
!define db  rectangle #Implementation

onlineBusiness "在线业务A" as onlineBusinessA
onlineBusiness "在线业务B" as onlineBusinessB
archimate #Strategy "数据中台" as dataCenter  <<technology-device>> 
db "数据库" as mysql 
db "第三方服务A" as thirdA
db "第三方服务B" as thirdB
rectangle  "离线业务A" as offlineBusinessA 
rectangle "缓存服务" as cache #orange
rectangle "索引服务" as index #orange
onlineBusinessA <-down-> dataCenter
onlineBusinessB <-down->dataCenter
dataCenter <-down->mysql
dataCenter <-down->thirdA
dataCenter <-down->thirdB
offlineBusinessA <-down-> dataCenter
cache <-left-> dataCenter
index <-right-> dataCenter
@enduml
```

## 时序图

```plantuml
@startuml
participant business as "在线业务服务"
collections dataCenter as "数据中台"
control auth as "鉴权服务" 
database cache as "缓存服务"
database index as "检索服务"
database db as "数据库/第三方接口"
queue mq as "消息队列"
participant backBusiness as "离线业务服务" 

business -> dataCenter : 变更数据(新增、修改、删除)
dataCenter->auth:验证权限
loop 1-N次
dataCenter->dataCenter: 校订数据\n二次加工(简单类型)
dataCenter -> db : 存储数据
dataCenter -> cache : 缓存处理
dataCenter -> index : 检索处理
dataCenter->mq: 发布数据变更消息
mq->backBusiness: 数据二次加工生产新数据(复杂类型)
backBusiness-->dataCenter: 变更新数据
end
dataCenter --> business : 反馈处理结果
====
business -> dataCenter: 查询数据
dataCenter->auth:验证权限
loop 1-N次
dataCenter -> dataCenter: 校验、补充请求数据
dataCenter -> cache: 查询缓存
dataCenter -> index: 检索数据
dataCenter -> db : 查询数据
dataCenter -> dataCenter: 校验、补充返回数据
end
dataCenter -->business: 返回数据
@enduml
```

## 仓库uml
```plantuml
@startuml
 class LineschemaMeta {
  Lineschema string
  JsonSchema string
  Tpl string
  DefaultJson string
  SchemaLoader *gojsonschema.JSONLoader
}
 class TemplateMeta {
  Name string
  ExecProvider ExecproviderInterface
  LineschemaMeta *LineschemaMeta
}
 class repository {
  template *template.Template
  metaMap map[string]*TemplateMeta
}
 interface VolumeInterface  {
  SetValue(key string,value interface  { })
  GetValue(key string,value interface  { })ok bool
}
 interface ExecproviderInterface  {
  Exec(identifier string,s string)(string,error)
  GetSource()source interface  { }
}
 interface RepositoryInterface  {
  AddTemplateByDir(dir string)addTplNames []string
  AddTemplateByFS(fsys fs.FS,root string)addTplNames []string
  AddTemplateByStr(name string,s string)addTplNames []string
  GetTemplate()*template.Template
  ExecuteTemplate(name string,volume VolumeInterface)(string,error)
  TemplateExists(name string)bool
  RegisterMeta(tplName string,meta *TemplateMeta)
  GetMeta(tplName string)(*TemplateMeta,bool)
}
.RepositoryInterface <|- .repository
@enduml
```

## 执行器uml
```plantuml
@startuml


 class RequestData {
  URL string
  Method string
  Header http.Header
  Cookies []*http.Cookie
  Body string
}

 class ResponseData {
  HttpStatus string
  Header http.Header
  Cookies []*http.Cookie
  Body string
  RequestData *RequestData
}

 class CURLExecProviderConfig {
  Proxy string
  LogLevel string
  Timeout int
  KeepAlive int
  MaxIdleConns int
  MaxIdleConnsPerHost int
  IdleConnTimeout int
}

 class CURLExecProvider {
  Config CURLExecProviderConfig
  client *http.Client
  clinetOnce sync.Once
}

 class DBExecProviderConfig {
  DSN string
  LogLevel string
  Timeout int
}

 class DBExecProvider {
  Config DBExecProviderConfig
  db *sql.DB
  dbOnce sync.Once
}


 interface ExecproviderInterface  {
  Exec(identifier string,s string)(string,error)
  GetSource()source interface  { }
}


.ExecproviderInterface <|- .CURLExecProvider
.ExecproviderInterface <|- .DBExecProvider
@enduml
```
## 软件执行流程图
```plantuml
@startuml

start
  :接收输入(input);
  if(鉴权) then(通过)
  if(模板未加载) then(是)
   :加载模板;
 endif
:格式化输入;
if (校验输入) equals(通过)then
:初始化容器(volume)数据;
:调用API模板主函数;
repeat: 执行内嵌模板/远程调用;
if(发生错误) then(抛出错误)
break
endif
repeat while (更多模板?)
else(不通过)
endif
endif
:格式化输出;
:输出(output);
stop
@enduml

```