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


