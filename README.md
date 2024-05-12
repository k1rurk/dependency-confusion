## Before usage
1) Register DNS for data exfiltration. Setup ns1 and ns2 then write your domain with ip into the config
2) Run docker for client SDK
3) Add data to:
* pip .pypirc in internal/upload_registry/pip/sources/.pypirc
* npm .pypirc in internal/upload_registry/npm/sources/.npmrc
* config.toml in config/. Values for keys like urlscan, otx, git, scrapeops, dns
