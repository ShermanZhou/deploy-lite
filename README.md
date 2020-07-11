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


TODO:
- implement the status api
