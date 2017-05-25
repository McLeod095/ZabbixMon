# Nginx VTS monitoring for Zabbix

# Usage

## build nginx 
with module [nginx-module-vts](https://github.com/vozlt/nginx-module-vts)

## Edit nginx.conf 
```
http {
 	vhost_traffic_status_zone;
 	...;
	server {
        listen *:8899;
        server_name stub.example.com;

        location / {q
                stub_status on;
                access_log off;
        }
        location /status {
                vhost_traffic_status_display;
                vhost_traffic_status_display_format html;
                access_log off;
        }
    }
}
```

## Build app

For Linux
GOOS=linux GOARCH=amd64 go build -ldflags "-s -w"
