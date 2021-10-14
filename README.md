## Introduce

scrape_configs for metrics server that register in etcd.


### Example 

```shell
make docker
```

```shell
docker run sd_etcd:1.0 /dist/sd_etcd -server 192.168.128.1:2379 -target-file /scrape/sd_etcd.json
```

with prometheus
```
version: "3.5"
services:
  prometheus:
    container_name: prometheus
    image: prom/prometheus
    ports:
      - 9090:9090
    volumes:
    - /etc/monitoring.d/prometheus.yaml:/etc/prometheus/prometheus.yml
    - /etc/monitoring.d/scrape:/etc/prometheus/scrape
    networks:
      - monitoring
    restart: always
  grafana:
    container_name: grafana
    image: grafana/grafana
    ports:
      - 3000:3000
    volumes:
    - /data/monitoring/grafana:/var/lib/grafana
    networks:
      - monitoring
    restart: always
  sd_etcd:
    container_name: sd_etcd 
    image: sd_etcd:1.0
    command: /dist/sd_etcd -server 192.168.128.1:2379 -target-file /scrape/sd_etcd.json
    volumes:
    - /etc/monitoring.d/scrape:/scrape/
    networks:
      - monitoring
    restart: always
networks:
  monitoring:
    driver: bridge
```