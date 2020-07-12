# deploy-lite
deployment api application with yaml.


CI tool is too heavy for home use.
This tool is for deployment stuff for home built server.

Acutally without all crazy hooks, this simple deployment tool are 3 part
 - pickup a package
 - run a script 
 - report logs to front-end
 
This tool's implemenation:
(1) it reads yaml deployment script from api (POST), script supports deploying multiple packages.
(2) assuming package is always 'tar' compressed, it will decompress then run a script.
The POST yaml deployment script only provides package names and script names to work on. But combining scp, whole thing can be automated.
(3) it will save whole built terminal environment into log, that can be pulled by api to see the status or history.

Feature: 
- Simple Token Authentication

Anti Feature:
- no github or any git repo integration
- no account
- not necessily do new build on server side (though can do build in the script). Why do I always need so for javascript project? would mac make difference from linux?


Usage
- 1. use cURL to trigger a deployment
(before triggering the deployment, scp the deploy-pkg.tar to a user folder e.g. /home/user/deploy-lite-pkgs/)

- 2. 
```
curl -v localhost:8080/api/v1/deploy -H "Content-Type:text/x-yaml" --data-binary @deploy.yaml 
```
( to reserve newline for yaml, use --data-binary)
The content of yaml
```
namespace: prod
authToken: some-secret-token
deploy:
  back-end-app:
    script: deploy-backend.sh
    package: backend-api.tar
    skip: true
  front-end-app:
    script: deploy-ui.sh
    package: frequency-agenda.tar
    skip: false
```
The return is the %namespace%-%sessionid% -- need this to get the status of the deployment.
- 3 namespace and status api

Choose a namespace e.g. 'prod', so you can get all deployment logs using 
```
curl localhost:8080/api/v1/status/prod
```
this will list all log names with timestamp, ordered by descendent of timestemp
like
```
prod-1234132543123213                      2020-07-15 14:23  UTC    
```
the number followed by 'prod-' is also returned by step 2's sessionID

get status of a deployment

```
curl localhost:8080/api/v1/status/prod-1234132543123213
```

- 4 the sessionID
it is the unix time in nano sec, for each namespace, this is likely unique enough and sequencial.

- 5 feature added to prevent double submit
Within two minutes, you can't submit the same yaml for the same namespace. Since script is running asynchronously, this is not 100% safe way to prevent. But for hobbies home server, this is good enough to prevent double trigger.

=========================

# deploy-lite
用yaml设置文件的部署工具

CI 工具用于个人服务器乃杀鸡之牛刀。此物小巧精悍。

无各类花哨回调，仅含三组件
- 部署包拾取
- 部署脚本运行
- 报告状态

实现
（1） 用POST接收yaml配置，支持部署多包
（2） 解压'tar'包，
 (3) 以部署指定脚本执行任意部署步骤，记录步骤结果入log文件，可用api读取。

功能
- 简单验证


非功能
- 没有github或bitbucket集成的必要
- 无需账户
- 无需编译步骤（如发布react/angular, 不必行此脑残一举， mac/Linux有何不同？

用法
- 1. cURL开启发布
（发布前，scp deploy-pkg.tar 至用户权限文件夹，如/home/user/deploy-lite-pkgs

- 2. 
```
curl -v localhost:8080/api/v1/deploy -H "Content-Type:text/x-yaml" --data-binary @deploy.yaml 
```
(yaml, 用 --data-binary 保留换行)

内详
```
namespace: prod
authToken: some-secret-token
deploy:
  back-end-app:
    script: deploy-backend.sh
    package: backend-api.tar
    skip: true
  front-end-app:
    script: deploy-ui.sh
    package: frequency-agenda.tar
    skip: false
```
回应为%namespace%-%sessionid% -- 以此调用api得log
The return is the %namespace%-%sessionid% -- need this to get the status of the deployment.
- 3 namespace 空间名 and status 日志 api

选空间名如'prod', 以此列出历史各日志 
```
curl localhost:8080/api/v1/status/prod
```
逆时列出：
```
prod-1234132543123213                      2020-07-15 14:23  UTC    
```
加入日志号，以得日志详细：
'prod-' + ‘1234132543123213’

```
curl localhost:8080/api/v1/status/prod-1234132543123213
```

- 4 日志号
为unix纳秒, 对于各个空间足够唯一足够顺序.

- 5 已防重复递交
两分钟内，以yaml哈希为比较，拒绝二次递交部署。因脚本异步性，完成时刻不易确定，但个人服务器，足以。




